package ui

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// switchTab moves to another screen and pulls in whatever that screen needs. The
// artwork changes with the tab, so it is synced immediately rather than debounced:
// a deliberate switch is not the cursor drifting.
func (m *Model) switchTab(t tabID) tea.Cmd {
	if t == m.tab {
		return nil
	}
	m.tab = t

	// What was open belonged to the screen it was opened from. Leaving it open
	// would put an album on top of the queue, and the way back would lead to a
	// list the reader is no longer looking at.
	m.closeOpen()

	cmds := []tea.Cmd{m.syncCover()}
	// Coming back to the player is the common way the trace becomes visible
	// again; waiting for the next second would be a visible gap.
	if cmd := m.startScope(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch {
	case t == tabLibrary:
		// Fetched every time rather than once: a playlist made or renamed
		// elsewhere would otherwise never appear until spindle was restarted,
		// and one page of a library is a cheap answer.
		m.library.paging().loading = true
		cmds = append(cmds, fetchLibraryCmd(m.player, m.library.kind, 0), fetchOpenCmd(m.player, openPlaylist, likedID, 0))
	case t == tabQueue:
		// The queue is kept for the sake of instant skipping, which only needs
		// the first entry to be right. Looking at the whole list is a different
		// promise, so it is fetched again.
		cmds = append(cmds, fetchQueueCmd(m.player))
	}
	return tea.Batch(cmds...)
}

// browseKey handles the keys belonging to the playlists and search tabs. It
// returns handled=false for anything the caller should deal with itself.
func (m *Model) browseKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	// Whatever is open is the screen, whichever tab it was opened from.
	if cmd, done := m.openKey(k); done {
		return cmd, true
	}

	switch m.tab {
	case tabQueue:
		return m.queueKey(k)
	case tabLibrary:
		return m.libraryKey(k)
	case tabSearch:
		return m.searchKey(k)
	default:
		return nil, false
	}
}

// openKey drives whatever page has been opened, whichever tab it was opened
// from. It answers before the tab does: while a page is open it is the screen,
// and the list underneath is only what is waiting to be come back to.
func (m *Model) openKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	page := m.openMut()
	if page == nil {
		return nil, false
	}
	if m.listKey(k, &page.cursor, page.count(), true) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.Enter):
		// An artist's list is of records, so choosing one opens it rather than
		// playing it: an artist page is somewhere to go from, not a list to
		// start.
		if page.holdsAlbums() {
			if a := atAlbum(page.albums, page.cursor.cursor); a != nil {
				return m.push(openedAlbum(*a)), true
			}
			return nil, true
		}

		play := m.playOpenList(page.cursor.cursor)
		if t := m.cursorTrack(); t != nil {
			play.track = *t
		}
		return m.startPlay(play), true

	case key.Matches(k, m.keys.Actions), key.Matches(k, m.keys.ActionsTyped):
		m.openActions()
		return nil, true

	case key.Matches(k, m.keys.PlayOne):
		// Enter plays the list from here, which is what the official client
		// does and what makes the rest of it follow. This is the other reading:
		// one track, and whatever was playing before it is let go. Nothing
		// follows it, because nothing was asked to.
		t := m.cursorTrack()
		if t == nil {
			return nil, true
		}
		id := t.ID
		return m.startPlay(playRequest{
			action: "play track",
			track:  *t,
			call:   func(ctx context.Context, p player.Player) error { return p.PlayNow(ctx, id) },
		}), true

	case key.Matches(k, m.keys.Enqueue), key.Matches(k, m.keys.EnqueueTyped):
		if a := m.cursorAlbum(); a != nil {
			return m.enqueueList(openAlbum, a.ID, a.Name), true
		}
		if t := m.cursorTrack(); t != nil {
			return m.enqueue(t.ID), true
		}
		return nil, true

	case key.Matches(k, m.keys.Back):
		m.pop()
		return m.syncCover(), true
	}
	return nil, false
}

func (m *Model) libraryKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if m.listKey(k, m.library.cursor(), m.library.count(), true) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.SearchKind), key.Matches(k, m.keys.SearchKindBack):
		delta := 1
		if key.Matches(k, m.keys.SearchKindBack) {
			delta = -1
		}
		return m.turnLibraryKind(delta), true

	case key.Matches(k, m.keys.Enter):
		return m.openLibraryRow(), true

	case key.Matches(k, m.keys.Actions):
		m.openActions()
		return nil, true

	case key.Matches(k, m.keys.Enqueue):
		// The whole thing, since the whole thing is what the cursor is on. An
		// artist is the exception: what "all of it" would mean there is every
		// record they ever made, which is not a queue anybody asked for.
		switch {
		case m.library.atAlbum() != nil:
			a := m.library.atAlbum()
			return m.enqueueList(openAlbum, a.ID, a.Name), true
		case m.library.selected() != nil:
			sel := m.library.selected()
			return m.enqueueList(openPlaylist, sel.ID, sel.Name), true
		}
		return nil, true
	}
	return nil, false
}

