package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// openLibraryList is a playlist open on the library tab, with its tracks in.
func openLibraryList(t *testing.T) Model {
	t.Helper()

	m := likedModel(t)
	pl := m.library.playlists[1]
	page, err := m.player.PlaylistTracksPage(context.Background(), pl.ID, 0)
	if err != nil || len(page.Items) < 3 {
		t.Fatalf("PlaylistTracksPage: %v (%d tracks)", err, len(page.Items))
	}
	showOpen(&m, pl, page.Items)
	return m
}

// / looks through the list on screen rather than asking Spotify anything: the
// rows are already here, and a playlist of three hundred is exactly where you
// want to go straight to a name.
func TestSlashLooksThroughTheListOnScreen(t *testing.T) {
	m := openLibraryList(t)
	want := m.open().tracks[2]

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !tm.(Model).find.typing {
		t.Fatal("/ did not open a query over the list")
	}

	for _, r := range strings.ToLower(want.Title)[:4] {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	got := tm.(Model)
	if len(got.find.matches) == 0 {
		t.Fatalf("%q matched nothing in the list", got.find.query)
	}
	if at := got.open().cursor.cursor; got.open().tracks[at].ID != want.ID {
		t.Errorf("the cursor landed on %q, want %q", got.open().tracks[at].Title, want.Title)
	}

	// Enter keeps what was found and hands the keyboard back to the list.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := tm.(Model); got.find.typing || len(got.find.matches) == 0 {
		t.Error("enter should keep the matches and stop typing")
	}
}

// n and N walk the matches, and wrap: a list read to the bottom is one you want
// to start again, not one that stops answering.
func TestNAndShiftNWalkTheMatches(t *testing.T) {
	m := openLibraryList(t)

	// A query every row matches, so the walk is over the whole list.
	m.find = find{query: "e"}
	m.refind()
	if len(m.find.matches) < 2 {
		t.Skipf("the mock playlist has %d rows matching, want a few", len(m.find.matches))
	}

	first := m.open().cursor.cursor
	m.findStep(1)
	if m.open().cursor.cursor == first {
		t.Error("n did not move to the next match")
	}

	m.findStep(-1)
	if got := m.open().cursor.cursor; got != first {
		t.Errorf("N landed on %d, want back on %d", got, first)
	}

	// From the last match, n comes round to the first.
	m.open().cursor.moveTo(m.find.matches[len(m.find.matches)-1], m.open().count())
	m.findStep(1)
	if got := m.open().cursor.cursor; got != m.find.matches[0] {
		t.Errorf("n from the last match landed on %d, want the first (%d)", got, m.find.matches[0])
	}
}

// The transport is on n and p, and has to work from a list too — where the
// letters could just as easily have been the query, and are not.
func TestTheTransportKeysWorkInAList(t *testing.T) {
	m := openLibraryList(t)
	m.ps = &player.State{TrackID: "now", Title: "playing", Playing: true}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd == nil {
		t.Error("n did not ask for the next track")
	}
	if got := tm.(Model); got.find.typing {
		t.Error("n opened a query")
	}
}

// On the search tab there is no list of one's own to look through: / is the
// query, as it has always been.
func TestSlashOnTheSearchTabStillTypes(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSearch
	m.width, m.height = 100, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	got := tm.(Model)
	if !got.search.typing || got.find.typing {
		t.Errorf("search typing = %v, find typing = %v, want the query", got.search.typing, got.find.typing)
	}
}

