package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// The field a list is searched in.
//
// A box over the head of the list rather than a line at the foot of the screen,
// where the query used to be written among the notices: there it read as one more
// thing the program had to say, rather than as the one place the keyboard was
// going.
//
// So it is drawn the way the pointer is — same pen, same rounded corners, same
// ground a step off the screen's — over rows the list has stood aside for. It
// covers nothing: the table steps down by the height of the box while a search
// is open and steps back up when it closes, so what a search costs is three rows
// of the list for as long as it is being made, and nothing at all after.
//
// What it holds is the query and how many rows matched, because a query that
// matches nothing and one that matches everything look the same on a list you
// cannot see the end of. Where they matched is not its business — that is what
// the marks in the rows are for. See find.go.

const (
	// finderRows is how tall the box is: the two edges and the line between.
	finderRows = 3

	// finderPad is the air inside the frame, and finderLeast the narrowest the
	// box is drawn at. Under that there is no room for a query and a count both,
	// and a box that shows one without the other is not worth the rows.
	finderPad   = 2
	finderLeast = 28

	// finderGap is the least air between what has been typed and the count, so
	// the two never read as one line of text.
	finderGap = 4

	// finderCaret is where the next letter will go.
	finderCaret = "▌"
)

// drawFinder lays the field over rows already laid out, given the row the body
// starts at.
func (m Model) drawFinder(lines []string, l layout, top int) []string {
	if !m.finding() {
		return lines
	}

	// The width of the list it searches, and the block's width rather than a
	// row's: the count beside the heading is set to the block, and a box a row
	// wide left the tail of it sticking out.
	//
	// The whole width rather than cut to its contents, because a box the size of
	// four letters floating in the middle of the heading reads as something
	// dropped on the screen. At the list's width it is the head of the list while
	// the search is open, which is what it is.
	box := min(queueBlockWidth(l), l.interior-leftMargin-rightMargin)
	if box < finderLeast {
		return lines
	}
	left := leftMargin

	// Directly over the heading, in the rows the block stood aside for — under
	// the band and the blank that goes with it. See listChrome.
	band := m.listBandRows(l)
	boxTop := top + band
	if band > 0 {
		boxTop += listBandGap
	}
	if boxTop < 0 || boxTop+finderRows > len(lines) {
		return lines
	}
	typed, count := m.finderParts()

	rule := strings.Repeat(pointerH, box-2)
	out := append([]string(nil), lines...)
	put := func(row int, s string) {
		out[row] = overwrite(out[row], left, s, l.interior)
	}
	edge := m.styles.Rule
	put(boxTop, edge.Render(pointerTL+rule+pointerTR))
	put(boxTop+1, edge.Render(pointerV)+finderLine(typed, count, box-2)+edge.Render(pointerV))
	put(boxTop+2, edge.Render(pointerElbow+rule+pointerBR))

	// No ground inside it. The rows it stands in are the list's own, held empty
	// for it, so there is nothing under the box to cover — and a panel painted a
	// shade off the screen, on a screen with a photograph at the top of it, reads
	// as a second window rather than as part of this one.
	return out
}

// finderTakes is what the field costs the list it is searching: its own height
// while it is up, and nothing when it is not.
func (m Model) finderTakes() int {
	if !m.finding() {
		return 0
	}
	return finderRows
}

// finding reports whether the field is up: a query being written, or one written
// and not yet let go of — its marks are still in the list and n still walks them,
// so the field is still what the screen is doing.
func (m Model) finding() bool {
	if !m.findable() || m.devices.open || m.actions.open {
		return false
	}
	return m.find.typing || m.find.query != ""
}

// finderParts is what the field holds: the prompt and what has been typed into
// it, and how it did.
func (m Model) finderParts() (typed, count string) {
	switch {
	case m.find.query == "":
	case len(m.find.matches) == 0:
		count = m.styles.Empty.Render("no match")
	default:
		count = m.styles.Album.Render(fmt.Sprintf("%d of %d", m.find.at+1, len(m.find.matches)))
	}

	// The caret only while the keyboard belongs to the field. Once the query is
	// let go of, the box is a report on what it found rather than somewhere to
	// type, and a caret sitting in it would be a lie about where the next key
	// goes.
	caret := ""
	if m.find.typing {
		caret = m.styles.Cursor.Render(finderCaret)
	}
	return m.styles.QueryPrompt.Render("/") + "  " + m.styles.Query.Render(m.find.query) + caret, count
}

// finderLine sets those two in the width the box settled on, the query from the
// left and the count from the right.
func finderLine(typed, count string, w int) string {
	pad := strings.Repeat(" ", finderPad)
	room := w - 2*finderPad - lipgloss.Width(count) - finderGap
	return pad + padRight(fit(typed, max(room, 1)), max(room, 1)) +
		strings.Repeat(" ", finderGap) + count + pad
}
