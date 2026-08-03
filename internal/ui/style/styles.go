package style

import "charm.land/lipgloss/v2"

// Styles is every Lipgloss style the UI renders with, derived from a Theme.
type Styles struct {
	Theme  Theme
	Border lipgloss.Border

	// Frame.
	Rule      lipgloss.Style // border runes
	AppName   lipgloss.Style // "spindle" in the top border
	DeviceOn  lipgloss.Style // device dot and name while playing
	DeviceOff lipgloss.Style // device dot and name while paused

	// Track information.
	Title  lipgloss.Style
	Artist lipgloss.Style
	Album  lipgloss.Style
	Time   lipgloss.Style

	// Transport row.
	Controls  lipgloss.Style
	ToggleOn  lipgloss.Style
	ToggleOff lipgloss.Style
	Volume    lipgloss.Style

	// Standalone screens and banners.
	Heading lipgloss.Style
	Detail  lipgloss.Style
	Error   lipgloss.Style
}

// New builds the styles for the given terminal background.
func New(isDark bool) Styles {
	t := NewTheme(isDark)
	return Styles{
		Theme:  t,
		Border: lipgloss.NormalBorder(),

		Rule:      lipgloss.NewStyle().Foreground(t.Border),
		AppName:   lipgloss.NewStyle().Foreground(t.Text),
		DeviceOn:  lipgloss.NewStyle().Foreground(t.Accent),
		DeviceOff: lipgloss.NewStyle().Foreground(t.Muted),

		Title:  lipgloss.NewStyle().Foreground(t.Text),
		Artist: lipgloss.NewStyle().Foreground(t.Accent),
		Album:  lipgloss.NewStyle().Foreground(t.Muted),
		Time:   lipgloss.NewStyle().Foreground(t.Muted),

		Controls:  lipgloss.NewStyle().Foreground(t.Text),
		ToggleOn:  lipgloss.NewStyle().Foreground(t.Accent),
		ToggleOff: lipgloss.NewStyle().Foreground(t.Faint),
		Volume:    lipgloss.NewStyle().Foreground(t.Text),

		Heading: lipgloss.NewStyle().Foreground(t.Text),
		Detail:  lipgloss.NewStyle().Foreground(t.Muted),
		Error:   lipgloss.NewStyle().Foreground(t.Error),
	}
}
