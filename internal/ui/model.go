package ui

import (
	"image/color"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/style"
)

const (
	// optimisticWindow is how long a local change outranks anything a poll
	// still in flight may report. See DESIGN.md 4.2.
	optimisticWindow = 2 * time.Second

	// idlePoll is the resting cadence. DESIGN.md 4.1 picked five seconds without
	// measuring; against a live account both five and two seconds ran with zero
	// 429s, so three keeps a comfortable margin over the fastest rate proved
	// safe while cutting the wait before an outside change is noticed.
	idlePoll = 3 * time.Second

	// activePoll is used for a while after a change nobody here asked for.
	// Somebody is driving from another device, and the next thing they do
	// should not take five seconds to arrive.
	activePoll = 1500 * time.Millisecond

	// followWindow is how long that lasts after the last such change.
	followWindow = 30 * time.Second

	// A track running out is the most common change nobody pressed a key for,
	// and it is the one whose timing the local clock already knows. It is
	// handled as an unpressed skip rather than by polling harder.

	seekStep   = 5 * time.Second
	volumeStep = 5
)

// Model is the whole application state. Nothing outside Update writes to it.
type Model struct {
	player player.Player
	covers *cover.Loader
	cell   cover.CellSize

	ps *player.State // last known server state

	// progressAt is when ps.Progress was true. The position on screen is the
	// two of them added together rather than a counter of its own: a counter
	// and a poll are two answers to the same question, and they disagree by
	// whatever the tick has drifted — which is visible as the playhead jumping
	// backwards every time the poll wins.
	progressAt      time.Time
	optimisticUntil time.Time // a poll must not overwrite before this

	tab       tabID
	queuePane queuePane
	library   libraryPane
	search    searchPane

	// settings is the screen for what spindle keeps between runs.
	settings settingsPane

	// find is a search through the list on screen, which is a different act
	// from the search tab: nothing is asked of Spotify, and what it walks is
	// already in front of you.
	find find

	// stack is what has been opened on top of a tab's own list: a playlist, an
	// album, an artist, and whatever was opened from those. It belongs to the
	// model rather than to a pane because the same page can be reached from the
	// library and from a search, and it must read the same either way.
	stack []openPage

	// noDevice is the normal entry path, not an error: nothing is playing
	// anywhere. Only the player tab cares — browsing works regardless.
	noDevice bool
	devices  devicePane

	cover coverState

	// nowCover is the second picture: what is playing, beside what the cursor
	// is on. Only the screens that show both fill it. It carries no accent —
	// the palette follows the cover the reader is looking at, and two of them
	// pulling at it would leave the colours changing for no reason.
	nowCover coverState

	// coverSeq debounces the artwork preview: arrowing down a list should not
	// fire an upload per row, only once the cursor settles.
	coverSeq int

	// volumeSeq debounces the volume keys, volumeSent is the last value actually
	// sent and volumeSentAt is when, so a run of presses can be collapsed
	// without delaying the first one.
	volumeSeq    int
	volumeSent   int
	volumeSentAt time.Time

	// devicesAt is when the device list was last asked for, so a screen showing
	// one keeps it current without asking on every tick.
	devicesAt time.Time

	// tookOwnDeviceAt is when the device spindle started for itself was last
	// claimed, so a claim that failed is tried again and one that worked is not
	// repeated every time the list arrives.
	tookOwnDeviceAt time.Time

	// mutedFrom is the volume to come back to, or zero when nothing is muted.
	// Muting is not the same as turning the volume down: the level that was
	// chosen is worth keeping, and the music keeps playing either way.
	mutedFrom int

	// rateLimitedUntil suspends polling. Spotify asked to be left alone, and
	// carrying on regardless is how a short throttle becomes a long one.
	rateLimitedUntil time.Time

	// noPremium is a standing explanation, not a transient error: a free account
	// can read playback but never change it.
	noPremium bool

	// play is the request to start something that is in flight, and the newest
	// one waiting behind it.
	playInFlight bool
	playPending  *playRequest
	playSentAt   time.Time

	// actions is the menu of verbs for whatever the cursor is on.
	actions actionsPane

	// said is a line about something that just happened and needs no answer,
	// and when it was said. It fades on its own, like the skipped notice.
	said   string
	saidAt time.Time

	// skipped is the track the device could not play, and when it said so. It
	// is shown for a moment and then let go: it is news, not a state.
	skipped   string
	skippedAt time.Time

	// awaitingTrack is the track a skip is trying to move away from, and
	// confirmUntil is when to stop asking whether it has. expecting is the one
	// it is trying to move to, where the caller knew which that would be.
	awaitingTrack string
	expecting     string
	confirmUntil  time.Time

	// nextPollAt is when the resting poll is next due, followUntil is how long
	// to keep the faster cadence, and endPolledFor stops a track that has run
	// out from being handled again on every following tick.
	nextPollAt   time.Time
	followUntil  time.Time
	endPolledFor string

	// queue is what Spotify says comes next. It exists so that pressing n can
	// put the right title on screen at once rather than half a second later.
	// queueFor is the track it was fetched for.
	queue    []player.Track
	queueFor string

	// order is the run of move keys waiting to be sent.
	order pendingOrder

	// nowQueued is the playing track as the queue reported it. The daemon says
	// what is playing sooner and more often, but only this carries the release
	// date and the rest of the detail panel's material.
	nowQueued *player.Track

	// stopDaemon records that the user asked to take the music with them. It is
	// read after the program ends rather than acted on inside it: stopping the
	// device while its own UI is still drawing would be a race for no reason.
	stopDaemon bool

	err       error
	tickCount int

	width, height int
	isDark        bool

	styles  style.Styles
	keys    keyMap
	help    help.Model
	spinner spinner.Model

	// device is the mark beside the device name. It turns while the music plays,
	// which is the one part of the screen that says "sound is coming out of this
	// machine right now" without needing to be read.
	device    spinner.Model
	deviceRun bool

	// scope is the waveform under the artwork, and whether it is being drawn.
	scope scopeState

	// stage is the visualiser with the whole terminal to itself. See stage.go.
	stage stageState

	// grain is the cover taken apart into dots, for the picture that moves it.
	// See grain.go.
	grain grainState

	// words is the line being sung, in dots. See words.go.
	words wordsState

	// chase is what walks across the screen through a solo. See chase.go.
	chase chaseState

	// lyrics is the words of the track playing, and whether they are on screen.
	lyrics lyricsState

	// peek is the glance at what is coming, above the artwork.
	peek peekState

	// rowsAreFlush drops the cursor column from list rows, for the lists that
	// have no cursor. Set while drawing one and cleared straight after.
	rowsAreFlush bool
}

