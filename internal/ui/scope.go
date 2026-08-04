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

	// scopeMinHeight is the terminal height below which the trace is not
	// offered at all: the player itself needs the rows more.
	scopeMinHeight = 28

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
}

// available reports whether the trace can be offered at all. A short terminal
// has better uses for five rows.
func (m Model) scopeAvailable() bool {
	return m.tab == tabPlayer && !m.noDevice && m.height >= scopeMinHeight
}

// scopeVisible reports whether the trace is on screen right now.
func (m Model) scopeVisible() bool {
	return m.scope.on && m.scopeAvailable()
}

// scopeHeight is the rows the trace occupies, blank separator included.
func (m Model) scopeHeight() int {
	if !m.scopeVisible() {
		return 0
	}
	return scopeRows + scopeChrome
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
		dy := int(math.Round((0.5 - float64(y)*0.46) * float64(dotsY-1)))
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
	if len(m.scope.frame) == 0 || dots <= 0 {
		return 0
	}
	i := x * len(m.scope.frame) / dots
	return m.scope.frame[min(i, len(m.scope.frame)-1)]
}

// scopeBlock is the trace with its blank separator, padded to the frame width.
func (m Model) scopeBlock(l layout) []string {
	w := l.interior - leftMargin - rightMargin
	out := make([]string, 0, scopeRows+scopeChrome)
	out = append(out, m.pad("", l))
	for _, line := range m.scopeLines(w) {
		out = append(out, m.pad(line, l))
	}
	return out
}

// nextScopeFrame asks the backend for the latest waveform. A backend with no
// samples to give leaves the trace resting on the centre line, which is what
// silence looks like — no error, nothing to explain.
func (m Model) nextScopeFrame() []float32 {
	source, ok := m.player.(player.Waveform)
	if !ok {
		return nil
	}
	return source.Waveform()
}
