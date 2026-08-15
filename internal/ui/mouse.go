package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// What the pointer is over.
//
// The screen is one string. There is no tree of boxes under it to ask, so there
// are two ways to answer where a click landed: write down what was drawn while
// drawing it, or work it out again from the layout. The first is not open to us
// — the view is a pure function and may not leave notes — and it is the worse of
// the two anyway, because two records of the same thing drift apart and the one
// that is wrong is the one nobody is looking at.
//
// So a point is answered from the same arithmetic the drawing uses, function for
// function: listChrome and listBandRows for a table, libraryShape and gridFor
// for the wall, the same labels and the same air for both rows of tabs. Where
// the two could still disagree, mouse_test.go finds a word on the drawn screen
// and asks what is there.

// spotKind is what sort of thing is under the pointer.
type spotKind int

const (
	// spotNothing is the screen itself: the margins, the picture, the help bar,
	// the rows a search field is standing in, and every screen with nothing on
	// it to point at.
	spotNothing spotKind = iota

	// spotTabs is the bar across the top, spotKinds the library's own bar of
	// kinds under it.
	spotTabs
	spotKinds

	// spotList is a table of rows, spotTile a wall of covers.
	spotList
	spotTile

	// spotHelp is the page of keys, which scrolls under its own head.
	spotHelp

	// spotDevice is one of the machines playback can be moved to.
	spotDevice

	// spotScroll is the thin bar down the right of a list. at is how far down
	// it the pointer is, in rows.
	spotScroll

	// The two places a question is typed: the field the catalogue is searched
	// from, and the box a list is searched in. Pressing either is asking for
	// the keyboard, which is the only thing a field can mean.
	spotQuery
	spotFinder

	// The three things on the player worth pointing at. at is the column
	// reached along the bar for the first two, and which control for the last:
	// a place on a meter is a place, and rounding it to an index would throw
	// away the only thing a click on a bar is saying.
	spotSeek
	spotVolume
	spotControl
)

// spot is what is under the pointer: what sort of thing, and which one of them.
//
// at is -1 where the region is known and the part of it is not: the air between
// two labels, the gap between two covers, the band over a list, a screen whose
// rows are its own business. The wheel wants the region and a click wants the
// part, so both answers are worth having and neither is a failure.
type spot struct {
	kind spotKind
	at   int
}

// point is a place the pointer has been and what was found there, for the debug
// bar to report. Nothing acts on it.
type point struct {
	x, y int
	at   spot
	seen bool
}

// String is how the bar puts it: where the terminal said the pointer was, and
// what this side of the screen made of it.
func (p point) String() string {
	if !p.seen {
		return "-"
	}
	names := map[spotKind]string{
		spotNothing: "nothing",
		spotTabs:    "tabs",
		spotKinds:   "kinds",
		spotList:    "row",
		spotTile:    "tile",
		spotHelp:    "help",
	}
	return fmt.Sprintf("%d,%d %s %d", p.x, p.y, names[p.at.kind], p.at.at)
}

// span is where one label sits on a row: the column it starts at, and how wide
// it is.
type span struct{ at, w int }

// spanAt is which of them holds a column, or -1 for the air between them.
func spanAt(spans []span, x int) int {
	for i, s := range spans {
		if x >= s.at && x < s.at+s.w {
			return i
		}
	}
	return -1
}

// labelSpans lays a run of labels out from a column, with the same air between
// them the drawing leaves.
func labelSpans(labels []string, gap, from int) []span {
	out := make([]span, len(labels))
	for i, label := range labels {
		w := lipgloss.Width(label)
		out[i] = span{at: from, w: w}
		from += w + gap
	}
	return out
}

// labelsWidth is what that run measures altogether.
func labelsWidth(labels []string, gap int) int {
	w := 0
	for i, label := range labels {
		if i > 0 {
			w += gap
		}
		w += lipgloss.Width(label)
	}
	return w
}

