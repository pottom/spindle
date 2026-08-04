package ui

import (
	"charm.land/bubbles/v2/textinput"

	"github.com/pottom/spindle/internal/player"
)

// playlistPane is the library tab. It has two levels: the playlists themselves,
// and the tracks inside whichever one is open.
type playlistPane struct {
	items  []player.Playlist
	cursor listState
	pages  paging

	open   *player.Playlist // nil while browsing the top level
	tracks []player.Track
	inner  listState
	within paging

	// liked is the first page of the saved tracks, read whenever the library
	// is, so the row at the top of it has a cover before it is opened and opens
	// without a wait. likedAll says that page was the whole list, which is the
	// only time the count on the row can be trusted.
	liked    []player.Track
	likedAll bool
}

// paging is what a list has read so far and what is left. Lists arrive fifty at
// a time and are read by scrolling, so the next page is fetched as the cursor
// approaches the end of what is loaded rather than by asking for it.
type paging struct {
	// more says another page exists, next is where to ask for it. Next is the
	// backend's answer rather than the number of items received: pages arrive
	// short — a podcast episode or a track unavailable here is dropped — and
	// counting what survived would re-read the same rows forever.
	more bool
	next int

	// loading stops a run of cursor keys from asking for the same page a dozen
	// times before the first answer lands.
	loading bool
}

// wants reports whether the cursor has come near enough to the end of what is
// loaded to send for the next page.
func (p paging) wants(cursor, loaded int) bool {
	return p.more && !p.loading && cursor >= loaded-pageAhead
}

// took records a page that has arrived.
func (p *paging) took(more bool, next int) {
	p.more, p.next, p.loading = more, next, false
}

// selected returns the playlist under the cursor at the top level.
func (p playlistPane) selected() *player.Playlist {
	if p.cursor.cursor < 0 || p.cursor.cursor >= len(p.items) {
		return nil
	}
	return &p.items[p.cursor.cursor]
}

// cover is the artwork this pane wants shown: the open playlist's, or the one
// under the cursor. The playing track's cover is deliberately not used here —
// it already has a whole tab of its own.
func (p playlistPane) cover() string {
	if p.open != nil {
		// Inside a playlist the panel describes the track under the cursor, so
		// the picture beside it has to be that track's. The playlist's own
		// cover is a picture of a list, which says nothing about the row being
		// read.
		if t := at(p.tracks, p.inner.cursor); t != nil {
			return t.CoverURL
		}
		return p.open.CoverURL
	}
	if sel := p.selected(); sel != nil {
		return sel.CoverURL
	}
	return ""
}

// queuePane is the queue tab. It holds no tracks of its own: the model already
// keeps the queue for the sake of instant skipping, and a second copy would be
// one more thing to keep in step.
type queuePane struct {
	cursor listState
}

// searchPane is the search tab: a query and what it matched.
//
// One kind is shown at a time, with the counts of the others beside the query,
// so what else matched is visible without spending rows on it. The four keep
// their own cursor and their own paging, because moving between kinds and
// coming back to where you were is the whole point of switching.
type searchPane struct {
	input textinput.Model

	// typing is whether the keyboard belongs to the query. Off to begin with,
	// so the tab answers the same keys as every other list until / is pressed:
	// a screen where the digits cannot reach the tabs and a full stop cannot
	// open the menu is a screen you have to escape from before you can use the
	// program.
	typing bool

	kind  player.SearchKind
	found map[player.SearchKind]*searchResults

	// seq rises with every query, so a slow search that lands after a newer one
	// can be thrown away.
	seq int
}

// searchResults is what one kind matched.
type searchResults struct {
	tracks    []player.Track
	albums    []player.Album
	artists   []player.Artist
	playlists []player.Playlist

	cursor listState
	pages  paging
}

// count is how many rows this kind has.
func (r *searchResults) count() int {
	if r == nil {
		return 0
	}
	return len(r.tracks) + len(r.albums) + len(r.artists) + len(r.playlists)
}

// of returns the results for a kind, making the room for them on first use.
//
// The empty kind is tracks: it is the zero value of the field and the kind a
// query asks for everything under, and two entries for one list would be a
// screen that quietly showed nothing.
func (s *searchPane) of(kind player.SearchKind) *searchResults {
	if kind == "" {
		kind = player.SearchTracks
	}
	if s.found == nil {
		s.found = map[player.SearchKind]*searchResults{}
	}
	if s.found[kind] == nil {
		s.found[kind] = &searchResults{}
	}
	return s.found[kind]
}

// current is the results of the kind on screen.
func (s *searchPane) current() *searchResults { return s.of(s.kind) }

func newSearchPane() searchPane {
	in := textinput.New()
	// Tracks to begin with, which is what nearly every search is for.
	in.Prompt = "⌕ "
	in.Placeholder = "title, artist or album"
	in.Focus()
	return searchPane{input: in, kind: player.SearchTracks}
}

// selected returns the track under the cursor, and nothing when the kind on
// screen is not tracks: the rest have their own accessors because they are not
// interchangeable.
func (s *searchPane) selected() *player.Track {
	r := s.current()
	if s.kind != player.SearchTracks && s.kind != "" {
		return nil
	}
	return at(r.tracks, r.cursor.cursor)
}

