package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// The one thing that can go wrong with a pointer in a program whose screen is a
// string: the arithmetic that answers a click drifting away from the arithmetic
// that drew the screen. Nothing catches that by reading — the two agree in every
// case anybody thought to check by hand.
//
// So these tests never work out where anything ought to be. They find a word on
// the drawn screen, take the column and row it is actually at, and ask what is
// there. If the hit test names something else, the two have drifted.

// wordAt is where a word is drawn: the column and row of its first cell.
func wordAt(t *testing.T, m Model, word string) (x, y int) {
	t.Helper()
	for row, line := range strings.Split(m.render(), "\n") {
		plain := ansi.Strip(line)
		if i := strings.Index(plain, word); i >= 0 {
			return lipgloss.Width(plain[:i]), row
		}
	}
	t.Fatalf("%q is not on the screen", word)
	return 0, 0
}

func wheelAt(x, y int, button tea.MouseButton) tea.MouseWheelMsg {
	return tea.MouseWheelMsg{X: x, Y: y, Button: button}
}

func clickAt(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}

// Every label in the bar across the top answers where it is drawn, from its
// first cell to its last.
func TestAClickLandsOnTheTabItIsOver(t *testing.T) {
	m := likedModel(t)

	for i, name := range tabNames {
		x, y := wordAt(t, m, name)
		for col := x; col < x+lipgloss.Width(name); col++ {
			at := m.spotAt(col, y)
			if at.kind != spotTabs || at.at != i {
				t.Fatalf("column %d of row %d is %q, and the pointer calls it %v/%d",
					col, y, name, at.kind, at.at)
			}
		}
		// And the rule under the label belongs to the label.
		if at := m.spotAt(x, y+1); at.kind != spotTabs || at.at != i {
			t.Errorf("the mark under %q is %v/%d, want the tab itself", name, at.kind, at.at)
		}
	}

	// The air between two labels is the bar and none of them.
	x, _ := wordAt(t, m, tabNames[tabQueue])
	if at := m.spotAt(x-2, 0); at.kind != spotTabs || at.at != -1 {
		t.Errorf("the air before %q is %v/%d, want the bar and no tab", tabNames[tabQueue], at.kind, at.at)
	}
}

// And a click on one goes there.
func TestClickingATabOpensIt(t *testing.T) {
	m := likedModel(t)
	x, y := wordAt(t, m, tabNames[tabSearch])

	got, _ := m.mouseClick(clickAt(x, y))
	if got.tab != tabSearch {
		t.Fatalf("clicking %q left the screen on %v", tabNames[tabSearch], got.tab)
	}

	// And the wheel over the bar turns it, in the direction it is turned.
	got, _ = m.mouseWheel(wheelAt(x, y, tea.MouseWheelDown))
	if got.tab != m.tab.next(1) {
		t.Errorf("the wheel over the tabs landed on %v, want %v", got.tab, m.tab.next(1))
	}
}

// The library's own bar of kinds, the same way: the label answers where it is
// drawn, and pressing it puts that kind on the wall.
func TestClickingALibraryKindShowsIt(t *testing.T) {
	m := likedModel(t)

	label := m.kindLabels()[1]
	x, y := wordAt(t, m, label)
	if at := m.spotAt(x, y); at.kind != spotKinds || at.at != 1 {
		t.Fatalf("%q is at column %d of row %d, and the pointer calls it %v/%d", label, x, y, at.kind, at.at)
	}

	got, cmd := m.mouseClick(clickAt(x, y))
	if got.library.kind != libraryOrder[1] {
		t.Fatalf("clicking %q left the library on %v", label, got.library.kind)
	}
	if cmd == nil {
		t.Error("a kind that has never been read was not asked for")
	}
}

