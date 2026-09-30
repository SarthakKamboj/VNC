package mac_capture

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=12.3
#cgo LDFLAGS: -framework ScreenCaptureKit -framework Foundation -framework CoreMedia -framework CoreVideo -framework CoreGraphics -framework IOSurface

#include "screen_capture_kit.h"

#include <math.h>
*/
import "C"

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"math"
	"os"
	"sync"
	"time"
	"unsafe"
)

// import "image/png"

type ColorInfo struct {
	c          color.Color
	palleteIdx int
}

type OctTreeNodeLoc struct {
	r float32
	g float32
	b float32

	w float32
	h float32
	d float32
}

type OctTreeNode struct {
	// only populated at leaf nodes
	colors     []ColorInfo
	childNodes []OctTreeNode
	loc        OctTreeNodeLoc
}

type VncPallete struct {
	// palette  color.Palette
	palette  []ColorInfo
	rootNode OctTreeNode
	cache    map[color.Color]int
}

func (vncPallete *VncPallete) Init(palette color.Palette) {
	// generate octree here
	vncPallete.palette = make([]ColorInfo, 0, len(palette))
	for i := 0; i < len(palette); i++ {
		vncPallete.palette = append(vncPallete.palette, ColorInfo{palette[i], i})
	}
	vncPallete.cache = make(map[color.Color]int)
	vncPallete.partitionPalette()
}

func (vncPallete *VncPallete) partitionPalette() {
	var rootNode OctTreeNode = OctTreeNode{}
	rootNode.loc = OctTreeNodeLoc{0, 0, 0, 256, 256, 256}

	vncPallete.partitionPaletteHelper(&rootNode, vncPallete.palette)

	vncPallete.rootNode = rootNode
}

func (vncPallete *VncPallete) partitionPaletteHelper(parentNode *OctTreeNode, parentNodeColors []ColorInfo) {
	var subDim [3]float32 = [3]float32{parentNode.loc.w / 2.0, parentNode.loc.h / 2.0, parentNode.loc.d / 2.0}

	var intersectingColors []ColorInfo = vncPallete.GetIntersectingColors(parentNode.loc, parentNodeColors)
	if len(intersectingColors) > 8 {

		parentNode.childNodes = make([]OctTreeNode, 0, 10)

		for x := 0; x < 2; x++ {
			for y := 0; y < 2; y++ {
				for z := 0; z < 2; z++ {
					var subNode OctTreeNode = OctTreeNode{}

					var subLocR = parentNode.loc.r + (subDim[0] * float32(x))
					var subLocG = parentNode.loc.g + (subDim[1] * float32(y))
					var subLocB = parentNode.loc.b + (subDim[2] * float32(z))
					var subLoc OctTreeNodeLoc = OctTreeNodeLoc{subLocR, subLocG, subLocB, subDim[0], subDim[1], subDim[2]}
					subNode.loc = subLoc

					vncPallete.partitionPaletteHelper(&subNode, intersectingColors)

					parentNode.childNodes = append(parentNode.childNodes, subNode)
				}
			}
		}
	} else {
		parentNode.colors = make([]ColorInfo, 0, 10)
		parentNode.colors = intersectingColors
	}
}

func (vncPallete *VncPallete) GetIntersectingColors(loc OctTreeNodeLoc, colorsToExamine []ColorInfo) []ColorInfo {
	var intersectingColors []ColorInfo = make([]ColorInfo, 0, 256)
	for _, colorInfo := range colorsToExamine {
		r32, g32, b32, _ := colorInfo.c.RGBA()
		r, g, b := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)
		var inBounds bool = vncPallete.Intersects(loc, r, g, b)
		if inBounds {
			intersectingColors = append(intersectingColors, colorInfo)
		}
	}
	return intersectingColors
}

func (vncPallete *VncPallete) FindClosestIndex(c color.Color) uint8 {
	// traverse octree here
	// return 123

	val, exists := vncPallete.cache[c]
	if exists {
		return uint8(val)
	}

	r32, g32, b32, _ := c.RGBA()
	r, g, b := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)

	// fmt.Printf("color is %v %v %v\n", r, g, b)

	// fmt.Println("root node has %d chilren", len(vncPallete.rootNode.childNodes))
	index := vncPallete.FindClosestIndexHelper(r, g, b, &vncPallete.rootNode)
	// fmt.Printf("Selected index is %v\n", index)
	vncPallete.cache[c] = index
	return uint8(index)
}

func (vncPallete *VncPallete) Intersects(loc OctTreeNodeLoc, r uint8, g uint8, b uint8) bool {
	return float32(r) >= loc.r && float32(g) >= loc.g && float32(b) >= loc.b && float32(r) <= loc.r+loc.w && float32(g) <= loc.g+loc.h && float32(b) <= loc.b+loc.d
}

