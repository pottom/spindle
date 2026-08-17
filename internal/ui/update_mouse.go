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

// How fast a run of notches has to arrive before a list moves by more than one
// row, and how far it moves then.
//
// A terminal reports notches and never says what turned them. A mouse wheel
// clicks a few times a second and every one of them is deliberate; a trackpad
// flick sends them as fast as the screen redraws, and each one means far less.
// The only thing that tells them apart from here is how close together they
// come, so that is what is measured.
//
// These numbers are a starting point and the debug bar carries the gap that was
// actually measured, so they can be set from a real hand rather than from this
// comment. Conservative on purpose: a list that runs away is worse to use than
// one that is a little slow.
const (
	wheelBrisk = 80 * time.Millisecond
	wheelFast  = 30 * time.Millisecond

	wheelBriskRows = 2
	wheelFastRows  = 4
)

// wheelStep is how many rows one notch moves, from how long it has been since
// the last one.
func (m *Model) wheelStep() int {
	gap := time.Since(m.lastNotchAt)
	m.lastNotchAt = time.Now()
	m.lastNotch = gap

	switch {
	case gap < wheelFast:
		return wheelFastRows
	case gap < wheelBrisk:
		return wheelBriskRows
	default:
		return 1
	}
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

	// A box answers everything while it is up, exactly as it does with the keys:
	// nothing underneath may be turned by a wheel over something standing on top
	// of it.
	if _, inside := m.menuVerbAt(m.layout(), e.X, e.Y); inside {
		if m.story {
			// Nothing in it to walk through, but plenty to read past.
			l := m.layout()
			m.storyAt = min(max(m.storyAt+delta*m.wheelStep(), 0), m.storyLast(l))
			return m, nil
		}
		if m.devices.open {
			m.devices.cursor.move(delta, len(m.devices.items))
		} else {
			m.actions.state.move(delta, len(m.actions.verbs))
		}
		return m, nil
	}
	if _, up := m.openPopup(); up {
		return m, nil
	}

	at := m.noteMouse(e.X, e.Y)

	// Held down, the wheel changes the thing rather than walking it. What that
	// means is whatever the thing under it is: the order of an ordered list, the
	// size of a wall of pictures. Two meanings for one gesture, and each is the
	// obvious one where it is.
	if e.Mod&tea.ModCtrl != 0 {
		switch {
		case at.kind == spotTile:
			// Turned down is more of them across and so smaller, which is what
			// turning away from yourself means on every other grid of pictures.
			return m.resizeWall(delta)
		case m.tab == tabSettings && at.at >= 0:
			// The switch under the pointer, turned the way the keys turn it.
			// Turned up is forward, as it is on every other bar and wheel here.
			m.settings.cursor.moveTo(at.at, settingsCount)
			return m, m.turnSetting(-delta)
		}
		return m.mouseReorder(at, delta)
	}

	switch at.kind {
	case spotTabs:
		return m, m.switchTab(m.tab.next(delta))

	case spotKinds:
		if m.tab == tabSearch {
			m.turnSearchKind(delta)
			return m, tea.Batch(m.previewCover(), m.readAhead())
		}
		return m, tea.Batch(m.turnLibraryKind(delta), m.syncGridCovers())

	case spotTile:
		// A row of covers at a time, which is how the wall is walked and how it
		// scrolls: a wheel that moved by one tile would put one row of the wall
		// across two rows of the screen. See gridWindow.
		g, items, state := m.wallUnderPointer()
		if !g.ok() {
			return m, nil
		}
		state.move(delta*m.wheelStep()*g.cols, len(items))
		return m, tea.Batch(m.previewCover(), m.readAhead(), m.syncGridCovers())

	case spotList:
		cursor, count := (&m).rowCursor()
		if cursor == nil {
			return m, nil
		}
		// A row a notch, or more when the notches are coming fast: which of a
		// wheel and a trackpad is in somebody's hand is not a thing a terminal
		// says, and how close together the notches arrive is the only evidence
		// there is. See wheelStep.
		cursor.move(delta*m.wheelStep(), count)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case spotHelp:
		// The page scrolls under its own head, exactly as the keys scroll it.
		m.helpAt = max(m.helpAt+delta, 0)
		return m, nil

	case spotDevice:
		m.devices.cursor.move(delta, len(m.devices.items))
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

// doubleWithin is how soon a second press on the same cell is the same press
// again rather than a new one.
//
// What every desktop calls a double click. Four hundred milliseconds is what
// they settle on, near enough, and it is the number to change if a deliberate
// second click ever fails to open anything.
const doubleWithin = 400 * time.Millisecond

// gripWithin is how long a cell goes on holding the track a run of ctrl+notches
// took hold of, once the notches stop.
//
// Long, because it is not what ends a hold — moving the pointer is. It is the
// backstop for the other case: coming back to a cell much later and expecting
// to take hold of whatever is there now.
const gripWithin = 2 * time.Second

// mouseClick acts on what was pressed.
//
// A label is pressed and the screen changes; a row or a cover is pressed and the
// cursor goes there; pressed again it opens or plays. One press does not,
// because a wall of records is a place where the pointer passes over a hundred
// covers on the way to one, and a single click that started the music would be a
// mistake nobody asked to be able to make.
//
// The right button raises the menu of verbs on the thing under it, which is what
// the right button does everywhere else.
func (m Model) mouseClick(e tea.MouseClickMsg) (Model, tea.Cmd) {
	// The big screen is watched rather than worked on, and a press of the
	// pointer is the way out of it — the same thing esc does, from the same
	// instinct: it is the first thing a hand reaches for to make something go
	// away. Nothing up there is worth pointing at.
	if m.stage.on {
		return m, m.leaveStage()
	}

	// A box answers everything while it is up. A press on one of its rows
	// chooses it, a press anywhere else puts it away — which is what pressing
	// away from an open menu means, and there is nothing else it could mean.
	//
	// One press rather than two, unlike a row of a list: this box was opened on
	// purpose and holds nothing but choices, so there is nothing here to brush
	// past by accident.
	if p, up := m.openPopup(); up {
		at, inside := m.menuVerbAt(m.layout(), e.X, e.Y)
		if p.plain || !inside || e.Button != tea.MouseLeft {
			// Nothing in it to choose, or a press away from it: either way it
			// goes. A paragraph is read and then put away.
			m.actions.open, m.devices.open, m.story = false, false, false
			return m, nil
		}
		if at < 0 {
			return m, nil
		}
		if m.devices.open {
			m.devices.cursor.moveTo(at, len(m.devices.items))
			return m, m.transfer()
		}
		m.actions.open = false
		return m, m.actions.verbs[at].do(&m)
	}

	if e.Button == tea.MouseRight {
		return m.mouseActions(e)
	}
	if e.Button != tea.MouseLeft {
		return m, nil
	}

	// Whatever was being held is let go of by pressing somewhere else, which
	// cannot happen with one button but can with two.
	m.drag = dragState{}

	// A second press on the cell the last one was on, soon enough after it.
	twice := m.lastClick.seen && m.lastClick.x == e.X && m.lastClick.y == e.Y &&
		time.Since(m.lastClickAt) < doubleWithin

	at := m.noteMouse(e.X, e.Y)
	m.lastClick, m.lastClickAt = m.lastPoint, time.Now()

	if twice && (at.kind == spotList || at.kind == spotTile || at.kind == spotDevice) && at.at >= 0 {
		// The cursor is already on it — the first press put it there — so this
		// is the key that acts on where the cursor is, asked for by hand rather
		// than pressed. One act, however it was asked for: what enter does on
		// this screen is what a second click does, whatever screen it is.
		return m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	}

	// A field is one thing rather than a row of things, so it is answered before
	// the check that there is a part under the pointer at all.
	switch at.kind {
	case spotQuery:
		// Pressing a field is asking for the keyboard, and there is nothing
		// else it could be asking for.
		if m.search.typing {
			return m, nil
		}
		return m, m.startTyping()

	case spotFinder:
		m.find.typing = true
		return m, nil
	}

	if at.at < 0 {
		// The region is known and the part of it is not: the air between two
		// labels, the gap between two covers. Nothing to press.
		return m, nil
	}

	switch at.kind {
	case spotTabs:
		return m, m.switchTab(tabID(at.at))

	case spotKinds:
		// The search's are the views of one answer, the library's are its four
		// lists. A press names one of them either way — see searchViewSpans and
		// kindSpans, which say where each of them sits.
		if m.tab == tabSearch {
			if view := m.viewAt(at.at); view != "" {
				m.search.kind = view
				return m, tea.Batch(m.previewCover(), m.readAhead())
			}
			return m, nil
		}
		if at.at < 0 || at.at >= len(libraryOrder) {
			return m, nil
		}
		return m, tea.Batch(m.setLibraryKind(libraryOrder[at.at]), m.syncGridCovers())

	case spotTile:
		_, items, state := m.wallUnderPointer()
		state.moveTo(at.at, len(items))
		return m, tea.Batch(m.previewCover(), m.readAhead(), m.syncGridCovers())

	case spotDevice:
		// Chosen with a second press, like everything else. Moving the music to
		// another machine is not something to do by brushing past a name.
		m.devices.cursor.moveTo(at.at, len(m.devices.items))
		return m, nil

	case spotList:
		cursor, count := (&m).rowCursor()
		if cursor == nil {
			return m, nil
		}
		cursor.moveTo(at.at, count)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case spotSeek, spotVolume:
		// A press on a bar takes hold of it rather than acting on it. What was
		// chosen is sent when the button comes up — a click is that with no
		// motion in between. See drag.go.
		m.takeHold(at.kind, at.at)

	case spotScroll:
		// And the same for the bar down the side of a list, which moves the
		// list as it is dragged rather than when it is let go.
		m.takeHold(at.kind, at.at)
		m.followScroll(e.Y)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case spotControl:
		return m.pressControl(control(at.at))
	}
	return m, nil
}

// mouseReorder moves the track under the pointer up or down the queue, which is
// what ctrl and the wheel do together.
//
// Two things make this work rather than something to regret. The first is that
// the queue already gathers a run of moves and sends it as one edit once it has
// come to rest — see moveQueued and orderDebounce — so a wheel spun ten notches
// costs one request, not ten.
//
// The second is which track it moves. Not the one under the pointer: the row
// climbs out from under it, so the second notch would take hold of a different
// track and three notches would leave three tracks scattered. It moves the one
// the cursor is on, and the cursor goes with it. The pointer decides only where
// the hold begins.
//
// What ends the hold is the pointer moving, not the clock — measured, because
// the first attempt ended it on the clock and it was wrong within a second. Two
// notches a second apart with the pointer sitting still is somebody turning a
// wheel slowly, and the track escaped from under them and swapped back. So the
// hold lasts as long as the notches keep arriving on the same cell; the timeout
// beside it is only there so a cell nobody has touched for a while is not still
// holding something.
//
// Only the queue's own tab. Nothing else on screen is a list somebody put in an
// order.
func (m Model) mouseReorder(at spot, delta int) (Model, tea.Cmd) {
	if at.kind != spotList || m.tab != tabQueue || m.open() != nil || m.devices.open {
		return m, nil
	}

	held := m.grip.seen && m.grip.x == m.lastPoint.x && m.grip.y == m.lastPoint.y &&
		time.Since(m.gripAt) < gripWithin
	m.grip, m.gripAt = m.lastPoint, time.Now()

	if !held && at.at >= 0 {
		m.queuePane.cursor.moveTo(at.at, len(m.queueRows()))
	}
	return m, m.moveQueued(delta)
}

// resizeWall makes the covers larger or smaller, which is what ctrl and the
// wheel do over a grid of pictures anywhere.
//
// By the column rather than by the cell. The width of a tile is what is left
// over once the columns are decided, so asking for two more cells does nothing
// at all until it happens to cross a boundary — a notch that leaves the screen
// exactly as it was reads as a gesture that is not supported. One notch, one
// column, every time.
//
// What it keeps is what the wall says it did, not what was asked for: the shape
// clamps the count against how narrow a cover may be drawn and how wide it may
// grow, and storing the ask rather than the answer would leave notches piling up
// against a wall that had stopped moving.
//
// Turned up is larger — fewer across — as turned up is louder and further along
// everywhere else here. The pictures are asked for again at the new size, so
// this is the one gesture that costs anything.
func (m Model) resizeWall(delta int) (Model, tea.Cmd) {
	l := m.layout()
	was := m.libraryShape(l, l.bodyHeight)
	if !was.ok() {
		return m, nil
	}

	m.library.cols = max(was.cols+delta, 1)
	now := m.libraryShape(l, l.bodyHeight)
	if !now.ok() || now.cols == was.cols {
		m.library.cols = was.cols
		return m, nil
	}
	m.library.cols = now.cols
	return m, tea.Batch(m.syncGridCovers(), m.savePrefs())
}

// mouseActions raises the menu of verbs on the thing under the pointer.
//
// The cursor goes there first. The menu is about what the cursor is on — that is
// how every screen builds it — so a menu raised somewhere the cursor is not
// would be a menu about the wrong record, and the surest way to have somebody
// take a track out of a queue they did not mean to.
func (m Model) mouseActions(e tea.MouseClickMsg) (Model, tea.Cmd) {
	at := m.noteMouse(e.X, e.Y)
	if at.at < 0 {
		return m, nil
	}

	var cmd tea.Cmd
	switch at.kind {
	case spotList:
		cursor, count := (&m).rowCursor()
		if cursor == nil {
			return m, nil
		}
		cursor.moveTo(at.at, count)
		cmd = tea.Batch(m.previewCover(), m.readAhead())
	case spotTile:
		m.library.cursor().moveTo(at.at, len(m.libraryTiles()))
		cmd = tea.Batch(m.previewCover(), m.readAhead(), m.syncGridCovers())
	default:
		// The tabs, the kinds, the transport: things that do one thing each and
		// have no second thing to offer.
		return m, nil
	}

	m.openActionsAt(e.X, e.Y)
	return m, cmd
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
