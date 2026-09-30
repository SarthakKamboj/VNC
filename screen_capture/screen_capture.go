package screen_capture

import (
	"fmt"
	"image"
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
	go RecordGif(100, &wg)

	wg.Wait()

	StopCapture()
}

func StopCapture() {
	mac_capture.StopCapture()
	for len(rawImageFrames) > 0 {
		<-rawImageFrames
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
		for row := imageBounds.Min.Y; row <= imageBounds.Max.Y; row++ {
			for col := imageBounds.Min.X; col <= imageBounds.Max.X; col++ {
				c := rawFrame.At(col, row)
				var index = skPalette.FindClosestIndex(c)
				palletedImage.SetColorIndex(col, row, index)
			}
		}

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
