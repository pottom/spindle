package ui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// listState is the cursor and scroll position of a list. It holds no items:
// the panes own those, and a list that outlives its contents is a bug waiting
// to happen.
type listState struct {
	cursor int
	top    int // index of the first visible row
}

// move shifts the cursor by delta over count items, clamping rather than
// wrapping — wrapping in a long list loses the user.
func (l *listState) move(delta, count int) {
	if count == 0 {
		l.cursor, l.top = 0, 0
		return
	}
	l.cursor = min(max(l.cursor+delta, 0), count-1)
}

// moveTo puts the cursor on one row, which is what a search through the list
// hands back: a place rather than a direction.
func (l *listState) moveTo(row, count int) {
	if count == 0 {
		l.cursor, l.top = 0, 0
		return
	}
	l.cursor = min(max(row, 0), count-1)
}

// reset returns to the top, for when the underlying items are replaced.
func (l *listState) reset() {
	l.cursor, l.top = 0, 0
}

// window returns the slice of indices to draw, scrolling only as far as it must
// to keep the cursor in view.
func (l *listState) window(count, height int) (from, to int) {
	if count == 0 || height <= 0 {
		return 0, 0
	}

	l.cursor = min(max(l.cursor, 0), count-1)
	l.top = min(max(l.top, 0), max(count-height, 0))

	if l.cursor < l.top {
		l.top = l.cursor
	}
	if l.cursor >= l.top+height {
		l.top = l.cursor - height + 1
	}
	return l.top, min(l.top+height, count)
}

// listChromeRows is what a list block spends on itself above the rows, once the
// artwork has taken its share: a blank line, the heading, and another blank.
const listChromeRows = 3

// listBodyRows is how many rows of the list itself a block of the given height
// has room for. listBlock draws exactly this many and the page keys move by
// exactly this many, which is the only way a page can mean a screenful.
func listBodyRows(height, artHeight int) int {
	return max(height-min(artHeight, height)-listChromeRows, 0)
}

// visibleListRows is how many rows the list on screen is showing. Only the view
// truly knows, and the view may not say: it is a pure function and writing the
// number down as it drew would make it something else. So both sides derive it
// from the layout instead, through the same arithmetic — the view to fill the
// rows, the keys to move by them.
func (m Model) visibleListRows() int {
	if !fitsMinimum(m.width, m.height) {
		return 0
	}
	// The device list is drawn whole rather than windowed, so its page is
	// however many devices there are: one press reaches the end, which is what
	// a page down means when the whole list is one page.
	if m.devices.open || (m.tab == tabPlayer && m.noDevice) {
		return len(m.devices.items)
	}

	l := m.layout()
	return listBodyRows(max(l.bodyHeight, 1), l.artHeight)
}

// listKey applies the movement every list shares: a row, a page, or all the way
// to an end. It reports whether the key was one of those, so each screen can go
// on to its own keys.
//
// vim says whether g and G are read here. On the search tab they are not: every
// printable key belongs to the query, and a key that types on one screen must
// not navigate on it.
func (m *Model) listKey(k tea.KeyPressMsg, state *listState, count int, vim bool) bool {
	page := max(m.visibleListRows(), 1)

	switch {
	case key.Matches(k, m.keys.Down):
		state.move(1, count)
	case key.Matches(k, m.keys.Up):
		state.move(-1, count)
	case key.Matches(k, m.keys.PageDown):
		state.move(page, count)
	case key.Matches(k, m.keys.PageUp):
		state.move(-page, count)
	case key.Matches(k, m.keys.HalfDown):
		state.move(max(page/2, 1), count)
	case key.Matches(k, m.keys.HalfUp):
		state.move(-max(page/2, 1), count)
	// The ends are asked for as a move the length of the list, which clamps to
	// them however long it is.
	case key.Matches(k, m.keys.First), vim && key.Matches(k, m.keys.FirstVim):
		state.move(-count, count)
	case key.Matches(k, m.keys.Last), vim && key.Matches(k, m.keys.LastVim):
		state.move(count, count)
	default:
		return false
	}
	return true
}
