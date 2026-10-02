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

	for j := 0; j < 1; j++ {
		const runs = 1
		var total time.Duration
		for i := 0; i < runs; i++ {
			captureStart := time.Now()

			if j == 0 {
				screen_capture.NotExist = false
				screen_capture.UseOctree = true
			} else if j == 1 {
				screen_capture.NotExist = true
				screen_capture.UseOctree = false
			} else if j == 2 {
				screen_capture.NotExist = false
				screen_capture.UseOctree = false
			} else if j == 3 {
				screen_capture.NotExist = true
				screen_capture.UseOctree = true
			}

			screen_capture.CaptureScreen()
			took := time.Since(captureStart)
			total += took
			fmt.Printf("CaptureScreen [NotExist: %t  UseOctree: %t] run %d/%d took %v\n", screen_capture.NotExist, screen_capture.UseOctree, i+1, runs, took)
		}
		// fmt.Printf("\nCaptureScreen [NotExist: %t  UseOctree: %t] average over %d runs: %v\n", screen_capture.NotExist, screen_capture.UseOctree, runs, total/runs)
	}
}