// openLibraryRow goes into whatever the cursor is on, whichever kind it is.
func (m *Model) openLibraryRow() tea.Cmd {
	switch {
	case m.library.atAlbum() != nil:
		return m.push(openedAlbum(*m.library.atAlbum()))
	case m.library.atArtist() != nil:
		return m.push(openedArtist(*m.library.atArtist()))
	case m.library.selected() != nil:
		return m.push(openedPlaylist(*m.library.selected()))
	default:
		return nil
	}
}

// turnLibraryKind moves to the next kind that has anything in it, and asks for
// it if it has never been read. Kinds nobody has opened cost no requests: the
// tab loads the one it is showing and no more.
func (m *Model) turnLibraryKind(delta int) tea.Cmd {
	next := libraryKind((int(m.library.kind) + delta + libraryKinds) % libraryKinds)
	m.library.kind = next

	cmds := []tea.Cmd{m.syncCover()}
	if m.library.countOf(next) == 0 && !m.library.pages[next].loading {
		m.library.pages[next].loading = true
		cmds = append(cmds, fetchLibraryCmd(m.player, next, 0))
	}
	return tea.Batch(cmds...)
}

func (m *Model) searchKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if m.search.typing {
		return m.searchTypingKey(k)
	}

	found := m.search.current()
	if m.listKey(k, &found.cursor, found.count(), true) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.SearchType):
		m.startTyping()
		return nil, true

	case key.Matches(k, m.keys.SearchKind), key.Matches(k, m.keys.SearchKindBack):
		delta := 1
		if key.Matches(k, m.keys.SearchKindBack) {
			delta = -1
		}
		m.turnSearchKind(delta)
		return tea.Batch(m.previewCover(), m.readAhead()), true

	case key.Matches(k, m.keys.Enter):
		return m.openSearchHit(), true

	case key.Matches(k, m.keys.Actions), key.Matches(k, m.keys.ActionsTyped):
		m.openActions()
		return nil, true

	case key.Matches(k, m.keys.Enqueue), key.Matches(k, m.keys.EnqueueTyped):
		if pl := m.cursorPlaylist(); pl != nil {
			return m.enqueueList(openPlaylist, pl.ID, pl.Name), true
		}
		if sel := m.search.selected(); sel != nil {
			return m.enqueue(sel.ID), true
		}
		return nil, true

	case key.Matches(k, m.keys.PlayOne):
		if sel := m.search.selected(); sel != nil {
			id := sel.ID
			return m.startPlay(playRequest{
				action: "play track",
				track:  *sel,
				call:   func(ctx context.Context, p player.Player) error { return p.PlayNow(ctx, id) },
			}), true
		}
		return nil, true

	case key.Matches(k, m.keys.Back):
		if m.search.input.Value() == "" {
			return m.switchTab(tabPlayer), true
		}
		m.search.input.SetValue("")
		m.search.found = nil
		return m.syncCover(), true
	}

	// Everything else belongs to whoever handles it next: the digits reach the
	// tabs, d opens the devices, v turns the trace. That is the whole point of
	// typing being a mode rather than the default.
	return nil, false
}

// searchTypingKey drives the query while the keyboard belongs to it.
func (m *Model) searchTypingKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	found := m.search.current()

	// No g and G while typing: they are letters, and every letter is the query.
	if m.listKey(k, &found.cursor, found.count(), false) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.Enter), key.Matches(k, m.keys.Back):
		// Enter hands the results to the keyboard; escape does the same, and
		// then clears the query on a second press. Neither leaves the tab
		// while there is something to come back to.
		m.stopTyping()
		return nil, true

	case key.Matches(k, m.keys.ActionsTyped):
		m.openActions()
		return nil, true

	case key.Matches(k, m.keys.EnqueueTyped):
		if sel := m.search.selected(); sel != nil {
			return m.enqueue(sel.ID), true
		}
		return nil, true
	}

	// Anything else is the query. It drives a fresh search on each keystroke;
	// the sequence number keeps a slow one from overwriting a newer answer.
	before := m.search.input.Value()

	var cmd tea.Cmd
	m.search.input, cmd = m.search.input.Update(k)

	if m.search.input.Value() == before {
		return cmd, true
	}
	// A new query starts over in every kind: the counts beside it have to be
	// about what is on screen now.
	m.search.seq++
	m.search.found = nil
	m.search.current().pages.loading = true
	return tea.Batch(cmd, searchCmd(m.player, m.search.input.Value(), "", m.search.seq, 0)), true
}

