package main

import "sk_vnc/screen_capture"

/*
#include <math.h>
*/
import "C"

func main() {
	screen_capture.CaptureScreen()
}
