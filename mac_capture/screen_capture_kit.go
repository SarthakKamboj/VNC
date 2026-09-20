package mac_capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=12.3
#cgo LDFLAGS: -framework ScreenCaptureKit -framework Foundation -framework CoreMedia -framework CoreVideo -framework CoreGraphics -framework IOSurface

#include "screen_capture_kit.h"

#include <math.h>
*/
import "C"

import "fmt"
import "os"
import "image"
import "image/gif"
// import "image/png"
import "image/color"
import "image/color/palette"
import "math"
import "unsafe"

var rawImageFrames = make(chan *image.RGBA, 3)

func recordGif(maxFrames int) {
	// gif should be 60fps probably
	var recordingGif *gif.GIF = &gif.GIF{}
	// recordingGif.Image = make([]*image.Paletted, 0, maxFrames)
	// recordingGif.Delay = make([]int, 0, maxFrames)

	for rawFrame := range rawImageFrames {

		fmt.Println("Received frame")

		if (len(recordingGif.Image) > maxFrames) {
			break
		}

		var imageBounds image.Rectangle = rawFrame.Bounds()
		// var pallete color.Palette = color.Palette{}

		// for row := imageBounds.Min.Y; row <= imageBounds.Max.Y; row++ {
		// 	for col := imageBounds.Min.X; col <= imageBounds.Max.X; col++ {
		// 		var imageColor color.Color = rawFrame.At(col, row)
		// 		pallete = append(pallete, imageColor)
		// 	}
		// }

		var palletedImage *image.Paletted = image.NewPaletted(imageBounds, palette.Plan9)
		for row := imageBounds.Min.Y; row <= imageBounds.Max.Y; row++ {
			for col := imageBounds.Min.X; col <= imageBounds.Max.X; col++ {
				palletedImage.Set(col, row, rawFrame.At(col, row))
			}
		}

		recordingGif.Image = append(recordingGif.Image, palletedImage)

		// var delay float = len(recordingGif.Image) * 1.0 / 60.0 * 100.0
		var delay int = int(math.Floor(0.5 + (1.0 / 60.0 * 100.0)))
		recordingGif.Delay = append(recordingGif.Delay, delay)

		fmt.Println("append frame at index %i", len(recordingGif.Delay) - 1)
	}

	gifFile, _ := os.Create("recording.gif")
	err := gif.EncodeAll(gifFile, recordingGif)
	if err != nil {
		fmt.Println("error has occured while trying to create gif")
	}
}

func StartCapture() {
	C.start_capture()
	go recordGif(30)
}

//export goHandleFrame
func goHandleFrame(frame C.frame_t) {
	fmt.Printf("Frame is %d by %d and %d planes\n", int(frame.width), int(frame.height), int(frame.num_planes));

	var topLeft image.Point = image.Point{0,0};
	var bottomRight image.Point = image.Point{int(frame.width)-1, int(frame.height)-1};

	var screenImage *image.RGBA = image.NewRGBA(image.Rectangle{topLeft, bottomRight})

	var numPixels int = int(frame.height) * int(frame.width)
	framePixelData := unsafe.Slice(frame.pixel_data, numPixels)

	for y := 0; y < int(frame.height); y++ {
		for x := 0; x < int(frame.width); x++ {
			var indexIntoBuffer int = (int(frame.width) * y) + x
			var pixel C.pixel_t = framePixelData[indexIntoBuffer]
			var color color.RGBA = color.RGBA{uint8(pixel.r), uint8(pixel.g), uint8(pixel.b), uint8(pixel.a)}
			screenImage.Set(x, y, color)
		}
	}

	fmt.Println("appending to rawImageFrames")
	rawImageFrames <- screenImage

	// _, err := os.Stat("image.png")
	// if err != nil {
	// 	f, _ := os.Create("image.png")
	// 	png.Encode(f, screenImage)
	// 	f.Close()
	// }

}