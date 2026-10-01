package main

import (
	"fmt"
	"sk_vnc/screen_capture"
	"time"
)

/*
#include <math.h>
*/
import "C"

func main() {
	captureStart := time.Now()
	screen_capture.CaptureScreen()
	fmt.Printf("CaptureScreen took %v\n", time.Since(captureStart))
}