// cover is the artwork of whatever the cursor rests on, whichever kind that is.
func (s *searchPane) cover() string {
	r := s.current()
	switch s.kind {
	case player.SearchAlbums:
		if a := atAlbum(r.albums, r.cursor.cursor); a != nil {
			return a.CoverURL
		}
	case player.SearchArtists:
		if a := atArtist(r.artists, r.cursor.cursor); a != nil {
			return a.ImageURL
		}
	case player.SearchPlaylists:
		if p := atPlaylist(r.playlists, r.cursor.cursor); p != nil {
			return p.CoverURL
		}
	default:
		if t := at(r.tracks, r.cursor.cursor); t != nil {
			return t.CoverURL
		}
	}
	return ""
}

// queueRows is what the queue tab lists: the track sounding now, then what
// follows it. The playing track is not part of the queue and never becomes one,
// so it is prepended here rather than kept in it — the skip logic depends on
// the queue starting at what comes next.
func (m Model) queueRows() []player.Track {
	now, ok := m.nowPlayingRow()
	if !ok {
		return m.queue
	}
	rows := make([]player.Track, 0, len(m.queue)+1)
	return append(append(rows, now), m.queue...)
}

// nowPlayingRow is the track sounding now. Its identity comes from the player
// state, which the daemon updates the moment anything changes; the queue's own
// copy is only borrowed for the detail it carries, and only while the two agree
// about what is playing.
func (m Model) nowPlayingRow() (player.Track, bool) {
	if m.ps == nil || m.ps.TrackID == "" {
		return player.Track{}, false
	}
	if m.nowQueued != nil && m.nowQueued.ID == m.ps.TrackID {
		now := *m.nowQueued
		// The live measurement is fresher than whatever was recorded last time
		// this track played, and is the only one there is on a first listen.
		if m.ps.Tempo > 0 {
			now.Tempo = m.ps.Tempo
		}
		return now, true
	}
	return player.Track{
		ID:       m.ps.TrackID,
		Title:    m.ps.Title,
		Artists:  m.ps.Artists,
		Album:    m.ps.Album,
		CoverURL: m.ps.CoverURL,
		Duration: m.ps.Duration,
	}, true
}

// queuedTrack is the row under the cursor, or nil when there is nothing to
// point at.
// cursorTrack is the track the current screen's cursor rests on, whichever
// screen that is. The panel above every list describes it, so the panel does
// not need to know which list it came from.
func (m Model) cursorTrack() *player.Track {
	switch {
	case m.tab == tabQueue:
		return m.queuedTrack()
	case m.tab == tabLibrary && m.playlists.open != nil:
		return at(m.playlists.tracks, m.playlists.inner.cursor)
	case m.tab == tabSearch:
		return m.search.selected()
	default:
		return nil
	}
}

// cursorPlaylist is the playlist under the cursor, where the cursor is on a
// playlist rather than on a track: the library's top level, and the search's
// playlists. Nil everywhere else, which is how the screens that have no such
// row say so.
func (m Model) cursorPlaylist() *player.Playlist {
	switch {
	case m.tab == tabLibrary && m.playlists.open == nil:
		return m.playlists.selected()
	case m.tab == tabSearch && m.search.kind == player.SearchPlaylists:
		found := m.search.current()
		return atPlaylist(found.playlists, found.cursor.cursor)
	default:
		return nil
	}
}

// at is the element under an index, or nil when the cursor has outrun the list.
// One for each kind, because Go has no way to say it once that is worth reading.
func at(tracks []player.Track, i int) *player.Track {
	if i < 0 || i >= len(tracks) {
		return nil
	}
	return &tracks[i]
}

func atAlbum(albums []player.Album, i int) *player.Album {
	if i < 0 || i >= len(albums) {
		return nil
	}
	return &albums[i]
}

func atArtist(artists []player.Artist, i int) *player.Artist {
	if i < 0 || i >= len(artists) {
		return nil
	}
	return &artists[i]
}

func atPlaylist(playlists []player.Playlist, i int) *player.Playlist {
	if i < 0 || i >= len(playlists) {
		return nil
	}
	return &playlists[i]
}

func (m Model) queuedTrack() *player.Track {
	rows := m.queueRows()
	i := m.queuePane.cursor.cursor
	if i < 0 || i >= len(rows) {
		return nil
	}
	return &rows[i]
}

// queueIndex maps the row under the cursor onto the queue itself, or -1 when
// the cursor is on the track already playing — which is in no queue, and so
// cannot be moved or dropped.
func (m Model) queueIndex() int {
	i := m.queuePane.cursor.cursor
	if _, playing := m.nowPlayingRow(); playing {
		i--
	}
	if i < 0 || i >= len(m.queue) {
		return -1
	}
	return i
}

// clampQueueCursor keeps the cursor inside the list after it changes length.
func (m *Model) clampQueueCursor() {
	m.queuePane.cursor.move(0, len(m.queueRows()))
}
