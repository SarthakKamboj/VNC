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

var rawImageFrames = make(chan *image.RGBA, 10)

func CaptureScreen() {
	var wg sync.WaitGroup

	mac_capture.StartCapture(&rawImageFrames)

	wg.Add(1)
	go RecordGif(60, &wg)
	wg.Wait()

	StopCapture()
}

func StopCapture() {
	mac_capture.StopCapture()
	for len(rawImageFrames) > 0 {
		<-rawImageFrames
	}
}

// lookupStats counts how one partition's pixels were resolved to palette indices.
type lookupStats struct {
	comparisons    int // palette color comparisons made by octree searches on cache misses
	cacheMisses    int // pixels that missed the cache and went through the octree
	cacheHits      int // pixels whose palette index was already in the cache
	maxComparisons int // most comparisons any single cache miss needed
}

func SetPalettedImage(startX int, startY int, width int, height int, rawFrame *image.RGBA, palettedImage *image.Paletted, skPalette *SkPalette, wg *sync.WaitGroup, mu *sync.Mutex, stats *lookupStats, cache *map[color.Color]uint8) {
	defer wg.Done()

	*stats = lookupStats{}

	for row := startY; row < startY+height; row++ {
		for col := startX; col < startX+width; col++ {
			c := rawFrame.At(col, row)
			var index uint8 = 0
			val, exists := (*cache)[c]
			if exists {
				index = val
				stats.cacheHits++
			} else {
				// need to see if this is faster or not compared to just naive eucledian checks
				compCount := 0
				index, compCount = skPalette.FindClosestIndex(c)
				stats.comparisons += compCount
				stats.cacheMisses++
				stats.maxComparisons = max(stats.maxComparisons, compCount)
				(*cache)[c] = index
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

	const xPartitions int = 4
	const yPartitions int = 4
	const totalPartitions int = xPartitions * yPartitions

	var pixelClosestMaps [totalPartitions]map[color.Color]uint8 = [totalPartitions]map[color.Color]uint8{}
	for i := 0; i < totalPartitions; i++ {
		pixelClosestMaps[i] = make(map[color.Color]uint8)
	}

	// Per-section totals across all frames; averages are printed once at the end.
	var frameCount int
	var idleFrames int
	var idleSum, newPalettedSum, setLoopSum, appendSum, frameSum time.Duration
	var compSum int
	var cacheMissSum int
	var cacheHitSum int
	var maxComp int
	var pixelSum int

	var cachePalettedImage *image.Paletted = nil

	for rawFrame := range rawImageFrames {

		frameStart := time.Now()
		var idle time.Duration
		if !lastFrameDone.IsZero() {
			idle = frameStart.Sub(lastFrameDone)
		}

		var imageBounds image.Rectangle = rawFrame.Bounds()

		var isEmpty = imageBounds.Max.X == 0 || imageBounds.Max.Y == 0

		var palettedImage *image.Paletted = nil

		if !isEmpty {
			palettedImage = image.NewPaletted(imageBounds, palette.Plan9)
		}

		newPalettedDone := time.Now()

		numPixels, totalComp, totalCacheMisses := 0, 0, 0

		if !isEmpty {
			// TODO: need to make this faster
			/*
				Use an octree to find the pixel index
			*/
			numPixels = imageBounds.Dx() * imageBounds.Dy()

			var wg sync.WaitGroup
			var mu sync.Mutex

			// TODO: see why adding more threads is not making this faster but rather slower

			var partitionStats [totalPartitions]lookupStats = [totalPartitions]lookupStats{}

			for x := 0; x < xPartitions; x++ {
				for y := 0; y < yPartitions; y++ {
					width := imageBounds.Dx() / xPartitions
					height := imageBounds.Dy() / yPartitions
					startX := x * width
					startY := y * height

					idx := (y * xPartitions) + x

					wg.Add(1)
					go SetPalettedImage(startX, startY, width, height, rawFrame, palettedImage, &skPalette, &wg, &mu, &partitionStats[idx], &pixelClosestMaps[idx])
				}
			}

			wg.Wait()

			for _, ps := range partitionStats {
				totalComp += ps.comparisons
				totalCacheMisses += ps.cacheMisses
				cacheHitSum += ps.cacheHits
				maxComp = max(maxComp, ps.maxComparisons)
			}
		}

		quantizeDone := time.Now()
		setLoop := quantizeDone.Sub(newPalettedDone)

		if isEmpty && cachePalettedImage != nil {
			recordingGif.Image = append(recordingGif.Image, cachePalettedImage)
		} else {
			recordingGif.Image = append(recordingGif.Image, palettedImage)
			cachePalettedImage = palettedImage
		}

		var delay int = int(math.Floor(0.5 + (1.0 / 60.0 * 100.0)))
		recordingGif.Delay = append(recordingGif.Delay, delay)

		lastFrameDone = time.Now()

		frameCount++
		// The first frame has no previous frame to measure idle time from.
		if idle > 0 {
			idleSum += idle
			idleFrames++
		}
		newPalettedSum += newPalettedDone.Sub(frameStart)
		setLoopSum += setLoop
		appendSum += lastFrameDone.Sub(quantizeDone)
		frameSum += lastFrameDone.Sub(frameStart)
		compSum += totalComp
		cacheMissSum += totalCacheMisses
		pixelSum += numPixels

		if len(recordingGif.Image) > maxFrames {
			break
		}
	}

	if frameCount > 0 {
		n := float64(frameCount)
		fmt.Printf("\n[gif] RecordGif: %.2f ms avg processing per frame over %d frames (+ %.2f ms avg waiting for the next frame)\n",
			ms(frameSum)/n, frameCount, ms(idleSum)/max(float64(idleFrames), 1))
		fmt.Printf("    %-22s %6.2f ms  %3.0f%%   %.1f ns/pixel\n", "map pixels to palette",
			ms(setLoopSum)/n, 100*float64(setLoopSum)/float64(frameSum), float64(setLoopSum.Nanoseconds())/float64(pixelSum))
		fmt.Printf("    %-22s %6.2f ms  %3.0f%%\n", "other",
			ms(frameSum-setLoopSum)/n, 100*float64(frameSum-setLoopSum)/float64(frameSum))
		fmt.Printf("    %-22s %8d/frame  (%.2f%% of pixels)\n", "cache hits",
			cacheHitSum/frameCount, 100*float64(cacheHitSum)/float64(max(cacheHitSum+cacheMissSum, 1)))
		fmt.Printf("    %-22s %8d/frame", "cache misses", cacheMissSum/frameCount)
		if cacheMissSum > 0 {
			fmt.Printf("  (octree search compares %.1f colors avg, worst %d, of %d)",
				float64(compSum)/float64(cacheMissSum), maxComp, len(skPalette.palette))
		}
		fmt.Println()
	}

	// fmt.Println("making gif file")
	encodeStart := time.Now()
	gifFile, _ := os.Create("recording.gif")
	err := gif.EncodeAll(gifFile, recordingGif)
	if err != nil {
		fmt.Println("error has occured while trying to create gif")
	}
	gifFile.Close()
	fmt.Printf("[gif] gif.EncodeAll of %d frames: %.2f ms\n",
		len(recordingGif.Image), ms(time.Since(encodeStart)))
}

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000000.0
}
