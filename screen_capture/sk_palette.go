package screen_capture

import (
	"fmt"
	"image/color"
	"math"
)

type ColorInfo struct {
	r8         uint8
	g8         uint8
	b8         uint8
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
	colors []ColorInfo
	// only populated at non-leaf nodes
	childNodes []OctTreeNode
	loc        OctTreeNodeLoc
}

type SkPalette struct {
	goPalette color.Palette
	palette   []ColorInfo
	rootNode  OctTreeNode
}

func (skPalette *SkPalette) Init(palette color.Palette) {
	skPalette.palette = make([]ColorInfo, 0, len(palette))
	for i := 0; i < len(palette); i++ {
		r32, g32, b32, _ := palette[i].RGBA()
		r8, g8, b8 := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)
		skPalette.palette = append(skPalette.palette, ColorInfo{r8, g8, b8, i})
	}

	skPalette.goPalette = palette

	skPalette.partitionPalette()
}

func (skPalette *SkPalette) partitionPalette() {
	var rootNode OctTreeNode = OctTreeNode{}
	rootNode.loc = OctTreeNodeLoc{0, 0, 0, 256, 256, 256}

	skPalette.partitionPaletteHelper(&rootNode, skPalette.palette)

	skPalette.rootNode = rootNode
}

func (skPalette *SkPalette) partitionPaletteHelper(parentNode *OctTreeNode, parentNodeColors []ColorInfo) {
	var subDim [3]float32 = [3]float32{parentNode.loc.w / 2.0, parentNode.loc.h / 2.0, parentNode.loc.d / 2.0}

	var intersectingColors []ColorInfo = skPalette.GetIntersectingColors(parentNode.loc, parentNodeColors)
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

					skPalette.partitionPaletteHelper(&subNode, intersectingColors)

					parentNode.childNodes = append(parentNode.childNodes, subNode)
				}
			}
		}
	} else {
		parentNode.colors = make([]ColorInfo, 0, 10)
		parentNode.colors = intersectingColors
	}
}

func (skPalette *SkPalette) GetIntersectingColors(loc OctTreeNodeLoc, colorsToExamine []ColorInfo) []ColorInfo {
	var intersectingColors []ColorInfo = make([]ColorInfo, 0, 256)
	for _, colorInfo := range colorsToExamine {
		r, g, b := colorInfo.r8, colorInfo.g8, colorInfo.b8
		var inBounds bool = skPalette.Intersects(loc, r, g, b)
		if inBounds {
			intersectingColors = append(intersectingColors, colorInfo)
		}
	}
	return intersectingColors
}

func (skPalette *SkPalette) FindClosestIndex(c color.Color) (uint8, int) {
	r32, g32, b32, _ := c.RGBA()
	r, g, b := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)

	var comparisonCount int = 0
	index := skPalette.FindClosestIndexHelper(r, g, b, &skPalette.rootNode, &comparisonCount)
	return uint8(index), comparisonCount
}

func (skPalette *SkPalette) Intersects(loc OctTreeNodeLoc, r uint8, g uint8, b uint8) bool {
	return float32(r) >= loc.r && float32(g) >= loc.g && float32(b) >= loc.b && float32(r) <= loc.r+loc.w && float32(g) <= loc.g+loc.h && float32(b) <= loc.b+loc.d
}

func (skPalette *SkPalette) FindClosestIndexHelper(r uint8, g uint8, b uint8, node *OctTreeNode, comparisonCount *int) int {

	var minDist float32 = float32(math.Pow(256.0, 2) * 3)
	var minIndex int = len(skPalette.palette)

	if len(node.childNodes) > 0 {

		for i := range node.childNodes {
			childNode := &node.childNodes[i]
			*comparisonCount++
			if skPalette.Intersects(childNode.loc, r, g, b) {

				var closestSubIdx int = skPalette.FindClosestIndexHelper(r, g, b, childNode, comparisonCount)
				var compareColorInfo ColorInfo = skPalette.palette[closestSubIdx]
				cr, cg, cb := compareColorInfo.r8, compareColorInfo.g8, compareColorInfo.b8
				var dist float32 = (float32(cr)-float32(r))*(float32(cr)-float32(r)) + (float32(cg)-float32(g))*(float32(cg)-float32(g)) + (float32(cb)-float32(b))*(float32(cb)-float32(b))

				if dist < minDist {
					minIndex = int(compareColorInfo.palleteIdx)
					minDist = dist
				}
			}

		}
	} else {
		for _, compareColorInfo := range node.colors {
			*comparisonCount++
			cr, cg, cb := compareColorInfo.r8, compareColorInfo.g8, compareColorInfo.b8
			var dist float32 = float32(math.Pow(float64(cr-r), 2) + math.Pow(float64(cg-g), 2) + math.Pow(float64(cb-b), 2))
			if dist < minDist {
				minIndex = int(compareColorInfo.palleteIdx)
				minDist = dist
			}
		}
	}

	return minIndex
}

func (skPalette *SkPalette) Print() {
	skPalette.PrintHelper(skPalette.rootNode)
}

func (skPalette *SkPalette) PrintHelper(node OctTreeNode) {
	fmt.Printf("r=%f, g=%f, b=%f, w=%f, h=%f, d=%f ", node.loc.r, node.loc.g, node.loc.b, node.loc.w, node.loc.h, node.loc.d)
	if len(node.childNodes) == 0 {
		fmt.Printf(" numColors=%d\n", len(node.colors))
	} else {
		fmt.Printf(" numChildren=%d\n", len(node.childNodes))
	}
}
