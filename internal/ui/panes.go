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

	open   *player.Playlist // nil while browsing the top level
	tracks []player.Track
	inner  listState
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

// searchPane is the search tab: a query and its results.
type searchPane struct {
	input   textinput.Model
	results []player.Track
	cursor  listState

	// seq rises with every query, so a slow search that lands after a newer one
	// can be thrown away.
	seq int
}

func newSearchPane() searchPane {
	in := textinput.New()
	in.Prompt = "⌕ "
	in.Placeholder = "title, artist or album"
	in.Focus()
	return searchPane{input: in}
}

// selected returns the result under the cursor.
func (s searchPane) selected() *player.Track {
	if s.cursor.cursor < 0 || s.cursor.cursor >= len(s.results) {
		return nil
	}
	return &s.results[s.cursor.cursor]
}

func (s searchPane) cover() string {
	if sel := s.selected(); sel != nil {
		return sel.CoverURL
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
		return *m.nowQueued, true
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