// startTyping hands the keyboard to the query, from whichever tab asked.
func (m *Model) startTyping() tea.Cmd {
	cmd := m.switchTab(tabSearch)
	m.search.typing = true
	m.search.input.Focus()
	return cmd
}

// stopTyping gives the keyboard back to the list.
func (m *Model) stopTyping() {
	m.search.typing = false
	m.search.input.Blur()
}

// turnSearchKind moves to the next kind that matched anything, so a query with
// no albums does not have an empty screen among its four.
func (m *Model) turnSearchKind(delta int) {
	at := 0
	for i, kind := range player.SearchKinds {
		if kind == m.search.kind {
			at = i
		}
	}

	for range player.SearchKinds {
		at = (at + delta + len(player.SearchKinds)) % len(player.SearchKinds)
		if m.search.of(player.SearchKinds[at]).count() > 0 {
			m.search.kind = player.SearchKinds[at]
			return
		}
	}
}

// openSearchHit acts on whatever the cursor is on. A track plays; an album, an
// artist or a playlist opens, because a list you have just found is one you
// want to look inside before you commit the speakers to it. Starting it whole
// is a key away either way — the menu offers it, and so does the row above the
// list once it is open.
func (m *Model) openSearchHit() tea.Cmd {
	found := m.search.current()
	switch m.search.kind {
	case player.SearchAlbums:
		if a := atAlbum(found.albums, found.cursor.cursor); a != nil {
			return m.push(openedAlbum(*a))
		}
	case player.SearchArtists:
		if a := atArtist(found.artists, found.cursor.cursor); a != nil {
			return m.push(openedArtist(*a))
		}
	case player.SearchPlaylists:
		if p := atPlaylist(found.playlists, found.cursor.cursor); p != nil {
			return m.push(openedPlaylist(*p))
		}
	default:
		if t := at(found.tracks, found.cursor.cursor); t != nil {
			id := t.ID
			return m.startPlay(playRequest{
				action: "play track",
				track:  *t,
				call:   func(ctx context.Context, p player.Player) error { return p.PlayNow(ctx, id) },
			})
		}
	}
	return nil
}

// playOpenList starts the open page at the given position, so the rest of it
// follows.
//
// A playlist or an album is named and Spotify plays it from there. The saved
// tracks have no name Spotify will take, so they are handed over as tracks —
// which is why only what has been read of them plays: the list is walked a page
// at a time, and what has not been scrolled to has not been asked for.
func (m *Model) playOpenList(offset int) playRequest {
	page := m.open()
	switch {
	case page.kind == openAlbum:
		uri := player.AlbumURI(page.id)
		return playRequest{
			action: "play album",
			call: func(ctx context.Context, p player.Player) error {
				return p.PlayContextAt(ctx, uri, offset)
			},
		}

	case !isLiked(page.id):
		id := page.id
		return playRequest{
			action: "play playlist",
			call: func(ctx context.Context, p player.Player) error {
				return p.PlayPlaylist(ctx, id, offset)
			},
		}

	default:
		ids := trackIDs(page.tracks)
		return playRequest{
			action: "play liked songs",
			call: func(ctx context.Context, p player.Player) error {
				return p.PlayTracks(ctx, ids, offset)
			},
		}
	}
}

// trackIDs is a list of tracks as the ids the player takes.
func trackIDs(tracks []player.Track) []string {
	out := make([]string, 0, len(tracks))
	for _, t := range tracks {
		out = append(out, t.ID)
	}
	return out
}

// playContext starts a whole album, artist or playlist. The track shown while
// the device catches up is a stand-in built from what was chosen: the real one
// arrives with the next state, and a screen that said nothing in between would
// read as a key that did not work.
func (m *Model) playContext(action, uri string, showing player.Track) tea.Cmd {
	return m.startPlay(playRequest{
		action: action,
		track:  showing,
		call:   func(ctx context.Context, p player.Player) error { return p.PlayContext(ctx, uri) },
	})
}

// playRequest is an ask for something to start, and the track it will land on.
type playRequest struct {
	action string
	track  player.Track
	call   func(ctx context.Context, p player.Player) error
}

