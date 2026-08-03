package style

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles is every Lipgloss style the UI renders with. The accent is not fixed:
// it is taken from the album artwork, so the whole screen picks up the colour of
// whatever is playing.
type Styles struct {
	Theme  Theme
	Accent color.Color

	// Tab header.
	TabActive lipgloss.Style
	TabIdle   lipgloss.Style
	TabRule   lipgloss.Style

	// Lists.
	Cursor       lipgloss.Style
	RowPrimary   lipgloss.Style
	RowSecondary lipgloss.Style
	RowTrailing  lipgloss.Style
	RowSelected  lipgloss.Style
	RowPlaying   lipgloss.Style
	Empty        lipgloss.Style

	// Search field.
	Query       lipgloss.Style
	QueryPrompt lipgloss.Style
	Placeholder lipgloss.Style

	// Track information.
	Title  lipgloss.Style
	Artist lipgloss.Style
	Album  lipgloss.Style
	Time   lipgloss.Style

	// Progress line.
	Elapsed   lipgloss.Style
	Remaining lipgloss.Style
	Knob      lipgloss.Style

	// Transport row.
	Controls  lipgloss.Style
	ToggleOn  lipgloss.Style
	ToggleOff lipgloss.Style
	MeterOn   lipgloss.Style
	MeterOff  lipgloss.Style
	Volume    lipgloss.Style

	// Status line.
	DeviceOn  lipgloss.Style
	DeviceOff lipgloss.Style
	Help      lipgloss.Style

	// Artwork area and standalone screens.
	NoCover lipgloss.Style
	Heading lipgloss.Style
	Detail  lipgloss.Style
	Error   lipgloss.Style
}

// New builds the styles for the given terminal background. A nil accent falls
// back to the theme's own, for the moments before any artwork has loaded.
func New(isDark bool, accent color.Color) Styles {
	t := NewTheme(isDark)
	if accent == nil {
		accent = t.Accent
	}
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

	return Styles{
		Theme:  t,
		Accent: accent,

		TabActive: fg(t.Text).Bold(true),
		TabIdle:   fg(t.Faint),
		TabRule:   fg(accent),

		Cursor:       fg(accent),
		RowPrimary:   fg(t.Muted),
		RowSecondary: fg(t.Faint),
		RowTrailing:  fg(t.Faint),
		RowSelected:  fg(t.Text).Bold(true),
		RowPlaying:   fg(accent),
		Empty:        fg(t.Faint),

		Query:       fg(t.Text),
		QueryPrompt: fg(accent),
		Placeholder: fg(t.Faint),

		Title:  fg(t.Text).Bold(true),
		Artist: fg(accent),
		Album:  fg(t.Muted),
		Time:   fg(t.Muted),

		Elapsed:   fg(accent),
		Remaining: fg(t.Border),
		Knob:      fg(accent),

		Controls:  fg(t.Text),
		ToggleOn:  fg(accent),
		ToggleOff: fg(t.Faint),
		MeterOn:   fg(t.Muted),
		MeterOff:  fg(t.Border),
		Volume:    fg(t.Muted),

		DeviceOn:  fg(accent),
		DeviceOff: fg(t.Faint),
		Help:      fg(t.Faint),

		NoCover: fg(t.Faint),
		Heading: fg(t.Text),
		Detail:  fg(t.Muted),
		Error:   fg(t.Error),
	}
}
