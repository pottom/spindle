package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

func TestListMoveClamps(t *testing.T) {
	var l listState

	l.move(-1, 5)
	if l.cursor != 0 {
		t.Errorf("cursor = %d, want 0 at the top", l.cursor)
	}

	l.move(99, 5)
	if l.cursor != 4 {
		t.Errorf("cursor = %d, want 4 at the bottom", l.cursor)
	}
}

func TestListMoveOnEmptyList(t *testing.T) {
	l := listState{cursor: 3, top: 2}
	l.move(1, 0)
	if l.cursor != 0 || l.top != 0 {
		t.Errorf("cursor, top = %d, %d; want 0, 0", l.cursor, l.top)
	}
}

// The window has to scroll only as far as it must: jumping the viewport around
// a cursor that is already visible is what makes a list feel unsteady.
func TestListWindowScrollsMinimally(t *testing.T) {
	const count, height = 20, 5
	var l listState

	if from, to := l.window(count, height); from != 0 || to != 5 {
		t.Fatalf("initial window = %d..%d, want 0..5", from, to)
	}

	// Moving inside the visible range must not scroll at all.
	l.move(4, count)
	if from, to := l.window(count, height); from != 0 || to != 5 {
		t.Errorf("window = %d..%d after moving to the last visible row, want 0..5", from, to)
	}

	// One step past it scrolls by exactly one.
	l.move(1, count)
	if from, to := l.window(count, height); from != 1 || to != 6 {
		t.Errorf("window = %d..%d, want 1..6", from, to)
	}

	// Jumping back to the top brings the window with it.
	l.move(-99, count)
	if from, to := l.window(count, height); from != 0 || to != 5 {
		t.Errorf("window = %d..%d after returning to the top, want 0..5", from, to)
	}
}

func TestListWindowClampsToContents(t *testing.T) {
	l := listState{cursor: 40, top: 30}
	from, to := l.window(3, 10)
	if from != 0 || to != 3 {
		t.Errorf("window = %d..%d, want 0..3 when the list shrank underneath", from, to)
	}
	if l.cursor != 2 {
		t.Errorf("cursor = %d, want 2", l.cursor)
	}
}

func TestListWindowHandlesNothingToShow(t *testing.T) {
	var l listState
	if from, to := l.window(0, 5); from != 0 || to != 0 {
		t.Errorf("window = %d..%d, want 0..0 for an empty list", from, to)
	}
	if from, to := l.window(10, 0); from != 0 || to != 0 {
		t.Errorf("window = %d..%d, want 0..0 with no room", from, to)
	}
}

// defaultTestCell is a measured cell of the shape a modern terminal reports, so
// layout-dependent tests do not silently exercise the fallback.
var defaultTestCell = cover.CellSize{Width: 17, Height: 41, Measured: true}

// pagedQueue is a queue tab holding more tracks than any terminal could show,
// which is the only situation in which paging means anything.
func pagedQueue(n int) Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "playing", Playing: true}
	m.tab = tabQueue
	for i := range n {
		m.queue = append(m.queue, trackAt(fmt.Sprintf("id%02d", i), fmt.Sprintf("track %02d", i)))
	}
	m.width, m.height = 100, 40
	m.resize()
	return m
}

// listRowPattern matches a drawn list row by its gutter: the cursor if it is
// there, then the ordinal or the playing mark, then the title.
var listRowPattern = regexp.MustCompile(`(?m)^ +(?:▸ +)?(?:♪|\d+) {2}\S`)

func drawnListRows(screen string) int {
	return len(listRowPattern.FindAllString(screen, -1))
}

// A page has to be a screenful, or the key lies about where it lands. The view
// draws the rows and the key handler moves by them without being able to ask
// the view anything, so this is what keeps the two of them saying one number.
func TestAPageIsTheRowsOnScreen(t *testing.T) {
	m := pagedQueue(60)

	page := m.visibleListRows()
	if drawn := drawnListRows(plain(m.render())); drawn != page {
		t.Fatalf("the queue drew %d rows but a page moves by %d", drawn, page)
	}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if got := tm.(Model).queuePane.cursor.cursor; got != page {
		t.Errorf("page down landed on row %d, want %d — one screenful", got, page)
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if got := tm.(Model).queuePane.cursor.cursor; got != 0 {
		t.Errorf("page up landed on row %d, want the screenful back", got)
	}
}

// However far a key asks the cursor to go, it stops at the list.
func TestPagingStopsAtTheEnds(t *testing.T) {
	m := pagedQueue(60)
	last := len(m.queueRows()) - 1

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if got := tm.(Model).queuePane.cursor.cursor; got != 0 {
		t.Errorf("page up from the top left the cursor on %d, want 0", got)
	}

	for range 20 {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	if got := tm.(Model).queuePane.cursor.cursor; got != last {
		t.Errorf("paging past the bottom left the cursor on %d, want the last row %d", got, last)
	}
}

// home and end reach the ends of whichever list is under them, and so do g and
// G — the library has two levels and both are driven by the same keys.
func TestTheEndsAreOneKeyAway(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabLibrary
	for i := range 30 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%02d", i), Name: fmt.Sprintf("playlist %02d", i),
		})
	}
	m.width, m.height = 100, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if got := tm.(Model).library.cursors[libraryPlaylists].cursor; got != 29 {
		t.Errorf("end landed on %d, want the last playlist", got)
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if got := tm.(Model).library.cursors[libraryPlaylists].cursor; got != 0 {
		t.Errorf("home landed on %d, want the first playlist", got)
	}

	// Inside a playlist the same keys drive the tracks.
	opened := tm.(Model)
	var tracks []player.Track
	for i := range 40 {
		tracks = append(tracks, trackAt(fmt.Sprintf("t%02d", i), "track"))
	}
	showOpen(&opened, player.Playlist{ID: "p00", Name: "playlist 00"}, tracks)

	tm = opened
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'G', Text: "G", Mod: tea.ModShift})
	if got := tm.(Model).open().cursor.cursor; got != 39 {
		t.Errorf("G landed on track %d, want the last one", got)
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if got := tm.(Model).open().cursor.cursor; got != 0 {
		t.Errorf("g landed on track %d, want the first one", got)
	}
}