// A cover on the wall answers where it is drawn, and pressing it moves the
// cursor there — and no further: a wall is where the pointer passes over a
// hundred records on the way to one, and a click that started the music would be
// a mistake nobody asked to be able to make.
func TestClickingACoverMovesTheCursorToIt(t *testing.T) {
	m := likedModel(t)

	items := m.libraryTiles()
	if len(items) < 3 {
		t.Skipf("the mock library holds %d things, want three to point at", len(items))
	}

	// Whichever of them is drawn under its own name — a long name is cut to the
	// tile, and a cut name is not a word to look for.
	found := 0
	for i, item := range items {
		x, y := 0, 0
		for row, line := range strings.Split(m.render(), "\n") {
			plain := ansi.Strip(line)
			if j := strings.Index(plain, item.name); j >= 0 && row > tabBarHeight {
				x, y = lipgloss.Width(plain[:j]), row
				break
			}
		}
		if y == 0 {
			continue
		}
		found++
		if at := m.spotAt(x, y); at.kind != spotTile || at.at != i {
			t.Fatalf("%q is at column %d of row %d, and the pointer calls it %v/%d",
				item.name, x, y, at.kind, at.at)
		}

		got, _ := m.mouseClick(clickAt(x, y))
		if got.library.cursor().cursor != i {
			t.Fatalf("clicking %q left the cursor on %d, want %d", item.name, got.library.cursor().cursor, i)
		}
		if got.open() != nil {
			t.Fatal("one click opened a record, want it to select one")
		}
	}
	if found == 0 {
		t.Fatal("no cover on the wall is drawn under a name this can look for")
	}
}

// A row of a table, likewise: the pointer names the row the words are on.
func TestAClickLandsOnTheRowItIsOver(t *testing.T) {
	m := queueModel(0, "alpha", "bravo", "charlie", "delta")
	m.width, m.height = 100, 40
	m.resize()

	for n, id := range []string{"alpha", "bravo", "charlie", "delta"} {
		x, y := wordAt(t, m, id)
		at := m.spotAt(x, y)
		if at.kind != spotList || at.at != queueRowOf(n) {
			t.Fatalf("%q is at column %d of row %d, and the pointer calls it %v/%d",
				id, x, y, at.kind, at.at)
		}

		got, _ := m.mouseClick(clickAt(x, y))
		if got.queuePane.cursor.cursor != queueRowOf(n) {
			t.Errorf("clicking %q left the cursor on %d, want %d",
				id, got.queuePane.cursor.cursor, queueRowOf(n))
		}
	}

	// The band over the list is the list's screen but no row of it.
	if at := m.spotAt(leftMargin, tabBarHeight); at.kind != spotList || at.at != -1 {
		t.Errorf("the band over the queue is %v/%d, want the list and no row", at.kind, at.at)
	}
}

// The wheel moves the cursor of whatever it is over, a row at a time.
func TestTheWheelTurnsTheListUnderIt(t *testing.T) {
	m := queueModel(0, "alpha", "bravo", "charlie", "delta")
	m.width, m.height = 100, 40
	m.resize()

	x, y := wordAt(t, m, "alpha")
	got, _ := m.mouseWheel(wheelAt(x, y, tea.MouseWheelDown))
	if got.queuePane.cursor.cursor != 1 {
		t.Errorf("a notch down left the cursor on %d, want one row down", got.queuePane.cursor.cursor)
	}
	got, _ = got.mouseWheel(wheelAt(x, y, tea.MouseWheelUp))
	if got.queuePane.cursor.cursor != 0 {
		t.Errorf("a notch back left the cursor on %d, want where it started", got.queuePane.cursor.cursor)
	}
}

// And over the wall it moves by a row of covers, because that is how the wall
// scrolls: by one tile it would put a row of it across two rows of the screen.
func TestTheWheelTurnsTheWallByARowOfCovers(t *testing.T) {
	m := wideLibrary(t, 60)

	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	if !g.ok() || g.cols < 2 {
		t.Skipf("the wall came out %d by %d, want something to scroll", g.cols, g.rows)
	}

	x, y := wordAt(t, m, m.kindLabels()[0])
	got, _ := m.mouseWheel(wheelAt(x, y+gridChromeRows+1, tea.MouseWheelDown))
	if got.library.cursor().cursor != g.cols {
		t.Errorf("a notch down the wall moved %d covers, want a row of %d",
			got.library.cursor().cursor, g.cols)
	}
}

