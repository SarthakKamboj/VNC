package screen_capture

import "sk_vnc/mac_capture"

// type ScreenPixel struct {
// 	float R
// 	float G
// 	float B
// 	float A
// }

// type ScreenFrame struct {
// 	int Width
// 	int Height
// }

func CaptureScreen() {
	mac_capture.StartCapture()
	// mac_capture.CreateScStreamConfiguration()
}