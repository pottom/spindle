package cover

import (
	"image"
	"testing"
)

// Both drawn covers sit in the middle of their square. There is nothing to
// measure them against on screen — no sleeve's own framing, no photographer's
// crop — so the arithmetic has to be right.
//
// Across by weight and down by extent, which is what the eye does with these two
// shapes. The note's outermost points are a head on the left and a stem on the
// right, and centring those left the beam and both stems — nearly all of the ink
// — to the right of the middle. The heart is the other way about: its mass sits
// high, in the lobes, and balancing that would hang the point below the square.
func TestTheDrawnCoversAreCentred(t *testing.T) {
	const size = 256
	for _, c := range []struct {
		name string
		art  image.Image
	}{
		{"the saved tracks' heart", likedArt(size)},
		{"the stand-in note", noneArt(size)},
	} {
		back := c.art.At(0, 0)
		top, bottom := size, -1
		var weight, ink float64
		for y := range size {
			for x := range size {
				if c.art.At(x, y) == back {
					continue
				}
				top, bottom = min(top, y), max(bottom, y)
				weight += float64(x)
				ink++
			}
		}
		if ink == 0 {
			t.Errorf("%s: nothing is drawn at all", c.name)
			continue
		}

		// Within a pixel or three: a curve sampled on a grid cannot land exactly.
		const slack = 3
		if at := int(weight / ink); abs(at-size/2) > slack {
			t.Errorf("%s: its weight falls at column %d of %d", c.name, at, size)
		}
		if over, under := top, size-1-bottom; abs(over-under) > slack {
			t.Errorf("%s: %d pixels of air over it and %d under", c.name, over, under)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
