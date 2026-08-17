package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/notes"
	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// searched is a screen that has just been answered, with the query given.
func searched(t *testing.T, query string, res player.Results) Model {
	t.Helper()

	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 140, 40
	m.tab = tabSearch
	m.search.input.SetValue(query)
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(msg.SearchResults{Seq: m.search.seq, Query: query, Matched: true, Results: res})
	return tm.(Model)
}

func page[T any](items ...T) player.Page[T] { return player.Page[T]{Items: items} }

// Searching for an artist and being given a list of their songs is nearly right
// and quietly wrong: what was asked for was the artist, and the answer is one
// row that has to be found among twenty that look like it.
//
// So the strongest answer is the first row, and the cursor arrives on it.
func TestTheTopResultIsTheFirstRow(t *testing.T) {
	m := searched(t, "queen", player.Results{
		Tracks:  page(player.Track{ID: "t1", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}}),
		Artists: page(player.Artist{ID: "a1", Name: "Queen", Followers: 42_000_000}),
	})

	if m.search.kind != searchAll {
		t.Fatalf("the query landed on %q", m.search.kind)
	}
	if !m.onTop() {
		t.Fatal("the cursor did not arrive on the top result")
	}
	if got := m.cursorArtist(); got == nil || got.Name != "Queen" {
		t.Errorf("what the cursor is on is %v, want the artist", got)
	}

	screen := ansi.Strip(m.render())
	if !strings.Contains(screen, "artist") {
		t.Error("the row does not say what it is")
	}
	if !strings.Contains(screen, "Bohemian Rhapsody") {
		t.Error("the songs are not under it")
	}

	// And it opens as what it is rather than playing something.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if page := tm.(Model).open(); page == nil || page.kind != openArtist {
		t.Error("enter on the top result did not open the artist")
	}
}

// A query that is a phrase rather than a name is answered by the songs, and
// naming the first of them as well would be the same track on two rows.
func TestAPhraseHasNoTopResultOfItsOwn(t *testing.T) {
	m := searched(t, "love of my life", player.Results{
		Tracks:  page(player.Track{ID: "t1", Title: "Love of My Life", Artists: []string{"Queen"}}),
		Artists: page(player.Artist{ID: "a1", Name: "Queen"}),
	})

	if m.search.top.found() {
		t.Errorf("a phrase was given a top result of kind %q", m.search.top.kind)
	}
	if m.counted() != 1 {
		t.Errorf("the view holds %d rows, want the one song", m.counted())
	}
	if got := m.search.selected(); got == nil || got.Title != "Love of My Life" {
		t.Errorf("the first row is %v, want the song", got)
	}
}

// The rows below the top result are songs, and the cursor moving off it says so.
func TestBelowTheTopResultAreSongs(t *testing.T) {
	m := searched(t, "queen", player.Results{
		Tracks: page(
			player.Track{ID: "t1", Title: "Bohemian Rhapsody"},
			player.Track{ID: "t2", Title: "Under Pressure"},
		),
		Artists: page(player.Artist{ID: "a1", Name: "Queen"}),
	})

	if m.counted() != 3 {
		t.Fatalf("the view holds %d rows, want the artist and the two songs", m.counted())
	}

	m.search.of(searchAll).cursor.cursor = 1
	if m.onTop() {
		t.Error("the second row is still the top result")
	}
	if got := m.search.selected(); got == nil || got.Title != "Bohemian Rhapsody" {
		t.Errorf("the second row is %v, want the first song", got)
	}
	if got := m.cursorArtist(); got != nil {
		t.Errorf("a song row answered as an artist: %v", got)
	}
}

// Nothing matched at all is still nothing: no row, and no answer to act on.
func TestNothingMatchedHasNoTopResult(t *testing.T) {
	m := searched(t, "zzzzzz", player.Results{})
	if m.search.top.found() || m.counted() != 0 {
		t.Errorf("a query that matched nothing has %d rows", m.counted())
	}
	if got := ansi.Strip(m.render()); !strings.Contains(got, "Nothing matched") {
		t.Error("the screen does not say that nothing matched")
	}
}