// Scrolling a row and coming back does not fetch, render and send every cover
// that went past the edge. Walking the wall with the keys never made this worth
// having; a wheel does it in a second.
func TestScrollingTheWallBackDoesNotAskAgain(t *testing.T) {
	m := wideLibrary(t, 60)
	// Somewhere for the wall to send its requests. Nothing runs them: a command
	// is a closure until the framework calls it, and what is being counted here
	// is how many the wall makes.
	m.covers = &cover.Loader{}

	if cmd := m.syncGridCovers(); cmd == nil {
		t.Fatal("the wall asked for no pictures at all")
	}
	if len(m.tiles) == 0 {
		t.Fatal("the wall filed no pictures")
	}

	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	count := len(m.libraryTiles())

	// A screenful down, which is far enough that the window under the cursor
	// really moves, and the pictures for it.
	m.library.cursor().move(g.page(), count)
	if from, _ := m.library.cursor().gridWindow(count, g); from == 0 {
		t.Fatal("a screenful down and the wall is still at the top")
	}
	m.syncGridCovers()

	// And back to where it was.
	m.library.cursor().move(-g.page(), count)
	if cmd := m.syncGridCovers(); cmd != nil {
		t.Error("scrolling the wall away and back asked again for pictures it already had")
	}
}

// The block names its last four rows from its foot, because what is above them
// is as tall as the caption needs. This is the only thing holding those two
// numbers to the rows they are about.
func TestTheBlockPutsTheBarAndTheTransportWhereTheyAreNamed(t *testing.T) {
	m := playerModel()

	lines := m.infoBlock(m.layout().infoWidth)
	bar := lines[len(lines)-1-playerBarUp]
	if !strings.Contains(bar, knob) {
		t.Errorf("the row named the bar holds %q, want the playhead in it", ansi.Strip(bar))
	}
	transport := lines[len(lines)-1-playerTransportUp]
	if !strings.Contains(transport, iconNext) {
		t.Errorf("the row named the transport holds %q, want the controls in it", ansi.Strip(transport))
	}
}

// The playhead, the meter and every glyph beside them answer where they are
// drawn — the player is the screen where the mouse has the most to reach for
// and the least in the way of rows to reach it by.
func TestThePointerFindsThePlayer(t *testing.T) {
	m := playerModel()

	x, y := wordAt(t, m, iconNext)
	if at := m.spotAt(x, y); at.kind != spotControl || at.at != int(ctlNext) {
		t.Fatalf("the skip glyph is at column %d of row %d, and the pointer calls it %v/%d", x, y, at.kind, at.at)
	}
	got, cmd := m.mouseClick(clickAt(x, y))
	if cmd == nil {
		t.Error("pressing the skip glyph asked the device for nothing")
	}
	_ = got

	// The play glyph is the one that changes what it is drawn as, so it is
	// worth pressing rather than only pointing at.
	x, y = wordAt(t, m, iconPause)
	if at := m.spotAt(x, y); at.kind != spotControl || at.at != int(ctlPlay) {
		t.Fatalf("the pause glyph is %v/%d, want the play control", at.kind, at.at)
	}
	if got, _ = m.mouseClick(clickAt(x, y)); got.ps.Playing {
		t.Error("pressing pause left it playing")
	}

	// The meter: pressed and let go at its far end it is full, and at its near
	// end empty. A click is a drag with no motion in the middle — see drag.go —
	// so it is the release that sets it.
	v := m.volumeSpan(m.layout().infoWidth)
	_, row := wordAt(t, m, iconNext)
	left := leftMargin + m.layout().artWidth + columnGap
	press := func(x int) Model {
		t.Helper()
		out, _ := m.mouseClick(clickAt(x, row))
		out, _ = out.mouseRelease(tea.MouseReleaseMsg{X: x, Y: row, Button: tea.MouseLeft})
		return out
	}
	if got = press(left + v.at + v.w - 1); got.ps.Volume != 100 {
		t.Errorf("pressing the far end of the meter set the volume to %d, want all of it", got.ps.Volume)
	}
	if got = press(left + v.at); got.ps.Volume != 0 {
		t.Errorf("pressing the near end of the meter set the volume to %d, want none of it", got.ps.Volume)
	}

	// And the wheel over it steps by what the keys step by.
	was := m.ps.Volume
	got, _ = m.mouseWheel(wheelAt(left+v.at+2, row, tea.MouseWheelUp))
	if got.ps.Volume != was+volumeStep {
		t.Errorf("a notch up the meter left the volume at %d, want %d", got.ps.Volume, was+volumeStep)
	}
}

