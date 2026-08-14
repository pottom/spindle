package cover

import (
	"image"
	"testing"
)

// Both drawn covers sit in the middle of their square. There is nothing to
// measure them against on screen — no sleeve's own framing, no photographer's
// crop — so the arithmetic has to be right.
func TestTheDrawnCoversAreCentred(t *testing.T) {
	const size = 256
	for _, c := range []struct {
		name string
		art  image.Image
	}{
		{"the saved tracks' heart", likedArt(size)},
		{"the stand-in note", noneArt(size)},
	} {
		top, bottom, left, right := size, -1, size, -1
		back := c.art.At(0, 0)
		for y := range size {
			for x := range size {
				if c.art.At(x, y) == back {
					continue
				}
				top, bottom = min(top, y), max(bottom, y)
				left, right = min(left, x), max(right, x)
			}
		}
		if bottom < 0 {
			t.Errorf("%s: nothing is drawn at all", c.name)
			continue
		}

		// Within a pixel or two: a curve sampled on a grid cannot land exactly.
		const slack = 3
		if over, under := top, size-1-bottom; abs(over-under) > slack {
			t.Errorf("%s: %d pixels of air over it and %d under", c.name, over, under)
		}
		if before, after := left, size-1-right; abs(before-after) > slack {
			t.Errorf("%s: %d pixels of air to its left and %d to its right", c.name, before, after)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
