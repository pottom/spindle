package cover

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// disc is a stand-in sleeve: something bright on something dark, which is what
// most covers are.
func disc() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := range 300 {
		for x := range 300 {
			d := math.Hypot(float64(x-150), float64(y-150))
			v := uint8(max(0, 230-d*1.6))
			img.Set(x, y, color.RGBA{R: v, G: v / 2, B: uint8(float64(v) * 0.8), A: 255})
		}
	}
	return img
}

// The picture has to still be the picture: bright in the middle where the cover
// is bright, dark at the corners where it is dark.
func TestGrindKeepsTheShape(t *testing.T) {
	g := Grind(disc(), 80, 20, 2, 4)

	if g.DotsX != 160 || g.DotsY != 80 || len(g.Lum) != 160*80 {
		t.Fatalf("ground to %dx%d dots with %d of them", g.DotsX, g.DotsY, len(g.Lum))
	}

	at := func(x, y int) int { return int(g.Lum[y*g.DotsX+x]) }
	middle := at(g.DotsX/2, g.DotsY/2)
	corner := at(2, 2)
	t.Logf("middle %d, corner %d", middle, corner)

	if middle < 200 {
		t.Errorf("the middle of the cover reads %d, want it bright", middle)
	}
	if corner > 60 {
		t.Errorf("the corner reads %d, want it dark", corner)
	}
}

// Half the covers ever printed are dark, and a dark one dithered against a fixed
// threshold is a black rectangle. The brightness is stretched so that every
// sleeve uses the whole range.
func TestGrindStretchesADarkCover(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := range 100 {
		for x := range 100 {
			// Nothing brighter than a quarter, and most of it much less.
			v := uint8(x * 60 / 100)
			img.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}

	g := Grind(img, 40, 10, 2, 4)

	var lo, hi uint8 = 255, 0
	var lit int
	for _, v := range g.Lum {
		lo, hi = min(lo, v), max(hi, v)
		if v >= 128 {
			lit++
		}
	}
	share := float64(lit) * 100 / float64(len(g.Lum))
	t.Logf("brightness runs %d to %d, %.0f%% of it over the middle", lo, hi, share)

	if hi < 200 {
		t.Errorf("the brightest dot of a dark cover is %d, want the picture stretched", hi)
	}
	if share < 25 || share > 75 {
		t.Errorf("%.0f%% of the dots are lit, want a picture rather than a silhouette", share)
	}
}

// Every cell carries a colour code when it differs from its neighbour, and a
// photograph differs in every cell. Rounding the colours together is what keeps
// a screenful drawable thirty times a second.
func TestGrindQuantisesTheColours(t *testing.T) {
	g := Grind(disc(), 80, 20, 2, 4)

	seen := map[color.RGBA]bool{}
	for _, c := range g.Cell {
		seen[c] = true
	}
	t.Logf("%d colours in %d cells", len(seen), len(g.Cell))

	if len(seen) > grainLevels*grainLevels*grainLevels {
		t.Errorf("%d different colours, want at most %d", len(seen), grainLevels*grainLevels*grainLevels)
	}

	// And they are on the grid, not near it.
	step := 255 / (grainLevels - 1)
	for c := range seen {
		for _, v := range []uint8{c.R, c.G, c.B} {
			if int(v)%step != 0 {
				t.Fatalf("a cell is %v, which is not on the %d-step grid", c, step)
			}
		}
	}
}

// A cover is square and a terminal is not, so it is cropped from the middle
// rather than squashed.
func TestGrindCropsRatherThanSquashes(t *testing.T) {
	// A tall image drawn in a wide picture: what survives has to be the middle
	// band of it, so the middle row is the middle row of the original.
	img := image.NewRGBA(image.Rect(0, 0, 100, 400))
	for y := range 400 {
		v := uint8(0)
		if y > 190 && y < 210 {
			v = 255 // a bright stripe across the middle
		}
		for x := range 100 {
			img.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}

	g := Grind(img, 40, 10, 2, 4)
	mid := int(g.Lum[(g.DotsY/2)*g.DotsX+g.DotsX/2])
	if mid < 200 {
		t.Errorf("the middle of the cropped cover reads %d, want the stripe that was there", mid)
	}
}
