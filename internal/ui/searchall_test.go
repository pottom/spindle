package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

// The year's column is eight cells because a year is four, so the kind of
// record cannot ride along in it: "1991 compilation" came out as "1991 co…" on
// a screen with sixty cells to spare. Reported from a real screen.
func TestAnAlbumsKindDoesNotCrowdItsYear(t *testing.T) {
	m := searched(t, "queen", player.Results{Albums: page(
		player.Album{ID: "al1", Name: "Greatest Hits II", Artists: []string{"Queen"}, Released: "1991-10-28", Tracks: 17, AlbumType: "compilation"},
	)})
	m.search.kind = player.SearchAlbums
	m.width, m.height = 165, 40
	m.resize()

	rows := m.searchPaneView(m.layout(), m.layout().bodyHeight)
	var row string
	for _, line := range rows {
		if strings.Contains(plain(line), "Greatest Hits II") {
			row = plain(line)
		}
	}
	if row == "" {
		t.Fatal("the record is not on the screen")
	}
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
