package mac_capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=12.3
#cgo LDFLAGS: -framework ScreenCaptureKit -framework Foundation -framework CoreMedia -framework CoreVideo -framework CoreGraphics -framework IOSurface

#include "screen_capture_kit.h"

#include <math.h>
*/
import "C"
// import "unsafe"
import "fmt"
import "os"
import "image"
import "image/png"
import "image/color"
import "unsafe"

func StartCapture() {
	C.start_capture()
}

//export goHandleFrame
func goHandleFrame(frame C.frame_t) {
	// fmt.Println("Received a frame");
	fmt.Printf("Frame is %d by %d and %d planes\n", int(frame.width), int(frame.height), int(frame.num_planes));

	var topLeft image.Point = image.Point{0,0};
	var bottomRight image.Point = image.Point{int(frame.width), int(frame.height)};

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

	f, _ := os.Create("image.png")
	png.Encode(f, screenImage)
	f.Close()

	// ProcessCMSampleBuffer(sampleBuffer)
}

// func goProcessFrame(frame C) {

// }

// func ProcessCMSampleBuffer(sampleBufferPtr unsafe.Pointer) {
	// var sampleBufferRef C.CMSampleBufferRef = C.CMSampleBufferRef(sampleBufferPtr)
	// var bufferReady C.Boolean = C.CMSampleBufferDataIsReady(sampleBufferRef)
	// if (bufferReady != 0) {
	// 	fmt.Println("Buffer is ready");
	// } else {
	// 	fmt.Println("Buffer is not ready");
	// }

	// imageBufferRef := C.CMSampleBufferGetImageBuffer(sampleBufferRef)
	// pixelBufferRef := C.CVPixelBufferRef(unsafe.Pointer(imageBufferRef))
	// ioSurfaceRef := C.CVPixelBufferGetIOSurface(pixelBufferRef)
	// ioSurfaceWidth := C.IOSurfaceGetWidth(ioSurfaceRef)
	// ioSurfaceHeight := C.IOSurfaceGetHeight(ioSurfaceRef)
	// fmt.Println("Buffer is %zu by %zu", ioSurfaceWidth, ioSurfaceHeight)
// }

// func CreateScStreamConfiguration() unsafe.Pointer {
// 	return C.create_sc_stream_configuration()	
// }
