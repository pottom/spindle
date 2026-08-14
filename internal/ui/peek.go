package ui

import "strings"

const (
	// peekRows is how many tracks the glance shows at most. Four is enough to
	// know what is coming without becoming a second queue screen.
	peekRows = 4

	// peekLeast is how few it will still bother with. Two rows of what is coming
	// is a glance; one is a caption on the queue.
	peekLeast = 2

	// peekChrome is what the block spends on itself: the names of its columns,
	// the line under them, and the blank between the list and the artwork.
	peekChrome = 3
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
	return m.artTop(l, l.bodyHeight) >= peekLeast+peekChrome
}

// peekVisible reports whether the glance is on screen right now.
func (m Model) peekVisible() bool { return m.peek.on && m.peekAvailable() }

// drawPeek writes the glance into the blank rows above the artwork, leaving
// everything below exactly where it was.
//
// It starts at the top of the body, a row under the blank the frame keeps
// between the header and everything else. It stood in that blank for a while and
// read as hanging off the header rather than as the first thing on the screen.
func (m Model) drawPeek(lines []string, at int, l layout) []string {
	// As many tracks as the blank above the artwork can hold, up to four. It was
	// four or nothing, which on the terminals where the room is five or six rows
	// meant nothing — and the glance is worth having at three.
	//
	// The room is measured to the top of the picture rather than to the end of
	// the rows laid out so far: the picture is in those rows, and a block that
	// counted them as room would draw itself over the sleeve.
	room := min(m.artTop(l, l.bodyHeight), len(lines)-at)
	if room < peekLeast+peekChrome {
		return lines
	}
	tall := min(room-1, peekRows+peekChrome-1)
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
	//
	// The row starts at leftMargin+inset and the column beside the picture at
	// leftMargin+artWidth+columnGap, so the title has to be artWidth+columnGap-
	// inset-1 wide for the space after it to land on that column. No gutter in
	// the arithmetic: this list is flush, having no cursor to stand anywhere.
	glance.rowsMainAt = max(l.artWidth+columnGap-inset-1, 0)

	for i, row := range glance.place(glance.upNextBlock(), w, tall) {
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
