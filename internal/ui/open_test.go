package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// searchedModel is the search tab with one kind of result on screen.
func searchedModel(t *testing.T, kind player.SearchKind) Model {
	t.Helper()

	p := player.NewMock()
	res, err := p.Search(context.Background(), "bowie", "", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	m := New(p, nil, defaultTestCell)
	m.tab = tabSearch
	m.width, m.height = 120, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(msg.SearchResults{Query: "bowie", Results: res})
	got := tm.(Model)
	got.search.kind = kind
	return got
}

// A record found by searching opens rather than plays: a list you have just
// found is one you want to look inside before committing the speakers to it.
func TestEnterOpensAnAlbumFromTheSearch(t *testing.T) {
	m := searchedModel(t, player.SearchAlbums)
	want := m.search.current().albums[0]

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	page := tm.(Model).open()
	if page == nil {
		t.Fatal("enter opened nothing")
	}
	if page.kind != openAlbum || page.id != want.ID {
		t.Errorf("opened %v %q, want the album %q", page.kind, page.id, want.ID)
	}
}

// An artist's page lists their records, and choosing one goes into it — which
// is what the stack is for. esc walks back the way it came.
func TestAnArtistOpensTheirRecordsAndEscWalksBack(t *testing.T) {
	m := searchedModel(t, player.SearchArtists)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("opening an artist asked for nothing")
	}
	runControls(cmd)

	// The page fills from the answer, as the runtime would deliver it.
	got := tm.(Model)
	albums, err := got.player.ArtistAlbums(context.Background(), got.open().id, 0)
	if err != nil {
		t.Fatalf("ArtistAlbums: %v", err)
	}
	tm, _ = tm.Update(msg.OpenedFetched{ID: got.open().id, Albums: albums.Items})
	got = tm.(Model)
	if !got.open().holdsAlbums() || got.open().count() == 0 {
		t.Fatalf("the artist page holds %d rows, want their records", got.open().count())
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = tm.(Model)
	if len(got.stack) != 2 || got.open().kind != openAlbum {
		t.Fatalf("the stack is %d deep on a %v, want an album on top of the artist", len(got.stack), got.open().kind)
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	got = tm.(Model)
	if len(got.stack) != 1 || got.open().kind != openArtist {
		t.Error("esc did not go back to the artist")
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := tm.(Model); got.open() != nil {
		t.Error("esc did not leave the last page")
	}
}

// An album plays as a record: choosing a track starts the album there, so what
// follows is the rest of the record rather than nothing.
func TestPlayingFromAnAlbumKeepsTheRecord(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabSearch

	res, err := p.Search(context.Background(), "bowie", player.SearchAlbums, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	album := res.Albums.Items[0]
	tracks, err := p.AlbumTracks(context.Background(), album.ID, 0)
	if err != nil || len(tracks.Items) < 2 {
		t.Fatalf("AlbumTracks: %v (%d tracks)", err, len(tracks.Items))
	}

	page := openedAlbum(album)
	page.tracks = tracks.Items
	page.cursor.move(1, len(page.tracks))
	m.stack = append(m.stack, page)

	cmd, handled := m.openKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !handled || cmd == nil {
		t.Fatal("enter did nothing on an album")
	}
	runControls(cmd)

	q, err := p.Queue(context.Background())
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if q.Current == nil || q.Current.ID != tracks.Items[1].ID {
		t.Errorf("playing %v, want the track the cursor was on", q.Current)
	}
	if len(q.Upcoming) == 0 {
		t.Error("nothing follows it, want the rest of the record")
	}
}

// The menu on a track offers the way to where it came from, and only where the
// backend said where that was.
func TestTheMenuGoesToTheAlbumAndTheArtist(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	verbs := m.actionsFor(player.Track{
		ID: "t1", Title: "Heroes", Album: "\"Heroes\"", AlbumID: "al1",
		Artists: []string{"David Bowie"}, ArtistIDs: []string{"ar1"},
	})

	var labels []string
	for _, v := range verbs {
		labels = append(labels, v.label)
	}
	joined := strings.Join(labels, " | ")
	if !strings.Contains(joined, "Go to the album") || !strings.Contains(joined, "Go to David Bowie") {
		t.Errorf("the menu offers %s, want the way to the record and the artist", joined)
	}

	// A queue entry from the daemon carries names and no ids, and a verb that
	// cannot act must not be offered.
	bare := m.actionsFor(player.Track{ID: "t1", Title: "Heroes", Artists: []string{"David Bowie"}})
	for _, v := range bare {
		if strings.HasPrefix(v.label, "Go to") {
			t.Errorf("the menu offers %q for a track with no ids", v.label)
		}
	}
}

// Inside an opened record every track carries its own sleeve, which costs the
// list half its rows: two rows a track, the title on one and the artists under
// it.
func TestAnOpenedRecordCarriesItsSleeves(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()

	var tracks []player.Track
	for i := range 20 {
		tracks = append(tracks, player.Track{
			ID: fmt.Sprintf("t%d", i), Title: fmt.Sprintf("Track %d", i),
			Artists: []string{"Someone"}, CoverURL: "http://example.invalid/a.jpg",
			Duration: 3 * time.Minute,
		})
	}
	m.stack = append(m.stack, openPage{kind: openPlaylist, id: "p0", name: "Deep Cuts", tracks: tracks})

	if !m.showsRowArt() {
		t.Fatal("a wide terminal does not draw the sleeves")
	}
	if got := m.listRowHeight(); got != openRowRows {
		t.Errorf("a track takes %d rows, want %d", got, openRowRows)
	}

	// The pager counts tracks, not rows: half as many fit.
	l := m.layout()
	rows := m.listBodyRows(l.bodyHeight, m.listBandRows(l))
	if got, want := m.visibleListRows(), rows/openRowRows; got != want {
		t.Errorf("the page holds %d tracks, want %d", got, want)
	}

	// Each track is drawn over two rows, its artists under its title.
	screen := strings.Split(plain(fmt.Sprint(m.View())), "\n")
	title, under := -1, -1
	for i, row := range screen {
		if strings.Contains(row, "Track 3") {
			title = i
			under = i + 1
			break
		}
	}
	if title < 0 || under >= len(screen) {
		t.Fatal("the third track is not on the screen")
	}
	if !strings.Contains(screen[under], "Someone") {
		t.Errorf("the row under the title is %q, want the artists", screen[under])
	}

	// And with no room for a sleeve it is a table again.
	m.width = rowArtFrom - 1
	m.resize()
	if m.showsRowArt() {
		t.Error("a narrow terminal still draws the sleeves")
	}
	if got := m.listRowHeight(); got != 1 {
		t.Errorf("a track takes %d rows without its sleeve, want 1", got)
	}
}