// The bar the playhead rides on is a place rather than a step: pressed halfway
// along, the track is halfway through.
func TestPressingTheBarSeeksToThatPlace(t *testing.T) {
	m := playerModel()

	x, y := wordAt(t, m, knob)
	at := m.spotAt(x, y)
	if at.kind != spotSeek {
		t.Fatalf("the playhead is at column %d of row %d, and the pointer calls it %v", x, y, at.kind)
	}

	// Where the playhead is drawn is where it says it is: pressing the cell it
	// is standing in seeks to where the track already is.
	bar := barCells(m.layout().infoWidth)
	want := atFraction(at.at, bar, m.ps.Duration)
	if got := m.elapsed(); absDuration(got-want) > m.ps.Duration/time.Duration(bar) {
		t.Errorf("the playhead is drawn at %s and pressing it asks for %s", got, want)
	}

	// And the far end of the bar is the end of the track.
	end := m.spotAt(x-at.at+bar, y)
	if end.kind != spotSeek || atFraction(end.at, bar, m.ps.Duration) != m.ps.Duration {
		t.Errorf("the end of the bar is %v/%d, want the end of the track", end.kind, end.at)
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// The menu is a box now, standing on the thing it is about. What it draws and
// what it answers a press with are the same rows, which is the only thing
// holding a click on a verb to the verb it runs.
func TestTheMenuAnswersWhereItsVerbsAreDrawn(t *testing.T) {
	m := likedModel(t)
	if !m.openActions() {
		t.Fatal("the cover under the cursor offered no verbs")
	}

	l := m.layout()
	for i, v := range m.actions.verbs {
		x, y := wordAt(t, m, v.label)
		at, inside := m.menuVerbAt(l, x, y)
		if !inside || at != i {
			t.Fatalf("%q is drawn at column %d of row %d, and the menu calls it %d (inside %v)",
				v.label, x, y, at, inside)
		}
	}

	// The box stands where the cursor is rather than in the middle of the
	// screen: on the wall, that is the cover it was raised over.
	cx, cy := m.cursorPoint(l)
	if box := m.menuShape(l); box.x != cx || box.y != cy {
		t.Errorf("the box is at %d,%d and the cover under the cursor at %d,%d", box.x, box.y, cx, cy)
	}

	// And a press on a verb runs it and puts the menu away.
	x, y := wordAt(t, m, m.actions.verbs[0].label)
	got, cmd := m.mouseClick(clickAt(x, y))
	if got.actions.open {
		t.Error("pressing a verb left the menu up")
	}
	if cmd == nil {
		t.Error("pressing a verb did nothing")
	}

	// A press anywhere else puts it away and does nothing else.
	if got, _ = m.mouseClick(clickAt(leftMargin, m.height-2)); got.actions.open {
		t.Error("pressing away from the menu left it up")
	}
}

// The right button raises it on whatever it is over, having moved the cursor
// there first: a menu about a record nobody pointed at is how a track gets taken
// out of a queue by accident.
func TestARightClickRaisesTheMenuOnWhatItIsOver(t *testing.T) {
	m := queueModel(0, "alpha", "bravo", "charlie", "delta")
	m.width, m.height = 100, 40
	m.resize()

	x, y := wordAt(t, m, "charlie")
	got, _ := m.mouseClick(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseRight})
	if !got.actions.open {
		t.Fatal("the right button raised no menu")
	}
	if got.actions.title != "charlie" {
		t.Errorf("the menu is about %q, want the row it was raised over", got.actions.title)
	}
	if got.queuePane.cursor.cursor != queueRowOf(2) {
		t.Errorf("the cursor is on %d, want the row the menu is about", got.queuePane.cursor.cursor)
	}
}

// A second press on the same cell is what enter does there. One is not: on a
// wall of records the pointer passes over a hundred covers on the way to one.
func TestASecondPressOpensWhatTheFirstChose(t *testing.T) {
	m := likedModel(t)

	items := m.libraryTiles()
	x, y := wordAt(t, m, items[0].name)

	got, _ := m.mouseClick(clickAt(x, y))
	if got.open() != nil {
		t.Fatal("one press opened a record")
	}
	twice, _ := got.mouseClick(clickAt(x, y))
	if twice.open() == nil {
		t.Fatal("two presses opened nothing")
	}

	// And two presses far enough apart are two presses.
	slow := got
	slow.lastClickAt = slow.lastClickAt.Add(-time.Second)
	if again, _ := slow.mouseClick(clickAt(x, y)); again.open() != nil {
		t.Error("two presses a second apart opened a record, want them read as two")
	}
}

// Ctrl and the wheel move the track rather than the cursor. The whole of the
// gesture is in the second notch: the row it moved has climbed out from under
// the pointer, and it is still the row that moves.
func TestCtrlAndTheWheelMoveTheTrackItStartedOn(t *testing.T) {
	m := queueModel(0, "alpha", "bravo", "charlie", "delta")
	m.width, m.height = 100, 40
	m.resize()

	x, y := wordAt(t, m, "alpha")
	got, _ := m.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl})
	if ids(got.queue)[0] != "bravo" || ids(got.queue)[1] != "alpha" {
		t.Fatalf("one notch left the queue %v, want alpha one place down", ids(got.queue))
	}
	if got.queuePane.cursor.cursor != queueRowOf(1) {
		t.Errorf("the cursor is on %d, want it to have gone with the track", got.queuePane.cursor.cursor)
	}

	// The pointer has not moved and "bravo" is under it now. The run is still
	// about alpha.
	got, _ = got.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl})
	if ids(got.queue)[2] != "alpha" {
		t.Fatalf("two notches left the queue %v, want alpha two places down", ids(got.queue))
	}

	// Slowly is still one hold: what ends it is the pointer moving, not the
	// clock. A second between notches is somebody turning a wheel.
	slow := got
	slow.gripAt = slow.gripAt.Add(-time.Second)
	slow, _ = slow.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl})
	if ids(slow.queue)[3] != "alpha" {
		t.Errorf("a slow third notch left the queue %v, want alpha still the one moving", ids(slow.queue))
	}

	// The queue is bravo, charlie, alpha, delta by now, and the pointer is on
	// the row bravo has ended up in.

	// A cell nobody has touched for a while is holding nothing: the next notch
	// takes whatever is under the pointer then, which is bravo.
	cold := got
	cold.gripAt = cold.gripAt.Add(-2 * gripWithin)
	cold, _ = cold.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl})
	if ids(cold.queue)[1] != "bravo" {
		t.Errorf("a cold notch left the queue %v, want bravo — the row under the pointer — to have moved", ids(cold.queue))
	}

	// And moving the pointer ends it too: a row down is charlie.
	moved := got
	moved, _ = moved.mouseWheel(tea.MouseWheelMsg{X: x, Y: y + 1, Button: tea.MouseWheelDown, Mod: tea.ModCtrl})
	if ids(moved.queue)[2] != "charlie" {
		t.Errorf("a notch a row down left the queue %v, want charlie to have moved instead", ids(moved.queue))
	}

	// Nowhere else: the library is not a list anybody put in an order.
	wall := likedModel(t)
	before := wall.libraryTiles()
	wx, wy := wordAt(t, wall, before[0].name)
	if after, _ := wall.mouseWheel(tea.MouseWheelMsg{X: wx, Y: wy, Button: tea.MouseWheelDown, Mod: tea.ModCtrl}); after.library.cursor().cursor != wall.library.cursor().cursor {
		t.Error("ctrl and the wheel moved something on the library wall")
	}
}