func (vncPallete *VncPallete) FindClosestIndexHelper(r uint8, g uint8, b uint8, node *OctTreeNode) int {
	// traverse octree here
	// return 123

	// fmt.Printf("FindClosestIndexHelper: color=(%d, %d, %d) node loc=%+v children=%d colors=%d\n", r, g, b, node.loc, len(node.childNodes), len(node.colors))

	var minDist float32 = float32(math.Pow(256.0, 2) * 3)
	var minIndex int = len(vncPallete.palette)

	if len(node.childNodes) > 0 {

		for _, childNode := range node.childNodes {

			if vncPallete.Intersects(childNode.loc, r, g, b) {

				// fmt.Printf("Looking at octree child node %d", i)

				var closestSubIdx int = vncPallete.FindClosestIndexHelper(r, g, b, &childNode)
				// fmt.Printf("FindClosestIndexHelper: child loc=%+v returned index %d\n", childNode.loc, closestSubIdx)
				var compareColorInfo ColorInfo = vncPallete.palette[closestSubIdx]
				cr32, cg32, cb32, _ := compareColorInfo.c.RGBA()
				cr, cg, cb := uint8(cr32>>8), uint8(cg32>>8), uint8(cb32>>8)
				var dist float32 = (float32(cr)-float32(r))*(float32(cr)-float32(r)) + (float32(cg)-float32(g))*(float32(cg)-float32(g)) + (float32(cb)-float32(b))*(float32(cb)-float32(b))

				// fmt.Printf("FindClosestIndexHelper: candidate index %d color=(%d, %d, %d) dist=%f\n", compareColorInfo.palleteIdx, cr, cg, cb, dist)

				if dist < minDist {
					minIndex = int(compareColorInfo.palleteIdx)
					minDist = dist
				}
			}

		}

		// fmt.Printf("FindClosestIndexHelper: returning index %d dist=%f\n", minIndex, minDist)
	} else {
		for _, compareColorInfo := range node.colors {
			cr32, cg32, cb32, _ := compareColorInfo.c.RGBA()
			cr, cg, cb := uint8(cr32>>8), uint8(cg32>>8), uint8(cb32>>8)
			var dist float32 = float32(math.Pow(float64(cr-r), 2) + math.Pow(float64(cg-g), 2) + math.Pow(float64(cb-b), 2))
			// fmt.Printf("FindClosestIndexHelper: leaf candidate index %d color=(%d, %d, %d) dist=%f\n", compareColorInfo.palleteIdx, cr, cg, cb, dist)
			if dist < minDist {
				minIndex = int(compareColorInfo.palleteIdx)
				minDist = dist
			}
		}
		// fmt.Printf("FindClosestIndexHelper: leaf returning index %d dist=%f\n", minIndex, minDist)
	}

	return minIndex
}

func (vncPallete *VncPallete) Print() {
	vncPallete.PrintHelper(vncPallete.rootNode)
}

func (vncPallete *VncPallete) PrintHelper(node OctTreeNode) {
	fmt.Printf("r=%f, g=%f, b=%f, w=%f, h=%f, d=%f ", node.loc.r, node.loc.g, node.loc.b, node.loc.w, node.loc.h, node.loc.d)
	if len(node.childNodes) == 0 {
		fmt.Printf(" numColors=%d\n", len(node.colors))
	} else {
		fmt.Printf(" numChildren=%d\n", len(node.childNodes))

		// for _, childNode := range node.childNodes {
		// 	vncPallete.PrintHelper()
		// }
	}
}

var rawImageFrames = make(chan *image.RGBA, 1)

func recordGif(maxFrames int, wg *sync.WaitGroup) {

	defer wg.Done()

	// gif should be 60fps probably
	var recordingGif *gif.GIF = &gif.GIF{}

	var lastFrameDone time.Time

	var vncPallete VncPallete = VncPallete{}
	vncPallete.Init(palette.Plan9)
	// vncPallete.Print()

	// return

	for rawFrame := range rawImageFrames {

		frameStart := time.Now()
		// How long this goroutine sat with nothing to do: if it is ~0 the
		// encoder is the bottleneck, if it is large the capture side is.
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
				// palletedImage.Set(col, row, rawFrame.At(col, row))
				c := rawFrame.At(col, row)
				// fmt.Printf("color is %v %v %v %v\n", uint8(pixel.r), uint8(pixel.g), uint8(pixel.b), uint8(pixel.a))
				// fmt.Printf("@@@ Looking at %v %v\n", col, row)
				var index = vncPallete.FindClosestIndex(c)
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

	StopCapture()
}

func Capture() {
	var wg sync.WaitGroup

	C.start_capture()

	wg.Add(1)
	go recordGif(100, &wg)

	wg.Wait()
}

func StopCapture() {
	C.stop_capture()
	for len(rawImageFrames) > 0 {
		<-rawImageFrames
	}
}

//export goHandleFrame
func goHandleFrame(frame C.frame_t) {
	start := time.Now()

	var topLeft image.Point = image.Point{0, 0}
	var bottomRight image.Point = image.Point{int(frame.width) - 1, int(frame.height) - 1}

	var screenImage *image.RGBA = image.NewRGBA(image.Rectangle{topLeft, bottomRight})

	var numPixels int = int(frame.height) * int(frame.width)
	framePixelData := unsafe.Slice(frame.pixel_data, numPixels)

	allocDone := time.Now()

	// TODO: need to make this faster
	for y := 0; y < int(frame.height); y++ {
		for x := 0; x < int(frame.width); x++ {
			var indexIntoBuffer int = (int(frame.width) * y) + x
			var pixel C.pixel_t = framePixelData[indexIntoBuffer]
			var color color.RGBA = color.RGBA{uint8(pixel.r), uint8(pixel.g), uint8(pixel.b), uint8(pixel.a)}
			// fmt.Printf("color is %v %v %v %v\n", uint8(pixel.r), uint8(pixel.g), uint8(pixel.b), uint8(pixel.a))
			screenImage.Set(x, y, color)
		}
	}

	convertDone := time.Now()

	// A full channel means recordGif is the bottleneck, and this send is what
	// stalls the ScreenCaptureKit delivery queue -- so time it on its own.
	rawImageFrames <- screenImage
	queueLen := len(rawImageFrames)

	sendDone := time.Now()
	fmt.Printf("[go]   alloc %.2f | convert %.2f | chan send %.2f (queue %d/%d) | total %.2f ms\n",
		ms(allocDone.Sub(start)),
		ms(convertDone.Sub(allocDone)),
		ms(sendDone.Sub(convertDone)),
		queueLen, cap(rawImageFrames),
		ms(sendDone.Sub(start)))
}

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000000.0
}
