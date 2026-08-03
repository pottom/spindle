package ui

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
