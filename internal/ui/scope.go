package ui

import (
	"math"
	"strings"

	"github.com/pottom/spindle/internal/player"
)

const (
	// scopeRows is how tall the trace is. Four rows of braille is sixteen dots,
	// which is enough for the shape of a waveform to read without the trace
	// becoming the loudest thing on the screen.
	scopeRows = 4

	// scopeChrome is the blank row kept above the trace, so it never butts
	// against the artwork.
	scopeChrome = 1

	// brailleBase is U+2800, where the eight-dot cells begin.
	brailleBase = 0x2800

	// A braille cell is two dots wide and four tall.
	dotsPerCellX = 2
	dotsPerCellY = 4
)

// brailleBit maps a dot's position inside a cell to its bit. The order is not
// sequential: the eight-dot cells were added to Unicode after the six-dot ones,
// so dots seven and eight sit above the rest.
var brailleBit = [dotsPerCellX][dotsPerCellY]uint8{
	{0, 1, 2, 6},
	{3, 4, 5, 7},
}

// scopeState is the waveform the player screen is drawing, and whether it is
// drawing one at all.
type scopeState struct {
	// on is what the key toggles. The trace is off until asked for: it costs a
	// redraw every 33ms, which is not something to spend without being asked.
	on bool

	// running guards the tick loop, so two callers cannot start it twice and
	// make the trace move at double speed.
	running bool

	// frame is the latest waveform, one value per horizontal dot, in -1..1. It
	// is resampled to whatever width the screen turns out to be.
	frame []float32

	// trail is what the last few frames drew, newest last. A cathode ray tube
	// leaves a glow behind the beam; without it a terminal trace looks redrawn
	// thirty times a second, because it is. The trigger is what makes this work
	// rather than smear: the older frames sit almost on top of the newest.
	trail [][]uint8

	// envelope follows the recent loudness so the trace can be scaled to it.
	// Measured against a live stream, peaks ran from 0.06 to 0.87 within one
	// track: at a fixed scale the quiet passages are a flat line and the loud
	// ones clip. It rises at once and falls slowly, as a meter does.
	envelope float32
}

const (
	// scopeDeflection is how much of the half-height a passage at the current
	// envelope fills, leaving room for something louder to still read as louder.
	scopeDeflection = 0.86

	// scopeRelease is the fraction of the envelope kept each frame while the
	// music is quieter than it was — about a second and a half to fall by half.
	scopeRelease = 0.985

	// scopeFloor stops the gain running away in silence, where the only thing
	// left to amplify is the noise.
	scopeFloor = 0.05

	// scopeTrail is how many frames the glow lasts. One frame behind the beam
	// reads as afterglow; more than that smears, because a thirtieth of a
	// second is long enough for the wave to have moved right across the screen.
	scopeTrail = 2
)

// follow updates the loudness envelope from a new frame.
func (s *scopeState) follow(frame []float32) {
	var peak float32
	for _, v := range frame {
		if v < 0 {
			v = -v
		}
		peak = max(peak, v)
	}

	s.envelope = max(peak, s.envelope*scopeRelease)
	s.envelope = max(s.envelope, scopeFloor)
}

// scopeAvailable reports whether the trace can be drawn without moving anything.
//
// It goes into the blank rows the player screen already has below the artwork,
// never into rows taken from it: shrinking the body would move the cover and
// the text every time the key was pressed, and a visualiser is not worth making
// the rest of the screen jump.
func (m Model) scopeAvailable() bool {
	if m.tab != tabPlayer || m.noDevice || m.ps == nil {
		return false
	}
	return m.scopeRoom(m.layout()) >= scopeRows+scopeChrome
}

// scopeRoom is how many blank rows sit below the artwork, which is what the
// trace has to fit into. scopeTop is the first of them.
func (m Model) scopeRoom(l layout) int {
	// Without a picture the body has no blank rows: the text and the list have
	// already taken them, and they are the reason the picture was dropped.
	if !l.hasArt() {
		return 0
	}
	return max(l.bodyHeight-m.scopeTop(l), 0)
}

