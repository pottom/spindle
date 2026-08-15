package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The boxes that stand over the screen: the menu of verbs, and the list of
// machines the music can be moved to.
//
// One shape for both, because they are one thing — a short list to choose from,
// about something named at the top of it, standing over whatever you were
// looking at rather than in place of it. The verbs open at the row or the cover
// they are about; the devices open under the name of the device in the header,
// which is the thing they change.
//
// The menu of verbs, drawn where the thing it is about is.
//
// It used to stand in the list's own rows, on the reasoning that this screen has
// no boxes and a panel floating over a list would need one to be legible. Two
// things have happened since: the screen has boxes now — the field a list is
// searched in and the frame round the cover under the cursor are both drawn with
// this pen — and the library stopped being a list. On a wall of covers there
// were no rows to stand in, so the menu opened, swallowed every key, and drew
// nothing at all. Measured, not reasoned: `.` on the wall and then j, and the
// frame did not move.
//
// So it is a box now, opened at the thing it is about: under the row on a list,
// beside the cover on the wall, and at the pointer when the pointer is what
// opened it. It covers what is under it — the cells are written over, so the
// picture behind it goes — which is what makes it readable without a ground of
// its own.

const (
	// menuPad is the air inside the box, left and right of the words.
	menuPad = 1

	// menuLeast is the narrowest it will be drawn: under this a verb is cut to
	// something nobody could choose from.
	menuLeast = 24

	// menuChrome is what the box spends on itself: the two edges, the name of
	// what it is about, the line under that, and the line it stands on.
	menuChrome = 5
)

// menuBox is where a box ends up and how big it is.
type menuBox struct {
	x, y, w, h int
}

// popup is what one of them holds: where it was opened, what it is about, and
// the rows to choose from — already styled, because which of them is marked is
// the caller's business.
type popup struct {
	x, y     int
	title    string
	subtitle string
	rows     []string
}

// openPopup is whichever box is up, and whether one is.
//
// The menu of verbs outranks the picker, because the menu is opened over
// whatever is already there and the picker is not opened over the menu.
func (m Model) openPopup() (popup, bool) {
	switch {
	case m.actions.open && len(m.actions.verbs) > 0:
		return popup{
			x: m.actions.x, y: m.actions.y,
			title: m.actions.title, subtitle: m.actions.subtitle,
			rows: m.verbLines(0),
		}, true
	case m.devices.open:
		return m.devicesPopup(), true
	}
	return popup{}, false
}

// menuShape measures the box for the verbs it holds and fits it on the screen.
//
// Opened at a point and moved from there only as far as it must: a menu that
// would run off the right edge is pulled back to it, and one that would run off
// the foot opens upwards instead — which is what every menu on every desktop
// does, and so the one thing nobody has to be told.
func (m Model) menuShape(l layout, p popup) menuBox {
	want := menuLeast
	for _, line := range append([]string{p.title, p.subtitle}, p.rows...) {
		want = max(want, lipgloss.Width(line)+2*menuPad+2)
	}

	left, right := leftMargin, l.interior-rightMargin
	w := min(want, right-left)
	h := len(p.rows) + menuChrome

	x := min(max(p.x, left), right-w)

	// The body's own rows, which is where a menu about something on the body
	// belongs: over the tabs it would read as belonging to them.
	top, foot := tabBarHeight, tabBarHeight+l.bodyHeight
	y := p.y
	if y+h > foot {
		// Upwards from the point rather than pinned to the foot, so the corner
		// stays with the thing it was opened on.
		y = p.y - h
	}
	y = min(max(y, top), max(foot-h, top))
	return menuBox{x: x, y: y, w: w, h: h}
}

// drawMenu writes it over rows already laid out.
func (m Model) drawMenu(lines []string, l layout) []string {
	p, ok := m.openPopup()
	if !ok {
		return lines
	}
	box := m.menuShape(l, p)
	if box.w < menuLeast || box.y+box.h > len(lines) {
		return lines
	}

	inner := box.w - 2
	rule := strings.Repeat(pointerH, inner)
	pen := m.styles.Rule

	out := append([]string(nil), lines...)
	row := box.y
	put := func(s string) {
		out[row] = overwrite(out[row], box.x, s, l.interior)
		row++
	}
	// A line of the box: the two uprights with the words held between them, and
	// the air inside written out rather than left to whatever was under it.
	line := func(s string) {
		put(pen.Render(pointerV) + fit(strings.Repeat(" ", menuPad)+s, inner) + pen.Render(pointerV))
	}

	put(pen.Render(pointerTL + rule + pointerTR))
	line(m.styles.Title.Render(fit(p.title, inner-2*menuPad)))
	line(m.styles.Artist.Render(fit(p.subtitle, inner-2*menuPad)))
	put(pen.Render(pointerTee + rule + pointerTeeR))
	for _, row := range p.rows {
		line(fit(row, inner-2*menuPad))
	}
	put(pen.Render(pointerElbow + rule + pointerBR))
	return out
}

// verbLines is the choosable part of the menu: the mark, the key that does the
// same thing without the menu, and what it does.
//
// Asked for at a width of nought to measure them, which is why they come back as
// lines rather than as a drawing: what the box has to be wide enough for is what
// will be written in it.
func (m Model) verbLines(w int) []string {
	out := make([]string, len(m.actions.verbs))
	for i, v := range m.actions.verbs {
		style, gutter := m.styles.RowPrimary, "  "
		if i == m.actions.state.cursor {
			style, gutter = m.styles.RowSelected, m.styles.Cursor.Render(rowCursor)+" "
		}

		shortcut := "  "
		if v.key != "" {
			shortcut = m.styles.FactLabel.Render(v.key) + " "
		}
		line := gutter + shortcut + style.Render(v.label)
		if w > 0 {
			line = fit(line, w)
		}
		out[i] = line
	}
	return out
}

// menuVerbAt is which verb a point is on, or -1 for the rest of the box and
// anywhere off it.
//
// The same rows drawMenu writes, counted rather than remembered.
func (m Model) menuVerbAt(l layout, x, y int) (int, bool) {
	p, ok := m.openPopup()
	if !ok {
		return -1, false
	}
	box := m.menuShape(l, p)
	if x < box.x || x >= box.x+box.w || y < box.y || y >= box.y+box.h {
		return -1, false
	}
	// menuChrome-1 of the rows the box spends on itself are above the choices:
	// the edge, the two lines naming what it is about, and the rule under them.
	if at := y - box.y - (menuChrome - 1); at >= 0 && at < len(p.rows) {
		return at, true
	}
	return -1, true
}
