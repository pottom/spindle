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
//
// It starts one row above the body, in the blank the frame keeps between the
// header and everything else — which is where every other screen's first line
// sits, and where this one looked a row low.
func (m Model) drawPeek(lines []string, at int, l layout) []string {
	if len(lines)-at < peekRows+peekChrome {
		return lines
	}
	// A column in from the frame's own margin. Flush with the artwork and the
	// device name, the glance reads as part of the chrome rather than as
	// something laid on top of the screen.
	const inset = 1
	w := l.interior - leftMargin - rightMargin - inset
	indent := strings.Repeat(" ", inset)

	// The title column is held to the picture's width, so the artists below the
	// heading start on the same column the track's own artists do further down
	// the screen. Left to the row's own arithmetic they landed wherever the
	// division fell, which is a column that lines up with nothing.
	glance := m
	glance.rowsMainAt = max(l.artWidth+columnGap-inset-rowGutter-1, 0)

	for i, row := range glance.place(glance.upNextBlock(), w, peekRows+1) {
		lines[at+i] = m.pad(indent+row, l)
	}
	return lines
}

// peekSubtitle is what goes beside the heading. Only the empty case says
// anything: "and more" was true of almost every queue there is, so it was a
// word that never varied and never told anybody anything.
func peekSubtitle(n int) string {
	if n == 0 {
		return "nothing queued"
	}
	return ""
}
