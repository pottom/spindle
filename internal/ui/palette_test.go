package ui

import (
	"testing"

	"github.com/pottom/spindle/internal/player"
)

// The hue a cell is drawn in travels through the drawing code as an int8, in
// the same array as the level it burns at. That caps the palette at 128 bands,
// and going past it would not fail loudly: the index would wrap negative and
// the water would come out in the wrong colour at the right-hand edge.
//
// The cap is not a problem to solve. A band narrower than a cell buys nothing,
// and no terminal is 128 cells to the band. This is here so that raising the
// palette's resolution again is a failing test rather than a bug on screen.
func TestPaletteFitsTheIndex(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	if n := len(m.styles.Bars); n > 128 {
		t.Fatalf("the spectrum has %d bands; the hue index is an int8 and holds 128", n)
	}
	if n := len(m.styles.Words); n > 128 {
		t.Fatalf("the lyric palette has %d bands; the hue index is an int8 and holds 128", n)
	}
}
