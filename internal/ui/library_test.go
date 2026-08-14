package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"

	"github.com/pottom/spindle/internal/ui/cover"
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

// The library is a wall of covers with no band over it, so there is nothing to
// mark: every tile is already showing its own picture.
func TestTheWallIsNotMarked(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	for i, n := range []string{"Deep House", "Chill", "Runners"} {
		m.library.playlists = append(m.library.playlists,
			player.Playlist{ID: fmt.Sprintf("p%d", i), Name: n, Owner: "pottom", Tracks: 30})
	}

	if got := plain(fmt.Sprint(m.View())); strings.Contains(got, pointerTL) {
		t.Error("the wall was marked as though it had a band")
	}

	// What is opened from it is a record with its tracks under it, and that is
	// marked like any other.
	m.stack = append(m.stack, openPage{kind: openPlaylist, id: "p0", name: "Deep House",
		tracks: []player.Track{{ID: "t0", Title: "One"}, {ID: "t1", Title: "Two"}}})
	m.stack[0].cursor.cursor = 1
	if got := plain(fmt.Sprint(m.View())); !strings.Contains(got, pointerTL) {
		t.Error("a page opened from the library was not marked")
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

// The wall is walked across as well as down, and it scrolls a whole row at a
// time: a row of covers split across two screenfuls is not a row any more.
func TestTheWallIsWalkedInTwoDirections(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	for i := range 24 {
		m.library.playlists = append(m.library.playlists,
			player.Playlist{ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom"})
	}

	l := m.layout()
	g := m.libraryShape(l, l.bodyHeight)
	if g.cols < 2 || g.rows < 1 {
		t.Fatalf("the wall is %d by %d", g.cols, g.rows)
	}

	press := func(name string, code rune) {
		t.Helper()
		if !m.libraryGridKey(tea.KeyPressMsg{Code: code}) {
			t.Fatalf("the wall did not take %s", name)
		}
	}
	at := func() int { return m.library.cursors[m.library.kind].cursor }

	press("right", tea.KeyRight)
	if at() != 1 {
		t.Errorf("right moved to %d, want the next tile", at())
	}
	press("down", tea.KeyDown)
	if want := 1 + g.cols; at() != want {
		t.Errorf("down moved to %d, want %d — a row down", at(), want)
	}
	press("left", tea.KeyLeft)
	if want := g.cols; at() != want {
		t.Errorf("left moved to %d, want %d", at(), want)
	}
	press("up", tea.KeyUp)
	if at() != 0 {
		t.Errorf("up moved to %d, want the first tile", at())
	}

	// The window scrolls by rows: whatever is at the top of the screen is the
	// start of a row.
	state := &m.library.cursors[m.library.kind]
	state.cursor = len(m.library.playlists) - 1
	from, _ := state.gridWindow(len(m.library.playlists), g)
	if from%g.cols != 0 {
		t.Errorf("the wall starts at tile %d, which is %d into a row", from, from%g.cols)
	}
}

// A list asked for again keeps the cursor on the thing it was on, wherever that
// has moved to. A refresh that sent the reader back to the top would make a
// library unreadable for longer than the refresh takes.
func TestARefreshKeepsTheCursorWhereItWas(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()

	lists := []player.Playlist{
		{ID: "p0", Name: "Deep House"}, {ID: "p1", Name: "Chill"}, {ID: "p2", Name: "Runners"},
	}
	m.library.adopt(msg.LibraryFetched{Kind: int(libraryPlaylists), Playlists: lists}, player.Playlist{ID: likedID})
	m.library.cursors[libraryPlaylists].moveTo(2, m.library.count()) // "Chill", after the liked row

	was := m.library.idAt(libraryPlaylists, m.library.cursors[libraryPlaylists].cursor)
	if was == "" {
		t.Fatal("the cursor is on nothing to begin with")
	}

	// The same page again, with something new at the front: the cursor follows
	// what it was on rather than staying at its index.
	m.library.adopt(msg.LibraryFetched{
		Kind:      int(libraryPlaylists),
		Playlists: append([]player.Playlist{{ID: "pNew", Name: "Made on the phone"}}, lists...),
	}, player.Playlist{ID: likedID})

	if now := m.library.idAt(libraryPlaylists, m.library.cursors[libraryPlaylists].cursor); now != was {
		t.Errorf("after the refresh the cursor is on %q, want %q", now, was)
	}

	// And something that has gone leaves it at the top rather than pointing at
	// whatever slid into its place.
	m.library.adopt(msg.LibraryFetched{Kind: int(libraryPlaylists), Playlists: lists[:1]},
		player.Playlist{ID: likedID})
	if got := m.library.cursors[libraryPlaylists].cursor; got != 0 {
		t.Errorf("the cursor is at %d after what it was on went, want the top", got)
	}
}

// A window of a different size divides the wall into tiles of a different size,
// and a cover rendered for the old one is the wrong picture. The resize has to
// send for them again, or some never arrive at all.
func TestTheWallAsksAgainWhenTheWindowChanges(t *testing.T) {
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 12 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i),
			CoverURL: fmt.Sprintf("http://example.invalid/%d.jpg", i),
		})
	}

	if cmd := m.syncGridCovers(); cmd == nil {
		t.Fatal("the wall sent for no covers at all")
	}
	was := m.libraryShape(m.layout(), m.layout().bodyHeight)

	// Wider: bigger tiles, and every cover held is for the old size.
	var tm tea.Model = m
	tm, cmd := tm.Update(tea.WindowSizeMsg{Width: 220, Height: 50})
	m = tm.(Model)
	if now := m.libraryShape(m.layout(), m.layout().bodyHeight); now.tileW == was.tileW {
		t.Skipf("the tiles are %d cells wide at both sizes", now.tileW)
	}
	if cmd == nil {
		t.Fatal("the resize sent for nothing")
	}

	// What is held now is asked for at the new size.
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	for id, tile := range m.tiles {
		if tile.width != g.tileW || tile.height != g.coverRows {
			t.Errorf("%s is held at %dx%d, want the new %dx%d",
				id, tile.width, tile.height, g.tileW, g.coverRows)
			break
		}
	}
}