func (m Model) scopeTop(l layout) int {
	block := l.artHeight
	return max((l.bodyHeight-block)/2, 0) + block
}

// scopeVisible reports whether the trace is on screen right now.
func (m Model) scopeVisible() bool {
	return m.scope.on && m.scopeAvailable()
}

// scopeLines renders the trace across w cells.
//
// The waveform is drawn as a line rather than a scatter of points: consecutive
// samples are joined vertically, so a steep slope stays continuous instead of
// breaking into dots. Without that a loud passage looks like static.
//
// Each cell is coloured by how loud that moment is, not by where it sits, so
// the trace flares on a hit and recedes between them.
func (m Model) scopeLines(w int) []string {
	return m.scopeLinesFrom(w, m.scopeTrigger(w*dotsPerCellX))
}

// scopeLinesFrom draws the frame beginning at a given sample. Where that sample
// is decided is scopeTrigger's business; this only draws.
func (m Model) scopeLinesFrom(w, start int) []string {
	if w <= 0 || len(m.styles.ScopeCore) == 0 {
		return nil
	}
	grid, loud := m.scopeGrid(w, start)
	return m.scopeDraw(w, grid, loud)
}

// scopeGrid plots the frame: which braille dots the trace lights, and how far
// the wave swings under each cell.
func (m Model) scopeGrid(w, start int) ([]uint8, []float32) {
	dotsX, dotsY := w*dotsPerCellX, scopeRows*dotsPerCellY
	grid := make([]uint8, w*scopeRows)
	loud := make([]float32, w)

	prev := -1
	for x := range dotsX {
		y := m.scopeSample(start, x, dotsX)
		if a := abs32(y); a > loud[x/dotsPerCellX] {
			loud[x/dotsPerCellX] = a
		}

		// Map -1..1 onto the rows, leaving a dot of headroom at each edge so a
		// clipping passage does not look like a flat line against the border.
		dy := int(math.Round((0.5 - float64(y)*0.5*scopeDeflection) * float64(dotsY-1)))
		dy = min(max(dy, 0), dotsY-1)

		from := dy
		if prev >= 0 {
			from = prev
		}
		for yy := min(from, dy); yy <= max(from, dy); yy++ {
			cell := (yy/dotsPerCellY)*w + x/dotsPerCellX
			grid[cell] |= 1 << brailleBit[x%dotsPerCellX][yy%dotsPerCellY]
		}
		prev = dy
	}

	return grid, loud
}

// scopeDraw turns a dot grid into rows, with the older frames glowing behind it.
//
// A cell the beam is on now is coloured by how loud that moment is; a cell only
// the afterglow reaches is drawn at the quiet end of the palette, so the glow
// reads as behind the trace rather than as part of it.
func (m Model) scopeDraw(w int, grid []uint8, loud []float32) []string {
	lines := make([]string, scopeRows)
	for r := range scopeRows {
		var sb strings.Builder

		// The extremes of the swing are drawn in the theme's cool grey and the
		// middle of the trace in the artwork's accent. Two families rather than
		// two strengths: a pale tip reads as a different colour, which is what
		// gives the line a lit core instead of a uniform band.
		ramp := m.styles.ScopeCore
		if r == 0 || r == scopeRows-1 {
			ramp = m.styles.ScopeEdge
		}

		// Runs of one colour are rendered together, so a row costs about as
		// much output as a line of text rather than one escape per cell.
		var run strings.Builder
		level := -1
		flush := func() {
			if run.Len() > 0 {
				sb.WriteString(ramp[level].Render(run.String()))
				run.Reset()
			}
		}

		for c := range w {
			at := r*w + c
			bits := grid[at]

			// Whatever the beam is not covering, the glow might be.
			want := scopeLevel(loud[c], len(ramp))
			for age, old := range m.scope.trail {
				if len(old) != len(grid) || old[at] == 0 {
					continue
				}
				if glow := len(m.scope.trail) - age - 1; bits == 0 {
					bits = old[at]
					want = min(glow, len(ramp)-1)
				} else {
					bits |= old[at]
				}
			}

			if bits == 0 {
				flush()
				level = -1
				sb.WriteByte(' ')
				continue
			}
			if want != level {
				flush()
				level = want
			}
			run.WriteRune(rune(brailleBase + int(bits)))
		}
		flush()
		lines[r] = fit(sb.String(), w)
	}
	return lines
}

