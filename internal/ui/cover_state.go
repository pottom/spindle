package ui

import (
	"image/color"
	"strings"
)

// coverState is what the view needs to know about the artwork area: which URL
// and cell size it was rendered for, the art itself, the colour the cover reads
// as, and whether the attempt failed.
// cursorSlot is the picture of what the cursor is resting on. It is the
// renderer's slot, which is why it is a number — see gridSlotFrom for the wall's.
const cursorSlot = 0

type coverState struct {
	url           string
	width, height int

	art string

	// lines is that art split into rows and padded to the width it was asked
	// for, done once when it arrives.
	//
	// Because measuring it is not cheap. A picture drawn the way a terminal that
	// can draw pictures wants it is a placeholder character, two combining marks
	// and a colour per cell, and asking how wide that is walks every byte of it
	// through an escape-sequence parser. Measured on a wall of sixty covers, one
	// frame came to two hundred milliseconds and a hundred and ten megabytes —
	// nearly all of it re-measuring strings whose width was already known.
	lines []string

	accent    color.RGBA
	hasAccent bool
	failed    bool
}

// loading reports whether the spinner belongs in the artwork area.
func (c coverState) loading() bool {
	return c.url != "" && c.art == "" && !c.failed
}

// took files arriving art, split into rows once so that drawing it is a
// concatenation.
//
// Split only — not squared off to the width it was asked for. What comes back is
// as wide as the renderer could draw it, which is a cell or two under the box on
// most cell shapes, and padding it to the box would put those cells on one side
// of the picture. Whoever lays it out pads to the width it really is.
func (c *coverState) took(art string) {
	c.art = art
	c.lines = strings.Split(art, "\n")
}

// matches reports whether a result belongs to what is currently on screen.
func (c coverState) matches(url string, w, h int) bool {
	return c.url == url && c.width == w && c.height == h
}
