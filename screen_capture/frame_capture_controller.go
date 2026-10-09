package screen_capture

import (
	"fmt"
	"image"
	"sk_vnc/mac_capture"
)

type FrameCaptureController struct {
	frameRate uint
	maxFrames uint

	curFrameCount uint

	frameCollector     FrameCollector
	frameCaptureSource FrameCaptureSource
}

func (frameCaptureController *FrameCaptureController) Init(frameRate uint, maxFrames uint) {
	frameCaptureController.frameRate = frameRate
	frameCaptureController.maxFrames = maxFrames
	frameCaptureController.curFrameCount = 0
	frameCaptureController.frameCaptureSource = &mac_capture.MacFrameCaptureSource{}
}

func (frameCaptureController *FrameCaptureController) ImageCallback(frameImage *image.Paletted) {
	frameCaptureController.curFrameCount++
	fmt.Printf("Received frame %d\n", frameCaptureController.curFrameCount)
	if frameCaptureController.curFrameCount == frameCaptureController.maxFrames {
		frameCaptureController.frameCaptureSource.StopCapture()
	}
}

func (frameCaptureController *FrameCaptureController) Capture() {
	frameCaptureController.frameCaptureSource.Init(frameCaptureController.ImageCallback, frameCaptureController.frameRate)
	frameCaptureController.frameCaptureSource.StartCapture()
}