// startPlay shows the track at once and asks for it, one request at a time.
//
// The requests are absolute — "play this" — so two of them overlapping can be
// applied in either order, and the device ends up playing whichever was asked
// for first. Holding the newest back until the last is answered makes the last
// press win, and collapses a run of them into two requests rather than one each.
func (m *Model) startPlay(req playRequest) tea.Cmd {
	// A request that already knows what will be playing says so at once; a skip
	// does not, and lets the answer tell it.
	if req.track.ID != "" {
		track := req.track
		m.showTrack(&track)
	}

	// One at a time, and never faster than the floor: both the answer and the
	// clock have to be in before the next one goes out.
	if m.playInFlight || time.Since(m.playSentAt) < playFloor {
		m.playPending = &req
		if !m.playInFlight {
			m.playInFlight = true
			return tea.Batch(playFloorCmd(), m.syncCover())
		}
		return m.syncCover()
	}
	m.playInFlight = true
	return tea.Batch(m.sendPlay(req), m.syncCover())
}

func (m *Model) sendPlay(req playRequest) tea.Cmd {
	p, call := m.player, req.call
	m.playSentAt = time.Now()
	return tea.Batch(
		playCmd(req.action, func(ctx context.Context) error { return call(ctx, p) }),
		// Without this the player tab shows the previous track until the next
		// resting poll, and the key looks like it did nothing at all.
		m.awaitTrackChange(),
	)
}

// readAhead sends for the next page of whatever list is being scrolled, once the
// cursor has come near enough to the end of what is loaded.
//
// Scrolling is the only way these lists are read, so it is also the only signal
// there is that more of them is wanted. Asking on the cursor rather than on a
// key of its own means nobody has to know the list was ever cut.
func (m *Model) readAhead() tea.Cmd {
	switch {
	case m.open() != nil:
		return m.openMut().readAhead(m.player)

	case m.tab == tabLibrary:
		if !m.library.paging().wants(m.library.cursor().cursor, m.library.count()) {
			return nil
		}
		m.library.paging().loading = true
		return fetchLibraryCmd(m.player, m.library.kind, m.library.paging().next)

	case m.tab == tabSearch:
		found := m.search.current()
		if !found.pages.wants(found.cursor.cursor, found.count()) {
			return nil
		}
		found.pages.loading = true
		return searchCmd(m.player, m.search.input.Value(), m.search.kind, m.search.seq, found.pages.next)

	default:
		return nil
	}
}

// previewCover schedules an artwork load for whatever the cursor now rests on,
// after the debounce.
func (m *Model) previewCover() tea.Cmd {
	m.coverSeq++
	return coverSettleCmd(m.coverSeq)
}

// applySearchResults adopts a result set if it is still the newest one.
//
// A fresh query answers every kind at once and each is taken; reading further
// into one kind answers only that one, and only that one is added to.
func (m *Model) applySearchResults(res msg.SearchResults) tea.Cmd {
	if res.Seq != m.search.seq {
		return nil
	}

	// An empty query clears the pane rather than leaving a stale cover behind.
	if strings.TrimSpace(res.Query) == "" {
		m.search.found = nil
		return m.syncCover()
	}

	if res.Kind == "" || res.Kind == player.SearchTracks {
		r := m.search.of(player.SearchTracks)
		r.tracks = adopt(r.tracks, res.Results.Tracks.Items, res.Offset, &r.cursor)
		r.pages.took(res.Results.Tracks.More, res.Results.Tracks.Next)
	}
	if res.Kind == "" || res.Kind == player.SearchAlbums {
		r := m.search.of(player.SearchAlbums)
		r.albums = adopt(r.albums, res.Results.Albums.Items, res.Offset, &r.cursor)
		r.pages.took(res.Results.Albums.More, res.Results.Albums.Next)
	}
	if res.Kind == "" || res.Kind == player.SearchArtists {
		r := m.search.of(player.SearchArtists)
		r.artists = adopt(r.artists, res.Results.Artists.Items, res.Offset, &r.cursor)
		r.pages.took(res.Results.Artists.More, res.Results.Artists.Next)
	}
	if res.Kind == "" || res.Kind == player.SearchPlaylists {
		r := m.search.of(player.SearchPlaylists)
		r.playlists = adopt(r.playlists, res.Results.Playlists.Items, res.Offset, &r.cursor)
		r.pages.took(res.Results.Playlists.More, res.Results.Playlists.Next)
	}

	// A query whose tracks came back empty lands on whatever did match, rather
	// than opening on an empty screen with the answer one key away.
	if m.search.current().count() == 0 {
		m.turnSearchKind(1)
	}
	return m.syncCover()
}

// adopt replaces a list with a first page and extends it with a later one. The
// cursor only goes home for the first, or reading past fifty would throw the
// reader back to the top.
func adopt[T any](have, page []T, offset int, cursor *listState) []T {
	if offset > 0 {
		return append(have, page...)
	}
	cursor.reset()
	return page
}