// spotAt is what the terminal's column and row are over.
func (m Model) spotAt(x, y int) spot {
	none := spot{spotNothing, -1}
	if !fitsMinimum(m.width, m.height) {
		return none
	}
	// The big screen is watched rather than worked on, and it outranks the tabs
	// exactly as it does for the keys.
	if m.stage.on {
		return none
	}

	l := m.layout()

	// The screen is placed in the middle of the terminal, so the column the
	// terminal names is not the column the screen was drawn in.
	x -= max((m.width-l.interior)/2, 0)
	if x < 0 || x >= l.interior {
		return none
	}

	// The labels, and the rule under them: a mark that belongs to the label over
	// it, and a click a row low is a click on the tab.
	if y < tabBarHeight-1 {
		return spot{spotTabs, spanAt(tabSpans(l), x)}
	}

	row := y - tabBarHeight
	if row < 0 || row >= l.bodyHeight {
		// The blank between the tabs and the body, and everything under the
		// body: the notice, the help bar, the air around them.
		return none
	}

	// Nothing underneath is reachable while the menu is up, exactly as with the
	// keys: what is open answers everything.
	if m.actions.open {
		return none
	}

	// And the list of devices is what is being looked at while it is up, whether
	// it was opened over a screen or is the screen.
	if m.devices.open || (m.tab == tabPlayer && m.noDevice) {
		return m.deviceSpot(l, x, row)
	}

	// The box a list is searched in stands over whatever is under it, so it is
	// asked before them. See finder.go.
	if m.finding() && row >= m.finderAt(l) && row < m.finderAt(l)+finderRows &&
		x >= leftMargin && x < leftMargin+finderWidth(l) {
		return spot{spotFinder, -1}
	}

	switch {
	case m.tab == tabPlayer:
		return m.playerSpot(l, x, row)
	case m.tab == tabHelp:
		return spot{spotHelp, -1}
	case m.tab == tabLibrary && m.open() == nil:
		return m.wallSpot(l, x, row)
	case m.tab == tabSettings:
		// A list with a cursor, laid out its own way: a heading, a line saying
		// what the screen is, a blank, and the switches under them. See
		// settingsPanel.
		if at := row - settingsChrome; at >= 0 && at < settingsCount && x >= leftMargin {
			return spot{spotList, at}
		}
		return spot{spotList, -1}
	case m.open() != nil, m.tab == tabQueue, m.tab == tabSearch:
		return m.listSpot(l, x, row)
	}
	return none
}

// listSpot is which row of a table is under the pointer.
//
// From the three numbers the table is drawn from: the band over it, what the
// list spends on its own head, and how far down its window the rows have got.
// See listBlock, which draws them, and pointAtCursor, which points at one — this
// is that arithmetic read backwards.
func (m Model) listSpot(l layout, x, row int) spot {
	here := spot{spotList, -1}

	band := m.listBandRows(l)
	head := band + m.listChrome(band)
	body := m.listBodyRows(max(l.bodyHeight, 1), band)

	// The field the catalogue is searched from is this screen's heading — three
	// rows above the first row of results, where the heading, the column names
	// and the line under them stand — or the top of the screen while nothing has
	// been found and there is no band to sit under. See searchPaneView.
	if m.tab == tabSearch && m.open() == nil {
		field := head - 3
		if m.search.current().count() == 0 {
			field = 0
		}
		if row == field && x >= leftMargin && x < leftMargin+searchFieldWidth(l) {
			return spot{spotQuery, -1}
		}
	}

	if row < head || row >= head+body {
		return here
	}
	if x < leftMargin || x >= leftMargin+queueBlockWidth(l) {
		return here
	}

	cursor, count := (&m).rowCursor()
	if cursor == nil {
		return here
	}

	// The bar down the right, where there is one. A column of its own, drawn
	// after the row and a blank — see listBlock — and there is no bar at all
	// while the whole list fits.
	if x == leftMargin+queueRowWidth(l)+1 && count > body {
		return spot{spotScroll, row - head}
	}
	// The window this screen is showing, asked for the way the drawing asks:
	// this is a copy of the model, so nothing here is decided for the next
	// frame.
	from, to := cursor.window(count, body)
	if at := from + row - head; at < to {
		return spot{spotList, at}
	}
	return here
}

// wallSpot is what the library's wall has at a point: the bar of kinds over it,
// or one of the covers.
func (m Model) wallSpot(l layout, x, row int) spot {
	if row < gridChromeRows {
		return spot{spotKinds, spanAt(m.kindSpans(), x)}
	}
	row -= gridChromeRows + m.finderTakes()
	if row < 0 {
		// The rows the wall stood aside for. What is there belongs to the field
		// being typed into. See finder.go.
		return spot{spotNothing, -1}
	}

	here := spot{spotTile, -1}
	g := m.libraryShape(l, l.bodyHeight)
	if !g.ok() {
		return spot{spotNothing, -1}
	}

	// A row of tiles carries a row of air over it, where the arms of the frame
	// stand. The air belongs to the tiles under it: a click a row high is a
	// click on the cover. See drawGrid.
	r := row / (g.tileH + tileRowGap)
	col := x - leftMargin - gridGutter
	step := g.tileW + g.gap
	if r >= g.rows || col < 0 || col%step >= g.tileW {
		return here
	}
	i := col / step
	if i >= g.cols {
		return here
	}

	items := m.libraryTiles()
	state := &m.library.cursors[m.library.kind]
	from, to := state.gridWindow(len(items), g)
	if at := from + r*g.cols + i; at < to {
		return spot{spotTile, at}
	}
	return here
}

