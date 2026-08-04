package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
