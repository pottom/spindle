package ui

import (
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/progress"
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

	ps              *player.State // last known server state
	localProgress   time.Duration // ticked locally, not ps.Progress
	optimisticUntil time.Time     // a poll must not overwrite before this

	cover coverState

	err       error
	tickCount int

	width, height int

	styles   style.Styles
	keys     keyMap
	help     help.Model
	progress progress.Model
	spinner  spinner.Model
}

// New wires a model around a playback backend and an artwork loader. The palette
// starts dark and is corrected once the terminal reports its background colour.
func New(p player.Player, covers *cover.Loader) Model {
	m := Model{
		player:   p,
		covers:   covers,
		keys:     newKeyMap(),
		help:     help.New(),
		progress: progress.New(progress.WithoutPercentage(), progress.WithFillCharacters(barRune, barRune)),
		spinner:  spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
	m.help.ShortSeparator = " · "
	m.applyBackground(true)
	return m
}

// applyBackground rebuilds every style that depends on the terminal background.
func (m *Model) applyBackground(isDark bool) {
	m.styles = style.New(isDark)
	m.help.Styles = help.DefaultStyles(isDark)
	m.spinner.Style = m.styles.Detail
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		tickCmd(),
		fetchStateCmd(m.player),
	)
}
