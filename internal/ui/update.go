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
		return m, nil

	case tea.BackgroundColorMsg:
		m.applyBackground(message.IsDark())
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(message)

	case msg.Tick:
		return m.handleTick()

	case msg.StateFetched:
		m.adopt(message.State)
		m.err = nil
		return m, nil

	case msg.Error:
		m.err = message.Err
		return m, nil

	case spinner.TickMsg:
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
	l := computeLayout(m.width, m.height, m.helpHeight(), m.err != nil)
	m.help.SetWidth(l.interior - 2)
	m.progress.SetWidth(l.infoWidth)
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
	switch {
	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(k, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.resize()
		return m, nil
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
