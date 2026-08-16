package ui

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A list read through is written down, and read back at once next time. What
// makes that worth anything is this: the live first page that follows says the
// same thing, and everything read past it stands.
//
// Without it the fifty tracks of the live first page would replace the three
// thousand that came off the disk, and the whole walk would happen again — every
// time, and on every refresh.
func TestAFirstPageThatRepeatsItselfKeepsTheRest(t *testing.T) {
	whole := make([]player.Track, 200)
	for i := range whole {
		whole[i] = player.Track{ID: string(rune('a'+i%26)) + string(rune('0'+i/26)), Title: "one"}
	}

	page := openPage{kind: openPlaylist, id: "p1"}
	page.adopt(msg.OpenedFetched{ID: "p1", Tracks: whole, Offset: 0, More: false})
	if !page.pages.whole || page.count() != len(whole) {
		t.Fatalf("the written-down list came back as %d tracks, whole=%v", page.count(), page.pages.whole)
	}

	// The live first page: the front of the same list.
	page.adopt(msg.OpenedFetched{ID: "p1", Tracks: whole[:50], Offset: 0, More: true, Next: 50})
	if page.count() != len(whole) {
		t.Errorf("a first page that said nothing new left %d tracks, want all %d",
			page.count(), len(whole))
	}
	if page.pages.more {
		t.Error("it went back to reading a list it already holds")
	}
}

// And a list that changed is read again. The head is where an edit shows: a
// track added, removed or moved shifts the ids about.
func TestAFirstPageThatDiffersThrowsTheRestAway(t *testing.T) {
	held := []player.Track{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}

	page := openPage{kind: openPlaylist, id: "p1"}
	page.adopt(msg.OpenedFetched{ID: "p1", Tracks: held, Offset: 0, More: false})

	// Something new at the top.
	fresh := []player.Track{{ID: "z"}, {ID: "a"}}
	page.adopt(msg.OpenedFetched{ID: "p1", Tracks: fresh, Offset: 0, More: true, Next: 2})

	if page.count() != len(fresh) {
		t.Errorf("a changed list kept %d tracks, want the %d that were fetched",
			page.count(), len(fresh))
	}
	if !page.pages.more || page.pages.whole {
		t.Error("a changed list was not read through again")
	}
}

// The same argument for the library's own kinds, where the playlists are held
// one along: the row that heads them is not one of them.
func TestTheLibraryKeepsWhatItHasReadThrough(t *testing.T) {
	liked := player.Playlist{ID: likedID, Name: "Liked Songs"}
	lists := []player.Playlist{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}}

	var pane libraryPane
	pane.adopt(msg.LibraryFetched{Kind: int(libraryPlaylists), Playlists: lists, Offset: 0}, liked)
	if pane.countOf(libraryPlaylists) != len(lists)+1 {
		t.Fatalf("the library holds %d, want the lists and the row that heads them",
			pane.countOf(libraryPlaylists))
	}

	pane.adopt(msg.LibraryFetched{
		Kind: int(libraryPlaylists), Playlists: lists[:2], Offset: 0, More: true, Next: 2,
	}, liked)
	if pane.countOf(libraryPlaylists) != len(lists)+1 {
		t.Errorf("a first page that said nothing new left %d, want all of them",
			pane.countOf(libraryPlaylists))
	}
}

// And the round trip itself: a list written down comes back as though it had
// just been fetched, whole, from the top.
func TestAListWrittenDownComesBack(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	held := heldList{Tracks: []player.Track{{ID: "a", Title: "one"}, {ID: "b", Title: "two"}}}
	cmd := keepList(openedName(openPlaylist, "p1"), held)
	if cmd == nil {
		t.Fatal("nowhere to write it down")
	}
	cmd()

	back := readOpened(openPlaylist, "p1")
	if back == nil {
		t.Fatal("what was written down did not come back")
	}
	got, ok := back().(msg.OpenedFetched)
	if !ok {
		t.Fatalf("it came back as %T", back())
	}
	if got.ID != "p1" || len(got.Tracks) != 2 || got.More || got.Offset != 0 {
		t.Errorf("it came back as %+v, want the whole list from the top", got)
	}

	// Another list is another file.
	if readOpened(openAlbum, "p1") != nil {
		t.Error("an album read a playlist's list")
	}

	// And one written down long enough ago is not read back at all: the cost of
	// being wrong is a wasted walk, but a list from a fortnight ago is a list
	// nobody should be shown while a live one is on its way.
	old := heldList{Tracks: held.Tracks, At: time.Now().Add(-2 * listsHeld)}
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if err := os.WriteFile(listPath(openedName(openPlaylist, "p2")), data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if readOpened(openPlaylist, "p2") != nil {
		t.Error("a list older than it is worth was read back")
	}
}
