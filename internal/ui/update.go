package ui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
		m.resize()
		return m, m.syncCover()

	case tea.BackgroundColorMsg:
		m.isDark = message.IsDark()
		m.restyle()
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(message)

	case msg.Tick:
		return m.handleTick()

	case msg.StateFetched:
		m.adopt(message.State)
		m.err = nil
		return m, m.syncCover()

	case msg.CoverReady:
		if m.cover.matches(message.URL, message.Width, message.Height) {
			m.cover.art = message.Art.Cells
			m.cover.accent, m.cover.hasAccent = message.Art.Accent, message.Art.HasAccent
			m.restyle()
		}
		return m, nil

	case msg.CoverFailed:
		if m.cover.matches(message.URL, message.Width, message.Height) {
			m.cover.failed = true
		}
		return m, nil

	case msg.CoverSettled:
		if message.Seq == m.coverSeq {
			return m, m.syncCover()
		}
		return m, nil

	case msg.PlaylistsFetched:
		m.playlists.items = message.Playlists
		m.playlists.cursor.reset()
		return m, m.syncCover()

	case msg.PlaylistTracksFetched:
		if m.playlists.open != nil && m.playlists.open.ID == message.PlaylistID {
			m.playlists.tracks = message.Tracks
			m.playlists.inner.reset()
		}
		return m, nil

	case msg.SearchResults:
		return m, m.applySearchResults(message)

	case msg.Error:
		m.err = message.Err
		return m, nil

	case spinner.TickMsg:
		// The spinner only exists to cover an artwork download. Letting it run
		// otherwise would mean a redraw every 100 ms for nothing.
		if !m.cover.loading() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(message)
		return m, cmd
	}

	return m, nil
}

// resize propagates the terminal size into the components that cache a width.
func (m *Model) resize() {
	if !fitsMinimum(m.width, m.height) {
		return
	}
	m.help.SetWidth(min(m.width, maxFrameWidth) - leftMargin - rightMargin)
}

// handleTick advances the local clock and resynchronises every fifth second.
func (m Model) handleTick() (tea.Model, tea.Cmd) {
	m.tickCount++
	if m.ps != nil && m.ps.Playing {
		m.localProgress += time.Second
		if m.localProgress > m.ps.Duration {
			m.localProgress = m.ps.Duration
		}
	}

	cmds := []tea.Cmd{tickCmd()}
	if m.tickCount%resyncEvery == 0 {
		cmds = append(cmds, fetchStateCmd(m.player))
	}
	return m, tea.Batch(cmds...)
}

// adopt merges a server snapshot. Inside the optimistic window only metadata is
// taken over, so a poll that has not yet seen a local change cannot undo it.
// See DESIGN.md 4.2.
func (m *Model) adopt(st *player.State) {
	if st == nil {
		return
	}
	if m.ps == nil || !time.Now().Before(m.optimisticUntil) {
		m.ps = st
		m.localProgress = st.Progress
		return
	}

	m.ps.TrackID = st.TrackID
	m.ps.Title = st.Title
	m.ps.Artists = st.Artists
	m.ps.Album = st.Album
	m.ps.CoverURL = st.CoverURL
	m.ps.Duration = st.Duration
	m.ps.DeviceID = st.DeviceID
	m.ps.DeviceName = st.DeviceName
}

