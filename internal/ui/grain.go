package ui

import (
	"image/color"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The record itself, in dots, moving.
//
// Every other picture here is drawn from the sound. This one is drawn from the
// sleeve: the cover taken apart into a few hundred thousand dots, lit where it
// is light and dark where it is dark, and then bent by what is playing. The
// spectrum lifts the columns it belongs to and a hit sends a ring out from the
// middle, so the record breathes and ripples without ever stopping being the
// record.
//
// It is only offered on the whole screen. In the strip under the artwork it
// would be four rows of a photograph, which is a smudge, and the artwork itself
// is already on that screen a few cells away.

const (
	// grainLit is where a dot's brightness has to sit before it lights, once
	// the ordered threshold under it has been added. Half: the picture is a
	// negative of itself the other side of this, and half is where a stretched
	// cover has half its dots.
	grainLit = 128

	// grainSpread is how far apart the ordered thresholds are set. It is what
	// turns a brightness into a density of dots: at nothing the picture would
	// be two flat tones, and at too much it is a grey fog with no picture in it.
	grainSpread = 14

	// grainBend is how far the loudest band lifts the columns it covers, as a
	// share of the picture's height. Enough to see the record move with the
	// music, little enough that it is never torn apart by it.
	grainBend = 0.06

	// grainEase is how fast the bend follows the spectrum. Slower than the
	// bands themselves: what is wanted is the record swaying, not every
	// twitch of a meter.
	grainEase = 0.25

	// A hit sends a ring out from the middle. grainRingSpeed is how fast it
	// travels in dot rows a frame, grainRingWidth how thick it is, grainRingPush
	// how far it shoves the picture as it passes, and grainRingFade what it
	// keeps of that each frame.
	grainRingSpeed = 9
	grainRingWidth = 14
	grainRingPush  = 5
	grainRingFade  = 0.94

	// grainHit is the rise in loudness that starts one.
	grainHit = 0.06
)

// bayer is the ordered threshold matrix the dots are dithered against.
//
// Ordered rather than diffused: error diffusion gives a better still picture and
// a terrible moving one, because a pixel's dot depends on its neighbours' and
// the whole field crawls the moment anything shifts. Here every dot's threshold
// depends on where it is on the screen and nothing else, so the picture holds
// still until it is deliberately moved.
var bayer = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// grainState is what the picture carries between frames.
type grainState struct {
	// have is the cover this was ground from, and the size it was ground for.
	have           cover.Grain
	url            string
	cellsX, cellsY int

	// asked stops a load being started again while one is in flight.
	asked string

	// bend is where each column of dots is being lifted to, and ring is the
	// shock travelling out from the middle after a hit.
	bend []float32

	ring     float32
	ringPush float32

	wasLoud float32
}

// grainLines draws the record, w cells across and rows deep.
func (m Model) grainLines(w, rows int) []string {
	g := m.grain.have
	if g.DotsX == 0 || g.CellsX != w || g.CellsY != rows {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	grid := make([]uint8, w*rows)

	midX, midY := float32(dotsX)/2, float32(dotsY)/2

	for y := range dotsY {
		for x := range dotsX {
			// Where this dot reads the record from: itself, lifted by the band
			// under it and shoved by whatever ring is passing.
			from := float32(y)
			if len(m.grain.bend) == dotsX {
				from += m.grain.bend[x]
			}
			if m.grain.ringPush > 0.02 {
				dx, dy := float32(x)-midX, float32(y)-midY
				if d := float32(math.Hypot(float64(dx), float64(dy))); d > 0 {
					// Only the crest of the ring moves anything, and it moves it
					// along the line out from the middle.
					if off := d - m.grain.ring; off > -grainRingWidth && off < grainRingWidth {
						fade := 1 - abs32(off)/grainRingWidth
						from -= dy / d * m.grain.ringPush * fade
					}
				}
			}

			at := int(from)
			if at < 0 || at >= dotsY {
				continue
			}

			// Lit where the record is brighter than the threshold under this
			// dot: the same middle for all of them, moved up or down by where
			// the dot sits in the ordered pattern.
			if int(g.Lum[at*dotsX+x]) < grainLit+(bayer[y%4][x%4]-8)*grainSpread {
				continue
			}

			cell := (y/dotsPerCellY)*w + x/dotsPerCellX
			grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		}
	}

	return m.grainDraw(w, rows, grid)
}

// grainDraw turns the dots into rows, each cell in the colour the cover has
// there.
func (m Model) grainDraw(w, rows int, grid []uint8) []string {
	cells := m.grain.have.Cell

	lines := make([]string, rows)
	for r := range rows {
		var sb strings.Builder

		var run strings.Builder
		var was color.RGBA
		lit := false
		flush := func() {
			if run.Len() > 0 {
				style := lipgloss.NewStyle().Foreground(lipgloss.Color(hex(was)))
				sb.WriteString(style.Render(run.String()))
				run.Reset()
			}
		}

		for c := range w {
			at := r*w + c
			if grid[at] == 0 {
				flush()
				lit = false
				sb.WriteByte(' ')
				continue
			}

			want := cells[at]
			if !lit || want != was {
				flush()
				was, lit = want, true
			}
			run.WriteRune(rune(brailleBase + int(grid[at])))
		}
		flush()
		lines[r] = fit(sb.String(), w)
	}
	return lines
}

// hex is a colour as a terminal wants it.
func hex(c color.RGBA) string {
	const digits = "0123456789abcdef"
	out := []byte("#000000")
	for i, v := range []uint8{c.R, c.G, c.B} {
		out[1+i*2] = digits[v>>4]
		out[2+i*2] = digits[v&0xf]
	}
	return string(out)
}

// grainFlow moves the picture on by a frame: the bend follows the spectrum, and
// a hit sends a ring out from the middle.
func (m *Model) grainFlow(w, rows int) {
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 {
		return
	}

	if len(m.grain.bend) != dotsX {
		m.grain.bend = make([]float32, dotsX)
	}

	bands := m.scope.bands
	reach := grainBend * float32(dotsY)
	for x := range dotsX {
		var want float32
		if len(bands) > 0 {
			want = bands[min(x*len(bands)/dotsX, len(bands)-1)] * reach
		}
		m.grain.bend[x] += (want - m.grain.bend[x]) * grainEase
	}

	// The ring: started by the music getting louder, the same signal the sparks
	// are thrown by.
	rise := max(m.scope.envelope-m.grain.wasLoud, 0) / max(m.scope.envelope, scopeFloor)
	m.grain.wasLoud = m.scope.envelope

	if rise > grainHit {
		m.grain.ring, m.grain.ringPush = 0, grainRingPush*min(rise*6, 1.6)
	}
	m.grain.ring += grainRingSpeed
	m.grain.ringPush *= grainRingFade
	if m.grain.ring > float32(dotsX+dotsY) {
		m.grain.ringPush = 0
	}
}

// grind asks for the cover to be taken apart, if what is held is not the record
// on now or not the size the screen is. It is called as the frames arrive rather
// than when the screen opens, because either can change under it.
func (m *Model) grind() tea.Cmd {
	url := m.cover.url
	if url == "" || m.covers == nil {
		return nil
	}

	if m.grain.url == url && m.grain.cellsX == m.width && m.grain.cellsY == m.height {
		return nil
	}
	// One at a time: grinding is a decode and a scale, and asking again every
	// frame while the first one is still running would spend the whole machine
	// on the same picture.
	if m.grain.asked == url {
		return nil
	}

	m.grain.asked = url
	return grindCmd(m.covers, url, m.width, m.height)
}
