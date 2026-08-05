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
	Volume    lipgloss.Style

	// Lyrics, indexed by how far a line is from the one being sung. Only that
	// line is at full strength; everything else recedes with distance, above
	// and below alike, so the eye is drawn to the words actually sounding
	// rather than to a wall of even text.
	LyricFade []lipgloss.Style

	// LyricAhead is the part of the line being sung that has not been reached
	// yet: the accent, held back, so the sweep across it reads as progress
	// through the line rather than as two different lines.
	LyricAhead lipgloss.Style

	// Detail panel.
	FactLabel lipgloss.Style
	// Queued marks a track already waiting in the queue, wherever else it is
	// being listed.
	Queued lipgloss.Style

	StarOn  lipgloss.Style
	StarOff lipgloss.Style

	// Scrollbar.
	ScrollThumb lipgloss.Style
	ScrollTrack lipgloss.Style

	// Bars is what both visualisers are drawn in: hue across the width,
	// strength up the height. Indexed [position][level].
	Bars [][]lipgloss.Style

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

		LyricFade:  lyricFade(t, accent),
		LyricAhead: lipgloss.NewStyle().Foreground(shift(accent, 0, 0.62, 0.6)),

		FactLabel: fg(t.Faint),
		Queued:    fg(accent),
		StarOn:    fg(accent),
		StarOff:   fg(t.Faint),

		ScrollThumb: fg(accent),
		ScrollTrack: fg(t.Faint),

		Bars: barPalette(t, accent),

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

		// The transport is the artwork's colour, like everything else on the
		// screen that acts rather than merely reports.
		Controls:  fg(accent),
		ToggleOn:  fg(accent),
		ToggleOff: fg(t.Faint),
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
const (
	lyricFadeSteps = 6

	// lyricFadeBias pulls the fall toward the middle of the window. Below one,
	// the rows next to the current line start receding at once instead of
	// holding their strength for two or three rows.
	lyricFadeBias = 0.45
)

// lyricFade builds the fade: the line being sung in the artwork's accent, then
// a fall away from it that keeps going past the theme's faintest text.
//
// The fall follows a cosine rather than a straight line, so the rows either side
// of the current one stay nearly as strong as it and the outermost ones drop
// away quickly. Lines of text falling off at the edges like that read as a
// surface curving away — the same shading that makes a cylinder look round.
func lyricFade(t Theme, accent color.Color) []lipgloss.Style {
	near := blend(t.Muted, t.Faint, 0.2)
	far := shift(t.Faint, 0, 0.85, 0.45)

	out := make([]lipgloss.Style, lyricFadeSteps)
	out[0] = lipgloss.NewStyle().Foreground(accent).Bold(true)
	for i := 1; i < lyricFadeSteps; i++ {
		t := float64(i-1) / float64(lyricFadeSteps-2)

		// The curve is biased toward the edge before the cosine is taken. A
		// plain cosine leaves three or four rows sitting at almost the same
		// strength around the middle, which flattens the very roundness it is
		// there to give; this pulls the fall forward so only the line being
		// sung and its immediate neighbour stay bright.
		f := 1 - math.Cos(math.Pow(t, lyricFadeBias)*math.Pi/2)
		out[i] = lipgloss.NewStyle().Foreground(blend(near, far, f))
	}
	return out
}

const (
	// barFreqSteps is how many hues the spectrum sweeps across its width, and
	// barLevelSteps how many strengths it climbs through.
	barFreqSteps  = 10
	barLevelSteps = 6

	// barHueArc is how far the hue travels from one end of the spectrum to the
	// other, in degrees.
	//
	// Wide enough for the sweep to be a colour rather than a shade of one, and
	// narrow enough that both ends still belong to the artwork: at eighty a
	// warm cover runs from red through orange to yellow across the width, which
	// reads as the record's own light spread out, where half of that read as a
	// single colour someone had dimmed at one end.
	barHueArc = 80
)

// barPalette builds the spectrum's colours: hue across the frequency range,
// strength up the height of a bar.
//
// Two dimensions rather than one, because a bar carries two facts. Where the
// energy sits is the horizontal one — the hue sweeps from a deeper shade at the
// bass end to a brighter one at the treble, so a mix reads as a colour before
// it reads as a shape. How loud that band is is the vertical one — a bar rises
// out of a dim base into a lit tip, which is what makes a meter look like it is
// burning rather than merely tall.
func barPalette(t Theme, accent color.Color) [][]lipgloss.Style {
	base, sat, light := toHSL(accent)

	out := make([][]lipgloss.Style, barFreqSteps)
	for f := range out {
		// Centred on the accent, so the middle of the spectrum is the album's
		// own colour and the ends lean either side of it.
		offset := (float64(f)/float64(barFreqSteps-1) - 0.5) * barHueArc
		hue := math.Mod(base+offset+360, 360)

		out[f] = make([]lipgloss.Style, barLevelSteps)
		for l := range out[f] {
			v := float64(l) / float64(barLevelSteps-1)

			// The foot of a bar sits close to the ground and the tip lifts well
			// past the accent, so height reads before colour does.
			c := fromHSL(hue, min(sat*(0.55+0.5*v), 1), light*(0.42+0.75*v))
			if v > 0.82 {
				// The last step goes toward white: a tip that only gets lighter
				// in its own hue stops registering as hotter.
				c = blend(c, t.Text, (v-0.82)/0.18*0.55)
			}
			out[f][l] = lipgloss.NewStyle().Foreground(c)
		}
	}
	return out
}
