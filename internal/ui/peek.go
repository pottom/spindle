package ui

import "strings"

const (
	// peekRows is how many tracks the glance shows. Four is enough to know what
	// is coming without becoming a second queue screen.
	peekRows = 4

	// peekChrome is the heading and the blank row under the list.
	peekChrome = 2
)

// peekState is the glance at what is coming next, shown above the artwork.
//
// Read only on purpose: this is for knowing what follows, not for changing it.
// Editing lives on the queue tab, where there is room to see what a change did.
type peekState struct {
	// on is what the key toggles. Off to begin with, like the words: the player
	// screen is about what is playing.
	on bool
}

// peekAvailable reports whether the glance fits in the band above the artwork.
func (m Model) peekAvailable() bool {
	if m.tab != tabPlayer || m.noDevice || m.ps == nil {
		return false
	}
	l := m.layout()
	return m.artTop(l, l.bodyHeight) >= peekRows+peekChrome
}

// peekVisible reports whether the glance is on screen right now.
func (m Model) peekVisible() bool { return m.peek.on && m.peekAvailable() }

// drawPeek writes the glance into the blank rows above the artwork, leaving
// everything below exactly where it was.
func (m Model) drawPeek(body []string, l layout) []string {
	if len(body) < peekRows+peekChrome {
		return body
	}
	// A column in from the frame's own margin. Flush with the artwork and the
	// device name, the glance reads as part of the chrome rather than as
	// something laid on top of the screen.
	const inset = 1
	w := l.interior - leftMargin - rightMargin - inset
	indent := strings.Repeat(" ", inset)

	for i, row := range m.place(m.upNextBlock(), w, peekRows+1) {
		body[i] = m.pad(indent+row, l)
	}
	return body
}

func peekSubtitle(n int) string {
	switch {
	case n == 0:
		return "nothing queued"
	case n <= peekRows:
		return ""
	default:
		return "and more"
	}
}
