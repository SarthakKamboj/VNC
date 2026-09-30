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

func StartCapture(_captureChannel *chan *image.RGBA) {
	captureChannel = _captureChannel
	C.start_capture()
}

func StopCapture() {
	captureChannel = nil
	C.stop_capture()
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

	var topLeft image.Point = image.Point{0, 0}
	var bottomRight image.Point = image.Point{int(frame.width) - 1, int(frame.height) - 1}

	var screenImage *image.RGBA = image.NewRGBA(image.Rectangle{topLeft, bottomRight})

	var numPixels int = int(frame.height) * int(frame.width)
	framePixelData := unsafe.Slice(frame.pixel_data, numPixels)

	allocDone := time.Now()

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

	convertDone := time.Now()

	// A full channel means recordGif is the bottleneck, and this send is what
	// stalls the ScreenCaptureKit delivery queue -- so time it on its own.
	if captureChannel != nil {
		*captureChannel <- screenImage
	}
	queueLen := len(*captureChannel)

	sendDone := time.Now()
	fmt.Printf("[go]   alloc %.2f | convert %.2f | chan send %.2f (queue %d/%d) | total %.2f ms\n",
		ms(allocDone.Sub(start)),
		ms(convertDone.Sub(allocDone)),
		ms(sendDone.Sub(convertDone)),
		queueLen, cap(*captureChannel),
		ms(sendDone.Sub(start)))
}

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000000.0
}
