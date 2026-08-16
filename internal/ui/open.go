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

// String names the kind, for the file a list of it is written down in.
func (k openKind) String() string {
	switch k {
	case openAlbum:
		return "album"
	case openArtist:
		return "artist"
	default:
		return "playlist"
	}
}

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

	// A search belongs to the list it was made in, and this is another list.
	m.find = find{}

	m.stack = append(m.stack, page)
	return tea.Batch(
		// What was read through last time, from the disk, at once — and the
		// live first page beside it, which is what decides whether it still
		// holds. See listcache.go.
		readOpened(page.kind, page.id),
		fetchOpenCmd(m.player, page.kind, page.id, 0),
		m.syncCover(), m.syncNotes(), m.spinner.Tick,
	)
}

// pop goes back one page, and reports whether there was one to go back from.
func (m *Model) pop() bool {
	if len(m.stack) == 0 {
		return false
	}
	m.stack = m.stack[:len(m.stack)-1]
	m.find = find{}
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
	case m.Offset == 0 && o.pages.whole && o.sameHead(m):
		// A first page that repeats what the head of the held list already says,
		// against a list that was read through: nobody has touched it, and
		// everything read past this page still stands.
		//
		// This is what makes writing a list down worth anything. Without it the
		// first live page would replace three thousand tracks with fifty and the
		// whole walk would happen again — every time, on every refresh. See
		// listcache.go.
		o.pages.took(false, 0)
		return

	case m.Offset == 0:
		// Where the cursor is, before the list under it is replaced. A first
		// page arrives when a record is opened and again every time it is
		// refreshed, and a refresh that sent the reader back to the top would
		// make a list impossible to read for longer than the refresh takes.
		was := o.at()

		o.tracks, o.albums = m.Tracks, m.Albums
		o.cursor.reset()
		o.keepOn(was)
		// A list read afresh is a list to read through again.
		o.pages.pages, o.pages.whole = 0, false
	case o.holdsAlbums():
		o.albums = append(o.albums, m.Albums...)
	default:
		o.tracks = append(o.tracks, m.Tracks...)
	}
	o.pages.took(m.More, m.Next)
}

// sameHead reports whether a freshly fetched first page says exactly what the
// head of the list already says.
//
// By identity and in order, because that is what a change looks like: a track
// added, removed or moved shifts the ids about, and the first page is where
// Spotify puts the front of the list. It is not proof — an edit past the fiftieth
// track leaves the head alone — and it does not have to be: the cost of being
// wrong is a stale tail until the next time the list is opened, and the cost of
// not doing it is reading every list from the beginning for ever.
//
// A page longer than what is held is a longer list, and so a different one.
func (o openPage) sameHead(m msg.OpenedFetched) bool {
	if o.holdsAlbums() {
		if len(m.Albums) == 0 || len(m.Albums) > len(o.albums) {
			return false
		}
		for i := range m.Albums {
			if m.Albums[i].ID != o.albums[i].ID {
				return false
			}
		}
		return true
	}

	if len(m.Tracks) == 0 || len(m.Tracks) > len(o.tracks) {
		return false
	}
	for i := range m.Tracks {
		if m.Tracks[i].ID != o.tracks[i].ID {
			return false
		}
	}
	return true
}

// at is what the cursor is resting on, by id, so it can be found again in a list
// that has been fetched afresh.
func (o openPage) at() string {
	if o.holdsAlbums() {
		if a := atAlbum(o.albums, o.cursor.cursor); a != nil {
			return a.ID
		}
		return ""
	}
	if t := at(o.tracks, o.cursor.cursor); t != nil {
		return t.ID
	}
	return ""
}

// keepOn puts the cursor back on the thing it was on, wherever that has moved
// to. A thing that is no longer in the list leaves the cursor at the top, which
// is where a list that has changed under somebody has to start again.
func (o *openPage) keepOn(id string) {
	if id == "" {
		return
	}
	if o.holdsAlbums() {
		for i := range o.albums {
			if o.albums[i].ID == id {
				o.cursor.moveTo(i, o.count())
				return
			}
		}
		return
	}
	for i := range o.tracks {
		if o.tracks[i].ID == id {
			o.cursor.moveTo(i, o.count())
			return
		}
	}
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
