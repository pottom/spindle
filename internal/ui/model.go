package ui

import (
	"image/color"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
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

	cover coverState

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
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
	m.help.ShortSeparator = " · "
	m.restyle()
	return m
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
}

// layout resolves the current geometry. It is pure, so View and Update can both
// ask for it without either of them owning the answer.
func (m Model) layout() layout {
	return computeLayout(m.width, m.height, m.helpHeight(), m.err != nil, m.cell)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		tickCmd(),
		fetchStateCmd(m.player),
	)
}