// The search's band says what the other databases know about the artist under
// the cursor: it is the same question their own page answers, and the band was
// empty here while the answer existed one tab away. See artistPanel.
func TestTheSearchBandCarriesTheArtistNotes(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabSearch
	m.search.kind = player.SearchArtists
	m.search.of(player.SearchArtists).artists = []player.Artist{
		{ID: "a1", Name: "Queen", Genres: []string{"glam rock"}, Followers: 45_000_000},
	}
	m.artists = map[string]notes.Artist{"a1": {
		Name:      "Queen",
		Line:      "British rock band",
		Area:      "London",
		Listeners: 5_000_000,
		Note:      "Formed in 1970 by Brian May, Roger Taylor and Freddie Mercury.",
	}}
	m.width, m.height = 150, 36
	m.resize()

	band := plain(strings.Join(m.searchDetail(60, 12), "\n"))
	for _, want := range []string{"British rock band", "London", "5.0M listeners", "Formed in 1970"} {
		if !strings.Contains(band, want) {
			t.Errorf("the band = %q, want %q in it", band, want)
		}
	}

	// And it is asked for once the cursor has stopped, not on the way past.
	if cmd := m.syncCursorNotes(); cmd == nil {
		t.Error("nothing was asked about the artist under the cursor")
	}

	// Nothing known is the screen it was: Spotify's own facts rather than a gap.
	m.artists = nil
	if band := plain(strings.Join(m.searchDetail(60, 12), "\n")); !strings.Contains(band, "glam rock") {
		t.Errorf("with nothing known the band = %q, want Spotify's own facts", band)
	}
}

// A screen with no query on it may not wear the furniture of a list of answers.
//
// Clearing the query threw away every kind's results and kept which of them had
// answered best, which is a place in a list rather than a copy of what was
// there: the screen went on counting one result it could no longer name, and
// drew a band, a bar of views and a row of column names round a blank row.
// Reported from a real screen. See forgetFound.
func TestClearingTheQueryLeavesNoFurniture(t *testing.T) {
	m := searched(t, "queen", player.Results{
		Tracks:  page(player.Track{ID: "t1", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}}),
		Artists: page(player.Artist{ID: "a1", Name: "Queen", Followers: 42_000_000}),
	})

	// Escape, twice: the first hands the results to the keyboard, the second
	// clears the query.
	for range 2 {
		tm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		m = tm.(Model)
	}
	if got := m.search.input.Value(); got != "" {
		t.Fatalf("the query = %q, want it cleared", got)
	}

	if n := m.counted(); n != 0 {
		t.Errorf("the screen counts %d results with no query", n)
	}
	screen := plain(strings.Join(m.searchPaneView(m.layout(), m.layout().bodyHeight), "\n"))
	for _, gone := range []string{"all ", "title", "artist", "album"} {
		if strings.Contains(screen, gone) {
			t.Errorf("the empty screen still shows %q:\n%s", gone, screen)
		}
	}
	if !strings.Contains(screen, "Type to search") {
		t.Errorf("the empty screen does not invite a search:\n%s", screen)
	}
}

