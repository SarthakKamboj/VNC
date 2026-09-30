package screen_capture

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"math"
	"os"
	"sk_vnc/mac_capture"
	"sync"
	"time"
)

/*
	This file should be the platform abstraction layer for capturing screens
*/

var rawImageFrames = make(chan *image.RGBA, 1)

func CaptureScreen() {
	var wg sync.WaitGroup

	mac_capture.StartCapture(&rawImageFrames)

	wg.Add(1)
	go RecordGif(1000, &wg)
	wg.Wait()

	StopCapture()
}

func StopCapture() {
	mac_capture.StopCapture()
	for len(rawImageFrames) > 0 {
		<-rawImageFrames
	}
}

func SetPalettedImage(startX int, startY int, width int, height int, rawFrame *image.RGBA, palettedImage *image.Paletted, skPalette *SkPalette, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()

	cache := make(map[color.Color]uint8)

	for row := startY; row < startY+height; row++ {
		for col := startX; col < startX+width; col++ {
			c := rawFrame.At(col, row)
			var index uint8 = 0
			val, exists := cache[c]
			if exists {
				index = val
			} else {
				// need to see if this is faster or not compared to just naive eucledian checks
				index = skPalette.FindClosestIndex(c)
				cache[c] = index
			}
			palettedImage.SetColorIndex(col, row, index)
		}
	}
}

func RecordGif(maxFrames int, wg *sync.WaitGroup) {

	defer wg.Done()

	// gif should be 60fps probably
	var recordingGif *gif.GIF = &gif.GIF{}

	var lastFrameDone time.Time

	var skPalette SkPalette = SkPalette{}
	skPalette.Init(palette.Plan9)

	for rawFrame := range rawImageFrames {

		frameStart := time.Now()
		var idle time.Duration
		if !lastFrameDone.IsZero() {
			idle = frameStart.Sub(lastFrameDone)
		}

		if len(recordingGif.Image) > maxFrames {
			break
		}

		var imageBounds image.Rectangle = rawFrame.Bounds()
		var palletedImage *image.Paletted = image.NewPaletted(imageBounds, palette.Plan9)

		newPalettedDone := time.Now()

		// TODO: need to make this faster
		/*
			Use an octree to find the pixel index
		*/
		numPixels := imageBounds.Dx() * imageBounds.Dy()

		var wg sync.WaitGroup
		var mu sync.Mutex

		// TODO: see why adding more threads is not making this faster but rather slower
		var xPartitions int = 4
		var yPartitions int = 4

		for x := 0; x < xPartitions; x++ {
			for y := 0; y < yPartitions; y++ {
				width := imageBounds.Dx() / xPartitions
				height := imageBounds.Dy() / yPartitions
				startX := x * width
				startY := y * height

				wg.Add(1)
				go SetPalettedImage(startX, startY, width, height, rawFrame, palletedImage, &skPalette, &wg, &mu)
			}
		}

		wg.Wait()

		quantizeDone := time.Now()
		setLoop := quantizeDone.Sub(newPalettedDone)

		recordingGif.Image = append(recordingGif.Image, palletedImage)

		// var delay float = len(recordingGif.Image) * 1.0 / 60.0 * 100.0
		var delay int = int(math.Floor(0.5 + (1.0 / 60.0 * 100.0)))
		recordingGif.Delay = append(recordingGif.Delay, delay)

		lastFrameDone = time.Now()
		fmt.Printf("[gif]  idle %.2f | NewPaletted %.2f | Set loop %.2f (%.0f ns/px over %d px) | total %.2f ms (frame %d)\n",
			ms(idle),
			ms(newPalettedDone.Sub(frameStart)),
			ms(setLoop),
			float64(setLoop.Nanoseconds())/float64(numPixels),
			numPixels,
			ms(lastFrameDone.Sub(frameStart)),
			len(recordingGif.Delay)-1)
	}

	fmt.Println("making gif file")
	encodeStart := time.Now()
	gifFile, _ := os.Create("recording.gif")
	err := gif.EncodeAll(gifFile, recordingGif)
	if err != nil {
		fmt.Println("error has occured while trying to create gif")
	}
	gifFile.Close()
	fmt.Printf("[gif]  EncodeAll of %d frames took %.2f ms\n",
		len(recordingGif.Image), ms(time.Since(encodeStart)))
}

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000000.0
}
