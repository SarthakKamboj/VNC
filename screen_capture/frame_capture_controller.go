package screen_capture

import (
	"image"
	"sk_vnc/mac_capture"
)

type FrameCaptureController struct {
	frameRate uint
	maxFrames uint

	curFrameCount uint

	frames             []*image.Paletted
	frameCaptureSource FrameCaptureSource
}

func (frameCaptureController *FrameCaptureController) Init(frameRate uint, maxFrames uint) {
	frameCaptureController.frameRate = frameRate
	frameCaptureController.maxFrames = maxFrames
	frameCaptureController.frames = make([]*image.Paletted, 0, maxFrames)
	frameCaptureController.curFrameCount = 0
	frameCaptureController.frameCaptureSource = &mac_capture.MacFrameCaptureSource{}
}

func (frameCaptureController *FrameCaptureController) ImageCallback(frameImage *image.Paletted) {
	frameCaptureController.curFrameCount++
	frameCaptureController.frames = append(frameCaptureController.frames, frameImage)
	if frameCaptureController.curFrameCount == frameCaptureController.maxFrames {
		frameCaptureController.frameCaptureSource.StopCapture()
	}
}

func (frameCaptureController *FrameCaptureController) Capture() {
	frameCaptureController.frameCaptureSource.Init(frameCaptureController.ImageCallback, frameCaptureController.frameRate)
	frameCaptureController.frameCaptureSource.StartCapture()
}

func (frameCaptureController *FrameCaptureController) GetFrames() []*image.Paletted {
	return frameCaptureController.frames
}
