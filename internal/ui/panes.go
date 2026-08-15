package ui

import (
	"charm.land/bubbles/v2/textinput"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// libraryPane is the library tab: what the account has kept, one kind at a
// time.
//
// Three lists rather than one, because they are three kinds of thing and a
// screen that mixed them would have to explain which row was which. Each keeps
// its own cursor and its own paging, so switching kinds and coming back lands
// where the reader left off — the same bargain the search tab makes.
type libraryPane struct {
	kind libraryKind

	playlists []player.Playlist
	albums    []player.Album
	artists   []player.Artist
	recent    []player.Track

	cursors [libraryKinds]listState
	pages   [libraryKinds]paging

	// cols is how many covers the wall puts across, which ctrl and the wheel
	// change the way they size the icons in a file manager. Nought is "as many
	// as look right", which is where it starts and where it stays until somebody
	// says otherwise. Kept between runs: how large somebody wants their shelf is
	// a way of working, not a passing look. See gridFor and prefs.go.
	cols int

	// liked is the first page of the saved tracks, read whenever the library
	// is, so the row at the top of it has a cover before it is opened and opens
	// without a wait. likedAll says that page was the whole list, which is the
	// only time the count on the row can be trusted.
	liked    []player.Track
	likedAll bool

	// likedIDs is the same page as a set, for the lists that ask a track at a
	// time whether it is saved. Built where the page is taken, so it cannot say
	// something the list beside it does not.
	//
	// A set rather than a walk down the slice: the glance asks once per row per
	// frame, and the whole saved collection can be thousands of tracks long.
	likedIDs map[string]bool
}

// adoptLiked takes a freshly read first page of the saved tracks.
func (p *libraryPane) adoptLiked(tracks []player.Track, all bool) {
	p.liked, p.likedAll = tracks, all
	p.likedIDs = make(map[string]bool, len(tracks))
	for _, t := range tracks {
		p.likedIDs[t.ID] = true
	}
}

// saved reports whether a track is one of the saved ones — as far as what has
// been read of them says.
//
// As far as: the Web API refuses to answer whether one track is saved for this
// client id (measured, 403 on /me/tracks/contains), so the only source is the
// list itself. What has not been read cannot be marked, which is why nothing is
// drawn for the tracks it does not know: a blank column says "not in the part I
// have read", and a mark on every row it did not check would say something
// stronger and wrong.
func (p libraryPane) saved(id string) bool { return p.likedIDs[id] }

// libraryKind is which of the three is on screen.
type libraryKind int

const (
	libraryPlaylists libraryKind = iota
	libraryAlbums
	libraryArtists
	libraryRecent

	// libraryKinds is how many there are, and the width of the arrays above.
	libraryKinds = 4
)

// String is what the strip under the heading calls the kind.
func (k libraryKind) String() string {
	switch k {
	case libraryAlbums:
		return "albums"
	case libraryArtists:
		return "artists"
	case libraryRecent:
		return "recent"
	default:
		return "playlists"
	}
}

// count is how many rows the kind has read so far.
func (p libraryPane) countOf(k libraryKind) int {
	switch k {
	case libraryAlbums:
		return len(p.albums)
	case libraryArtists:
		return len(p.artists)
	case libraryRecent:
		return len(p.recent)
	default:
		return len(p.playlists)
	}
}

func (p libraryPane) count() int { return p.countOf(p.kind) }

// cursor and paging are the state of whichever kind is on screen.
func (p *libraryPane) cursor() *listState { return &p.cursors[p.kind] }
func (p *libraryPane) paging() *paging    { return &p.pages[p.kind] }

// at is where the cursor rests on the current kind, and nil when it rests on a
// row of another kind — which is how the actions menu tells what it is over.
func (p libraryPane) atPlaylist() *player.Playlist {
	if p.kind != libraryPlaylists {
		return nil
	}
	return atPlaylist(p.playlists, p.cursors[libraryPlaylists].cursor)
}

func (p libraryPane) atAlbum() *player.Album {
	if p.kind != libraryAlbums {
		return nil
	}
	return atAlbum(p.albums, p.cursors[libraryAlbums].cursor)
}

func (p libraryPane) atArtist() *player.Artist {
	if p.kind != libraryArtists {
		return nil
	}
	return atArtist(p.artists, p.cursors[libraryArtists].cursor)
}

// atTrack is the same for the one list here that is of tracks: what has been
// played lately.
func (p libraryPane) atTrack() *player.Track {
	if p.kind != libraryRecent {
		return nil
	}
	return at(p.recent, p.cursors[libraryRecent].cursor)
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

	// at is when the first page last arrived, which is how old what is on screen
	// is. A list somebody edits on their phone is out of date here and nothing
	// says so: this is what a refresh is measured against.
	at time.Time
}

// stale reports whether what is held is old enough to be asked for again.
func (p paging) stale(after time.Duration) bool {
	return !p.loading && time.Since(p.at) >= after
}

// wants reports whether the cursor has come near enough to the end of what is
// loaded to send for the next page.
func (p paging) wants(cursor, loaded int) bool {
	return p.more && !p.loading && cursor >= loaded-pageAhead
}

// took records a page that has arrived.
func (p *paging) took(more bool, next int) {
	p.more, p.next, p.loading = more, next, false
	p.at = time.Now()
}

// adopt takes a page of one of the lists. The first page replaces what was
// read and sends that kind's cursor home; a later one is added to it.
//
// liked is the row that heads the playlists and is not one: it is passed in
// rather than built here, because what is known about it comes from a request
// of its own.
func (p *libraryPane) adopt(m msg.LibraryFetched, liked player.Playlist) {
	kind := libraryKind(m.Kind)
	first := m.Offset == 0
	was := p.idAt(kind, p.cursors[kind].cursor)

	switch kind {
	case libraryAlbums:
		if first {
			p.albums = m.Albums
		} else {
			p.albums = append(p.albums, m.Albums...)
		}
	case libraryArtists:
		if first {
			p.artists = m.Artists
		} else {
			p.artists = append(p.artists, m.Artists...)
		}
	case libraryRecent:
		// The history is not a page: Spotify keeps some fifty entries and walks
		// them by timestamp rather than by offset, so what arrives is all of it.
		p.recent = m.Tracks
	default:
		if first {
			p.playlists = append([]player.Playlist{liked}, m.Playlists...)
		} else {
			p.playlists = append(p.playlists, m.Playlists...)
		}
	}

	if first {
		// A first page arrives on the way in and again on every refresh, and a
		// refresh that sent the reader back to the top of the wall would make it
		// unreadable for longer than the refresh takes. So the cursor goes back
		// on the thing it was on, wherever that has moved to.
		p.cursors[kind].reset()
		p.keepOn(kind, was)
	}
	p.pages[kind].took(m.More, m.Next)
}

// idAt is what the thing at a place in one of the library's lists is called, so
// a cursor can be put back on it after the list has been fetched afresh.
func (p libraryPane) idAt(kind libraryKind, i int) string {
	switch kind {
	case libraryAlbums:
		if a := atAlbum(p.albums, i); a != nil {
			return a.ID
		}
	case libraryArtists:
		if a := atArtist(p.artists, i); a != nil {
			return a.ID
		}
	case libraryRecent:
		if t := at(p.recent, i); t != nil {
			return t.ID
		}
	default:
		if pl := atPlaylist(p.playlists, i); pl != nil {
			return pl.ID
		}
	}
	return ""
}

// keepOn puts a cursor back on the thing it was on. Something that has gone from
// the list leaves it at the top, which is where a list that changed under
// somebody has to start again.
func (p *libraryPane) keepOn(kind libraryKind, id string) {
	if id == "" {
		return
	}
	for i := range p.countOf(kind) {
		if p.idAt(kind, i) == id {
			p.cursors[kind].moveTo(i, p.countOf(kind))
			return
		}
	}
}

// selected is the playlist under the cursor at the top level, which is the one
// row kind the library had before it had three.
func (p libraryPane) selected() *player.Playlist { return p.atPlaylist() }

// cover is the artwork this pane wants shown: the open playlist's, or the one
// under the cursor. The playing track's cover is deliberately not used here —
// it already has a whole tab of its own.
func (p libraryPane) cover() string {
	switch {
	case p.atTrack() != nil:
		return p.atTrack().CoverURL
	case p.atAlbum() != nil:
		return p.atAlbum().CoverURL
	case p.atArtist() != nil:
		return p.atArtist().ImageURL
	case p.selected() != nil:
		return p.selected().CoverURL
	default:
		return ""
	}
}

// queuePane is the queue tab. It holds no tracks of its own: the model already
// keeps the queue for the sake of instant skipping, and a second copy would be
// one more thing to keep in step.
type queuePane struct {
	cursor listState

	// room is which of the two blocks above the list are open. See queueRoom.
	room queueRoom
}

// queueRoom is how much of the screen the queue is given.
//
// The tab is a list with a band across the top of it: what is playing on the
// left, the picture of it on the right. Both are worth having and neither is
// worth having always — a queue is read by looking down it, and on an ordinary
// terminal the band costs a third of the rows it could be read in.
//
// So the key walks all four ways of arranging that, rather than switching one
// thing off. Two blocks make four states and a key that only toggled one of them
// would need a second key for the other, which is two keys for one question.
type queueRoom int

const (
	queueRoomBoth  queueRoom = iota // what is playing, and the picture of it
	queueRoomNow                    // what is playing, and the list
	queueRoomTrace                  // the picture, and the list
	queueRoomList                   // the list, and the whole screen for it
	queueRooms
)

// next walks to the arrangement after this one, and round.
func (r queueRoom) next() queueRoom { return (r + 1) % queueRooms }

// showsNow and showsTrace are which of the two blocks this arrangement has.
func (r queueRoom) showsNow() bool   { return r == queueRoomBoth || r == queueRoomNow }
func (r queueRoom) showsTrace() bool { return r == queueRoomBoth || r == queueRoomTrace }

func (r queueRoom) String() string {
	switch r {
	case queueRoomNow:
		return "the player"
	case queueRoomTrace:
		return "the picture"
	case queueRoomList:
		return "the list alone"
	default:
		return "the player and the picture"
	}
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

		// The measurement of what is sounding, which is in hand either way.
		// Left off, this row and the panel beside it said the tempo was unknown
		// while the header two lines above printed it — the number was there
		// and dropped on the way past. It is the first listen this branch is
		// for, and a first listen is exactly when nothing has it written down.
		Tempo: m.ps.Tempo,
	}, true
}

