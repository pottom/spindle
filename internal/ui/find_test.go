package ui

import (
	"context"
	"strings"
	"testing"

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