// What the query matched is marked in the list, in every row that matched and
// in every column that is searched.
func TestTheQueryIsLitInTheList(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 200, 40
	m.resize()
	m.queue[1] = player.Track{ID: "b", Title: "Titanium", Artists: []string{"Titov"},
		Album: "Titanium", Duration: 3 * time.Minute}
	m.startFind()
	m.find.query = "tit"
	m.refind()

	row := m.trackRow(m.queue[1], 190, false, 2)
	if got := plain(row); !strings.Contains(got, "Titanium") {
		t.Fatalf("the row lost its words: %q", got)
	}

	// Three of them: the title, the artist, the album.
	open := m.styles.Found.Render("")
	if at := strings.Index(open, "m"); at >= 0 {
		open = open[:at+1]
	}
	if n := strings.Count(row, open); n != 3 {
		t.Errorf("%d of the row's cells are lit, want 3: %q", n, row)
	}

	// And the letters lit are the ones that matched, in their own case.
	for _, want := range []string{"Tit", "Tit", "Tit"} {
		at := strings.Index(row, open+want)
		if at < 0 {
			t.Errorf("the mark does not sit on %q: %q", want, row)
			break
		}
		row = row[at+len(open):]
	}
}

func TestASpanIsFoundWhateverTheCase(t *testing.T) {
	for _, c := range []struct {
		hay, needle string
		want        [][2]int
	}{
		{"Titanium", "tit", [][2]int{{0, 3}}},
		{"Titanium", "IUM", [][2]int{{5, 8}}},
		{"aaaa", "aa", [][2]int{{0, 2}, {2, 4}}},
		{"Titanium", "zz", nil},
		{"Titanium", "", nil},
		{"Árvíztűrő", "ÍZTŰ", [][2]int{{4, 10}}},
		{"Árvíztűrő", "Ő", [][2]int{{11, 13}}},
	} {
		got := litSpans(c.hay, c.needle)
		if len(got) != len(c.want) {
			t.Errorf("%q in %q: %v, want %v", c.needle, c.hay, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q in %q: %v, want %v", c.needle, c.hay, got, c.want)
				break
			}
		}
		for _, span := range got {
			if !strings.EqualFold(c.hay[span[0]:span[1]], c.needle) {
				t.Errorf("%q in %q marks %q", c.needle, c.hay, c.hay[span[0]:span[1]])
			}
		}
	}
}

// searchingQueue is a queue of named tracks with a query open over it.
func searchingQueue(t *testing.T, query string) Model {
	t.Helper()
	m := queueModel(0, "a", "b", "c", "d", "e", "f", "g", "h")
	m.width, m.height = 140, 34
	m.resize()
	names := []string{"Titanium", "Mockingbird", "Messy", "Titans", "Cold", "Flowers", "Hero", "Style"}
	for i := range m.queue {
		m.queue[i] = player.Track{ID: fmt.Sprintf("t%d", i), Title: names[i],
			Artists: []string{"Titov"}, Album: "Nightfall", Duration: 3 * time.Minute}
	}
	m.queuePane.room = queueRoomList
	m.startFind()
	m.find.query = query
	m.refind()
	return m
}

// The field is a box above the list, and it says what the query did.
func TestTheFieldStandsOverTheList(t *testing.T) {
	m := searchingQueue(t, "tit")
	screen := plain(fmt.Sprint(m.View()))
	if !strings.Contains(screen, pointerTL) {
		t.Fatal("the field is not on the screen")
	}
	// Every row here has Titov on it, so the query is in all of them.
	for _, want := range []string{"/", "tit", finderCaret, fmt.Sprintf("1 of %d", len(m.find.matches))} {
		if !strings.Contains(screen, want) {
			t.Errorf("the field does not show %q", want)
		}
	}

	// It takes its rows from the list rather than lying over them: the table
	// steps down by exactly the height of the box.
	m.find = find{}
	was := strings.Split(plain(fmt.Sprint(m.View())), "\n")
	m = searchingQueue(t, "tit")
	now := strings.Split(plain(fmt.Sprint(m.View())), "\n")

	find := func(rows []string, want string) int {
		for i, row := range rows {
			if strings.Contains(row, want) {
				return i
			}
		}
		return -1
	}
	for _, of := range []string{"tracks", "title", "Titanium"} {
		before, after := find(was, of), find(now, of)
		if before < 0 || after < 0 {
			t.Errorf("%q is on row %d without the field and %d with it", of, before, after)
			continue
		}
		if after != before+finderRows {
			t.Errorf("%q moved from row %d to %d, want it %d down", of, before, after, finderRows)
		}
	}
}

