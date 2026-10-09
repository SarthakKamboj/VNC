package mac_capture

import (
	"image"
	"image/color"
	"image/color/palette"
	"math"
	"sync"
	"time"
)

type MacFrameCaptureSource struct {
	imageCallback func(*image.Paletted)
	frameRate     uint

	stopped bool

	macFrames chan MacFrame
}

func (macFrameCaptureSource *MacFrameCaptureSource) Init(imageCallback func(*image.Paletted), frameRate uint) {
	macFrameCaptureSource.frameRate = frameRate
	macFrameCaptureSource.imageCallback = imageCallback
	macFrameCaptureSource.stopped = true
	macFrameCaptureSource.macFrames = make(chan MacFrame, 10)
}

func (macFrameCaptureSource *MacFrameCaptureSource) StartCapture() {
	macFrameCaptureSource.stopped = false
	StartCapture(&macFrameCaptureSource.macFrames)
	macFrameCaptureSource.listenToFrames()
}

func (macFrameCaptureSource *MacFrameCaptureSource) SetPalettedImage(startX int, startY int, width int, height int, rawFrame *image.RGBA, palettedImage *image.Paletted, skPalette *SkPalette, wg *sync.WaitGroup, mu *sync.Mutex, cache *map[color.Color]uint8) {
	defer wg.Done()

	for row := startY; row < startY+height; row++ {
		for col := startX; col < startX+width; col++ {
			c := rawFrame.At(col, row)
			var index uint8 = 0
			val, exists := (*cache)[c]
			if exists {
				index = val
			} else {
				index, _ = skPalette.FindClosestIndex(c)
				(*cache)[c] = index
			}
			palettedImage.SetColorIndex(col, row, index)
		}
	}
}

func (macFrameCaptureSource *MacFrameCaptureSource) ticksBetweenTimes(time1 time.Time, time2 time.Time) int {
	msBetween := time1.UnixMilli() - time2.UnixMilli()
	expectedMsBetweenTicks := 1000.0 / float32(macFrameCaptureSource.frameRate)
	ticks := float32(msBetween) / expectedMsBetweenTicks
	return int(math.Round(float64(ticks)))
}

func (macFrameCaptureSource *MacFrameCaptureSource) listenToFrames() {
	const xPartitions int = 4
	const yPartitions int = 4
	const totalPartitions int = xPartitions * yPartitions

	var prevTimeStamp time.Time
	var cachedPalettedImage *image.Paletted = nil

	var skPalette SkPalette = SkPalette{}
	skPalette.Init(palette.Plan9)

	var pixelClosestMaps [totalPartitions]map[color.Color]uint8 = [totalPartitions]map[color.Color]uint8{}
	for i := 0; i < totalPartitions; i++ {
		pixelClosestMaps[i] = make(map[color.Color]uint8)
	}

	for !macFrameCaptureSource.stopped {
		macFrame, ok := <-macFrameCaptureSource.macFrames

		if !ok {
			break
		}

		ticks := macFrameCaptureSource.ticksBetweenTimes(macFrame.timestamp, prevTimeStamp)
		if prevTimeStamp.IsZero() {
			ticks = 0
		}

		if ticks > 1 && cachedPalettedImage != nil {
			for ticks > 1 {
				macFrameCaptureSource.imageCallback(cachedPalettedImage)
				ticks--
			}
		}

		var imageBounds image.Rectangle = macFrame.image.Bounds()

		var isEmpty = imageBounds.Max.X == 0 || imageBounds.Max.Y == 0
		if isEmpty {
			if cachedPalettedImage != nil {
				macFrameCaptureSource.imageCallback(cachedPalettedImage)
				prevTimeStamp = macFrame.timestamp
			}
			continue
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var palettedImage *image.Paletted = image.NewPaletted(imageBounds, palette.Plan9)

		for x := 0; x < xPartitions; x++ {
			for y := 0; y < yPartitions; y++ {
				width := imageBounds.Dx() / xPartitions
				height := imageBounds.Dy() / yPartitions
				startX := x * width
				startY := y * height

				idx := (y * xPartitions) + x

				wg.Add(1)
				go macFrameCaptureSource.SetPalettedImage(startX, startY, width, height, macFrame.image, palettedImage, &skPalette, &wg, &mu, &pixelClosestMaps[idx])
			}
		}

		wg.Wait()

		macFrameCaptureSource.imageCallback(palettedImage)

		cachedPalettedImage = palettedImage
		prevTimeStamp = macFrame.timestamp
	}
}

func (macFrameCaptureSource *MacFrameCaptureSource) StopCapture() {
	StopCapture()
	macFrameCaptureSource.stopped = true
}
