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
	// No g and G on this tab: they are letters, and every letter here is part
	// of the query.
	if m.listKey(k, &m.search.cursor, len(m.search.results), false) {
		return tea.Batch(m.previewCover(), m.readAhead()), true
	}

	switch {
	case key.Matches(k, m.keys.Enter):
		sel := m.search.selected()
		if sel == nil {
			return nil, true
		}
		id := sel.ID
		return m.startPlay(playRequest{
			action: "play track",
			track:  *sel,
			call:   func(ctx context.Context, p player.Player) error { return p.PlayNow(ctx, id) },
		}), true

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
		m.search.results = nil
		m.search.cursor.reset()
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
	m.search.seq++
	m.search.pages = paging{loading: true}
	return tea.Batch(cmd, searchCmd(m.player, m.search.input.Value(), m.search.seq, 0)), true
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
		if !m.search.pages.wants(m.search.cursor.cursor, len(m.search.results)) {
			return nil
		}
		m.search.pages.loading = true
		return searchCmd(m.player, m.search.input.Value(), m.search.seq, m.search.pages.next)

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
func (m *Model) applySearchResults(res msg.SearchResults) tea.Cmd {
	if res.Seq != m.search.seq {
		return nil
	}
	if res.Offset == 0 {
		m.search.results = res.Tracks
		m.search.cursor.reset()
	} else {
		m.search.results = append(m.search.results, res.Tracks...)
	}
	m.search.pages.took(res.More, res.Next)

	// An empty query clears the pane rather than leaving a stale cover behind.
	if strings.TrimSpace(res.Query) == "" {
		m.search.results = nil
	}
	return m.syncCover()
}