// New wires a model around a playback backend and an artwork loader. The palette
// starts dark and is corrected once the terminal reports its background colour.
func New(p player.Player, covers *cover.Loader, cell cover.CellSize) Model {
	m := Model{
		player: p,
		covers: covers,
		cell:   cell,
		isDark: true,
		keys:   newKeyMap(),
		help:   help.New(),
		search: newSearchPane(),
		// The waveform is on to begin with: it is the thing that makes the
		// screen feel alive, and a feature nobody knows to ask for may as well
		// not exist. The key is there to put it away.
		scope: scopeState{modes: [tabCount]scopeMode{
			tabPlayer: scopeWave,
			tabQueue:  scopeWave,
		}},
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		device:  spinner.New(spinner.WithSpinner(deviceSpinner)),
	}
	m.help.ShortSeparator = " · "
	m.restyle()
	return m
}

// coverTarget is the artwork the current tab wants on the left. Each tab answers
// differently: the player shows what is sounding, the browsers show what the
// cursor is resting on.
func (m Model) coverTarget() string {
	// Whatever is open is the screen, so its cursor is what the picture follows.
	if page := m.open(); page != nil {
		return page.cover()
	}

	switch m.tab {
	case tabQueue:
		if t := m.queuedTrack(); t != nil {
			return t.CoverURL
		}
		return ""
	case tabLibrary:
		return m.library.cover()
	case tabSearch:
		return m.search.cover()
	default:
		if m.ps == nil || m.noDevice {
			return ""
		}
		return m.ps.CoverURL
	}
}

