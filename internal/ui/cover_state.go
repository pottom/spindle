package ui

import "image/color"

// coverState is what the view needs to know about the artwork area: which URL
// and cell size it was rendered for, the art itself, the colour the cover reads
// as, and whether the attempt failed.
// cursorSlot is the picture of what the cursor is resting on. It is the
// renderer's slot, which is why it is a number — see gridSlotFrom for the wall's.
const cursorSlot = 0

type coverState struct {
	url           string
	width, height int

	art       string
	accent    color.RGBA
	hasAccent bool
	failed    bool
}

// loading reports whether the spinner belongs in the artwork area.
func (c coverState) loading() bool {
	return c.url != "" && c.art == "" && !c.failed
}

// matches reports whether a result belongs to what is currently on screen.
func (c coverState) matches(url string, w, h int) bool {
	return c.url == url && c.width == w && c.height == h
}
