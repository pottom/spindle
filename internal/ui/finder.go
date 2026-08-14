package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// The field a list is searched in.
//
// It floats over the list rather than standing in a row of its own, which is the
// whole of why it is drawn this way: a field with a row of its own would have to
// take that row from somewhere, and every list on this screen is already as long
// as the terminal allows. Opening a search to look through a list, and losing a
// line of the list to do it, is paying in the currency you came to spend.
//
// So it is laid over the rows the way the pointer is laid over the band — same
// pen, same rounded corners, same ground a step off the screen's — and the rows
// under it are still there the moment it closes. It stands over the top of the
// list, and moves to the foot of it when the row the cursor found is one of the
// three it would otherwise cover: a search that hides what it found has answered
// a question by taking the answer away.
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

	rows := m.visibleListRows()
	if rows <= finderRows {
		// The box would be the list. On a terminal this short the query goes
		// unshown rather than the rows it is searching.
		return lines
	}

	width := min(queueRowWidth(l), l.interior-leftMargin-rightMargin)
	if width < finderLeast {
		return lines
	}

	// As wide as what is in it, between a floor and a ceiling. A field the width
	// of the list would be a banner across the screen for four letters; one cut
	// to the letters would twitch on every key. The floor is where it opens and
	// stays for any ordinary query, and past that it grows.
	typed, count := m.finderParts()
	box := lipgloss.Width(typed) + lipgloss.Width(count) + finderGap + 2*finderPad + 2
	box = min(max(box, finderLeast), max(width*2/3, finderLeast))
	left := leftMargin + (width-box)/2

	// Over the head of the list, unless the row it found is up there — and then
	// over the foot of it, which is the foot of the rows a list actually has
	// rather than of the room it was given: a box floating in the blank under a
	// list of eight is a box floating in nothing.
	bodyTop := top + m.listBandRows(l) + listChromeRows
	row, filled := m.finderAt(rows)
	if filled < finderRows {
		return lines
	}

	boxTop := bodyTop
	foot := bodyTop + filled - finderRows
	if at := bodyTop + row; at >= boxTop && at < boxTop+finderRows &&
		(at < foot || at >= foot+finderRows) {
		boxTop = foot
	}
	if boxTop < 0 || boxTop+finderRows > len(lines) {
		return lines
	}

	rule := strings.Repeat(pointerH, box-2)
	out := append([]string(nil), lines...)
	put := func(row int, s string) {
		out[row] = overwrite(out[row], left, s, l.interior)
	}
	edge := m.styles.Rule
	put(boxTop, edge.Render(pointerTL+rule+pointerTR))
	put(boxTop+1, edge.Render(pointerV)+finderLine(typed, count, box-2)+edge.Render(pointerV))
	put(boxTop+2, edge.Render(pointerElbow+rule+pointerBR))

	// The ground last, over the whole of it: a card lying on the screen rather
	// than a frame drawn on it. It goes on after the letters because a style
	// wrapped round text that carries styles of its own does not survive them —
	// see raise, which puts the ground back after every reset inside the span.
	for row := boxTop; row < boxTop+finderRows; row++ {
		out[row] = raise(out[row], left, left+box, m.styles.Raised, l.interior)
	}
	return out
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

// finderAt is how far down the visible list the cursor is sitting, and how many
// of the rows on screen have anything in them.
func (m Model) finderAt(rows int) (row, filled int) {
	state, count := (&m).listCursor()
	if state == nil {
		return -1, 0
	}
	from, to := state.window(count, rows)
	return state.cursor - from, to - from
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
