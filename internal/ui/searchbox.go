package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
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

// searchBoxRows is how tall it is: the two edges and the line between, exactly
// as the finder's box.
const searchBoxRows = finderRows

// searchBox draws it, at the width of the block underneath.
func (m Model) searchBox(w int) []string {
	if w < finderLeast {
		return nil
	}

	edge := m.styles.Rule
	rule := strings.Repeat(pointerH, w-2)

	// The field draws itself: the caret is its own, and so is the window it
	// keeps on a query longer than the room. All that is added here is the frame
	// and what the query matched, set against it from the right.
	kinds := m.searchKinds()
	room := w - 2 - 2*finderPad - lipgloss.Width(kinds) - finderGap

	return []string{
		edge.Render(pointerTL + rule + pointerTR),
		edge.Render(pointerV) + finderLine(m.searchField(max(room, 8)), kinds, w-2) + edge.Render(pointerV),
		edge.Render(pointerElbow + rule + pointerBR),
	}
}

// searchViewSpans is where each of the counts sits inside the box, so a press on
// one can be answered by turning to that view.
//
// Read back from the same pieces the line is drawn from — the labels, the gap
// between them, and the width they were set into from the right. See
// finderLine, which does the setting.
func (m Model) searchViewSpans(w int) []span {
	labels := m.viewLabels()
	if len(labels) == 0 || w < finderLeast {
		return nil
	}

	line := strings.Join(labels, kindGap)
	// The frame, the padding inside it, and then the labels set flush right.
	at := 1 + (w - 2) - finderPad - lipgloss.Width(line)

	return labelSpans(labels, len(kindGap), at)
}