// queuedTrack is the row under the cursor, or nil when there is nothing to
// point at.
// cursorTrack is the track the current screen's cursor rests on, whichever
// screen that is. The panel above every list describes it, so the panel does
// not need to know which list it came from.
func (m Model) cursorTrack() *player.Track {
	switch {
	case m.open() != nil:
		page := m.open()
		if page.holdsAlbums() {
			return nil
		}
		return at(page.tracks, page.cursor.cursor)
	case m.tab == tabQueue:
		return m.queuedTrack()
	case m.tab == tabLibrary:
		return m.library.atTrack()
	case m.tab == tabSearch:
		return m.search.selected()
	default:
		return nil
	}
}

// cursorAlbum and cursorArtist are the same for the rows that are not tracks:
// an artist's records, and what a search matched.
func (m Model) cursorAlbum() *player.Album {
	if page := m.open(); page != nil {
		if page.holdsAlbums() {
			return atAlbum(page.albums, page.cursor.cursor)
		}
		return nil
	}
	if m.tab == tabSearch && m.search.kind == player.SearchAlbums {
		found := m.search.current()
		return atAlbum(found.albums, found.cursor.cursor)
	}
	return nil
}

func (m Model) cursorArtist() *player.Artist {
	if m.open() != nil || m.tab != tabSearch || m.search.kind != player.SearchArtists {
		return nil
	}
	found := m.search.current()
	return atArtist(found.artists, found.cursor.cursor)
}

// cursorPlaylist is the playlist under the cursor, where the cursor is on a
// playlist rather than on a track: the library's top level, and the search's
// playlists. Nil everywhere else, which is how the screens that have no such
// row say so.
func (m Model) cursorPlaylist() *player.Playlist {
	switch {
	case m.open() != nil:
		return nil
	case m.tab == tabLibrary:
		return m.library.selected()
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
