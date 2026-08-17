package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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