// A bar is taken hold of, moved, and let go. What is on screen follows the
// pointer the whole way; what is sent is sent once, at the end.
func TestDraggingThePlayheadShowsWhereItWillLand(t *testing.T) {
	m := playerModel()
	l := m.layout()
	at, w, ok := m.barSpan(l, spotSeek)
	if !ok {
		t.Fatal("there is no bar to take hold of")
	}

	_, row := wordAt(t, m, knob)
	held, cmd := m.mouseClick(clickAt(at+2, row))
	if !held.drag.on {
		t.Fatal("pressing the bar took hold of nothing")
	}
	if cmd != nil {
		t.Error("pressing the bar sent something, want it sent on release")
	}

	// Dragged to the far end, and away from the row it started on: a scrub is
	// not a test of aim.
	held, _ = held.mouseMotion(tea.MouseMotionMsg{X: at + w, Y: row + 3, Button: tea.MouseLeft})
	if held.playhead() != m.ps.Duration {
		t.Errorf("dragged to the end the bar shows %s, want %s", held.playhead(), m.ps.Duration)
	}
	// And the clock beside it says the same thing.
	if !strings.Contains(ansi.Strip(strings.Join(held.infoBlock(l.infoWidth), "\n")), formatDuration(m.ps.Duration)+" ") {
		t.Error("the clock does not follow the bar it is written under")
	}

	// Nothing has been asked of the device yet.
	if held.elapsed() == m.ps.Duration {
		t.Error("the drag moved the track itself, want it moved on release")
	}

	done, cmd := held.mouseRelease(tea.MouseReleaseMsg{X: at + w, Y: row + 3, Button: tea.MouseLeft})
	if cmd == nil {
		t.Error("letting go asked for nothing")
	}
	if done.drag.on {
		t.Error("letting go left the bar held")
	}
}

