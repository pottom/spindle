package ui

import (
	"math"
	"strings"
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
func (m Model) scopeLines(w int) []string {
	if w <= 0 {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, scopeRows*dotsPerCellY
	grid := make([]uint8, w*scopeRows)

	prev := -1
	for x := range dotsX {
		y := m.scopeSample(x, dotsX)
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

	lines := make([]string, scopeRows)
	for r := range scopeRows {
		var sb strings.Builder
		// A run of blank cells carries no style, so the rows compress to about
		// as much output as a line of text.
		style := m.styles.ScopeNear
		if r == 0 || r == scopeRows-1 {
			style = m.styles.ScopeFar
		}

		var run strings.Builder
		flush := func() {
			if run.Len() > 0 {
				sb.WriteString(style.Render(run.String()))
				run.Reset()
			}
		}
		for c := range w {
			bits := grid[r*w+c]
			if bits == 0 {
				flush()
				sb.WriteByte(' ')
				continue
			}
			run.WriteRune(rune(brailleBase + int(bits)))
		}
		flush()
		lines[r] = fit(sb.String(), w)
	}
	return lines
}

// scopeSample reads the waveform at one horizontal dot, resampling whatever the
// daemon sent to however wide the terminal is. With no frame at all the trace
// rests on the centre line, which is what silence looks like.
func (m Model) scopeSample(x, dots int) float32 {
	if len(m.scope.frame) == 0 || dots <= 0 || m.scope.envelope <= 0 {
		return 0
	}
	i := x * len(m.scope.frame) / dots
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
