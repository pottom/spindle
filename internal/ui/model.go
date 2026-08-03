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

	// resyncEvery is measured in ticks: one real State() call per five seconds.
	resyncEvery = 5

	seekStep   = 5 * time.Second
	volumeStep = 5
)

// Model is the whole application state. Nothing outside Update writes to it.
type Model struct {
	player player.Player
	covers *cover.Loader
	cell   cover.CellSize

	ps              *player.State // last known server state
	localProgress   time.Duration // ticked locally, not ps.Progress
	optimisticUntil time.Time     // a poll must not overwrite before this

	tab       tabID
	playlists playlistPane
	search    searchPane

	// noDevice is the normal entry path, not an error: nothing is playing
	// anywhere. Only the player tab cares — browsing works regardless.
	noDevice bool
	devices  devicePane

	cover coverState

	// coverSeq debounces the artwork preview: arrowing down a list should not
	// fire an upload per row, only once the cursor settles.
	coverSeq int

	// volumeSeq debounces the volume keys the same way.
	volumeSeq int

	// rateLimitedUntil suspends polling. Spotify asked to be left alone, and
	// carrying on regardless is how a short throttle becomes a long one.
	rateLimitedUntil time.Time

	// noPremium is a standing explanation, not a transient error: a free account
	// can read playback but never change it.
	noPremium bool

	err       error
	tickCount int

	width, height int
	isDark        bool

	styles  style.Styles
	keys    keyMap
	help    help.Model
	spinner spinner.Model
}

// New wires a model around a playback backend and an artwork loader. The palette
// starts dark and is corrected once the terminal reports its background colour.
func New(p player.Player, covers *cover.Loader, cell cover.CellSize) Model {
	m := Model{
		player:  p,
		covers:  covers,
		cell:    cell,
		isDark:  true,
		keys:    newKeyMap(),
		help:    help.New(),
		search:  newSearchPane(),
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
	m.help.ShortSeparator = " · "
	m.restyle()
	return m
}

// coverTarget is the artwork the current tab wants on the left. Each tab answers
// differently: the player shows what is sounding, the browsers show what the
// cursor is resting on.
func (m Model) coverTarget() string {
	switch m.tab {
	case tabPlaylists:
		return m.playlists.cover()
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

// helpKeys is the key set the help bar should advertise right now.
func (m Model) helpKeys() tabKeys {
	switch {
	case m.tab == tabPlayer && m.noDevice:
		return m.keys.forNoDevice()
	case m.devices.open:
		return m.keys.forDevices()
	default:
		return m.keys.forTab(m.tab)
	}
}

// layout resolves the current geometry. It is pure, so View and Update can both
// ask for it without either of them owning the answer.
func (m Model) layout() layout {
	return computeLayout(m.width, m.height, m.helpHeight(), m.hasNotice(), m.tab != tabPlayer, m.cell)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		tickCmd(),
		fetchStateCmd(m.player),
	)
}
