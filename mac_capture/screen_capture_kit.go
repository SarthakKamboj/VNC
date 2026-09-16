package mac_capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=12.3
#cgo LDFLAGS: -framework ScreenCaptureKit -framework Foundation -framework CoreMedia

#include "screen_capture_kit.h"

#include <math.h>
*/
import "C"
// import "unsafe"

func StartCapture() {
	C.start_capture()
}

// func CreateScStreamConfiguration() unsafe.Pointer {
// 	return C.create_sc_stream_configuration()	
// }
