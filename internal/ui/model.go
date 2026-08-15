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

	// lastKey is the press the debug bar reports, kept for no other reason: what
	// a terminal says about a key is not knowable from here without looking.
	lastKey tea.KeyPressMsg

	// queueAt is when the queue was last asked for. It comes from our own daemon
	// rather than from Spotify, so it is asked again often — see refresh.go.
	queueAt time.Time

	// tiles are the library's wall: one picture per thing on it, keyed by the
	// thing rather than by where it sits, so scrolling a row does not re-fetch
	// what only moved. What has scrolled away is dropped — see syncGridCovers.
	tiles map[string]coverState

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

	// run is the record's beats, counted forwards. See beatrun.go.
	run beatRun

	// helpAt is how far the help screen's keys have been scrolled under its
	// head. See view_help.go.
	helpAt int

	// splash is the program's own picture, up while the device is awaited.
	// See splash.go.
	splash splashState

	// sign is the one who walks past with a placard when a switch moves.
	// See sign.go.
	sign signState

	// volume is the stack of lamps the big screen says the level with, and
	// whatever has fallen off it. See volume.go.
	volume volumeState

	// mutedAt is when the room went quiet, which is what the row of marks that
	// says so is stamped with — so the company is dealt once rather than afresh
	// on every frame. See wordsComing.
	mutedAt time.Time

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

	// deviceLostAt is when the local daemon was last found missing, so that
	// starting a replacement is tried at a sensible interval rather than on
	// every tick. See revive.go.
	deviceLostAt time.Time

	width, height int
	isDark        bool

	// ground is the terminal's own background colour, as it reported it. Nil
	// until it has answered, and only the raised block asks for it.
	ground color.Color

	styles style.Styles

	// coverStyles is the same set in the colour of the cover on screen rather
	// than of the record sounding. Only the panel that describes that cover
	// wears it. See restyle and tone.go.
	coverStyles style.Styles

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

	// debug is the bar of numbers on ctrl+shift+d. See debug.go.
	debug debugState

	// joins is where the record changes, which is when the picture changes with
	// it. See joins.go.
	joins joinsState

	// face is what goes in the marks' place when the record is given a face
	// rather than three notes. See face.go.
	face faceState

	// words is the line being sung, in dots. See words.go.
	words wordsState

	// tide is the colour of the record coming next, arriving before the sound
	// of it. See tide.go.
	tide tideState

	// tone is the colour of the record that is sounding, which is what the
	// whole program is drawn in. See tone.go.
	tone toneState

	// lyrics is the words of the track playing, and whether they are on screen.
	lyrics lyricsState

	// peek is the glance at what is coming, above the artwork.
	peek peekState

	// rowsAreFlush drops the cursor column from list rows, for the lists that
	// have no cursor. Set while drawing one and cleared straight after.
	rowsAreFlush bool

	// rowsMainAt fixes the width of a row's first column, for the one list that
	// has to line its second column up with something else on the screen rather
	// than with the row's own arithmetic. Zero leaves the row to divide itself.
	rowsMainAt int

	// rowsAreTheQueue says every row of this list is already in the queue, so
	// the column that marks queued tracks has nothing to say and is dropped.
	rowsAreTheQueue bool
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
		// And the picture keeps time with the record from the start: a screen
		// that only answers the loudness is what this used to be, and it is on
		// the key for anybody who wants to see the difference.
		stage:   stageState{loose: true},
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
	if rgb, ok := m.toneAccent(); ok {
		accent = cover.Readable(rgb, m.isDark)
	}
	m.styles = style.New(m.isDark, accent).On(m.ground)

	// And a second set in the colour of the cover on screen, for the panel that
	// describes it. Everything else is the sounding record's — see tone.go — but
	// the band at the top is explicitly about another record when the cursor has
	// moved off, and the frame round it says so. Saying it in that record's own
	// colour as well costs one palette a cover rather than a comparison a frame.
	m.coverStyles = m.styles
	if m.cover.hasAccent {
		m.coverStyles = style.New(m.isDark, cover.Readable(m.cover.accent, m.isDark)).On(m.ground)
	}

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

// loaded reports that there is a track on the device to say anything about.
//
// A device can be active with nothing on it — one just switched to, or one that
// has been left alone — and the difference matters to everything that draws a
// position: a bar, a clock and a length are all statements about a track, and
// there is no track to make them about.
func (m Model) loaded() bool {
	return m.ps != nil && (m.ps.TrackID != "" || m.ps.Duration > 0)
}

// elapsed is where the current track has reached: the last position the backend
// reported, carried forward by the clock.
func (m Model) elapsed() time.Duration {
	if m.ps == nil {
		return 0
	}

	// Nothing loaded, nowhere to be. A device can be active with no track on
	// it — one just switched to, say — and until this asked, whatever number
	// arrived in the position was drawn as a time. What arrived after one such
	// switch was a timestamp, and the screen said the track was fifty-six years
	// in.
	if !m.loaded() {
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
	case m.finding() && !m.actions.open:
		// A search open over a list answers the keyboard, so the bar is its
		// keys — including the one that takes it off again.
		return m.keys.forFinding(m.find.typing)
	case m.open() != nil:
		// What is open is the screen, and its keys are the same wherever it was
		// opened from — which is the whole reason it is one screen.
		return m.keys.forOpen(m.open().holdsAlbums(), scope)
	case m.tab == tabQueue && !m.editable():
		// Against a device that is not ours the queue can only be read. Listing
		// keys that do nothing would be worse than a shorter bar.
		return m.keys.forReadOnlyQueue()
	case m.tab == tabPlayer:
		return m.keys.forPlayer(scope, lyrics, peek, m.width)
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
	switch {
	case m.tab == tabPlayer:
		return modePlayer
	case m.tab == tabLibrary && m.open() == nil:
		// The library's band is two previews side by side rather than one
		// picture with facts beside it. Whatever is opened from it is the other
		// thing again — a record, with its tracks under it — and takes the
		// picture the queue takes.
		return modeLibrary
	default:
		return modeList
	}
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
