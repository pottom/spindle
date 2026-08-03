package ui

import (
	"context"
	"errors"
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
		if m.drivenFromElsewhere(message.State) {
			m.followUntil = time.Now().Add(followWindow)
			m.nextPollAt = time.Now().Add(activePoll)
		}
		m.adopt(message.State)
		m.noDevice = false
		m.err = nil

		cmds := []tea.Cmd{m.syncCover()}
		if m.stillWaitingForTrackChange(message.State) {
			cmds = append(cmds, refetchCmd(confirmRetry))
		}
		// Keep the queue tied to whatever the server says is playing. Comparing
		// against the confirmed track rather than the displayed one stops an
		// optimistic skip from re-asking for a queue it already has.
		if message.State != nil && message.State.TrackID != m.queueFor {
			m.queueFor = message.State.TrackID
			cmds = append(cmds, fetchQueueCmd(m.player))
		}
		return m, tea.Batch(cmds...)

	case msg.NoActiveDevice:
		// Not an error, and not worth clearing the last known track for: the
		// player screen explains itself and offers the device list instead.
		m.noDevice, m.ps, m.err = true, nil, nil
		return m, tea.Batch(fetchDevicesCmd(m.player), m.syncCover())

	case msg.DevicesFetched:
		m.devices.adopt(message.Devices)
		return m, nil

	case msg.QueueFetched:
		m.queue = message.Tracks
		return m, nil

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

	case msg.StateChanged:
		// Re-arm before fetching: the next change may land while this one is
		// still being read, and a dropped wake-up would stall the screen until
		// the next unprompted poll.
		cmds := []tea.Cmd{fetchStateCmd(m.player)}
		if w, ok := m.player.(player.Watcher); ok {
			cmds = append(cmds, watchCmd(w))
		}
		return m, tea.Batch(cmds...)

	case msg.Refetch:
		if m.throttled() {
			return m, nil
		}
		return m, fetchStateCmd(m.player)

	case msg.VolumeSettled:
		// Only the newest keystroke counts, and only if the value has moved
		// since the leading request went out.
		if message.Seq != m.volumeSeq || m.ps == nil || m.ps.Volume == m.volumeSent {
			return m, nil
		}
		return m, m.sendVolume()

	case msg.ControlDone:
		// Whatever was standing in the way is evidently no longer standing.
		m.noPremium, m.err = false, nil
		return m, nil

	case msg.RateLimited:
		m.rateLimitedUntil = time.Now().Add(message.RetryAfter)
		m.err = nil
		return m, nil

	case msg.Error:
		if errors.Is(message.Err, player.ErrPremiumRequired) {
			m.noPremium = true
			return m, nil
		}
		m.err = message.Err
		return m, nil

	case spinner.TickMsg:
		if message.ID == m.device.ID() {
			// Silence should cost nothing: a mark that turns while nothing plays
			// is a redraw every 125 ms for a lie.
			if m.ps == nil || !m.ps.Playing {
				m.deviceRun = false
				return m, nil
			}
			var cmd tea.Cmd
			m.device, cmd = m.device.Update(message)
			return m, cmd
		}

		// The other spinner only exists to cover an artwork download. Letting it
		// run otherwise would mean a redraw every 100 ms for nothing.
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
	cmds := []tea.Cmd{tickCmd()}
	if cmd := m.spinDevice(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// The tick no longer counts anything: elapsed derives the position from the
	// clock. It is still what notices a track running out, and what keeps the
	// screen redrawing once a second.
	if m.ps != nil && m.ps.Playing && m.ps.Duration > 0 && m.elapsed() >= m.ps.Duration {
		if cmd := m.trackRanOut(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if m.shouldResync() {
		m.notePolled()
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

	// While a skip is in flight, a snapshot still showing the track we are
	// leaving is stale by definition. Adopting it would undo what the queue
	// already told us and flash the old title back onto the screen.
	if m.awaitingTrack != "" && st.TrackID == m.awaitingTrack {
		return
	}
	if m.ps == nil || !time.Now().Before(m.optimisticUntil) {
		m.ps = st
		m.progressAt = time.Now()
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
	m.ps.Bitrate = st.Bitrate
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

	if cmd, handled := m.deviceKey(k); handled {
		return m, cmd
	}
	if cmd, handled := m.browseKey(k); handled {
		return m, cmd
	}

	switch {
	case key.Matches(k, m.keys.QuitAll):
		// Leaving is the common case and gets the easy key; taking the music
		// with you has to be asked for.
		m.stopDaemon = true
		return m, tea.Quit

	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(k, m.keys.Help):
		// Expanding the help shortens the body, which shrinks the artwork, so
		// the cover has to be rendered again at the new size.
		m.help.ShowAll = !m.help.ShowAll
		m.resize()
		return m, m.syncCover()
	}

	// With nothing playing there is nothing to control.
	if m.noDevice {
		return m, nil
	}
	if m.ps == nil {
		return m, nil
	}

	p := m.player
	switch {
	case key.Matches(k, m.keys.PlayPause):
		// Pin the position before the flag flips. Pausing stops the clock from
		// being carried forward, so the anchor has to already hold everything
		// that had accumulated, or the playhead drops back to wherever the last
		// poll left it.
		m.setProgress(m.elapsed())
		m.ps.Playing = !m.ps.Playing
		m.hold()
		play := m.ps.Playing
		return m, tea.Batch(m.spinDevice(), controlCmd("toggle playback", func(ctx context.Context) error {
			if play {
				return p.Play(ctx)
			}
			return p.Pause(ctx)
		}))

	case key.Matches(k, m.keys.Next):
		cmd := m.skip("skip to next track", p.Next)
		// The queue already says what is coming, so show it now instead of
		// half a second from now. The confirming fetch demotes to a check.
		if next := m.takeFromQueue(); next != nil {
			m.showTrack(next)
			return m, tea.Batch(cmd, m.syncCover())
		}
		return m, cmd

	case key.Matches(k, m.keys.Prev):
		return m, m.skip("skip to previous track", p.Previous)

	case key.Matches(k, m.keys.SeekFwd):
		return m, m.seek(m.elapsed() + seekStep)

	case key.Matches(k, m.keys.SeekBack):
		return m, m.seek(m.elapsed() - seekStep)

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

// setProgress records a position and when it was true, which is what elapsed
// carries forward.
func (m *Model) setProgress(pos time.Duration) {
	m.ps.Progress = pos
	m.progressAt = time.Now()
}

// hold opens the window during which local state outranks the server's.
func (m *Model) hold() {
	m.optimisticUntil = time.Now().Add(optimisticWindow)
}

func (m *Model) seek(pos time.Duration) tea.Cmd {
	pos = min(max(pos, 0), m.ps.Duration)
	m.setProgress(pos)
	m.hold()

	p := m.player
	return controlCmd("seek", func(ctx context.Context) error {
		return p.Seek(ctx, pos)
	})
}

// setVolume moves the reading and asks the device to follow.
//
// The first press of a run goes out at once: waiting out a debounce before
// anything is audible makes the key feel broken, which is exactly how it felt.
// Presses that follow within the quiet period are collapsed into a single
// request once they stop, so holding the key still costs two calls rather than
// one per repeat.
func (m *Model) setVolume(pct int) tea.Cmd {
	m.ps.Volume = min(max(pct, 0), 100)
	m.hold()
	m.volumeSeq++

	cmds := []tea.Cmd{volumeSettleCmd(m.volumeSeq)}
	if time.Since(m.volumeSentAt) >= volumeDebounce {
		cmds = append(cmds, m.sendVolume())
	}
	return tea.Batch(cmds...)
}

// sendVolume dispatches the current reading and records what went out.
func (m *Model) sendVolume() tea.Cmd {
	pct, p := m.ps.Volume, m.player
	m.volumeSent, m.volumeSentAt = pct, time.Now()

	return controlCmd("set volume", func(ctx context.Context) error {
		return p.SetVolume(ctx, pct)
	})
}

// skip changes track and starts chasing the confirmation.
func (m *Model) skip(action string, call func(context.Context) error) tea.Cmd {
	m.setProgress(0)
	m.hold()
	return tea.Batch(controlCmd(action, call), m.awaitTrackChange())
}

// takeFromQueue pops the next track off the local queue, if there is one.
func (m *Model) takeFromQueue() *player.Track {
	if len(m.queue) == 0 {
		return nil
	}
	next := m.queue[0]
	m.queue = m.queue[1:]
	return &next
}

// showTrack puts a track on screen before the server has confirmed it. Only the
// metadata is touched: the playback flags stay whatever they were.
func (m *Model) showTrack(t *player.Track) {
	if m.ps == nil {
		return
	}
	m.ps.TrackID = t.ID
	m.ps.Title = t.Title
	m.ps.Artists = t.Artists
	m.ps.Album = t.Album
	m.ps.CoverURL = t.CoverURL
	m.ps.Duration = t.Duration
	m.setProgress(0)
}

// awaitTrackChange notes which track we are leaving and asks for the first
// confirmation. Whatever is playing now is what we expect to stop seeing.
func (m *Model) awaitTrackChange() tea.Cmd {
	m.awaitingTrack = ""
	if m.ps != nil {
		m.awaitingTrack = m.ps.TrackID
	}
	m.confirmUntil = time.Now().Add(confirmWindow)
	return refetchCmd(confirmFirst)
}

// stillWaitingForTrackChange reports whether the snapshot that just arrived is
// still the old track, and there is time left to ask again.
func (m *Model) stillWaitingForTrackChange(st *player.State) bool {
	if m.awaitingTrack == "" {
		return false
	}
	if st == nil || st.TrackID != m.awaitingTrack || time.Now().After(m.confirmUntil) {
		m.awaitingTrack = ""
		return false
	}
	return true
}

// shouldResync decides whether this tick carries a real state fetch. Polling
// through a throttle is how a short one becomes a long one.
func (m Model) shouldResync() bool {
	if m.throttled() {
		return false
	}
	return !time.Now().Before(m.nextPollAt)
}

// trackRanOut handles a track finishing on its own. It is a skip nobody pressed:
// the local clock knows when it happened, and the queue knows what follows, so
// the next track goes up at once and the confirmation follows. Waiting for the
// resting poll would leave the finished track on screen for seconds.
func (m *Model) trackRanOut() tea.Cmd {
	if m.endPolledFor == m.ps.TrackID {
		return nil
	}
	m.endPolledFor = m.ps.TrackID

	cmd := m.awaitTrackChange()

	// Under repeat-one the same track starts again, so the queue says nothing
	// useful about what is coming.
	if m.ps.Repeat == player.RepeatTrack {
		return cmd
	}
	if next := m.takeFromQueue(); next != nil {
		m.showTrack(next)
		return tea.Batch(cmd, m.syncCover())
	}
	return cmd
}

// notePolled records that a poll has just gone out and works out when the next
// resting one is due.
func (m *Model) notePolled() {
	interval := idlePoll
	if time.Now().Before(m.followUntil) {
		interval = activePoll
	}
	m.nextPollAt = time.Now().Add(interval)
}

// drivenFromElsewhere reports that the snapshot differs from what is on screen
// in a way nothing here asked for. Someone is using another device, so follow
// them closely for a while rather than making them wait out the resting cadence.
func (m Model) drivenFromElsewhere(st *player.State) bool {
	if st == nil || m.ps == nil {
		return false
	}
	if m.awaitingTrack != "" || time.Now().Before(m.optimisticUntil) {
		return false
	}
	return st.TrackID != m.ps.TrackID || st.Playing != m.ps.Playing
}

// throttled reports whether Spotify has asked to be left alone.
func (m Model) throttled() bool {
	return time.Now().Before(m.rateLimitedUntil)
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

// spinDevice starts the device mark turning if it is not already, and returns
// nil otherwise. Two tick loops for one spinner would make it turn twice as
// fast, so the flag is what keeps them from doubling up.
func (m *Model) spinDevice() tea.Cmd {
	if m.deviceRun || m.ps == nil || !m.ps.Playing {
		return nil
	}
	m.deviceRun = true
	return m.device.Tick
}
