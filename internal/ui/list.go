package ui

import (
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
// artwork has taken its share: a blank line under the band, the heading, the
// names of the columns, and the line under those.
const listChromeRows = 4

// listChrome is that, given the band over it, and the field where a list is
// being searched.
//
// The blank goes with the band. It is the air under the picture, and folded away
// there is no picture for it to be under — it was a row of nothing between the
// top of the screen and the table, which on the screen that is only a table is
// the one row nobody asked for.
//
// The field takes its rows rather than lying over them. It lay over the heading
// first, which cost nothing and hid the heading — and a list you are searching
// is a list whose heading says which list it is and what its columns are, which
// is worth more while you are reading it through than at any other time. So the
// table steps down and makes room, and steps back up when the search closes.
func (m Model) listChrome(band int) int {
	rows := listChromeRows
	if band <= 0 {
		rows -= listBandGap
	}
	return rows + m.finderTakes()
}

// listBandGap is that blank: the air between the band and the heading under it.
const listBandGap = 1

// listBodyRows is how many rows of the list itself a block of the given height
// has room for. listBlock draws exactly this many and the page keys move by
// exactly this many, which is the only way a page can mean a screenful.
func (m Model) listBodyRows(height, artHeight int) int {
	return max(height-min(artHeight, height)-m.listChrome(artHeight), 0)
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
	return m.listBodyRows(max(l.bodyHeight, 1), m.listBandRows(l))
}

// listBandRows is how tall the band above a list is: the artwork's height, and
// nothing at all where both of the blocks in it have been folded away.
//
// Read by the view to draw it and by the keys to page by it. Neither may work it
// out for itself — the view is a pure function and may not write down what it
// drew, so the two derive the same number from the same place or they disagree
// about how far a page is. See visibleListRows.
func (m Model) listBandRows(l layout) int {
	if m.tab == tabQueue && m.open() == nil && !m.devices.open && m.queuePane.room == queueRoomList {
		return 0
	}

	// The picture's own height rather than the box the layout gave it. A cover
	// keeps its shape inside that box and so comes out a row shorter than it,
	// and the band was the box: a blank row under the picture that lined up with
	// nothing, and a frame drawn round the band standing clear of the thing it
	// was drawn round. The row goes to the list instead.
	if l.artRows > 0 {
		return l.artRows
	}
	return l.artHeight
}

// listLoading reports whether the list on screen is waiting for a page. It is
// what the spinner beside the heading turns for, and what stops a list that has
// not arrived yet from claiming to be empty.
func (m Model) listLoading() bool {
	switch {
	case m.open() != nil:
		return m.open().pages.loading
	case m.tab == tabLibrary:
		return m.library.pages[m.library.kind].loading
	case m.tab == tabSearch:
		return m.search.current().pages.loading
	default:
		return false
	}
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
	case m.pressed(k, m.keys.Down):
		state.move(1, count)
	case m.pressed(k, m.keys.Up):
		state.move(-1, count)
	case m.pressed(k, m.keys.PageDown):
		state.move(page, count)
	case m.pressed(k, m.keys.PageUp):
		state.move(-page, count)
	case m.pressed(k, m.keys.HalfDown):
		state.move(max(page/2, 1), count)
	case m.pressed(k, m.keys.HalfUp):
		state.move(-max(page/2, 1), count)
	// The ends are asked for as a move the length of the list, which clamps to
	// them however long it is.
	case m.pressed(k, m.keys.First), vim && m.pressed(k, m.keys.FirstVim):
		state.move(-count, count)
	case m.pressed(k, m.keys.Last), vim && m.pressed(k, m.keys.LastVim):
		state.move(count, count)
	default:
		return false
	}
	return true
}