// The meter is the same gesture at another width, and the number beside it has
// to agree with it.
func TestDraggingTheMeterMovesTheVolume(t *testing.T) {
	m := playerModel()
	l := m.layout()
	at, w, ok := m.barSpan(l, spotVolume)
	if !ok {
		t.Fatal("there is no meter to take hold of")
	}

	_, row := wordAt(t, m, iconNext)
	held, _ := m.mouseClick(clickAt(at, row))
	if !held.drag.on || held.drag.kind != spotVolume {
		t.Fatalf("pressing the meter took hold of %v", held.drag)
	}

	held, _ = held.mouseMotion(tea.MouseMotionMsg{X: at + w/2, Y: row, Button: tea.MouseLeft})
	if got := held.heldVolume(); got < 45 || got > 55 {
		t.Errorf("dragged to the middle the meter shows %d, want about half", got)
	}
	if !strings.Contains(ansi.Strip(held.transportLine(l.infoWidth)), fmt.Sprintf("%3d", held.heldVolume())) {
		t.Error("the reading beside the meter disagrees with the meter")
	}

	done, cmd := held.mouseRelease(tea.MouseReleaseMsg{X: at + w, Y: row, Button: tea.MouseLeft})
	if cmd == nil {
		t.Error("letting go of the meter asked for nothing")
	}
	if done.ps.Volume != 100 {
		t.Errorf("letting go at the far end set the volume to %d, want all of it", done.ps.Volume)
	}
}

