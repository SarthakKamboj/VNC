package mac_capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=12.3
#cgo LDFLAGS: -framework ScreenCaptureKit -framework Foundation -framework CoreMedia -framework CoreVideo -framework CoreGraphics -framework IOSurface

#include "screen_capture_kit.h"

#include <math.h>
*/
import "C"

import (
	"fmt"
	"image"
	"sync"
	"time"
	"unsafe"
)

var captureChannel *chan *image.RGBA = nil
var totalFrames int = 0

// GoTransformFrame runs on ScreenCaptureKit's queue while StopCapture reads
// these from another goroutine, so they are guarded by a mutex.
type transformStats struct {
	mu                                   sync.Mutex
	frames                               int
	allocSum, convertSum, sendSum, total time.Duration
}

var goStats transformStats

func StartCapture(_captureChannel *chan *image.RGBA, totalFramesIn int) {
	captureChannel = _captureChannel
	totalFrames = totalFramesIn
	C.start_capture()
}

func StopCapture() {
	captureChannel = nil
	C.stop_capture()
	printTransformAverages()
}

func printTransformAverages() {
	goStats.mu.Lock()
	defer goStats.mu.Unlock()

	if goStats.frames == 0 {
		return
	}
	n := float64(goStats.frames)
	// A high send time means RecordGif is falling behind and frames are backing up.
	fmt.Printf("      GoTransformFrame = convert %.2f ms + alloc %.2f ms + send to RecordGif %.2f ms\n",
		ms(goStats.convertSum)/n, ms(goStats.allocSum)/n, ms(goStats.sendSum)/n)

	// Start the next capture's averages from zero.
	goStats.frames = 0
	goStats.allocSum, goStats.convertSum, goStats.sendSum, goStats.total = 0, 0, 0, 0
}

func swivelSCKBuffer(sckImageData []C.pixel_t, goImageData *image.RGBA, frame *C.frame_t, startX int, startY int, width int, height int, wg *sync.WaitGroup) {

	defer wg.Done()

	for y := startY; y < startY+height; y++ {
		for x := startX; x < startX+width; x++ {
			var indexIntoBuffer int = (int(frame.width) * y) + x
			var pixel C.pixel_t = sckImageData[indexIntoBuffer]
			// var color color.RGBA = color.RGBA{uint8(pixel.r), uint8(pixel.g), uint8(pixel.b), uint8(pixel.a)}
			// fmt.Printf("color is %v %v %v %v\n", uint8(pixel.r), uint8(pixel.g), uint8(pixel.b), uint8(pixel.a))
			// goImageData.Set(x, y, color)
			i := goImageData.PixOffset(x, y)
			if i < len(goImageData.Pix) {
				goImageData.Pix[i+0] = uint8(pixel.r)
			}
			if i+1 < len(goImageData.Pix) {
				goImageData.Pix[i+1] = uint8(pixel.g)
			}
			if i+2 < len(goImageData.Pix) {
				goImageData.Pix[i+2] = uint8(pixel.b)
			}
			if i+3 < len(goImageData.Pix) {
				goImageData.Pix[i+3] = uint8(pixel.a)
			}
		}
	}
}

//export GoTransformFrame
func GoTransformFrame(frame C.frame_t) {
	start := time.Now()

	if goStats.frames > totalFrames {
		return
	}

	var topLeft image.Point = image.Point{0, 0}

	var isEmpty bool = (frame.width == 0) || (frame.height == 0)

	var bottomRight image.Point = image.Point{int(frame.width) - 1, int(frame.height) - 1}
	if isEmpty {
		bottomRight = image.Point{0, 0}
	}

	var screenImage *image.RGBA = image.NewRGBA(image.Rectangle{topLeft, bottomRight})

	var numPixels int = int(frame.height) * int(frame.width)
	framePixelData := unsafe.Slice(frame.pixel_data, numPixels)

	allocDone := time.Now()

	if !isEmpty {
		var wg sync.WaitGroup

		var xPartitions int = 4
		var yPartitions int = 2

		for x := 0; x < xPartitions; x++ {
			for y := 0; y < yPartitions; y++ {
				width := int(frame.width) / xPartitions
				height := int(frame.height) / yPartitions
				startX := x * width
				startY := y * height

				wg.Add(1)
				go swivelSCKBuffer(framePixelData, screenImage, &frame, startX, startY, width, height, &wg)
			}
		}

		wg.Wait()
	}

	convertDone := time.Now()

	// A full channel means recordGif is the bottleneck, and this send is what
	// stalls the ScreenCaptureKit delivery queue -- so time it on its own.
	if captureChannel != nil {
		*captureChannel <- screenImage
	}

	sendDone := time.Now()

	goStats.mu.Lock()
	goStats.frames++
	goStats.allocSum += allocDone.Sub(start)
	goStats.convertSum += convertDone.Sub(allocDone)
	goStats.sendSum += sendDone.Sub(convertDone)
	goStats.total += sendDone.Sub(start)
	goStats.mu.Unlock()
}

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000000.0
}
