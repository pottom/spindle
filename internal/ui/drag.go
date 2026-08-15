package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Taking hold of a bar and moving it.
//
// The one gesture a terminal can report that a key cannot imitate: press, move,
// release. `MouseModeCellMotion` already sends the motion between the two, so
// nothing had to be turned on for this — what was missing is the holding.
//
// While a bar is held, the screen shows where the pointer is rather than where
// the track is. That is the whole of why this is worth having over a click: you
// can see where you are going to land before you land there, and let go
// somewhere else if it is not where you meant. Nothing is sent until the button
// comes up, so a scrub across the whole track is one request rather than forty.
//
// A plain click is the same gesture with no motion in the middle of it, which is
// why there is no separate path for one.

// dragState is what the pointer has hold of.
type dragState struct {
	on   bool
	kind spotKind // spotSeek or spotVolume

	// at is how far along the bar the pointer is, in the bar's own cells, and w
	// how many of those there are. Cells rather than a fraction because a cell
	// is what was pressed — the bar is drawn from the same two numbers, so the
	// knob lands under the pointer and not a rounding away from it.
	at, w int
}

// barSpan is where a bar the pointer can take hold of stands on the screen: the
// column it starts at, and how many cells long it is.
//
// Derived, like everything else the pointer is answered from, and derived again
// on every report rather than remembered from the press: a window resized
// mid-drag would otherwise go on scrubbing against a bar that is no longer
// there.
func (m Model) barSpan(l layout, kind spotKind) (at, w int, ok bool) {
	if m.ps == nil || m.noDevice {
		return 0, 0, false
	}
	left := leftMargin
	if l.hasArt() {
		left += l.artWidth + columnGap
	}

	switch kind {
	case spotSeek:
		return left, barCells(l.infoWidth), m.loaded()
	case spotVolume:
		v := m.volumeSpan(l.infoWidth)
		return left + v.at, barCells(v.w), true
	}
	return 0, 0, false
}

// takeHold starts a drag on the bar that was pressed.
func (m *Model) takeHold(kind spotKind, cell int) {
	if kind == spotScroll {
		m.drag = dragState{on: true, kind: kind}
		return
	}
	_, w, ok := m.barSpan(m.layout(), kind)
	if !ok {
		return
	}
	m.drag = dragState{on: true, kind: kind, at: min(max(cell, 0), w), w: w}
}

// mouseMotion follows the pointer while it has hold of something.
//
// Only the one direction the bar runs in matters: the pointer is free to wander
// across it mid-drag without letting go, which is what every slider anywhere
// does and what stops a scrub from being a test of aim.
func (m Model) mouseMotion(e tea.MouseMotionMsg) (Model, tea.Cmd) {
	if !m.drag.on || e.Button == tea.MouseNone {
		return m, nil
	}
	m.followBar(e.X, e.Y)
	return m, nil
}

// followBar puts the hold where the pointer is, along the bar being held.
func (m *Model) followBar(x, y int) {
	// The scrollbar is the one that runs down rather than across, and the one
	// that acts as it is dragged rather than when it is let go: there is nothing
	// to send, only a cursor to move, and a list that only jumped once the
	// button came up would be a list you were dragging blind.
	if m.drag.kind == spotScroll {
		m.followScroll(y)
		return
	}

	at, w, ok := m.barSpan(m.layout(), m.drag.kind)
	if !ok {
		m.drag = dragState{}
		return
	}
	m.drag.at, m.drag.w = min(max(x-at, 0), w), w
}

// followScroll puts the cursor where the thumb has been dragged to: the top of
// the bar is the first row of the list and the foot of it the last, which is
// what a scrollbar means everywhere.
func (m *Model) followScroll(y int) {
	l := m.layout()
	band := m.listBandRows(l)
	head := tabBarHeight + band + m.listChrome(band)
	body := m.listBodyRows(max(l.bodyHeight, 1), band)

	cursor, count := m.rowCursor()
	if cursor == nil || body <= 1 || count == 0 {
		m.drag = dragState{}
		return
	}

	at := min(max(y-head, 0), body-1)
	m.drag.at, m.drag.w = at, body-1
	cursor.moveTo(at*(count-1)/(body-1), count)
}

// mouseRelease lets go, and that is when what was chosen is sent.
func (m Model) mouseRelease(e tea.MouseReleaseMsg) (Model, tea.Cmd) {
	if !m.drag.on {
		return m, nil
	}
	// Where the button came up, which is not always somewhere a motion was
	// reported: a terminal reports a move when the pointer changes cell, and the
	// last thing it does is let go. What was let go of is what is sent.
	m.followBar(e.X, e.Y)

	held := m.drag
	m.drag = dragState{}
	if !held.on {
		return m, nil
	}

	switch held.kind {
	case spotScroll:
		// Nothing to send: the list moved as it was dragged.
		return m, tea.Batch(m.previewCover(), m.readAhead())
	case spotSeek:
		if !m.loaded() {
			return m, nil
		}
		return m, m.seek(atFraction(held.at, held.w, m.ps.Duration))
	case spotVolume:
		return m, m.setVolume(atShare(held.at, held.w, 100))
	}
	return m, nil
}

// playhead is where the bar draws the track as having got to: under the pointer
// while the pointer has hold of it, and where the track really is otherwise.
func (m Model) playhead() time.Duration {
	if m.drag.on && m.drag.kind == spotSeek && m.ps != nil {
		return atFraction(m.drag.at, m.drag.w, m.ps.Duration)
	}
	return m.elapsed()
}

// heldVolume is the same for the meter.
func (m Model) heldVolume() int {
	if m.drag.on && m.drag.kind == spotVolume {
		return atShare(m.drag.at, m.drag.w, 100)
	}
	if m.ps == nil {
		return 0
	}
	return m.ps.Volume
}