// restyle rebuilds every style from the current background and album accent.
func (m *Model) restyle() {
	var accent color.Color
	if m.cover.hasAccent {
		accent = cover.Readable(m.cover.accent, m.isDark)
	}
	m.styles = style.New(m.isDark, accent)
	m.help.Styles = help.DefaultStyles(m.isDark)
	m.spinner.Style = m.styles.Detail

	// The search field is a bubble with opinions of its own; overrule them so it
	// belongs to the same screen as everything else.
	in := textinput.DefaultStyles(m.isDark)
	for _, s := range []*textinput.StyleState{&in.Focused, &in.Blurred} {
		s.Text = m.styles.Query
		s.Placeholder = m.styles.Placeholder
		s.Prompt = m.styles.QueryPrompt
	}
	in.Cursor.Color = m.styles.Accent
	m.search.input.SetStyles(in)
}

// StopDaemonRequested reports whether the session ended with Q rather than q.
func (m Model) StopDaemonRequested() bool { return m.stopDaemon }

// elapsed is where the current track has reached: the last position the backend
// reported, carried forward by the clock.
func (m Model) elapsed() time.Duration {
	if m.ps == nil {
		return 0
	}

	pos := m.ps.Progress
	if m.ps.Playing && !m.progressAt.IsZero() {
		pos += time.Since(m.progressAt)
	}
	if m.ps.Duration > 0 && pos > m.ps.Duration {
		pos = m.ps.Duration
	}
	return max(pos, 0)
}

// helpKeys is the key set the help bar should advertise right now.
// helpKeysWith is helpKeys with the waveform key's availability passed in. The
// layout is what decides that, and the layout needs the help's height, so the
// two cannot ask each other — helpHeight passes false and relies on the help
// coming out the same height either way.
func (m Model) helpKeysWith(scope, lyrics, peek bool) tabKeys {
	switch {
	case m.tab == tabPlayer && m.noDevice:
		return m.keys.forNoDevice()
	case m.devices.open:
		return m.keys.forDevices()
	case m.open() != nil:
		// What is open is the screen, and its keys are the same wherever it was
		// opened from — which is the whole reason it is one screen.
		return m.keys.forOpen(m.open().holdsAlbums(), scope)
	case m.tab == tabQueue && !m.editable():
		// Against a device that is not ours the queue can only be read. Listing
		// keys that do nothing would be worse than a shorter bar.
		return m.keys.forReadOnlyQueue()
	case m.tab == tabPlayer:
		return m.keys.forPlayer(scope, lyrics, peek)
	default:
		return m.keys.forTab(m.tab, scope)
	}
}

// helpKeys is the help for what is on screen.
func (m Model) helpKeys() tabKeys {
	return m.helpKeysWith(m.scopeAvailable(), m.lyricsAvailable(), m.peekAvailable())
}

// layoutMode is how the current tab divides its body.
func (m Model) layoutMode() layoutMode {
	if m.tab == tabPlayer {
		return modePlayer
	}
	return modeList
}

// layout resolves the current geometry. It is pure, so View and Update can both
// ask for it without either of them owning the answer.
func (m Model) layout() layout {
	return computeLayout(m.width, m.height, m.helpHeight(), m.hasNotice(), m.layoutMode(), m.cell)
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.RequestBackgroundColor,
		tickCmd(),
		fetchStateCmd(m.player),
		loadPrefsCmd(),
		loadSettingsCmd(),
	}
	// A backend that reports its own changes is followed rather than polled.
	if w, ok := m.player.(player.Watcher); ok {
		cmds = append(cmds, watchCmd(w))
	}
	return tea.Batch(cmds...)
}
