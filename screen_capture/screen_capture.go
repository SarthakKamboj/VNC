package screen_capture

import (
	"fmt"
	"image/gif"
	"math"
	"os"
)

func CaptureScreen() {
	var totalFrames uint = 120

	captureController := FrameCaptureController{}
	captureController.Init(60, totalFrames)

	captureController.Capture()

	RecordGif(captureController)
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
