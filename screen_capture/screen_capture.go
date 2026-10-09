package screen_capture

import (
	"fmt"
	"image/gif"
	"math"
	"os"
	"time"
)

/*
	This file should be the platform abstraction layer for capturing screens
*/

func CaptureScreen() {
	// var wg sync.WaitGroup

	var totalFrames uint = 120

	// mac_capture.StartCapture(&rawImageFrames, totalFrames)
	captureController := FrameCaptureController{}
	captureController.Init(60, totalFrames)

	captureController.Capture()

	RecordGif(captureController)

	// wg.Add(1)
	// go RecordGif(totalFrames, &wg)
	// wg.Wait()
}

func StopCapture() {
	// mac_capture.StopCapture()
	// for len(rawImageFrames) > 0 {
	// 	<-rawImageFrames
	// }
}

func RecordGif(captureController FrameCaptureController) {
	var recordingGif *gif.GIF = &gif.GIF{}

	recordingGif.Image = captureController.GetFrames()

	for range len(recordingGif.Image) {
		var delay int = int(math.Floor(0.5 + (1.0 / 60.0 * 100.0)))
		recordingGif.Delay = append(recordingGif.Delay, delay)
	}

	gifFile, _ := os.Create("recording.gif")
	err := gif.EncodeAll(gifFile, recordingGif)
	if err != nil {
		fmt.Println("error has occured while trying to create gif")
	}
	gifFile.Close()
}

// func RecordGif(maxFrames int, wg *sync.WaitGroup) {

// defer wg.Done()

// // gif should be 60fps probably
// var recordingGif *gif.GIF = &gif.GIF{}

// var skPalette SkPalette = SkPalette{}
// skPalette.Init(palette.Plan9)

// const xPartitions int = 4
// const yPartitions int = 4
// const totalPartitions int = xPartitions * yPartitions

// var pixelClosestMaps [totalPartitions]map[color.Color]uint8 = [totalPartitions]map[color.Color]uint8{}
// for i := 0; i < totalPartitions; i++ {
// 	pixelClosestMaps[i] = make(map[color.Color]uint8)
// }

// var cachePalettedImage *image.Paletted = nil

// for rawFrame := range rawImageFrames {

// 	var imageBounds image.Rectangle = rawFrame.Bounds()

// 	var isEmpty = imageBounds.Max.X == 0 || imageBounds.Max.Y == 0

// 	var palettedImage *image.Paletted = nil

// 	if !isEmpty {
// 		palettedImage = image.NewPaletted(imageBounds, palette.Plan9)
// 	}

// 	if !isEmpty {
// 		// TODO: need to make this faster
// 		var wg sync.WaitGroup
// 		var mu sync.Mutex

// 		for x := 0; x < xPartitions; x++ {
// 			for y := 0; y < yPartitions; y++ {
// 				width := imageBounds.Dx() / xPartitions
// 				height := imageBounds.Dy() / yPartitions
// 				startX := x * width
// 				startY := y * height

// 				idx := (y * xPartitions) + x

// 				wg.Add(1)
// 				go SetPalettedImage(startX, startY, width, height, rawFrame, palettedImage, &skPalette, &wg, &mu, &pixelClosestMaps[idx])
// 			}
// 		}

// 		wg.Wait()
// 	}

// 	if isEmpty && cachePalettedImage != nil {
// 		recordingGif.Image = append(recordingGif.Image, cachePalettedImage)
// 	} else {
// 		recordingGif.Image = append(recordingGif.Image, palettedImage)
// 		cachePalettedImage = palettedImage
// 	}

// 	var delay int = int(math.Floor(0.5 + (1.0 / 60.0 * 100.0)))
// 	recordingGif.Delay = append(recordingGif.Delay, delay)

// 	if len(recordingGif.Image) > maxFrames {
// 		break
// 	}
// }

// gifFile, _ := os.Create("recording.gif")
// err := gif.EncodeAll(gifFile, recordingGif)
// if err != nil {
// 	fmt.Println("error has occured while trying to create gif")
// }
// gifFile.Close()
// }

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000000.0
}
