package ui

import (
	"strings"
)

// The box the catalogue is searched from.
//
// The same box a list is searched with — see finder.go — because it is the same
// act: somewhere to type, with what it found set against it. It was a bare line
// before, which made the one screen that is a field into the one screen that did
// not look like one.
//
// It stands at the head of the results rather than in a band of its own. What
// this screen is about is the query, and a heading belongs where the eye already
// goes for one.

// searchPrompt is the mark the field wears, and the one the query is drawn
// after wherever it is quoted back — see searchDetail.
const searchPrompt = "⌕"

// searchBoxRows is how much of the screen it takes: the two edges and the line
// between, exactly as the finder's box, and a row of air under it.
//
// The air is not decoration. The mark that says which row the band belongs to
// hangs its head in the blank row above the band — see pointAtCursor — and
// without one it was drawn over the box's own foot, which left the box open at
// the bottom.
const searchBoxRows = finderRows + 1

// searchBox draws it, at the width of the block underneath.
func (m Model) searchBox(w int) []string {
	if w < finderLeast {
		return nil
	}

	edge := m.styles.Rule
	rule := strings.Repeat(pointerH, w-2)

	// The field draws itself: the caret is its own, and so is the window it keeps
	// on a query longer than the room. All the box adds is the frame.
	//
	// Nothing else in it. What the query matched used to be set against the field
	// from the right, which put the names of the views a screen's width away from
	// the list they change — see searchKinds, which draws them over it now.
	return []string{
		edge.Render(pointerTL + rule + pointerTR),
		edge.Render(pointerV) + finderLine(m.searchField(searchRoom(w)), "", w-2) + edge.Render(pointerV),
		edge.Render(pointerElbow + rule + pointerBR),
		strings.Repeat(" ", w),
	}
}

// searchRoom is how much of the box the field itself has: the frame, the air
// inside it, and the gap finderLine keeps at the right.
func searchRoom(w int) int { return max(w-2-2*finderPad-finderGap, 8) }

// searchViewSpans is where each of the views sits on the row over the list, so
// a press on one can be answered by turning to it.
//
// Read back from the same pieces the row is drawn from — the labels and the gap
// between them, set flush left against the margin every block here keeps. See
// searchKinds, which draws them, and kindSpans, which does this for the
// library's own.
func (m Model) searchViewSpans() []span {
	labels := m.viewLabels()
	if len(labels) == 0 {
		return nil
	}
	return labelSpans(labels, len(kindGap), leftMargin)
}
