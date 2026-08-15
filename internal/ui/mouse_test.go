package ui

import (
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

	// The meter: pressed at its far end it is full, and at its near end empty.
	v := m.volumeSpan(m.layout().infoWidth)
	_, row := wordAt(t, m, iconNext)
	left := leftMargin + m.layout().artWidth + columnGap
	if got, _ = m.mouseClick(clickAt(left+v.at+v.w-1, row)); got.ps.Volume != 100 {
		t.Errorf("pressing the far end of the meter set the volume to %d, want all of it", got.ps.Volume)
	}
	if got, _ = m.mouseClick(clickAt(left+v.at, row)); got.ps.Volume != 0 {
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
