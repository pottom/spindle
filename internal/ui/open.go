package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// openPage is a thing the reader has gone into: a playlist, an album, or an
// artist. One type for all three because they are one act — a row was chosen
// and what is inside it is now the list — and three versions of that act meant
// three sets of keys, three cursors and three ways back out.
//
// What is inside differs: a playlist and an album hold tracks, an artist holds
// records. Both fields are here rather than behind an interface because the
// difference is two lines at the three places that draw or move the list, and
// an interface would spread it across as many types as there are kinds.
type openPage struct {
	kind openKind

	// id is what the backend calls it, and name and cover what the screen does.
	id, name, coverURL string

	// from is what the row said about itself where it was chosen, so the panel
	// above the list can say it too without asking the backend again.
	subtitle string

	tracks []player.Track
	albums []player.Album

	cursor listState
	pages  paging
}

type openKind int

const (
	openPlaylist openKind = iota
	openAlbum
	openArtist
)

// holdsAlbums reports whether the page's list is of records rather than tracks.
func (o openPage) holdsAlbums() bool { return o.kind == openArtist }

// count is how many rows the page has.
func (o openPage) count() int {
	if o.holdsAlbums() {
		return len(o.albums)
	}
	return len(o.tracks)
}

// cover is the artwork the page wants shown: whatever the cursor is resting on,
// and the page's own only while the list is empty.
//
// The row rather than the page, because the panel beside the picture describes
// the row. A sleeve that stayed put while the cursor moved down a track listing
// would be a picture of the heading.
func (o openPage) cover() string {
	if o.holdsAlbums() {
		if a := atAlbum(o.albums, o.cursor.cursor); a != nil {
			return a.CoverURL
		}
		return o.coverURL
	}
	if t := at(o.tracks, o.cursor.cursor); t != nil {
		return t.CoverURL
	}
	return o.coverURL
}

// openedPlaylist, openedAlbum and openedArtist are the three ways in. Each
// takes what the row already knew, so the page has a name, a cover and a
// subtitle before its first page of contents arrives.
func openedPlaylist(p player.Playlist) openPage {
	return openPage{
		kind: openPlaylist, id: p.ID, name: p.Name,
		coverURL: p.CoverURL, subtitle: p.Owner,
	}
}

func openedAlbum(a player.Album) openPage {
	return openPage{
		kind: openAlbum, id: a.ID, name: a.Name,
		coverURL: a.CoverURL, subtitle: albumLine(a),
	}
}

func openedArtist(a player.Artist) openPage {
	return openPage{
		kind: openArtist, id: a.ID, name: a.Name,
		coverURL: a.ImageURL, subtitle: artistLine(a),
	}
}

// albumLine and artistLine are what a page says under its name: enough of the
// row it was opened from that the heading does not have to be read twice.
func albumLine(a player.Album) string {
	who := strings.Join(a.Artists, ", ")
	year := releaseYear(a.Released)
	if year == "" {
		return who
	}
	if who == "" {
		return year
	}
	return who + " · " + year
}

func artistLine(a player.Artist) string {
	if a.Followers > 0 {
		return fmt.Sprintf("%s followers", formatCount(a.Followers))
	}
	return strings.Join(a.Genres, ", ")
}

// open is the page being read, or nil at the top level of a tab. Only the top
// of the stack is on screen; the rest is the way back.
func (m Model) open() *openPage {
	if len(m.stack) == 0 {
		return nil
	}
	return &m.stack[len(m.stack)-1]
}

// openMut is open for the code that changes it. Two methods rather than a
// pointer receiver on the model, because Update takes the model by value and
// the view must not be able to move a cursor.
func (m *Model) openMut() *openPage {
	if len(m.stack) == 0 {
		return nil
	}
	return &m.stack[len(m.stack)-1]
}

// push opens a page, and asks for its first contents. Anything already open
// stays underneath: going from an artist to one of their records and back is
// the whole point of the stack.
func (m *Model) push(page openPage) tea.Cmd {
	// A playlist that was read once already shows itself while the request
	// that refreshes it is in flight.
	if isLiked(page.id) {
		page.tracks = m.library.liked
	}
	page.pages = paging{loading: true}

	m.stack = append(m.stack, page)
	return tea.Batch(fetchOpenCmd(m.player, page.kind, page.id, 0), m.syncCover(), m.spinner.Tick)
}

// pop goes back one page, and reports whether there was one to go back from.
func (m *Model) pop() bool {
	if len(m.stack) == 0 {
		return false
	}
	m.stack = m.stack[:len(m.stack)-1]
	return true
}

// closeOpen leaves every page at once, which is what changing tabs means: the
// way back belongs to the screen it was opened from.
func (m *Model) closeOpen() { m.stack = nil }

// adopt takes a page of contents. The first replaces what was there and sends
// the cursor home; a later one is added to it, or reading past fifty would
// throw the reader back to the top of the list.
func (o *openPage) adopt(m msg.OpenedFetched) {
	switch {
	case m.Offset == 0:
		o.tracks, o.albums = m.Tracks, m.Albums
		o.cursor.reset()
	case o.holdsAlbums():
		o.albums = append(o.albums, m.Albums...)
	default:
		o.tracks = append(o.tracks, m.Tracks...)
	}
	o.pages.took(m.More, m.Next)
}

// readAhead sends for the next page once the cursor has come near enough to the
// end of what is loaded.
func (o *openPage) readAhead(p player.Player) tea.Cmd {
	if !o.pages.wants(o.cursor.cursor, o.count()) {
		return nil
	}
	o.pages.loading = true
	return fetchOpenCmd(p, o.kind, o.id, o.pages.next)
}
