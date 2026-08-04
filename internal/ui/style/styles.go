package style

import (
	"image/color"
	"math"

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

	// Lyrics, indexed by how far a line is from the one being sung. Only that
	// line is at full strength; everything else recedes with distance, above
	// and below alike, so the eye is drawn to the words actually sounding
	// rather than to a wall of even text.
	LyricFade []lipgloss.Style

	// Detail panel.
	FactLabel lipgloss.Style
	StarOn    lipgloss.Style
	StarOff   lipgloss.Style

	// Scrollbar.
	ScrollThumb lipgloss.Style
	ScrollTrack lipgloss.Style

	// The waveform is drawn in two families: the middle of the trace runs
	// through the artwork's accent, and its extremes through the theme's cool
	// grey. The contrast between the two is what gives the line a lit core and
	// pale tips rather than one flat colour top to bottom.
	//
	// Within each family the step is chosen by how loud the moment is, so the
	// trace also flares on a hit and recedes between them.
	ScopeCore []lipgloss.Style
	ScopeEdge []lipgloss.Style

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

		LyricFade: lyricFade(t, accent),

		FactLabel: fg(t.Faint),
		StarOn:    fg(accent),
		StarOff:   fg(t.Faint),

		ScrollThumb: fg(accent),
		ScrollTrack: fg(t.Faint),

		ScopeCore: coreRamp(accent),
		ScopeEdge: edgeRamp(t),

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

// scopeLevels is how many strengths each waveform family is drawn in. Four is
// enough for the trace to flare and recede without breaking a row into so many
// runs that it stops compressing.
const scopeLevels = 4

// coreRamp is the middle of the trace: the artwork's accent, swept through a
// little of the hue on either side of it. A quiet passage sits a touch cooler
// and darker, a hit lifts warmer and brighter — the colour moves with the music
// without ever leaving the accent's family.
func coreRamp(accent color.Color) []lipgloss.Style {
	quiet := shift(accent, -14, 0.80, 0.86)
	hot := shift(accent, 13, 1.12, 1.14)
	return ramp(quiet, accent, hot)
}

// edgeRamp is the extremes of the swing: the theme's own cool grey, which reads
// against the accent as a different colour rather than merely a dimmer one.
func edgeRamp(t Theme) []lipgloss.Style {
	return ramp(t.Faint, t.Muted, blend(t.Muted, t.Text, 0.35))
}

// ramp builds the palette between three stops, the middle one two thirds of the
// way up so most of the trace sits on it and only the peaks lift past.
func ramp(low, mid, high color.Color) []lipgloss.Style {
	const knee = 0.66

	out := make([]lipgloss.Style, scopeLevels)
	for i := range out {
		f := float64(i) / float64(scopeLevels-1)
		var c color.Color
		if f <= knee {
			c = blend(low, mid, f/knee)
		} else {
			c = blend(mid, high, (f-knee)/(1-knee))
		}
		out[i] = lipgloss.NewStyle().Foreground(c)
	}
	return out
}

// shift rotates a colour's hue by the given degrees and scales its saturation
// and lightness, which keeps a derived colour recognisably related to the one
// it came from — blending toward white or grey does not.
func shift(c color.Color, degrees, sat, light float64) color.Color {
	h, s, l := toHSL(c)
	h = math.Mod(h+degrees+360, 360)
	return fromHSL(h, min(s*sat, 1), min(l*light, 1))
}

func toHSL(c color.Color) (h, s, l float64) {
	r0, g0, b0, _ := c.RGBA()
	r, g, b := float64(r0>>8)/255, float64(g0>>8)/255, float64(b0>>8)/255

	maxc, minc := max(r, g, b), min(r, g, b)
	l = (maxc + minc) / 2
	if maxc == minc {
		return 0, 0, l
	}

	d := maxc - minc
	if l > 0.5 {
		s = d / (2 - maxc - minc)
	} else {
		s = d / (maxc + minc)
	}

	switch maxc {
	case r:
		h = math.Mod((g-b)/d+6, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}

func fromHSL(h, s, l float64) color.Color {
	if s == 0 {
		v := uint8(l*255 + 0.5)
		return color.RGBA{R: v, G: v, B: v, A: 0xff}
	}

	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q

	channel := func(t float64) uint8 {
		t = math.Mod(t+1, 1)
		switch {
		case t < 1.0/6:
			return uint8((p+(q-p)*6*t)*255 + 0.5)
		case t < 1.0/2:
			return uint8(q*255 + 0.5)
		case t < 2.0/3:
			return uint8((p+(q-p)*(2.0/3-t)*6)*255 + 0.5)
		default:
			return uint8(p*255 + 0.5)
		}
	}

	hk := h / 360
	return color.RGBA{R: channel(hk + 1.0/3), G: channel(hk), B: channel(hk - 1.0/3), A: 0xff}
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

// lyricFadeSteps is how many strengths a lyric is drawn in: one for the line
// being sung and one for each row between it and the edge of the window. Fewer
// and the fall shows as bands; more and the deepest are never reached.
const lyricFadeSteps = 6

// lyricFade builds the fade: the line being sung in the artwork's accent, then
// a fall away from it that keeps going past the theme's faintest text.
//
// The fall follows a cosine rather than a straight line, so the rows either side
// of the current one stay nearly as strong as it and the outermost ones drop
// away quickly. Lines of text falling off at the edges like that read as a
// surface curving away — the same shading that makes a cylinder look round.
func lyricFade(t Theme, accent color.Color) []lipgloss.Style {
	near := blend(t.Muted, t.Text, 0.15)
	far := shift(t.Faint, 0, 0.85, 0.45)

	out := make([]lipgloss.Style, lyricFadeSteps)
	out[0] = lipgloss.NewStyle().Foreground(accent).Bold(true)
	for i := 1; i < lyricFadeSteps; i++ {
		t := float64(i-1) / float64(lyricFadeSteps-2)
		f := 1 - math.Cos(t*math.Pi/2)
		out[i] = lipgloss.NewStyle().Foreground(blend(near, far, f))
	}
	return out
}
