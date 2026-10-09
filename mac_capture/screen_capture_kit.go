package mac_capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=12.3
#cgo LDFLAGS: -framework ScreenCaptureKit -framework Foundation -framework CoreMedia -framework CoreVideo -framework CoreGraphics -framework IOSurface

#include "screen_capture_kit.h"

#include <math.h>
*/
import "C"

import (
	"image"
	"sync"
	"time"
	"unsafe"
)

type MacFrame struct {
	image     *image.RGBA
	timestamp time.Time
}

var captureChannel *chan MacFrame = nil

// var totalFrames int = 0

// Frames GoTransformFrame has sent on; it stops sending once this passes totalFrames.
// var framesTransformed int = 0

func StartCapture(_captureChannel *chan MacFrame /*, totalFramesIn int*/) {
	captureChannel = _captureChannel
	// totalFrames = totalFramesIn
	C.start_capture()
}

func StopCapture() {
	captureChannel = nil
	C.stop_capture()
	// framesTransformed = 0
}

func swivelSCKBuffer(sckImageData []C.pixel_t, goImageData *image.RGBA, frame *C.frame_t, startX int, startY int, width int, height int, wg *sync.WaitGroup) {

	defer wg.Done()

	for y := startY; y < startY+height; y++ {
		for x := startX; x < startX+width; x++ {
			var indexIntoBuffer int = (int(frame.width) * y) + x
			var pixel C.pixel_t = sckImageData[indexIntoBuffer]
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
	// if framesTransformed > totalFrames {
	// 	return
	// }

	var topLeft image.Point = image.Point{0, 0}

	var isEmpty bool = (frame.width == 0) || (frame.height == 0)

	var bottomRight image.Point = image.Point{int(frame.width) - 1, int(frame.height) - 1}
	if isEmpty {
		bottomRight = image.Point{0, 0}
	}

	var screenImage *image.RGBA = image.NewRGBA(image.Rectangle{topLeft, bottomRight})

	var numPixels int = int(frame.height) * int(frame.width)
	framePixelData := unsafe.Slice(frame.pixel_data, numPixels)

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

	if captureChannel != nil {
		macFrame := MacFrame{screenImage, time.Now()}
		*captureChannel <- macFrame
	}

	// framesTransformed++
}
