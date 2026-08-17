package ui

import (
	"image/color"
	"strings"
	"time"
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

	// want is the width whatever is laying this out has reserved for it. What
	// comes back from a renderer can be narrower — a square picture in whole
	// cells is at the mercy of the cell's own shape, and every renderer rounds
	// its own way — and a line that is not the width the layout believes it is
	// slides everything after it along the row.
	want int

	art string

	// slot is the renderer slot this tile's picture was sent to. It belongs to
	// the tile rather than to where the tile sits, because a slot is one picture
	// to the terminal: two tiles holding the same slot is one picture drawn
	// twice, under whichever name was uploaded last. Nought is none yet.
	slot int

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

	// failedAt is when it failed, because a failure is not for ever. A picture
	// can be missed for a moment — a request that timed out, a format nothing
	// here could read until it could — and a tile that never asks again is a
	// hole in the wall that only a resize will mend. See coverRetryAfter.
	failedAt time.Time
}

// loading reports whether the spinner belongs in the artwork area.
func (c coverState) loading() bool {
	return c.url != "" && c.art == "" && !c.failed
}

// took files arriving art, split into rows and squared off to the width it is
// being laid out in, once, so that drawing it is a concatenation.
//
// Squared off to that width and not to the box it was asked for: the two are not
// the same, and the layout only ever knows the first. A line an inch short of
// what the layout believes shifts every tile after it along the row, and the
// marks drawn at reckoned columns — the frame round a tile, the cursor's own —
// then stand beside nothing.
func (c *coverState) took(art string) {
	c.art = art
	c.lines = strings.Split(art, "\n")
	if c.want <= 0 {
		return
	}
	for i, line := range c.lines {
		c.lines[i] = fit(line, c.want)
	}
}

// matches reports whether a result belongs to what is currently on screen.
// coverRetryAfter is how long a picture that would not come is left alone
// before it is asked for again. Long enough that a wall of missing covers is
// not a wall of requests, short enough that somebody who looks back at the
// screen sees it mended.
const coverRetryAfter = 30 * time.Second

// worthRetrying reports that this tile failed long enough ago to be worth
// asking about again.
func (c coverState) worthRetrying() bool {
	return c.failed && time.Since(c.failedAt) >= coverRetryAfter
}

func (c coverState) matches(url string, w, h int) bool {
	return c.url == url && c.width == w && c.height == h
}
