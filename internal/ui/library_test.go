package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"

	"github.com/pottom/spindle/internal/ui/msg"
)

// The library holds three lists and shows one at a time. A kind nobody has
// asked for costs no request until it is switched to.
func TestTheLibraryTurnsBetweenItsKinds(t *testing.T) {
	m := likedModel(t)
	if m.library.kind != libraryPlaylists {
		t.Fatalf("the library opens on %v, want the playlists", m.library.kind)
	}
	if len(m.library.albums) != 0 {
		t.Error("the saved albums were fetched before anybody asked for them")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	got := tm.(Model)
	if got.library.kind != libraryAlbums {
		t.Fatalf("ctrl+t landed on %v, want the albums", got.library.kind)
	}
	if cmd == nil {
		t.Fatal("switching to a list never read asked for nothing")
	}

	// The answer fills that kind and leaves the others alone.
	albums, err := got.player.SavedAlbums(context.Background(), 0)
	if err != nil {
		t.Fatalf("SavedAlbums: %v", err)
	}
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryAlbums), Albums: albums.Items, More: albums.More, Next: albums.Next})
	got = tm.(Model)
	if got.library.count() != len(albums.Items) {
		t.Errorf("the albums list holds %d, want %d", got.library.count(), len(albums.Items))
	}
	if len(got.library.playlists) == 0 {
		t.Error("switching kinds threw the playlists away")
	}

	// Each kind keeps its own cursor, so coming back lands where it was left.
	got.library.cursors[libraryAlbums].move(1, got.library.count())
	got.library.kind = libraryPlaylists
	if got.library.cursor().cursor != 0 {
		t.Error("the playlists' cursor moved when the albums' did")
	}
}

// A saved album opens like any other, and lands on the same page a search
// result would.
func TestOpeningASavedAlbum(t *testing.T) {
	m := likedModel(t)
	albums, err := m.player.SavedAlbums(context.Background(), 0)
	if err != nil || len(albums.Items) == 0 {
		t.Fatalf("SavedAlbums: %v (%d albums)", err, len(albums.Items))
	}
	m.library.kind = libraryAlbums
	m.library.albums = albums.Items

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	page := tm.(Model).open()
	if page == nil || page.kind != openAlbum || page.id != albums.Items[0].ID {
		t.Fatalf("enter opened %v, want the album under the cursor", page)
	}
}

// The strip beside the heading says which list is on screen and how much of it
// there is, so the other two are visible without spending a row on them.
func TestTheLibraryNamesItsKinds(t *testing.T) {
	m := likedModel(t)
	strip := plain(m.libraryKinds())

	for _, want := range []string{"playlists", "albums", "artists"} {
		if !strings.Contains(strip, want) {
			t.Errorf("the strip reads %q, want it to name %q", strip, want)
		}
	}
	if !strings.Contains(strip, "playlists "+itoa(len(m.library.playlists))) {
		t.Errorf("the strip reads %q, want the number of playlists read so far", strip)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

// The history is one of the library's lists, and it is of tracks: a row there
// plays rather than opens, and the queue survives it.
func TestTheHistoryPlaysATrackAndKeepsTheQueue(t *testing.T) {
	m := likedModel(t)
	recent, err := m.player.RecentlyPlayed(context.Background(), recentMost)
	if err != nil || len(recent) < 2 {
		t.Fatalf("RecentlyPlayed: %v (%d tracks)", err, len(recent))
	}

	m.library.kind = libraryRecent
	m.library.recent = recent
	m.library.cursors[libraryRecent].move(1, len(recent))

	if got := m.cursorTrack(); got == nil || got.ID != recent[1].ID {
		t.Fatalf("cursorTrack = %v, want the row under the cursor", got)
	}

	cmd, handled := m.libraryKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !handled || cmd == nil {
		t.Fatal("enter did nothing on the history")
	}
	runControls(cmd)

	q, err := m.player.Queue(context.Background())
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if q.Current == nil || q.Current.ID != recent[1].ID {
		t.Errorf("playing %v, want the track that was chosen", q.Current)
	}
}

// The history is not a page: Spotify keeps a fixed few and walks them by
// timestamp, so what arrives replaces what was there rather than adding to it.
func TestTheHistoryIsNotPaged(t *testing.T) {
	m := likedModel(t)
	recent, _ := m.player.RecentlyPlayed(context.Background(), recentMost)

	var tm tea.Model = m
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryRecent), Tracks: recent})
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryRecent), Tracks: recent})

	if got := tm.(Model).library.countOf(libraryRecent); got != len(recent) {
		t.Errorf("the history holds %d rows after two answers, want %d", got, len(recent))
	}
}

// A request that failed is not a request still in flight. Every list marks
// itself as reading so a run of cursor keys cannot ask for the same page a
// dozen times; a failure that left the mark set would end the list there for
// good, and it would read as a list that simply stops.
func TestAFailedPageDoesNotEndTheList(t *testing.T) {
	m := likedModel(t)
	m.library.pages[libraryPlaylists].loading = true
	m.library.pages[libraryPlaylists].more = true
	m.library.cursors[libraryPlaylists].cursor = len(m.library.playlists) - 1

	var tm tea.Model = m
	tm, _ = tm.Update(msg.Error{Err: errors.New("network is down")})
	got := tm.(Model)

	if got.library.pages[libraryPlaylists].loading {
		t.Error("the list is still waiting for a page that failed")
	}
	if got.readAhead() == nil {
		t.Error("the list will not ask again after a failure")
	}
}

// The library's band is marked the same way the queue's is: a bracket down its
// left and a line to the row it describes.
//
// Always, rather than only sometimes. On the queue the band is the cursor's
// track while the list begins with the sounding one, so the mark answers a
// question that only comes up off that row; here the band follows the cursor and
// nothing else, so what it belongs to is worth saying wherever the cursor is.
func TestTheLibraryBandIsMarkedToo(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	for i, n := range []string{"Deep House", "Chill", "Runners", "Party"} {
		m.library.playlists = append(m.library.playlists,
			player.Playlist{ID: fmt.Sprintf("p%d", i), Name: n, Owner: "pottom", Tracks: 30})
	}

	for _, at := range []int{0, 2, 3} {
		m.library.cursors[m.library.kind].cursor = at

		rows := strings.Split(plain(fmt.Sprint(m.View())), "\n")
		head, elbow := -1, -1
		for i, row := range rows {
			if strings.Contains(row, pointerTL) {
				head = i
			}
			if strings.Contains(row, pointerElbow) {
				elbow = i
			}
		}
		if head < 0 || elbow < 0 {
			t.Fatalf("cursor %d: the band was not marked: bracket at %d, line ends at %d", at, head, elbow)
		}

		// It ends on the row the band is about.
		want := []string{"Deep House", "Runners", "Party"}[map[int]int{0: 0, 2: 1, 3: 2}[at]]
		if !strings.Contains(rows[elbow], want) {
			t.Errorf("cursor %d: the line points at %q, want %q", at, strings.TrimSpace(rows[elbow]), want)
		}
	}
}

// And the search's is not: its heading is the field being typed into, and a
// bracket round the results while the question is still being written is an
// answer arriving early.
func TestTheSearchBandIsNotMarked(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabSearch
	m.resize()

	if got := plain(fmt.Sprint(m.View())); strings.Contains(got, pointerTL) {
		t.Error("the search's band was marked")
	}
}
