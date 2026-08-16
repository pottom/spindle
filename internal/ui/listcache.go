package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
	"github.com/pottom/spindle/internal/xdg"
)

// Keeping the lists that were expensive to read.
//
// A list is read to its end now — see walk.go — and a long one is sixty requests
// against a quota that turned out to be a day long. Doing that again every time
// somebody opens the same playlist is the shape of spending it, so what was read
// is written down and read back at once on the way in.
//
// It shows immediately, which is the part worth having on its own: a playlist
// with three thousand tracks in it used to arrive fifty at a time while you
// watched, and now it is simply there.
//
// It is not trusted, though. The live first page is asked for at the same time,
// and what it says decides: a first page that repeats what the head of the held
// list already says is a list nobody has touched, and everything read past it
// stands. A first page that differs is a list that changed, so it is thrown away
// and read through again. That is the freshness check spindle can afford —
// Spotify's own `snapshot_id` would be cheaper still and is not in the player
// interface yet.

// listsHeld is how long a written-down list is worth reading back.
//
// A week. What it costs to be wrong is one wasted walk, because the live first
// page is what decides; what it saves is every opening in between.
const listsHeld = 7 * 24 * time.Hour

// heldList is a list as it was last read through.
type heldList struct {
	Tracks []player.Track    `json:"tracks,omitempty"`
	Albums []player.Album    `json:"albums,omitempty"`
	Lists  []player.Playlist `json:"playlists,omitempty"`
	People []player.Artist   `json:"artists,omitempty"`
	At     time.Time         `json:"at"`
}

// listsDir is where they are kept, or "" where nothing can be.
func listsDir() string {
	base, err := xdg.CacheDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(base, "lists")
	if os.MkdirAll(dir, 0o755) != nil {
		return ""
	}
	return dir
}

// listPath is the file one list is written down in.
func listPath(name string) string {
	dir := listsDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, safeName(name)+".json")
}

// safeName keeps a key to what is certainly a filename on every platform spindle
// builds for.
func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('.')
		}
	}
	return b.String()
}

// keepList writes a list down, once it has been read through.
func keepList(name string, held heldList) tea.Cmd {
	path := listPath(name)
	if path == "" {
		return nil
	}
	held.At = time.Now()

	return func() tea.Msg {
		if data, err := json.Marshal(held); err == nil {
			_ = os.WriteFile(path, data, 0o600)
		}
		return nil
	}
}

// readList reads one back, or nothing.
func readList(name string) (heldList, bool) {
	path := listPath(name)
	if path == "" {
		return heldList{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return heldList{}, false
	}

	var held heldList
	if json.Unmarshal(data, &held) != nil || time.Since(held.At) >= listsHeld {
		return heldList{}, false
	}
	return held, true
}

// openedName is what an opened page is filed under.
func openedName(kind openKind, id string) string {
	return "open-" + kind.String() + "-" + id
}

// readOpened hands a written-down page back as though it had just been fetched.
//
// As though, because it is the same thing: a list, from the top, complete. The
// live first page that follows it decides whether it was right — see adopt.
func readOpened(kind openKind, id string) tea.Cmd {
	held, ok := readList(openedName(kind, id))
	if !ok || (len(held.Tracks) == 0 && len(held.Albums) == 0) {
		return nil
	}
	return func() tea.Msg {
		return msg.OpenedFetched{
			ID: id, Tracks: held.Tracks, Albums: held.Albums, Offset: 0, More: false,
		}
	}
}

// keepOpened writes one down, once its last page has arrived.
func (m Model) keepOpened() tea.Cmd {
	page := m.open()
	if page == nil || !page.pages.whole || page.count() == 0 {
		return nil
	}
	return keepList(openedName(page.kind, page.id), heldList{
		Tracks: page.tracks, Albums: page.albums,
	})
}

// libraryName is what one of the library's kinds is filed under.
func libraryName(kind libraryKind) string { return "library-" + kind.String() }

// readLibrary and keepLibrary are the same for the library's own lists.
func readLibrary(kind libraryKind) tea.Cmd {
	held, ok := readList(libraryName(kind))
	if !ok {
		return nil
	}
	out := msg.LibraryFetched{Kind: int(kind), Offset: 0}
	switch kind {
	case libraryAlbums:
		out.Albums = held.Albums
	case libraryArtists:
		out.Artists = held.People
	case libraryRecent:
		out.Tracks = held.Tracks
	default:
		out.Playlists = held.Lists
	}
	if len(out.Albums)+len(out.Artists)+len(out.Tracks)+len(out.Playlists) == 0 {
		return nil
	}
	return func() tea.Msg { return out }
}

func (m Model) keepLibrary() tea.Cmd {
	kind := m.library.kind
	if !m.library.pages[kind].whole || m.library.countOf(kind) == 0 {
		return nil
	}

	held := heldList{}
	switch kind {
	case libraryAlbums:
		held.Albums = m.library.albums
	case libraryArtists:
		held.People = m.library.artists
	case libraryRecent:
		held.Tracks = m.library.recent
	default:
		// Without the row that heads the list and is not a playlist: it is built
		// from a request of its own every time. See likedRow.
		if len(m.library.playlists) > 1 {
			held.Lists = m.library.playlists[1:]
		}
	}
	return keepList(libraryName(kind), held)
}
