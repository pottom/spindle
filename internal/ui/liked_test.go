package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// likedModel is the library with both its requests answered: the playlists, and
// the first page of the saved tracks.
func likedModel(t *testing.T) Model {
	t.Helper()

	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabLibrary
	m.width, m.height = 100, 40
	m.resize()

	page, err := p.PlaylistsPage(context.Background(), 0)
	if err != nil {
		t.Fatalf("PlaylistsPage: %v", err)
	}
	liked, err := p.LikedTracks(context.Background(), 0)
	if err != nil {
		t.Fatalf("LikedTracks: %v", err)
	}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryPlaylists), Playlists: page.Items, More: page.More, Next: page.Next})
	tm, _ = tm.Update(msg.OpenedFetched{
		ID: likedID, Tracks: liked.Items, More: liked.More, Next: liked.Next,
	})
	return tm.(Model)
}

// The saved tracks head the library, and they are read once as it loads so the
// row can say how many there are and show a cover before it is opened.
func TestLikedSongsHeadTheLibrary(t *testing.T) {
	m := likedModel(t)

	first := m.library.playlists[0]
	if !isLiked(first.ID) {
		t.Fatalf("the library starts with %q, want the saved tracks", first.Name)
	}
	if first.CoverURL == "" {
		t.Error("the saved tracks have no cover, want the most recently saved track's")
	}
	// The count is only shown once the list has been read to its end: it
	// arrives a page at a time, and a number that grows beside the fixed ones
	// under it reads as a mistake rather than as progress.
	if first.Tracks != 0 {
		t.Errorf("the row says %d tracks after one page of many, want no count yet", first.Tracks)
	}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.OpenedFetched{
		ID: likedID, Tracks: m.library.liked, More: false,
	})
	if got := tm.(Model).library.playlists[0].Tracks; got != len(m.library.liked) {
		t.Errorf("the row says %d tracks once the list is read out, want %d", got, len(m.library.liked))
	}

	row := plain(m.playlistRow(first, 80, false))
	if !strings.Contains(row, likedMark) || !strings.Contains(row, "Liked Songs") {
		t.Errorf("the row reads %q, want the heart and the name", row)
	}
}

// Opening it shows the tracks that were already read, rather than an empty list
// waiting on a request that has been made once already.
func TestOpeningLikedSongsShowsThemAtOnce(t *testing.T) {
	m := likedModel(t)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := tm.(Model)

	if got.open() == nil || !isLiked(got.open().id) {
		t.Fatal("enter did not open the saved tracks")
	}
	if len(got.open().tracks) == 0 {
		t.Error("the list came up empty, want the page that was already read")
	}
}

// Liked songs have no uri anybody can name, so playing one hands the list over
// track by track — and the rest of what has been read follows it.
func TestPlayingALikedSongCarriesTheRest(t *testing.T) {
	m := likedModel(t)
	showOpen(&m, player.Playlist{ID: likedID, Name: "Liked Songs"}, m.library.liked)

	req := m.playOpenList(1)
	if err := req.call(context.Background(), m.player); err != nil {
		t.Fatalf("play: %v", err)
	}

	state, err := m.player.State(context.Background())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if want := m.library.liked[1].ID; state.TrackID != want {
		t.Errorf("playing %q, want the track the cursor was on (%q)", state.TrackID, want)
	}

	q, err := m.player.Queue(context.Background())
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if len(q.Upcoming) == 0 {
		t.Error("nothing follows the track that was played, want the rest of the list")
	}
}

// The menu over the saved tracks offers what can be done to them, and not the
// link, which they do not have.
func TestTheMenuOverLikedSongsHasNoLink(t *testing.T) {
	m := likedModel(t)
	if !m.openActions() {
		t.Fatal("no menu over the saved tracks")
	}

	for _, v := range m.actions.verbs {
		if strings.Contains(v.label, "link") {
			t.Errorf("the menu offers %q, and the saved tracks have no address", v.label)
		}
	}
}
