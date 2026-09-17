package mac_capture

/*
#include <stdint.h>
*/
// import "C"
// import "unsafe"
// import "fmt"

// The preamble of a file containing //export is copied into two generated C
// files, so it must hold declarations only -- we can't include the
// ScreenCaptureKit headers here. Take the buffer as an opaque pointer and let
// screen_capture_kit.go, which does have CoreMedia in scope, cast it back.
//