// While the query has the keyboard every printable key belongs to it, so g has
// to type rather than jump. The keys that are not letters still move the
// results.
func TestGTypesOnTheSearchTabAndPageUpDoesNot(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSearch
	found := m.search.of(player.SearchTracks)
	for i := range 40 {
		found.tracks = append(found.tracks, trackAt(fmt.Sprintf("r%02d", i), "result"))
	}
	found.cursor.cursor = 20
	m.width, m.height = 100, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	got := tm.(Model)
	if q := got.search.input.Value(); q != "g" {
		t.Errorf("query = %q, want the letter in it rather than a jump", q)
	}

	// Typing starts the query over, so put a list back before testing the keys
	// that are not letters.
	found = got.search.of(player.SearchTracks)
	for i := range 40 {
		found.tracks = append(found.tracks, trackAt(fmt.Sprintf("r%02d", i), "result"))
	}
	found.cursor.cursor = 20

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	paged := tm.(Model)
	if c := paged.search.current().cursor.cursor; c >= 20 {
		t.Errorf("page up left the cursor on %d, want it a screenful higher", c)
	}
}

// The picker is a list like any other, and the keys that move a list move it.
func TestThePickerTakesTheSameKeys(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "a", Playing: true}
	m.devices.open = true
	m.devices.items = devices("", "laptop", "phone", "speaker")
	m.width, m.height = 100, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if sel := tm.(Model).devices.selected(); sel == nil || sel.ID != "speaker" {
		t.Errorf("end landed on %v, want the last device", sel)
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if sel := tm.(Model).devices.selected(); sel == nil || sel.ID != "laptop" {
		t.Errorf("page up landed on %v, want the first device", sel)
	}
}

// The row under the heading names the columns, and stands over the words it
// names rather than beside them.
func TestTheColumnsAreNamed(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 32
	m.resize()
	for i := range m.queue {
		m.queue[i] = player.Track{ID: fmt.Sprintf("t%d", i), Title: "Mockingbird",
			Artists: []string{"ReMan"}, Album: "Nightfall", Duration: 3 * time.Minute, Tempo: 108}
	}
	m.queuePane.room = queueRoomList

	rows := strings.Split(plain(fmt.Sprint(m.View())), "\n")
	head, first := -1, -1
	for i, row := range rows {
		if head < 0 && strings.Contains(row, "title") && strings.Contains(row, "artist") {
			head = i
		}
		if head >= 0 && first < 0 && strings.Contains(row, "Mockingbird") {
			first = i
		}
	}
	if head < 0 || first < 0 {
		t.Fatalf("the columns are named on row %d and the first track is on %d", head, first)
	}

	// A line under the names, and the list under that.
	if got := strings.TrimSpace(rows[head+1]); !strings.HasPrefix(got, pointerH) {
		t.Errorf("row %d under the names is %q, want the line", head+1, got)
	}
	if got := rows[head+2]; strings.TrimSpace(got) == "" {
		t.Errorf("row %d is blank, want the list to start there", head+2)
	}

	// Each name over its own column, to the column.
	for name, cell := range map[string]string{
		"title": "Mockingbird", "artist": "ReMan", "album": "Nightfall", "time": "3:00",
	} {
		at, under := strings.Index(rows[head], name), strings.Index(rows[first], cell)
		if at < 0 || under < 0 {
			t.Errorf("%q or %q is missing from the screen", name, cell)
			continue
		}
		if name == "time" {
			// Set to the right, like the cell under it.
			at, under = at+len(name), under+len(cell)
		}
		if at != under {
			t.Errorf("%q starts at %d and %q at %d", name, at, cell, under)
		}
	}

	// Nothing to name over an empty list.
	m.queue, m.ps = nil, nil
	m.queuePane.cursor = listState{}
	if got := plain(fmt.Sprint(m.View())); strings.Contains(got, " title ") {
		t.Error("an empty list still names its columns")
	}
}