// playerSpot is what the player screen has at a point: the bar the playhead
// rides on, the volume meter, or one of the transport glyphs.
//
// Those three and nothing else. The picture is a picture, the title is a title,
// and a screen where every word does something when it is pressed is a screen
// nobody presses anything on.
func (m Model) playerSpot(l layout, x, row int) spot {
	none := spot{spotNothing, -1}
	if m.ps == nil || m.noDevice {
		// Nothing loaded, or nowhere to send it. Both draw something else
		// entirely in this space. See body.
		return none
	}

	// The column beside the picture, and where in it the block came to rest.
	left := leftMargin
	if l.hasArt() {
		left += l.artWidth + columnGap
	}
	x -= left
	if x < 0 || x >= l.infoWidth {
		return none
	}

	lines := m.infoBlock(l.infoWidth)
	up := len(lines) - 1 - (row - m.playerTop(l, len(lines)))
	switch up {
	case playerBarUp:
		// Only where there is one. With nothing loaded the block holds a blank
		// row there rather than a bar at nought against a length of nought.
		if m.loaded() {
			return spot{spotSeek, x}
		}
	case playerTransportUp:
		if at := spanAt(m.controlSpans(), x); at >= 0 {
			return spot{spotControl, at}
		}
		if v := m.volumeSpan(l.infoWidth); x >= v.at && x < v.at+v.w {
			return spot{spotVolume, x - v.at}
		}
	}
	return none
}

// playerTop is the row of the body the player's block of text begins on, given
// how tall the block is.
//
// Two centrings, one inside the other: the column is placed in the body and the
// block is placed in the column. With the words up there is only one — the block
// hangs from the top of the picture and the words take everything under it. See
// body and infoWithLyrics.
func (m Model) playerTop(l layout, block int) int {
	rows := m.playerPaneRows(l)
	top := max((l.bodyHeight-rows)/2, 0)
	if m.lyricsVisible() {
		return top + m.artTop(l, rows)
	}
	return top + max((rows-block)/2, 0)
}

// deviceSpot is which device is under the pointer.
//
// The one list in the program that is only ever pointed at — you open it to
// choose one thing and it closes again — and for a long time it was the one list
// the pointer could not reach.
//
// The screen that is nothing but this list, because nothing is playing anywhere.
// It builds its lines and says where in them the devices start, and centres what
// it built — so the row is that offset, that start, and how far down the list
// the pointer is. See noDeviceLines. The picker opened with a key is a box, and
// boxes answer for themselves.
func (m Model) deviceSpot(l layout, x, row int) spot {
	none := spot{spotNothing, -1}
	if len(m.devices.items) == 0 {
		return none
	}

	// The picker opened with a key is a box standing over the screen, and the
	// box answers for itself — see menuVerbAt. This is the other one: the screen
	// that stands in for the player when nothing is playing anywhere.
	if m.devices.open {
		return none
	}

	// It keeps a row back from the body for the status line under it, which is
	// what the panel is laid out in. See body.
	lines, at := m.noDeviceLines(l)
	rows := max(l.bodyHeight-1, 1)
	left, width := leftMargin, min(l.interior-leftMargin-rightMargin, deviceListCols)

	if x < left || x >= left+width {
		return none
	}
	if i := row - stackTop(len(lines), rows, 0) - at; i >= 0 && i < len(m.devices.items) {
		return spot{spotDevice, i}
	}
	return none
}

// cursorPoint is where on the screen the thing under the cursor is drawn.
//
// The other direction from spotAt, out of the same arithmetic: that one is given
// a point and names what is there, this one is given the cursor and says where
// it ended up. It is what lets a menu raised by a key stand where a menu raised
// by a click stands — one box, opened at the thing it is about, however it was
// asked for.
//
// A row's mark is at the left margin and a cover's at its own left edge; the box
// opens a row below and a column in from either, so what was chosen is still
// readable above it.
func (m Model) cursorPoint(l layout) (x, y int) {
	if m.tab == tabLibrary && m.open() == nil {
		g := m.libraryShape(l, l.bodyHeight)
		state := &m.library.cursors[m.library.kind]
		if from, _ := state.gridWindow(len(m.libraryTiles()), g); g.ok() {
			at := state.cursor - from
			col, row := at%max(g.cols, 1), at/max(g.cols, 1)
			return leftMargin + gridGutter + col*(g.tileW+g.gap) + 1,
				tabBarHeight + gridChromeRows + m.finderTakes() + row*(g.tileH+tileRowGap) + 1
		}
		return leftMargin + 1, tabBarHeight + gridChromeRows + 1
	}

	band := m.listBandRows(l)
	head := tabBarHeight + band + m.listChrome(band)
	if cursor, count := (&m).rowCursor(); cursor != nil {
		from, _ := cursor.window(count, m.listBodyRows(max(l.bodyHeight, 1), band))
		return leftMargin + 1, head + cursor.cursor - from + 1
	}
	return leftMargin + 1, head
}

// rowCursor is the cursor of whatever list is on screen.
//
// The search keeps one per kind of hit and the settings keep their own, so this
// is find.go's answer with those two added: the mouse reaches screens a query
// does not.
func (m *Model) rowCursor() (*listState, int) {
	switch {
	case m.tab == tabSearch && m.open() == nil:
		found := m.search.current()
		return &found.cursor, found.count()
	case m.tab == tabSettings:
		return &m.settings.cursor, settingsCount
	}
	return m.listCursor()
}