func (m Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab switching outranks everything, including the search field: it is the
	// one key that has to work wherever you are.
	switch {
	case key.Matches(k, m.keys.NextTab):
		return m, m.switchTab(m.tab.next(1))
	case key.Matches(k, m.keys.PrevTab):
		return m, m.switchTab(m.tab.next(-1))
	}

	if cmd, handled := m.browseKey(k); handled {
		return m, cmd
	}

	switch {
	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(k, m.keys.Help):
		// Expanding the help shortens the body, which shrinks the artwork, so
		// the cover has to be rendered again at the new size.
		m.help.ShowAll = !m.help.ShowAll
		m.resize()
		return m, m.syncCover()
	}

	if m.ps == nil {
		return m, nil
	}

	p := m.player
	switch {
	case key.Matches(k, m.keys.PlayPause):
		m.ps.Playing = !m.ps.Playing
		m.hold()
		play := m.ps.Playing
		return m, controlCmd("toggle playback", func(ctx context.Context) error {
			if play {
				return p.Play(ctx)
			}
			return p.Pause(ctx)
		})

	case key.Matches(k, m.keys.Next):
		m.localProgress = 0
		m.ps.Progress = 0
		m.hold()
		return m, controlCmd("skip to next track", p.Next)

	case key.Matches(k, m.keys.Prev):
		m.localProgress = 0
		m.ps.Progress = 0
		m.hold()
		return m, controlCmd("skip to previous track", p.Previous)

	case key.Matches(k, m.keys.SeekFwd):
		return m, m.seek(m.localProgress + seekStep)

	case key.Matches(k, m.keys.SeekBack):
		return m, m.seek(m.localProgress - seekStep)

	case key.Matches(k, m.keys.VolUp):
		return m, m.setVolume(m.ps.Volume + volumeStep)

	case key.Matches(k, m.keys.VolDown):
		return m, m.setVolume(m.ps.Volume - volumeStep)

	case key.Matches(k, m.keys.Shuffle):
		m.ps.Shuffle = !m.ps.Shuffle
		m.hold()
		on := m.ps.Shuffle
		return m, controlCmd("set shuffle", func(ctx context.Context) error {
			return p.SetShuffle(ctx, on)
		})

	case key.Matches(k, m.keys.Repeat):
		m.ps.Repeat = nextRepeat(m.ps.Repeat)
		m.hold()
		mode := m.ps.Repeat
		return m, controlCmd("set repeat", func(ctx context.Context) error {
			return p.SetRepeat(ctx, mode)
		})
	}

	return m, nil
}

// syncCover starts an artwork load whenever the cover or the area it has to fill
// has changed. The artwork now scales with the window, so a resize invalidates
// what was rendered just as a track change does.
func (m *Model) syncCover() tea.Cmd {
	if m.covers == nil || !fitsMinimum(m.width, m.height) {
		return nil
	}

	l := m.layout()
	url := m.coverTarget()
	if m.cover.matches(url, l.artWidth, l.artHeight) {
		return nil
	}

	// Keep the accent from the outgoing cover until the new one arrives, so the
	// palette does not flash back to its default mid-swap.
	m.cover = coverState{
		url:       url,
		width:     l.artWidth,
		height:    l.artHeight,
		accent:    m.cover.accent,
		hasAccent: m.cover.hasAccent,
	}
	if m.cover.url == "" {
		m.cover.failed = true
		return nil
	}
	return tea.Batch(
		coverCmd(m.covers, m.cover.url, l.artWidth, l.artHeight),
		m.spinner.Tick,
	)
}

// hold opens the window during which local state outranks the server's.
func (m *Model) hold() {
	m.optimisticUntil = time.Now().Add(optimisticWindow)
}

func (m *Model) seek(pos time.Duration) tea.Cmd {
	pos = min(max(pos, 0), m.ps.Duration)
	m.localProgress = pos
	m.ps.Progress = pos
	m.hold()

	p := m.player
	return controlCmd("seek", func(ctx context.Context) error {
		return p.Seek(ctx, pos)
	})
}

func (m *Model) setVolume(pct int) tea.Cmd {
	pct = min(max(pct, 0), 100)
	m.ps.Volume = pct
	m.hold()

	p := m.player
	return controlCmd("set volume", func(ctx context.Context) error {
		return p.SetVolume(ctx, pct)
	})
}

func nextRepeat(mode string) string {
	switch mode {
	case player.RepeatOff:
		return player.RepeatContext
	case player.RepeatContext:
		return player.RepeatTrack
	default:
		return player.RepeatOff
	}
}
