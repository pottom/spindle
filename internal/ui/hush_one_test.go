package ui

import "testing"

// The hush is one drawing, and it is drawn as large as the screen allows.
//
// A set can be smaller than a company. Asked for four of one the deal used to
// find one, call that a failure at every size and fall through to the smallest —
// so the one thing on screen was drawn as small as it can be drawn.
func TestTheHushIsOneAndItIsBig(t *testing.T) {
	set, ok := markSets[markHush]
	if !ok {
		t.Fatal("no hush")
	}
	const w, rows = 200, 44
	var share float64 = wordsMark
	band := int(share * float64(rows*dotsPerCellY))
	size, crowd, _, ok := markCrowdFor(set, band, w*dotsPerCellX, 7)
	if !ok {
		t.Fatal("no row")
	}
	if len(crowd) != 1 {
		t.Errorf("the hush came up as %d drawings", len(crowd))
	}
	// The largest baked size the band has room for, which is what one drawing
	// with the screen to itself should be given.
	var fits int
	for _, s := range set.sizes {
		if s.tall <= band {
			fits = max(fits, s.tall)
		}
	}
	if size.tall < fits {
		t.Errorf("the hush was drawn at %d dots with %d baked and the room for it", size.tall, fits)
	}
	t.Logf("one drawing, %d dots tall, %d wide", size.tall, crowd[0].wide)
}
