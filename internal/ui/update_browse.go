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
		m.playlists.pages.loading = true
		cmds = append(cmds, fetchPlaylistsCmd(m.player, 0))
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
	switch m.tab {
	case tabQueue:
		return m.queueKey(k)
	case tabLibrary:
		return m.playlistKey(k)
	case tabSearch:
		return m.searchKey(k)
	default:
		return nil, false
	}
}

func (m *Model) playlistKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	// The tab has two levels and the same keys drive both: the playlists
	// themselves, or the tracks inside whichever one is open.
	state, count := &m.playlists.cursor, len(m.playlists.items)
	if m.playlists.open != nil {
		state, count = &m.playlists.inner, len(m.playlists.tracks)
	}
	if m.listKey(k, state, count, true) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.Enter):
		if m.playlists.open == nil {
			sel := m.playlists.selected()
			if sel == nil {
				return nil, true
			}
			open := *sel
			m.playlists.open = &open
			m.playlists.tracks = nil
			m.playlists.inner.reset()
			m.playlists.within = paging{loading: true}
			return tea.Batch(fetchPlaylistTracksCmd(m.player, open.ID, 0), m.syncCover()), true
		}

		id, offset := m.playlists.open.ID, m.playlists.inner.cursor
		play := playRequest{
			action: "play playlist",
			call: func(ctx context.Context, p player.Player) error {
				return p.PlayPlaylist(ctx, id, offset)
			},
		}
		if t := m.cursorTrack(); t != nil {
			play.track = *t
		}
		return m.startPlay(play), true

	case key.Matches(k, m.keys.Actions):
		m.openActions()
		return nil, true

	case key.Matches(k, m.keys.PlayOne):
		// Enter plays the list from here, which is what the official client
		// does and what makes the rest of it follow. This is the other reading:
		// one track, and whatever was playing before it is let go. Nothing
		// follows it, because nothing was asked to.
		t := m.cursorTrack()
		if m.playlists.open == nil || t == nil {
			return nil, true
		}
		id := t.ID
		return m.startPlay(playRequest{
			action: "play track",
			track:  *t,
			call:   func(ctx context.Context, p player.Player) error { return p.PlayNow(ctx, id) },
		}), true

	case key.Matches(k, m.keys.Enqueue):
		if m.playlists.open == nil {
			return nil, true
		}
		if i := m.playlists.inner.cursor; i >= 0 && i < len(m.playlists.tracks) {
			return m.enqueue(m.playlists.tracks[i].ID), true
		}
		return nil, true

	case key.Matches(k, m.keys.Back):
		if m.playlists.open == nil {
			return nil, true
		}
		m.playlists.open = nil
		m.playlists.tracks = nil
		return m.syncCover(), true
	}
	return nil, false
}

func (m *Model) searchKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	found := m.search.current()

	// No g and G on this tab: they are letters, and every letter here is part
	// of the query.
	if m.listKey(k, &found.cursor, found.count(), false) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.SearchKind), key.Matches(k, m.keys.SearchKindBack):
		delta := 1
		if key.Matches(k, m.keys.SearchKindBack) {
			delta = -1
		}
		m.turnSearchKind(delta)
		return tea.Batch(m.previewCover(), m.readAhead()), true

	case key.Matches(k, m.keys.Enter):
		return m.playSearchHit(), true

	case key.Matches(k, m.keys.ActionsTyped):
		m.openActions()
		return nil, true

	case key.Matches(k, m.keys.EnqueueTyped):
		if sel := m.search.selected(); sel != nil {
			return m.enqueue(sel.ID), true
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

	// Anything else is typing. The query drives a fresh search each keystroke;
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

// playSearchHit starts whatever the cursor is on. A track plays on its own and
// keeps the queue; an album, an artist or a playlist is a list, and plays as
// one.
func (m *Model) playSearchHit() tea.Cmd {
	found := m.search.current()
	switch m.search.kind {
	case player.SearchAlbums:
		if a := atAlbum(found.albums, found.cursor.cursor); a != nil {
			return m.playContext("play album", player.AlbumURI(a.ID), player.Track{
				Title: a.Name, Artists: a.Artists, CoverURL: a.CoverURL,
			})
		}
	case player.SearchArtists:
		if a := atArtist(found.artists, found.cursor.cursor); a != nil {
			return m.playContext("play artist", player.ArtistURI(a.ID), player.Track{
				Title: a.Name, CoverURL: a.ImageURL,
			})
		}
	case player.SearchPlaylists:
		if p := atPlaylist(found.playlists, found.cursor.cursor); p != nil {
			return m.playContext("play playlist", player.PlaylistURI(p.ID), player.Track{
				Title: p.Name, Artists: []string{p.Owner}, CoverURL: p.CoverURL,
			})
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
	case m.tab == tabLibrary && m.playlists.open != nil:
		if !m.playlists.within.wants(m.playlists.inner.cursor, len(m.playlists.tracks)) {
			return nil
		}
		m.playlists.within.loading = true
		return fetchPlaylistTracksCmd(m.player, m.playlists.open.ID, m.playlists.within.next)

	case m.tab == tabLibrary:
		if !m.playlists.pages.wants(m.playlists.cursor.cursor, len(m.playlists.items)) {
			return nil
		}
		m.playlists.pages.loading = true
		return fetchPlaylistsCmd(m.player, m.playlists.pages.next)

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
