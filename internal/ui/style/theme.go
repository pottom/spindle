package style

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is the palette from SCREENS.md section 2, resolved for one background.
type Theme struct {
	Accent      color.Color
	AccentDim   color.Color
	Text        color.Color
	Muted       color.Color
	Faint       color.Color
	Border      color.Color
	BorderFocus color.Color
	Error       color.Color
	Warning     color.Color
}

// NewTheme resolves the palette against the terminal background. Lipgloss v2
// keeps AdaptiveColor in an impure compat package that queries the terminal at
// init time, so the light/dark choice is threaded through explicitly instead.
func NewTheme(isDark bool) Theme {
	pick := lipgloss.LightDark(isDark)
	return Theme{
		Accent:      pick(lipgloss.Color("#128A3E"), lipgloss.Color("#1DB954")),
		AccentDim:   pick(lipgloss.Color("#1DB954"), lipgloss.Color("#14833B")),
		Text:        pick(lipgloss.Color("#1A1A1A"), lipgloss.Color("#E8E8E8")),
		Muted:       pick(lipgloss.Color("#57606A"), lipgloss.Color("#8B949E")),
		Faint:       pick(lipgloss.Color("#8C959F"), lipgloss.Color("#484F58")),
		Border:      pick(lipgloss.Color("#D0D7DE"), lipgloss.Color("#30363D")),
		BorderFocus: pick(lipgloss.Color("#128A3E"), lipgloss.Color("#1DB954")),
		Error:       pick(lipgloss.Color("#CF222E"), lipgloss.Color("#F85149")),
		Warning:     pick(lipgloss.Color("#9A6700"), lipgloss.Color("#D29922")),
	}
}
