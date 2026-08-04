package ui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

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
	case t == tabPlaylists && m.playlists.items == nil:
		cmds = append(cmds, fetchPlaylistsCmd(m.player))
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
	case tabPlaylists:
		return m.playlistKey(k)
	case tabSearch:
		return m.searchKey(k)
	default:
		return nil, false
	}
}

func (m *Model) playlistKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(k, m.keys.Down), key.Matches(k, m.keys.Up):
		delta := 1
		if key.Matches(k, m.keys.Up) {
			delta = -1
		}
		if m.playlists.open != nil {
			m.playlists.inner.move(delta, len(m.playlists.tracks))
			return m.previewCover(), true
		}
		m.playlists.cursor.move(delta, len(m.playlists.items))
		return m.previewCover(), true

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
			return tea.Batch(fetchPlaylistTracksCmd(m.player, open.ID), m.syncCover()), true
		}

		id, offset := m.playlists.open.ID, m.playlists.inner.cursor
		p := m.player
		return tea.Batch(
			controlCmd("play playlist", func(ctx context.Context) error {
				return p.PlayPlaylist(ctx, id, offset)
			}),
			// Without this the player tab shows the previous track until the
			// next five-second poll, and pressing enter looks like it did
			// nothing at all.
			m.awaitTrackChange(),
		), true

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
	switch {
	case key.Matches(k, m.keys.Down), key.Matches(k, m.keys.Up):
		delta := 1
		if key.Matches(k, m.keys.Up) {
			delta = -1
		}
		m.search.cursor.move(delta, len(m.search.results))
		return m.previewCover(), true

	case key.Matches(k, m.keys.Enter):
		sel := m.search.selected()
		if sel == nil {
			return nil, true
		}
		id, p := sel.ID, m.player
		return tea.Batch(
			controlCmd("play track", func(ctx context.Context) error {
				return p.PlayTrack(ctx, id)
			}),
			m.awaitTrackChange(),
		), true

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
	return tea.Batch(cmd, searchCmd(m.player, m.search.input.Value(), m.search.seq)), true
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
	m.search.results = res.Tracks
	m.search.cursor.reset()

	// An empty query clears the pane rather than leaving a stale cover behind.
	if strings.TrimSpace(res.Query) == "" {
		m.search.results = nil
	}
	return m.syncCover()
}
