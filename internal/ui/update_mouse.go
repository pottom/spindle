package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// What the mouse does.
//
// One rule, and everything here follows from it: the wheel belongs to whatever
// is under the pointer. Not to the screen, not to "the list", but to the thing
// the pointer is resting on — the tabs turn under it, the library's kinds turn
// under it, a wall scrolls a row of covers at a time and a table a row of tracks.
// Nobody has to learn that, which is the point.
//
// Only the wheel and the plain click, in the mode that reports them: cells, not
// motion. Following the pointer around the screen at sixty frames a second would
// be a message a frame for something nothing on this screen wants to know.
//
// What it costs is the terminal's own drag-to-select, which shift+drag gives
// back on every terminal this program is drawn on.

// noteMouse works out what is under a point and keeps it for the debug bar.
//
// The bar is the only way to see what this side of the screen thinks the pointer
// is over, and a click on the wrong thing and a click reported at the wrong
// place look the same from here. It is what settled the keyboard question; the
// same field settles this one. See debugSelf.
func (m *Model) noteMouse(x, y int) spot {
	at := m.spotAt(x, y)
	m.lastPoint = point{x: x, y: y, at: at, seen: true}
	return at
}

// mouseWheel turns whatever the pointer is over.
func (m Model) mouseWheel(e tea.MouseWheelMsg) (Model, tea.Cmd) {
	var delta int
	switch e.Button {
	case tea.MouseWheelDown:
		delta = 1
	case tea.MouseWheelUp:
		delta = -1
	default:
		// Sideways. Nothing on this screen runs across, and a wheel that did
		// something arbitrary with it would be worse than one that did nothing.
		return m, nil
	}

	at := m.noteMouse(e.X, e.Y)
	switch at.kind {
	case spotTabs:
		return m, m.switchTab(m.tab.next(delta))

	case spotKinds:
		return m, tea.Batch(m.turnLibraryKind(delta), m.syncGridCovers())

	case spotTile:
		// A row of covers at a time, which is how the wall is walked and how it
		// scrolls: a wheel that moved by one tile would put one row of the wall
		// across two rows of the screen. See gridWindow.
		g := m.libraryShape(m.layout(), m.layout().bodyHeight)
		if !g.ok() {
			return m, nil
		}
		m.library.cursor().move(delta*g.cols, len(m.libraryTiles()))
		return m, tea.Batch(m.previewCover(), m.readAhead(), m.syncGridCovers())

	case spotList:
		cursor, count := (&m).rowCursor()
		if cursor == nil {
			return m, nil
		}
		// One row a notch. A mouse wheel is coarser than that and a trackpad
		// finer, and which of the two is in somebody's hand is not a thing a
		// terminal says — so the honest answer is the smallest step, and going
		// faster when the notches come fast is a measurement rather than a
		// guess. Not made yet.
		cursor.move(delta, count)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case spotHelp:
		// The page scrolls under its own head, exactly as the keys scroll it.
		m.helpAt = max(m.helpAt+delta, 0)
		return m, nil

	case spotSeek:
		// The same step the keys take. A wheel over the bar is somebody nudging
		// the playhead, and a notch that jumped a different distance from the
		// key beside it would be a second answer to one question.
		//
		// Turned up is further along, as turned up is louder on the meter
		// beside it: both bars fill to the right, and a wheel that read one of
		// them backwards would be two rules on one row.
		return m, m.seek(m.elapsed() - time.Duration(delta)*seekStep)

	case spotVolume:
		return m, m.setVolume(m.ps.Volume - delta*volumeStep)
	}
	return m, nil
}

// mouseClick acts on what was pressed.
//
// A label is pressed and the screen changes; a row or a cover is pressed and the
// cursor goes there. Nothing plays and nothing opens: a wall of records is a
// place where the pointer passes over a hundred covers on the way to one, and a
// single click that started the music would be a mistake nobody asked to be able
// to make. What opens a thing is the next round's question.
func (m Model) mouseClick(e tea.MouseClickMsg) (Model, tea.Cmd) {
	if e.Button != tea.MouseLeft {
		return m, nil
	}

	at := m.noteMouse(e.X, e.Y)
	if at.at < 0 {
		// The region is known and the part of it is not: the air between two
		// labels, the gap between two covers. Nothing to press.
		return m, nil
	}

	switch at.kind {
	case spotTabs:
		return m, m.switchTab(tabID(at.at))

	case spotKinds:
		return m, tea.Batch(m.setLibraryKind(libraryOrder[at.at]), m.syncGridCovers())

	case spotTile:
		m.library.cursor().moveTo(at.at, len(m.libraryTiles()))
		return m, tea.Batch(m.previewCover(), m.readAhead(), m.syncGridCovers())

	case spotList:
		cursor, count := (&m).rowCursor()
		if cursor == nil {
			return m, nil
		}
		cursor.moveTo(at.at, count)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case spotSeek:
		// Where along the bar it was pressed, as a share of the track. The bar
		// is drawn from the same fraction, so the playhead lands under the
		// pointer. See progressLine.
		if !m.loaded() {
			return m, nil
		}
		return m, m.seek(atFraction(at.at, barCells(m.layout().infoWidth), m.ps.Duration))

	case spotVolume:
		// And the same for the meter, which is the same shape at another width.
		return m, m.setVolume(atShare(at.at, barCells(volumeCells), 100))

	case spotControl:
		return m.pressControl(control(at.at))
	}
	return m, nil
}

// pressControl does what the glyph under the pointer says, which is what the key
// bound to it does: one act, two ways of asking for it. See transport.go.
func (m Model) pressControl(c control) (Model, tea.Cmd) {
	if m.ps == nil || m.noDevice {
		return m, nil
	}
	switch c {
	case ctlPrev:
		return m, m.skipPrev()
	case ctlPlay:
		return m, m.togglePlay()
	case ctlNext:
		return m, m.skipNext()
	case ctlShuffle:
		return m, m.toggleShuffle()
	case ctlRepeat:
		return m, m.turnRepeat()
	}
	return m, nil
}

// atShare is where along a bar of that many cells a press landed, as a share of
// a whole — the arithmetic the bars are drawn by, read backwards.
func atShare(cell, cells, whole int) int {
	if cells <= 0 {
		return 0
	}
	return min(max(cell*whole/cells, 0), whole)
}

// atFraction is the same for a length of time.
func atFraction(cell, cells int, whole time.Duration) time.Duration {
	if cells <= 0 {
		return 0
	}
	return min(max(time.Duration(cell)*whole/time.Duration(cells), 0), whole)
}