// It covers no track at all — the list makes room for it — and it stands in the
// same place whatever the search has found.
func TestTheFieldCoversNoTrack(t *testing.T) {
	for _, at := range []int{0, 3, 7} {
		m := searchingQueue(t, "tit")
		m.queuePane.cursor.moveTo(at, len(m.queueRows()))

		rows := strings.Split(plain(fmt.Sprint(m.View())), "\n")
		box := -1
		for i, row := range rows {
			if strings.Contains(row, pointerTL) && box < 0 {
				box = i
			}
		}
		if box < 0 {
			t.Fatalf("cursor %d: the field is not on the screen", at)
		}

		// Nothing of the list under it: the three rows it covers are the heading,
		// the names of the columns and the line under them.
		for _, name := range []string{"Titanium", "Mockingbird", "Messy", "Titans"} {
			for i := box; i < box+finderRows; i++ {
				if strings.Contains(rows[i], name) {
					t.Errorf("cursor %d: the field covers %q on row %d", at, name, i)
				}
			}
		}

		// And the first track is the row under it.
		if got := strings.TrimSpace(rows[box+finderRows]); got == "" {
			t.Errorf("cursor %d: the list does not start under the field", at)
		}
	}
}

// A query with nothing to show for it says so.
func TestTheFieldSaysWhenNothingMatched(t *testing.T) {
	m := searchingQueue(t, "zzz")
	if got := plain(fmt.Sprint(m.View())); !strings.Contains(got, "no match") {
		t.Error("a query that matched nothing does not say so")
	}
}

// Esc takes it off — the field, the marks and all — whether it is being typed
// into or has been let go of.
func TestEscapeTakesTheSearchOff(t *testing.T) {
	for _, typing := range []bool{true, false} {
		m := searchingQueue(t, "tit")
		m.find.typing = typing

		cmd, handled := m.findKey(tea.KeyPressMsg{Code: tea.KeyEscape})
		if !handled || cmd != nil {
			t.Errorf("typing=%v: esc was not taken", typing)
		}
		if m.find.query != "" || m.finding() {
			t.Errorf("typing=%v: the search is still up after esc", typing)
		}
		if got := plain(fmt.Sprint(m.View())); strings.Contains(got, pointerTL) {
			t.Errorf("typing=%v: the field is still drawn after esc", typing)
		}
	}
}

// And so does leaving the list it was made in.
func TestTheSearchBelongsToItsList(t *testing.T) {
	m := searchingQueue(t, "tit")
	m.switchTab(tabLibrary)
	if m.find.query != "" {
		t.Error("the query outlived the list it was made in")
	}
}

// The bar under the screen says how to be rid of a search, because nothing else
// did: a field that shows a count and no way out is a mode with no door.
func TestTheBarSaysHowToClearASearch(t *testing.T) {
	for _, typing := range []bool{true, false} {
		m := searchingQueue(t, "tit")
		m.find.typing = typing

		bar := plain(m.help.View(m.helpKeys()))
		if !strings.Contains(bar, "clear") {
			t.Errorf("typing=%v: the bar does not offer to clear the search: %q", typing, bar)
		}
		if !strings.Contains(bar, "next match") {
			t.Errorf("typing=%v: the bar does not offer the next match: %q", typing, bar)
		}
	}

	// And it is the list's own bar again once the search is off.
	m := searchingQueue(t, "tit")
	m.find = find{}
	if bar := plain(m.help.View(m.helpKeys())); strings.Contains(bar, "clear") {
		t.Errorf("the search's keys outlived it: %q", bar)
	}
}
