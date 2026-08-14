package style

import (
	"image/color"
	"math"
	"strings"

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

	// Raised is the background of a block that stands apart from the screen.
	Raised lipgloss.Style

	// Found is what a search match is drawn in: the accent laid behind the
	// letters that matched, the way grep colours what it found.
	Found lipgloss.Style

	// Rule is a quiet line drawn over the screen rather than in it: a frame
	// round a block, a leader from one thing to another. The border grey, which
	// is what the unplayed half of the progress bar is drawn in — so a rule
	// reads as part of the furniture rather than as something with an opinion.
	Rule lipgloss.Style

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

	// Ladder is the segmented meter's colours: one for each step of the climb,
	// coolest at the foot and hottest at the top. Indexed by height alone —
	// on that picture the colour is what tells you how high a bar has gone,
	// which is the whole of how a stack of lamps has ever been read.
	Ladder []lipgloss.Style

	// Words is what a lyric is set in: hue by the sound a word was sung on,
	// strength by how loud it was. Indexed [hue][level], like the spectrum, and
	// cut from the same accent — but around a far wider arc, because on that
	// picture the colour is the only thing separating one word from the next,
	// and a narrow sweep leaves a whole line one colour.
	Words [][]lipgloss.Style

	// The same two palettes again, as the escape sequences they come out as.
	//
	// A picture of braille is thirty thousand cells a second, and every run of
	// them in one colour used to be handed to a style to render — which builds
	// the sequence, allocates a string and joins it, over and over, for a colour
	// that was decided when the artwork was loaded. Measured at 200x50: eleven
	// and a half thousand allocations a frame, three hundred and seventy
	// megabytes over thirty seconds, a hundred and fifty collections, and a tail
	// that missed the frame outright twice in nine hundred.
	//
	// So the sequences are cut once, here, and the drawing writes them straight
	// out. See Wrap and drawCells.
	BarsSeq   [][]Seq
	WordsSeq  [][]Seq
	LadderSeq []Seq

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
// groundStep is how far the raised block's ground sits off the screen's own, in
// lightness.
//
// Small on purpose: enough that the block has an edge without a frame, not so
// much that it reads as a panel dropped on the screen. Measured against the
// grounds a terminal actually comes with — pure black, GitHub's #0D1117,
// Catppuccin, Solarized Light — where it lands a step away and in the same hue
// on every one of them.
const groundStep = 0.03

// On is the same styles with the block's ground taken from the terminal's own
// colour rather than assumed.
//
// The terminal says what it is when it is asked, and until this it was asked
// only whether it was dark. That is enough for the text and wrong for a ground:
// a light theme is not white — Solarized's is cream — and a raised block painted
// in a cool grey on it is a card from another program. Taking the hue from the
// screen it sits on makes it that screen's card.
func (s Styles) On(ground color.Color) Styles {
	if ground == nil {
		return s
	}
	h, sat, l := toHSL(ground)
	if l < 0.5 {
		l = min(l+groundStep, 1)
	} else {
		l = max(l-groundStep, 0)
	}
	s.Raised = lipgloss.NewStyle().Background(fromHSL(h, sat, l))
	return s
}

// found is how a search match is marked: the accent behind the letters, and an
// ink taken from the accent's own hue at whichever end of the scale it is not.
//
// A background rather than a colour, because the colour is already spoken for.
// The row under the cursor is drawn in the accent to begin with, and a match
// coloured accent on it would be a highlight that disappears on exactly the row
// being looked at — which is the row it is most wanted on.
//
// The ink is picked off the accent rather than off the theme because the accent
// is a photograph's average and arrives at any lightness at all: dark text on a
// dark cover's accent is a black bar, and the theme's own text would be that on
// half the records in a library.
func found(accent color.Color) lipgloss.Style {
	h, sat, l := toHSL(accent)
	ink := fromHSL(h, min(sat, 0.25), 0.08)
	if l < 0.45 {
		ink = fromHSL(h, min(sat, 0.25), 0.96)
	}
	return lipgloss.NewStyle().Background(accent).Foreground(ink)
}

func New(isDark bool, accent color.Color) Styles {
	t := NewTheme(isDark)
	if accent == nil {
		accent = t.Accent
	}
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

	out := Styles{
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

		FactLabel: fg(t.Faint),
		Queued:    fg(accent),
		StarOn:    fg(accent),
		StarOff:   fg(t.Faint),

		ScrollThumb: fg(accent),
		ScrollTrack: fg(t.Faint),

		Bars:   barPalette(t, accent),
		Ladder: ladderPalette(t, accent),
		Words:  huePalette(t, accent, wordsHueArc),

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
		Rule:      fg(t.Border),
		Found:     found(accent),
		Raised:    lipgloss.NewStyle().Background(t.Raised),
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
	out.BarsSeq, out.WordsSeq = Sequences(out.Bars), Sequences(out.Words)
	out.LadderSeq = Sequences([][]lipgloss.Style{out.Ladder})[0]
	return out
}

// Seq is what a style comes out as: what is written before the text it dresses,
// and what is written after to put the terminal back.
type Seq struct{ Open, Close string }

// Wrap writes text in the sequence, which is what a style's own Render does —
// except that the sequence was cut once rather than being built again for every
// run of braille on the screen.
func (s Seq) Wrap(text string) string { return s.Open + text + s.Close }

// Sequences cuts a palette into the escape sequences it renders as.
//
// Taken from the style itself rather than assembled here: a style is asked to
// dress one character nobody will ever print, and what it put either side of it
// is what every run of that colour needs. That way this cannot drift from what
// lipgloss would have done — and a test holds the two against each other.
func Sequences(palette [][]lipgloss.Style) [][]Seq {
	const sentinel = "\x00"

	out := make([][]Seq, len(palette))
	for i, row := range palette {
		out[i] = make([]Seq, len(row))
		for j, st := range row {
			dressed := st.Render(sentinel)
			at := strings.Index(dressed, sentinel)
			if at < 0 {
				continue
			}
			out[i][j] = Seq{Open: dressed[:at], Close: dressed[at+len(sentinel):]}
		}
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
const (
	lyricFadeSteps = 6

	// lyricFadeBias pulls the fall toward the middle of the window. Below one,
	// the rows next to the current line start receding at once instead of
	// holding their strength for two or three rows.
	// Taken down twice, from 0.45 to 0.32 and then to here, both times because
	// there was still more grey than the window wanted. Most of it now sits near
	// the dark end, so the line being sung stands well clear of what surrounds
	// it and only its immediate neighbours hold enough to be read at a glance.
	lyricFadeBias = 0.22
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
	// Darker than the theme's faintest text by some way. Ending at the faintest
	// left the far rows the same weight as the chrome around them, so the window
	// had no edge — it wants to fall off into the background, not stop at it.
	far := shift(t.Faint, 0, 0.85, 0.20)

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
	// A hundred and twenty-eight, which is where the hue index stops: it travels
	// through the drawing code as an int8. See TestPaletteFitsTheIndex.
	//
	// Measured on a saturated accent, as the strongest neighbouring pair of
	// bands differ by, out of 255: ten steps put 55 between them across the
	// lyric palette's arc — a picture visibly cut into sections — sixty-four put
	// 8, and this puts 4.
	barFreqSteps  = 128
	barLevelSteps = 6

	// wordsHueArc is how far a lyric's colours travel. Much wider than the
	// spectrum's: there the sweep is a gradient across one shape, here it is
	// what separates one word from the next.
	wordsHueArc = 170

	// ladderSteps is how many colours the segmented meter climbs through, and
	// ladderHueArc how far round the wheel that climb travels.
	//
	// The arc is wide because on that picture the colour *is* the reading: a
	// hundred and twenty degrees from the accent's cool side to its warm one is
	// four or five colours the eye can name, which is what makes a stack of
	// lamps legible from across a room.
	ladderSteps  = 24
	ladderHueArc = 120

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

// ladderPalette builds the segmented meter's climb.
//
// The colour runs up the bar rather than across the picture, because that is
// what a ladder of lamps is: the eye reads how high the stack has gone from
// where its colour has got to, and it can do that on one bar alone. The arc is
// much wider than the spectrum's — a meter that climbed through two shades of
// the same colour would be a gradient rather than a scale — but it is still cut
// from the artwork, swinging either side of the cover's own hue, so a warm
// record keeps a warm meter.
func ladderPalette(t Theme, accent color.Color) []lipgloss.Style {
	base, sat, light := toHSL(accent)
	sat = max(min(sat*hueLift, 1), hueLeast)

	out := make([]lipgloss.Style, ladderSteps)
	for i := range out {
		v := float64(i) / float64(ladderSteps-1)

		hue := math.Mod(base+(v-0.5)*ladderHueArc+360, 360)
		c := fromHSL(hue, min(sat*(0.7+0.45*v), 1), light*(0.62+0.6*v))
		if v > 0.86 {
			// The last rungs go toward white, the way the tip of a bar does:
			// the top of a meter is meant to look like too much.
			c = blend(c, t.Text, (v-0.86)/0.14*0.45)
		}
		out[i] = lipgloss.NewStyle().Foreground(c)
	}
	return out
}

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
	return huePalette(t, accent, barHueArc)
}

// huePalette builds one of those: hue across a given arc, strength up the
// height.
// hueLift is how far past the accent's own saturation the pictures are drawn,
// and hueLeast how little colour they are allowed however grey the cover was.
//
// Measured on real covers: the accent comes back a fifth to a half saturated,
// because it is an average of a photograph. At a fifth, a screenful of single
// dots reads as grey with a tint whatever it is multiplied by — which is why
// there is a floor as well as a lift. The hue is still the record's own; what
// is taken from it is only the greyness.
const (
	hueLift  = 1.5
	hueLeast = 0.55
)

func huePalette(t Theme, accent color.Color, arc float64) [][]lipgloss.Style {
	base, sat, light := toHSL(accent)
	sat = max(min(sat*hueLift, 1), hueLeast)

	out := make([][]lipgloss.Style, barFreqSteps)
	for f := range out {
		// Centred on the accent, so the middle of the spectrum is the album's
		// own colour and the ends lean either side of it.
		offset := (float64(f)/float64(barFreqSteps-1) - 0.5) * arc
		hue := math.Mod(base+offset+360, 360)

		out[f] = make([]lipgloss.Style, barLevelSteps)
		for l := range out[f] {
			v := float64(l) / float64(barLevelSteps-1)

			// The foot of a bar sits close to the ground and the tip lifts well
			// past the accent, so height reads before colour does.
			//
			// The colour is pushed past the accent's own saturation rather than
			// held under it. A cover's accent is an average of a photograph and
			// comes out washed; a meter drawn in it looks like a meter that has
			// been left in the sun, and the dots are small enough that a weak
			// colour reads as grey.
			c := fromHSL(hue, min(sat*(0.62+0.45*v), 1), light*(0.42+0.75*v))
			// No whitening at the top, and that is where a line of type parts
			// company with a meter. A bar's tip goes toward white because a tip
			// that only gets lighter in its own hue stops reading as hotter — but
			// a line reaches that step whenever the record is loud, so the same
			// rule washed the colour out of it at exactly the moments it should
			// have been strongest. Watched as a line going colourful and then
			// paling back, over and over.
			out[f][l] = lipgloss.NewStyle().Foreground(c)
		}
	}
	return out
}
