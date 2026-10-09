package screen_capture

import (
	"image"
)

type FrameCaptureSource interface {
	Init(imageCallback func(*image.Paletted), frameRate uint)
	StartCapture()
	StopCapture()
}
