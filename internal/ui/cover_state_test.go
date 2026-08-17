package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// A cover is square and a cell is not, so a picture fitted into a box of whole
// cells comes out a few cells short of the tile more often than not. Left where
// it landed it sat against the left edge of its own frame, with the shortfall in
// one lump on the right. Reported from a real screen.
func TestAShortPictureStandsInTheMiddleOfItsTile(t *testing.T) {
	c := coverState{want: 14}
	c.took("▀▀▀▀▀▀▀▀▀▀\n▀▀▀▀▀▀▀▀▀▀")

	for i, line := range c.lines {
		if got := lipgloss.Width(line); got != 14 {
			t.Fatalf("line %d is %d cells wide, want the tile's 14", i, got)
		}
		if want := "  ▀▀▀▀▀▀▀▀▀▀  "; line != want {
			t.Errorf("line %d = %q, want %q", i, line, want)
		}
	}

	// And a picture that fills the tile is left exactly as it arrived: measuring
	// and re-cutting every line of a wall of covers is what this was written to
	// avoid.
	full := coverState{want: 4}
	full.took("▀▀▀▀")
	if full.lines[0] != "▀▀▀▀" {
		t.Errorf("a picture that fits came out as %q", full.lines[0])
	}
}
