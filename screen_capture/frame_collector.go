package screen_capture

import (
	"image"
	"math"
)

type FrameCollector struct {
	FrameRate uint

	frames             []*image.Paletted
	prevTimeStampInSec float32
}

func (frameCollector *FrameCollector) Init(frameRate uint, maxFrames uint) {
	frameCollector.FrameRate = frameRate

	frameCollector.frames = make([]*image.Paletted, 0, maxFrames)
	frameCollector.prevTimeStampInSec = 0
}

func (frameCollector *FrameCollector) ReceiveFrame(newImage *image.Paletted, timestampInSec float32) {
	expectedTimeBetweenFrames := 1.0 / float32(frameCollector.FrameRate)

	ticksSinceLastFrame := (timestampInSec - frameCollector.prevTimeStampInSec) / expectedTimeBetweenFrames

	if frameCollector.prevTimeStampInSec != 0 && ticksSinceLastFrame > 1.5 {
		// we will need to back copy the frames
		ceiledTicks := math.Ceil(float64(ticksSinceLastFrame - 1))
		frameCollector.backCopyFrames(uint(ceiledTicks - 1))
	}

	frameCollector.frames = append(frameCollector.frames, newImage)
	frameCollector.prevTimeStampInSec = timestampInSec
}

func (frameCollector *FrameCollector) backCopyFrames(numBackCopies uint) {
	for range numBackCopies {
		framesLength := len(frameCollector.frames)
		frameCollector.frames = append(frameCollector.frames, frameCollector.frames[framesLength-1])
	}
}
