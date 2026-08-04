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

	// Detail panel.
	FactLabel lipgloss.Style
	StarOn    lipgloss.Style
	StarOff   lipgloss.Style

	// Scrollbar.
	ScrollThumb lipgloss.Style
	ScrollTrack lipgloss.Style

	// Scope is the waveform's palette, quietest first. The trace is coloured by
	// how loud each moment is rather than by where it sits on the screen, so it
	// breathes with the music instead of reading as a fixed band.
	Scope []lipgloss.Style

	// Status line.
	DeviceOn  lipgloss.Style
	DeviceOff lipgloss.Style
	Quality   lipgloss.Style
	Help      lipgloss.Style

	// Artwork area and standalone screens.
	NoCover lipgloss.Style
	Heading lipgloss.Style
	Detail  lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
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

		FactLabel: fg(t.Faint),
		StarOn:    fg(accent),
		StarOff:   fg(t.Faint),

		ScrollThumb: fg(accent),
		ScrollTrack: fg(t.Faint),

		Scope: scopeRamp(t, accent),

		Quality: fg(t.Faint),

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
		Warning: fg(t.Warning),
	}
}

// scopeLevels is how many strengths the waveform is drawn in. Five is enough
// for the trace to flare and recede without breaking it into so many runs that
// the row stops compressing.
const scopeLevels = 5

// scopeRamp builds the waveform's palette: from a colour that recedes into the
// background at the quiet end, through the artwork's accent, to a lifted
// version of it for the moments that hit hardest.
func scopeRamp(t Theme, accent color.Color) []lipgloss.Style {
	quiet := blend(t.Faint, accent, 0.35)
	loud := blend(accent, t.Text, 0.45)

	out := make([]lipgloss.Style, scopeLevels)
	for i := range out {
		// The accent sits two thirds of the way up, so most of the trace is the
		// artwork's colour and only the peaks lift out of it.
		var c color.Color
		if f := float64(i) / float64(scopeLevels-1); f <= 0.66 {
			c = blend(quiet, accent, f/0.66)
		} else {
			c = blend(accent, loud, (f-0.66)/0.34)
		}
		out[i] = lipgloss.NewStyle().Foreground(c)
	}
	return out
}

// blend mixes two colours, t running from a to b.
func blend(a, b color.Color, t float64) color.Color {
	t = min(max(t, 0), 1)
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()

	mix := func(x, y uint32) uint8 {
		return uint8((float64(x>>8)*(1-t) + float64(y>>8)*t))
	}
	return color.RGBA{R: mix(ar, br), G: mix(ag, bg), B: mix(ab, bb), A: 0xff}
}
