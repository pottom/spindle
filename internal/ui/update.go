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

	case deviceRestarted:
		m.settings.restarting = false
		m.settings.changed = false
		m.said, m.saidAt = "The device is back", time.Now()
		return m, tea.Batch(fetchStateCmd(m.player), fetchDevicesCmd(m.player))

	case settingsMsg:
		m.settings.quality = message.quality
		m.settings.crossfade = message.crossfade
		m.settings.notify = message.notify
		m.settings.loaded = true
		return m, nil

	case prefsMsg:
		m.applyPrefs(prefs(message))
		return m, tea.Batch(m.startScope(), m.fetchLyrics())

	case msg.Tick:
		return m.handleTick()

	case msg.StateFetched:
		if m.drivenFromElsewhere(message.State) {
			m.followUntil = time.Now().Add(followWindow)
			m.nextPollAt = time.Now().Add(activePoll)
		}
		m.adopt(message.State)
		m.noteUnplayable(message.State)
		m.fillFromQueue()
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
		} else if m.ps != nil && m.ps.Title == "" && m.ps.Playing {
			// Nothing to go on and music coming out: the queue is the only
			// other thing that knows what is on.
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
		return m, m.takeOwnDevice()

	case msg.QueueFetched:
		// A move waiting to go out is not in the device's answer yet, and
		// drawing that answer would undo it under the user's hand.
		if !m.order.live {
			m.queue = message.Tracks
			m.clampQueueCursor()
		}
		m.nowQueued = nil
		if len(message.Current) > 0 {
			m.nowQueued = &message.Current[0]
		}
		m.fillFromQueue()
		if m.tab == tabQueue {
			return m, m.syncCover()
		}
		return m, nil

	case msg.CoverReady:
		if message.Slot == tideSlot {
			m.tideTook(message.Art)
			return m, nil
		}
		if message.Slot == nowSlot {
			if m.nowCover.matches(message.URL, message.Width, message.Height) {
				m.nowCover.art = message.Art.Cells
			}
			return m, nil
		}
		if m.cover.matches(message.URL, message.Width, message.Height) {
			m.cover.art = message.Art.Cells
			m.cover.accent, m.cover.hasAccent = message.Art.Accent, message.Art.HasAccent
			m.restyle()
		}
		return m, nil

	case msg.CoverFailed:
		if message.Slot == tideSlot {
			// No colour to hand over: the record changes the way it always did.
			return m, nil
		}
		if message.Slot == nowSlot {
			if m.nowCover.matches(message.URL, message.Width, message.Height) {
				m.nowCover.failed = true
			}
			return m, nil
		}
		if m.cover.matches(message.URL, message.Width, message.Height) {
			m.cover.failed = true
		}
		return m, nil

	case msg.CoverSettled:
		if message.Seq == m.coverSeq {
			return m, m.syncCover()
		}
		return m, nil

	case msg.LibraryFetched:
		m.library.adopt(message, m.likedRow())
		return m, m.syncCover()

	case msg.OpenedFetched:
		// The saved tracks are read whether or not they are open: the library
		// asks for the first page as it loads, for the cover and the count.
		if isLiked(message.ID) && message.Offset == 0 {
			m.library.liked, m.library.likedAll = message.Tracks, !message.More
			m.refreshLikedRow()
		}

		// Anything else that arrives for a page nobody is on is an answer the
		// reader has already walked away from.
		if page := m.openMut(); page != nil && page.id == message.ID {
			page.adopt(message)
			return m, m.syncCover()
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

	case msg.OrderSettled:
		// Only the last press of the run counts.
		if message.Seq != m.order.seq {
			return m, nil
		}
		return m, m.sendOrder()

	case msg.VolumeSettled:
		// Only the newest keystroke counts, and only if the value has moved
		// since the leading request went out.
		if message.Seq != m.volumeSeq || m.ps == nil || m.ps.Volume == m.volumeSent {
			return m, nil
		}
		return m, m.sendVolume()

	case msg.PlayDone:
		// The answer releases the next request, and is then handled as any
		// other control's would be: it may be an error worth showing.
		m.playInFlight = false
		var next tea.Cmd
		if pending := m.playPending; pending != nil {
			m.playPending = nil
			m.playInFlight = true
			next = m.sendPlay(*pending)
		}
		if message.Result == nil {
			return m, next
		}
		model, cmd := m.Update(message.Result)
		return model, tea.Batch(next, cmd)

	case msg.ControlDone:
		// Whatever was standing in the way is evidently no longer standing.
		m.noPremium, m.err = false, nil
		return m, nil

	case msg.RateLimited:
		m.rateLimitedUntil = time.Now().Add(message.RetryAfter)
		m.err = nil
		return m, nil

	case msg.Error:
		// A restart that failed is a restart that is over.
		m.settings.restarting = false

		// Whatever was in flight is not in flight any more. Every list marks
		// itself as reading so a run of cursor keys cannot ask for the same
		// page a dozen times; without clearing that here, one failed request —
		// a moment offline, a refused call — would leave the mark set for good
		// and the list would never ask for anything again. It looks like a list
		// that simply ends, which is the worst kind of failure: a silent one.
		m.stopLoading()

		if errors.Is(message.Err, player.ErrPremiumRequired) {
			m.noPremium = true
			return m, nil
		}
		m.err = message.Err
		return m, nil

	case msg.LyricsFetched:
		m.adoptLyrics(message)
		return m, nil

	case msg.WordsReady:
		m.words.asked = ""
		m.wordsAdopt(message.Grain, message.Words, message.Text)
		m.words.cellsX, m.words.cellsY = message.CellsX, message.CellsY
		return m, nil

	case msg.WaveformReady:
		// The trace stops the moment it leaves the screen: a redraw every 33ms
		// for something nobody is looking at is the whole cost of the feature.
		if !m.scopeVisible() {
			m.scope.running = false
			return m, nil
		}
		// Paused, what arrives is the last thing that was heard, over and over:
		// the analyser is fed by the output, and a stopped output feeds it
		// nothing. Drawn as it comes, the picture freezes mid-beat and stays
		// there, which reads as a crash rather than as a pause. So the music
		// stopping is drawn as the music stopping — everything sinks to where
		// silence would have left it.
		if m.ps != nil && !m.ps.Playing {
			m.scope.settle()
		} else {
			m.scope.frame = message.Samples
			m.scope.follow(message.Samples)
			m.scope.adoptBands(message.Bands)
			if message.Beat.Found() {
				m.scope.beat, m.scope.beatAt = message.Beat, time.Now()
			}
		}
		if m.scopeMode().wave() {
			m.rememberScope()
			if m.stage.on {
				m.throwSparks(m.width, m.height)
			} else {
				m.throwSparks(m.scopeWidth(m.layout()), scopeRows)
			}
		}
		// The colour of the record coming next, arriving before the sound of it.
		// See tide.go.
		m.tideFlow()
		tideAsk := m.tideCmd()

		if m.stage.on && m.scopeMode().words() {
			m.joinsFlow(int(time.Second / scopeInterval))
			m.wordsFlow(m.width, m.height)
			m.faceFlow()
			m.wordsEase(m.width, m.height)
			m.figureSpray(m.width, m.height)
			m.figureSweep(m.width, m.height)
			// Thrown from the tips of the band along the foot, and given the
			// whole terminal to cross rather than the band it came from.
			_, tall := m.wordsRoom(m.height)
			m.stageFlowIn(m.width, m.height, stageThrows{
				span:  float32(m.height * dotsPerCellY),
				reach: stageReach * float32(tall*dotsPerCellY),
				lift:  wordsLift,
			})
			if cmd := m.wordsGrind(); cmd != nil {
				return m, tea.Batch(cmd, tideAsk, scopeFrameCmd(m.player, m.frameMode()))
			}
		}
		// The water is only stirred where it is being drawn, and the big screen
		// falls back to it when the strip is switched off.
		if mirror := m.scopeMode().mirror() || (m.stage.on && m.scopeMode() == scopeOff); mirror {
			if m.stage.on {
				m.stageFlow(m.width, m.height)
			} else {
				m.stageFlow(m.scopeWidth(m.layout()), scopeRows)
			}
		}
		if tideAsk != nil {
			return m, tea.Batch(tideAsk, scopeFrameCmd(m.player, m.frameMode()))
		}
		return m, scopeFrameCmd(m.player, m.frameMode())

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

		// The other spinner covers a wait: an artwork download, or a page of a
		// list that has not arrived. Letting it run otherwise would mean a
		// redraw every 100 ms for nothing.
		if !m.cover.loading() && !m.listLoading() {
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
	m.help.SetWidth(m.layout().interior - leftMargin - rightMargin)
}

// handleTick advances the local clock and resynchronises every fifth second.
func (m Model) handleTick() (tea.Model, tea.Cmd) {
	m.tickCount++
	cmds := []tea.Cmd{tickCmd()}
	if cmd := m.spinDevice(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	// The trace stops whenever it leaves the screen — another tab, no device,
	// no state yet — and nothing else would ever start it again. Picking it up
	// here means it resumes on its own rather than waiting to be switched off
	// and on.
	if cmd := m.startScope(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	// The words belong to whatever is playing, so a track change is what sends
	// for them. The tick is where that is noticed.
	if cmd := m.fetchLyrics(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	// A banner appearing or clearing changes the height of the body, and with
	// it the artwork area. Nothing else notices, and a cover drawn into an area
	// that has since changed size is shown with a corner cut off. This costs
	// nothing when the size is unchanged.
	if cmd := m.syncCover(); cmd != nil {
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
	if cmd := m.refreshDevices(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// refreshDevices keeps the device list current while a screen of it is up.
//
// It used to be fetched only when a state poll came back with nothing playing,
// which left the screen showing whatever the list was when it opened: a daemon
// started a moment later did not appear on it, and neither did anything else,
// and the key that asks for a refresh was the only way back — when it worked at
// all. A list of what is out there has to keep looking.
func (m *Model) refreshDevices() tea.Cmd {
	if !m.noDevice && !m.devices.open {
		return nil
	}
	if time.Since(m.devicesAt) < devicesEvery {
		return nil
	}

	m.devicesAt = time.Now()
	return fetchDevicesCmd(m.player)
}

// devicesEvery is how often that happens. Spotify lists a device within a few
// seconds of it appearing, so this is about as often as it can say anything new.
const devicesEvery = 3 * time.Second

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

	// Where the track asked for is known, anything else arriving inside the
	// window is a snapshot from before the request — including the answer to a
	// request that has already been overtaken, which is what pressing play on
	// one track after another produces. Adopting those marks a row that is not
	// the one about to sound.
	if m.expecting != "" && time.Now().Before(m.confirmUntil) {
		if st.TrackID != m.expecting {
			return
		}
	}
	m.expecting = ""
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
	m.ps.Tempo = st.Tempo
}

// noteUnplayable picks up a track the device gave up on. Spotify refuses the
// audio key for some tracks inside the list they belong to, and the device
// moves past them — which on screen is a list skipping a track by itself, and
// reads as a fault here rather than there.
func (m *Model) noteUnplayable(st *player.State) {
	if st == nil || st.Unplayable == "" || st.Unplayable == m.skipped {
		return
	}
	m.skipped, m.skippedAt = st.Unplayable, time.Now()
}

// unplayableNotice is what to say about it, while it is still news.
func (m Model) unplayableNotice() (string, bool) {
	if m.skipped == "" || time.Since(m.skippedAt) > unplayableWindow {
		return "", false
	}

	name := m.skipped
	for _, t := range m.queue {
		if t.ID == m.skipped && t.Title != "" {
			name = t.Title
			break
		}
	}
	return name, true
}

// fillFromQueue borrows what the device left out of its answer. The names come
// off the audio stream, and there is not always one — a session picked up from
// elsewhere is playing before it has loaded anything of its own — but the queue
// still knows what is on. A screen gone blank over music that is plainly
// playing is worse than one that is a moment behind.
func (m *Model) fillFromQueue() {
	if m.ps == nil || m.ps.Title != "" || m.nowQueued == nil {
		return
	}
	// Mid-change the queue is stale by definition: it still names the track
	// being left, and borrowing that would put the mark on the wrong row until
	// the device caught up.
	if m.awaitingTrack != "" {
		return
	}
	if m.ps.TrackID != "" && m.ps.TrackID != m.nowQueued.ID {
		return
	}

	t := m.nowQueued
	m.ps.TrackID = t.ID
	m.ps.Title = t.Title
	m.ps.Artists = t.Artists
	m.ps.Album = t.Album
	m.ps.CoverURL = t.CoverURL
	if m.ps.Duration == 0 {
		m.ps.Duration = t.Duration
	}
}

func (m Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// The big screen outranks even the tabs: it is watched rather than worked
	// on, so the next key is the way out of it whatever that key was.
	if cmd, handled := m.stageKey(k); handled {
		return m, cmd
	}

	// Tab switching outranks everything, including the search field: it is the
	// one key that has to work wherever you are.
	switch {
	case key.Matches(k, m.keys.NextTab):
		return m, m.switchTab(m.tab.next(1))
	case key.Matches(k, m.keys.PrevTab):
		return m, m.switchTab(m.tab.next(-1))
	case key.Matches(k, m.keys.GoTab) && !m.search.typing && !m.find.typing:
		// Not on the search tab: there the digits belong to the query, and a
		// key that types on one screen must not navigate on it.
		if t, ok := tabAt(k.String()); ok {
			return m, m.switchTab(t)
		}
	}

	// A query being written answers everything, the way the menu does: every
	// printable key belongs to it while it is open.
	if cmd, handled := m.findKey(k); handled {
		return m, cmd
	}

	// The menu answers everything while it is up: nothing underneath may act on
	// a screen that is not the one being looked at.
	if cmd, handled := m.actionsKey(k); handled {
		return m, cmd
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

	case key.Matches(k, m.keys.Scope):
		if !m.scopeAvailable() {
			return m, nil
		}
		// Nothing about the geometry changes, so the cover is left alone: the
		// trace only fills rows that were already blank.
		m.scope.modes[m.tab] = m.scopeMode().next()
		for m.scopeMode().big() {
			m.scope.modes[m.tab] = m.scopeMode().next()
		}
		return m, tea.Batch(m.startScope(), m.savePrefs())

	case key.Matches(k, m.keys.Stage):
		if m.tab != tabPlayer || m.ps == nil {
			return m, nil
		}
		m.stage.on = true
		return m, m.startScope()

	case key.Matches(k, m.keys.Lyrics):
		if !m.lyricsAvailable() {
			return m, nil
		}
		m.lyrics.on = !m.lyrics.on
		return m, tea.Batch(m.fetchLyrics(), m.savePrefs())

	case key.Matches(k, m.keys.Peek):
		if !m.peekAvailable() {
			return m, nil
		}
		m.peek.on = !m.peek.on
		return m, m.savePrefs()

	case key.Matches(k, m.keys.SearchType):
		// / looks for something where you are. On a list that means the list —
		// no request, no waiting, the rows are already here. Where there is no
		// list to look through it means the other thing: go and ask Spotify.
		if m.findable() {
			m.startFind()
			return m, nil
		}
		return m, m.startTyping()

	case key.Matches(k, m.keys.FindNext):
		m.findStep(1)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case key.Matches(k, m.keys.FindPrev):
		m.findStep(-1)
		return m, tea.Batch(m.previewCover(), m.readAhead())

	case key.Matches(k, m.keys.Help):
		// A screen rather than a taller bar: unfolding it pushed the list off
		// the bottom of the tabs with the longest lists, which are exactly the
		// ones whose keys somebody is looking up.
		return m, m.switchTab(tabHelp)
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

	case key.Matches(k, m.keys.Mute):
		return m, m.toggleMute()

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
// syncCover keeps both pictures in step with what the screen is showing. The
// second one is asked for from here rather than from its own call sites: it
// changes when the track does, and the track changes in a dozen places that
// have no reason to know a second picture exists.
func (m *Model) syncCover() tea.Cmd {
	return tea.Batch(m.syncCursorCover(), m.syncNowCover())
}

// syncCursorCover is the picture of whatever the screen is about: the track
// playing, or the row under the cursor.
func (m *Model) syncCursorCover() tea.Cmd {
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
		coverCmd(m.covers, m.cover.url, l.artWidth, l.artHeight, cursorSlot),
		m.spinner.Tick,
	)
}

// syncNowCover is the same for the second picture, the one of what is playing.
// It is only asked for on the screens that show it: everywhere else the field
// is left as it is, so coming back to such a screen does not start from a blank
// square.
func (m *Model) syncNowCover() tea.Cmd {
	if m.covers == nil || !fitsMinimum(m.width, m.height) || !m.showsNowPanel() {
		return nil
	}

	l := m.layout()
	w, h := nowCoverBox(l)
	url := ""
	if m.ps != nil && !m.noDevice {
		url = m.ps.CoverURL
	}
	if m.nowCover.matches(url, w, h) {
		return nil
	}

	m.nowCover = coverState{url: url, width: w, height: h}
	if url == "" {
		m.nowCover.failed = true
		return nil
	}
	return coverCmd(m.covers, url, w, h, nowSlot)
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

// toggleMute silences the device and remembers what to come back to. The music
// carries on: this is the volume control, not the transport.
func (m *Model) toggleMute() tea.Cmd {
	if m.mutedFrom > 0 {
		back := m.mutedFrom
		m.mutedFrom = 0
		return m.setVolume(back)
	}
	if m.ps.Volume == 0 {
		// Already silent by hand. Nothing to remember and nothing to do.
		return nil
	}

	m.mutedFrom = m.ps.Volume
	return m.setVolume(0)
}

// setVolume moves the reading and asks the device to follow.
//
// The first press of a run goes out at once: waiting out a debounce before
// anything is audible makes the key feel broken, which is exactly how it felt.
// Presses that follow within the quiet period are collapsed into a single
// request once they stop, so holding the key still costs two calls rather than
// one per repeat.
func (m *Model) setVolume(pct int) tea.Cmd {
	if pct > 0 {
		// Reaching for the volume is a decision about the level, so there is
		// nothing left to come back to.
		m.mutedFrom = 0
	}
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

	// Through the same gate as everything else that starts a track. Spotify's
	// audio key service is what actually limits this, and it does not care
	// which key was pressed: fourteen starts in six seconds took the session
	// down with it, measured. See startPlay.
	return m.startPlay(playRequest{
		action: action,
		call:   func(ctx context.Context, _ player.Player) error { return call(ctx) },
	})
}

// takeFromQueue pops the next track off the local queue, if there is one.
func (m *Model) takeFromQueue() *player.Track {
	if len(m.queue) == 0 {
		return nil
	}
	next := m.queue[0]
	m.queue = m.queue[1:]
	m.clampQueueCursor()
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
	m.expecting = t.ID
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

// stopLoading takes the reading mark off every list. It says nothing about
// whether the page arrived — only that nothing is waiting for it any more, so
// scrolling to the end will ask again.
func (m *Model) stopLoading() {
	for i := range m.library.pages {
		m.library.pages[i].loading = false
	}
	for _, found := range m.search.found {
		found.pages.loading = false
	}
	for i := range m.stack {
		m.stack[i].pages.loading = false
	}
}

// takeOwnDevice claims the playback device spindle started itself.
//
// Somebody who opens spindle should not have to pick their own machine out of a
// list of speakers: the daemon is this program's own, it was started by this
// program a second ago, and the device list is the only reason it is on screen
// at all. So the first time it appears with nothing playing anywhere, it is
// taken.
//
// Only with nothing playing anywhere, and that is the whole safety of it: music
// coming out of a phone stays there, because a transfer carries the state
// across and taking a playing session would move it here.
//
// Not only once, though. The first version claimed the device a single time and
// never again, so a daemon that was restarted underneath a running interface
// came back to a screen asking which device to use — with one device on it, the
// one spindle had just started. A device that vanishes and returns is not a
// device somebody left for another speaker.
func (m *Model) takeOwnDevice() tea.Cmd {
	owner, ok := m.player.(player.Owner)
	if !ok || !m.noDevice || time.Since(m.tookOwnDeviceAt) < takeAgainAfter {
		return nil
	}

	id := owner.OwnDevice()
	if id == "" || !m.devices.has(id) {
		// The daemon has not registered with Spotify yet. It takes a few
		// seconds, and the next answer will carry it.
		return nil
	}

	m.tookOwnDeviceAt = time.Now()
	p := m.player
	return tea.Batch(
		controlCmd("use this device", func(ctx context.Context) error {
			return p.TransferTo(ctx, id, false)
		}),
		refetchCmd(confirmFirst),
	)
}

// takeAgainAfter is how long to leave it before claiming the device again. It
// stops a claim that cannot land — a daemon still coming up, a Spotify that
// refuses — from being retried on every answer, and is short enough that a
// restarted daemon is picked up while the reader is still looking at the screen.
const takeAgainAfter = 10 * time.Second

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

// startScope begins the waveform's tick loop if it is not already running, and
// returns nil otherwise: two loops would drive the trace at twice the rate.
func (m *Model) startScope() tea.Cmd {
	if m.scope.running || !m.scopeVisible() {
		return nil
	}
	m.scope.running = true
	return scopeFrameCmd(m.player, m.frameMode())
}