// An empty box is not a question, so nothing is waiting on one.
//
// Backspacing the last letter away marked the screen as waiting and then never
// asked — the settling drops a blank query, and nothing was left to answer and
// clear it — so the spinner turned under an empty field for as long as the tab
// was open. Reported from a real screen.
func TestBackspacingTheQueryAwayStopsTheSpinner(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 140, 40
	m.tab = tabSearch
	m.search.typing = true
	m.resize()

	var tm tea.Model = m
	for _, r := range "qu" {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if !tm.(Model).listLoading() {
		t.Fatal("a query being typed is not waiting for an answer")
	}

	for range 2 {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	got := tm.(Model)
	if got.search.input.Value() != "" {
		t.Fatalf("the query = %q, want it backspaced away", got.search.input.Value())
	}
	if got.listLoading() {
		t.Error("the screen is still waiting for an answer to an empty box")
	}
	if screen := plain(strings.Join(got.searchPaneView(got.layout(), got.layout().bodyHeight), "\n")); strings.Contains(screen, "Asking Spotify") {
		t.Errorf("the spinner is still turning:\n%s", screen)
	}
}

// The year's column is eight cells because a year is four, so the kind of record
// cannot ride along in it: "1991 compilation" came out as "1991 co…" on a screen
// with sixty cells to spare. Reported from a real screen.
//
// An artist's own records are the list this is read on now — everywhere else
// records are a wall of sleeves. See searchwall.go.
func TestAnAlbumsKindDoesNotCrowdItsYear(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 165, 40
	m.resize()
	m.rowsUnrated = true

	row := plain(m.albumRow("", player.Album{
		ID: "al1", Name: "Greatest Hits II", Artists: []string{"Queen"},
		Released: "1991-10-28", Tracks: 17, AlbumType: "compilation",
	}, 150, false))

	if !strings.Contains(row, "17 · compilation") {
		t.Errorf("the row = %q, want the count and the kind together", row)
	}
	if !strings.HasSuffix(strings.TrimRight(row, " "), "1991") {
		t.Errorf("the row = %q, want the year alone at its end", row)
	}
	if strings.Contains(row, "…") {
		t.Errorf("the row = %q, want nothing cut at this width", row)
	}
}

// The answers that are things with sleeves are a wall, and the songs are not.
// What the pointer finds on it is what was drawn.
func TestTheAnswersCanBeAWall(t *testing.T) {
	m := searched(t, "queen", player.Results{
		Tracks: page(player.Track{ID: "t1", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}}),
		Albums: page(
			player.Album{ID: "al1", Name: "A Night at the Opera", Artists: []string{"Queen"}, Released: "1975", Tracks: 12},
			player.Album{ID: "al2", Name: "Hot Space", Artists: []string{"Queen"}, Released: "1982", Tracks: 11},
			player.Album{ID: "al3", Name: "The Game", Artists: []string{"Queen"}, Released: "1980", Tracks: 10},
		),
	})
	m.width, m.height = 150, 42
	m.resize()

	// The songs are never a wall: twenty times the same sleeve, and what tells
	// two songs apart is their names.
	m.search.kind = searchAll
	if m.searchWall() {
		t.Error("the songs are being shown as covers")
	}

	m.search.kind = player.SearchAlbums
	if !m.searchWall() {
		t.Fatal("the records are not being shown as covers")
	}
	if got := len(m.searchTiles()); got != 3 {
		t.Fatalf("the wall holds %d records, want 3", got)
	}

	// Every tile is where the pointer says it is, and pressing one moves the
	// cursor there and no further.
	screen := strings.Split(m.render(), "\n")
	for i, tile := range m.searchTiles() {
		x, y := -1, -1
		for row, line := range screen {
			if j := strings.Index(plain(line), tile.name); j >= 0 && row > tabBarHeight {
				x, y = lipgloss.Width(plain(line)[:j]), row
				break
			}
		}
		if y < 0 {
			t.Errorf("%q is not on the wall", tile.name)
			continue
		}
		if at := m.spotAt(x, y); at.kind != spotTile || at.at != i {
			t.Errorf("%q is at column %d of row %d, and the pointer calls it %v/%d", tile.name, x, y, at.kind, at.at)
			continue
		}
		got, _ := m.mouseClick(clickAt(x, y))
		if at := got.search.current().cursor.cursor; at != i {
			t.Errorf("pressing %q left the cursor on %d, want %d", tile.name, at, i)
		}
	}
}

// The query stands above everything else the screen holds, so every row under
// it is that much further down — for the pointer as much as for the drawing.
func TestTheSearchPointerCountsFromUnderTheQuery(t *testing.T) {
	tracks := make([]player.Track, 0, 6)
	for i := range 6 {
		tracks = append(tracks, player.Track{ID: string(rune('a' + i)), Title: "Song " + string(rune('A'+i)), Artists: []string{"Queen"}})
	}
	m := searched(t, "queen", player.Results{Tracks: page(tracks...)})
	m.search.kind = player.SearchTracks
	m.width, m.height = 150, 42
	m.resize()

	// The last row it appears on: the first is the band, which describes
	// whatever the cursor is resting on rather than being a row of the list.
	screen := strings.Split(m.render(), "\n")
	for i, track := range tracks {
		x, y := -1, -1
		for row, line := range screen {
			if j := strings.Index(plain(line), track.Title); j >= 0 && row > tabBarHeight {
				x, y = lipgloss.Width(plain(line)[:j]), row
			}
		}
		if y < 0 {
			t.Errorf("%q is not on the screen", track.Title)
			continue
		}
		if at := m.spotAt(x, y); at.kind != spotList || at.at != i {
			t.Errorf("%q is drawn on row %d, and the pointer calls it %v/%d", track.Title, y, at.kind, at.at)
		}
	}
}
