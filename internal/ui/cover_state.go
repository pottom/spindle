package ui

import "image/color"

// coverState is what the view needs to know about the artwork area: which URL
// and cell size it was rendered for, the art itself, the colour the cover reads
// as, and whether the attempt failed.
// cursorSlot and nowSlot are the two pictures a screen may show at once: what
// the cursor is resting on, and what is playing. They are the renderer's slots,
// which is why they are numbers.
const (
	cursorSlot = 0
	nowSlot    = 1
)

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