// remember keeps a frame's dots so the next few can glow behind the beam.
func (s *scopeState) remember(grid []uint8) {
	s.trail = append(s.trail, grid)
	if len(s.trail) > scopeTrail {
		s.trail = s.trail[len(s.trail)-scopeTrail:]
	}
}

// scopeLevel picks a colour for how far a moment swings.
func scopeLevel(amplitude float32, levels int) int {
	// The square root spreads the quiet end out: most music sits low, and a
	// linear scale would leave the whole trace in one or two colours.
	t := math.Sqrt(float64(min(amplitude, 1)))
	return min(max(int(t*float64(levels)), 0), levels-1)
}

// scopeTrigger finds where in the frame to start drawing, so that a held note
// stands still on screen instead of shimmering.
//
// It looks for a rising zero crossing, the way an oscilloscope does, and
// requires the wave to have dipped below a small negative threshold first so
// that noise around zero cannot trigger it. Failing to find one is not a
// failure: the trace free-runs from the start of the frame, which is what a
// scope does with no trigger.
func (m Model) scopeTrigger(dots int) int {
	frame := m.scope.frame
	slack := len(frame) - dots
	if slack <= 0 || m.scope.envelope <= 0 {
		return 0
	}

	// A tenth of the recent loudness: high enough to ignore the noise floor,
	// low enough that a quiet passage still triggers.
	threshold := m.scope.envelope * 0.1

	armed := false
	for i := range slack {
		v := frame[i]
		if v < -threshold {
			armed = true
			continue
		}
		if armed && v >= 0 {
			return i
		}
	}
	return 0
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// scopeSample reads one sample of the frame, scaled to the recent loudness.
// With no frame at all the trace rests on the centre line, which is what
// silence looks like.
func (m Model) scopeSample(start, x, dots int) float32 {
	if len(m.scope.frame) == 0 || dots <= 0 || m.scope.envelope <= 0 {
		return 0
	}

	// A whole window is spread across the width, however wide that is, so the
	// screen always shows the same slice of time — about a thirtieth of a
	// second. Reading one sample per dot instead would zoom in on a wide
	// terminal and out on a narrow one, and the trace would appear to slow down
	// or speed up with nothing but the window size.
	window := min(player.WaveformWindow, len(m.scope.frame)-start)
	i := start + x*window/dots

	v := m.scope.frame[min(i, len(m.scope.frame)-1)]

	// Scaled to the recent loudness, then clamped: a sudden hit louder than
	// anything before it is drawn at the edge rather than off it.
	return min(max(v/m.scope.envelope, -1), 1)
}

// drawScope writes the trace into the blank rows directly under the artwork,
// leaving everything above exactly where it was.
//
// It hugs the cover rather than sitting at the foot of the screen: the trace
// belongs to the record being played, and a band of empty rows between the two
// would read as an unrelated meter.
func (m Model) drawScope(body []string, l layout) []string {
	at := m.scopeTop(l) + scopeChrome
	if at+scopeRows > len(body) {
		return body
	}
	w := l.interior - leftMargin - rightMargin
	for i, line := range m.scopeLines(w) {
		body[at+i] = m.pad(line, l)
	}
	return body
}

// rememberScope keeps this frame's dots so the next few draw a glow behind the
// beam. It is done here rather than while rendering because View has to stay a
// pure function of the model, and a trail is state.
func (m *Model) rememberScope() {
	l := m.layout()
	w := l.interior - leftMargin - rightMargin
	if w <= 0 {
		return
	}
	grid, _ := m.scopeGrid(w, m.scopeTrigger(w*dotsPerCellX))
	m.scope.remember(grid)
}