// Ctrl and the wheel over a wall of pictures size them, which is what they do
// over a grid of icons anywhere.
func TestCtrlAndTheWheelSizeTheWall(t *testing.T) {
	m := wideLibrary(t, 60)
	m.covers = &cover.Loader{}
	// Wide enough that there is more than one step in either direction: on a
	// hundred columns the ceiling is a notch away.
	m.width, m.height = 180, 44
	m.resize()

	x, y := wordAt(t, m, m.kindLabels()[0])
	y += gridChromeRows + 1

	was := m.libraryShape(m.layout(), m.layout().bodyHeight).tileW
	bigger, cmd := m.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp, Mod: tea.ModCtrl})
	if got := bigger.libraryShape(bigger.layout(), bigger.layout().bodyHeight).tileW; got <= was {
		t.Errorf("a notch up left the covers %d cells wide, want more than %d", got, was)
	}
	if cmd == nil {
		t.Error("the covers were not asked for again at the new size")
	}

	// Every notch does something until it runs out of room, and what it keeps is
	// what the wall could actually do — not the ask.
	seen := map[int]bool{bigger.library.cols: true}
	for range 40 {
		before := bigger.library.cols
		bigger, _ = bigger.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp, Mod: tea.ModCtrl})
		if bigger.library.cols == before {
			break
		}
		seen[bigger.library.cols] = true
	}
	if g := bigger.libraryShape(bigger.layout(), bigger.layout().bodyHeight); g.tileW > tileMost {
		t.Errorf("turned as far as it goes the covers are %d cells wide, want no more than %d", g.tileW, tileMost)
	}
	if len(seen) < 2 {
		t.Errorf("only %d sizes of wall were reachable turning up", len(seen))
	}

	smaller := m
	for range 40 {
		before := smaller.library.cols
		smaller, _ = smaller.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown, Mod: tea.ModCtrl})
		if smaller.library.cols == before {
			break
		}
	}
	if g := smaller.libraryShape(smaller.layout(), smaller.layout().bodyHeight); g.tileW < tileLeast {
		t.Errorf("turned the other way the covers are %d cells wide, want no less than %d", g.tileW, tileLeast)
	}
}

// Every switch on the settings screen answers where it is drawn, and ctrl and
// the wheel turn the one under the pointer.
func TestCtrlAndTheWheelTurnTheSwitchUnderIt(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSettings
	m.width, m.height = 100, 40
	m.resize()

	rows := m.settingRows()
	for i, row := range rows {
		x, y := wordAt(t, m, row.name)
		if at := m.spotAt(x, y); at.kind != spotList || at.at != i {
			t.Fatalf("%q is at column %d of row %d, and the pointer calls it %v/%d",
				row.name, x, y, at.kind, at.at)
		}
	}

	// The second switch, turned from over itself rather than from the cursor.
	x, y := wordAt(t, m, rows[1].name)
	got, _ := m.mouseWheel(tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelUp, Mod: tea.ModCtrl})
	if got.settings.cursor.cursor != 1 {
		t.Errorf("the cursor is on %d, want the switch under the pointer", got.settings.cursor.cursor)
	}
	if got.settingRows()[1].value == rows[1].value {
		t.Errorf("%q still reads %q, want it turned", rows[1].name, rows[1].value)
	}
}

// The big screen is watched rather than worked on, so a press of the pointer is
// the way out of it.
func TestAPressLeavesTheBigScreen(t *testing.T) {
	m := playerModel()
	m.stage.on = true

	if at := m.spotAt(10, 10); at.kind != spotNothing {
		t.Errorf("the big screen has %v under the pointer, want nothing to point at", at.kind)
	}
	got, _ := m.mouseClick(clickAt(10, 10))
	if got.stage.on {
		t.Error("a press left the big screen up")
	}
}

// playerModel is the player tab with something playing on it, at a size where
// there is room for the picture beside the words.
func playerModel() Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{
		TrackID: "now", Title: "playing", Artists: []string{"someone"}, Album: "somewhere",
		Duration: 5 * time.Minute, Volume: 40, Playing: true,
	}
	m.setProgress(2 * time.Minute)
	m.width, m.height = 120, 40
	m.resize()
	return m
}

// wideLibrary is the library tab with enough playlists on it to scroll.
func wideLibrary(t *testing.T, count int) Model {
	t.Helper()

	m := likedModel(t)
	for i := len(m.library.playlists); i < count; i++ {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID:       string(rune('a'+i%26)) + string(rune('a'+i/26)),
			Name:     "list " + string(rune('a'+i%26)) + string(rune('a'+i/26)),
			Owner:    "someone",
			CoverURL: "https://example.invalid/cover.jpg",
			Tracks:   10,
		})
	}
	return m
}
