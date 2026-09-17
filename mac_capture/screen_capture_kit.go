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

func StartCapture() {
	C.start_capture()
}

//export goHandleFrame
func goHandleFrame(frame C.frame_t) {
	// fmt.Println("Received a frame");
	fmt.Printf("Frame is %d by %d and %d planes\n", int(frame.width), int(frame.height), int(frame.num_planes));
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
