package ui

import (
	"strings"
	"testing"
)

// ladderModel is a spectrum with a shape to draw, on the ladder.
func ladderModel(w, h int) Model {
	m := scopeModel(w, h)
	m.scope.modes[tabPlayer] = scopeLadder

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.9
	}
	m.scope.adoptBands(bands)
	return m
}

// Lamps, not bars: every lit cell is half a block, so there is air under each
// one. A meter drawn in whole blocks is a filled shape, and a filled shape is
// the one thing this picture is not.
func TestLadderIsLampsWithAirBetween(t *testing.T) {
	m := ladderModel(100, 44)

	var lamps int
	for r, line := range m.ladderLines(96, 20) {
		for _, ch := range ansiOff(line) {
			switch ch {
			case ladderTop, ladderBottom:
				lamps++
			case ' ':
			default:
				t.Fatalf("row %d is drawn with %q, want lamps and gaps", r, string(ch))
			}
		}
	}
	if lamps == 0 {
		t.Error("nothing was lit")
	}
}

// The colour climbs: the foot of a tall bar and its top are different colours,
// because on this picture the colour is the reading.
func TestLadderClimbsThroughItsColours(t *testing.T) {
	m := ladderModel(100, 44)

	lines := m.ladderLines(96, 20)
	foot, top := "", ""
	for i, line := range lines {
		if strings.TrimSpace(ansiOff(line)) == "" {
			continue
		}
		if top == "" {
			top = line
		}
		foot = lines[len(lines)-1-i]
		break
	}
	if foot == top || foot == "" {
		t.Fatalf("could not find both ends of a bar: %q / %q", top, foot)
	}
	if colourUsed(foot) == colourUsed(top) {
		t.Errorf("the foot and the top of the ladder are both %q, want the colour to climb", colourUsed(foot))
	}
}

// The marker floats: after a bar falls, what it reached is still shown, one
// lamp on its own above the stack.
func TestLadderFloatsTheMarker(t *testing.T) {
	m := ladderModel(100, 44)

	quiet := make([]float32, 28)
	for i := range quiet {
		quiet[i] = 0.2
	}
	m.scope.adoptBands(quiet)

	lines := m.ladderLines(96, 20)

	var lit []int
	for r, line := range lines {
		if strings.TrimSpace(ansiOff(line)) != "" {
			lit = append(lit, r)
		}
	}
	if len(lit) < 2 {
		t.Fatalf("the picture lit %d rows, want a stack and a marker over it", len(lit))
	}
	if lit[1]-lit[0] < 2 {
		t.Errorf("the marker sits at row %d and the stack starts at %d, want air between them", lit[0], lit[1])
	}
}

// Whatever the size, it fills what it is given.
func TestLadderFillsWhatItIsGiven(t *testing.T) {
	m := ladderModel(100, 44)
	for _, size := range [][2]int{{40, 4}, {96, 20}, {200, 50}} {
		lines := m.ladderLines(size[0], size[1])
		if len(lines) != size[1] {
			t.Fatalf("%dx%d: drew %d rows", size[0], size[1], len(lines))
		}
		for i, line := range lines {
			if got := len([]rune(ansiOff(line))); got != size[0] {
				t.Errorf("%dx%d: row %d is %d cells wide", size[0], size[1], i, got)
			}
		}
	}
}

// In the strip under the artwork there are four rows, and a lamp with air under
// it would leave four rungs to show a whole spectrum in — four dashes moving
// about rather than a meter. There the rungs are packed two to a cell instead,
// each half of it a lamp of its own colour, which is twice the reading in the
// same space.
func TestLadderPacksTheRungsWhenItIsShallow(t *testing.T) {
	m := ladderModel(100, 44)

	rungs := func(rows int) int {
		quiet := make([]float32, 28)
		for i := range quiet {
			quiet[i] = 0.5
		}
		m.scope.adoptBands(quiet)

		// How many distinct heights the picture can tell apart: step the level
		// up and count how many different pictures come out.
		seen := map[string]bool{}
		for level := 0.0; level < 1; level += 0.02 {
			for i := range quiet {
				quiet[i] = float32(level)
			}
			m.scope.bands = quiet
			seen[strings.Join(m.ladderLines(96, rows), "\n")] = true
		}
		return len(seen)
	}

	shallow, tall := rungs(4), rungs(20)
	t.Logf("the strip tells %d heights apart, the screen %d", shallow, tall)

	if shallow < 8 {
		t.Errorf("the strip can only draw %d different heights, want the rungs packed", shallow)
	}
	if tall <= shallow {
		t.Errorf("a screen of 20 rows draws %d heights and a strip of 4 draws %d", tall, shallow)
	}
}

// The gaps are the look, and they survive the strip: where there is no room to
// leave them empty they are drawn dark instead, so the bar is still a stack of
// lamps rather than a stripe of colour.
func TestLadderKeepsItsGapsWhenItIsShallow(t *testing.T) {
	m := ladderModel(100, 44)

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.95
	}
	m.scope.adoptBands(bands)

	// A tall bar in the strip: every cell of it has to carry two different
	// colours, the lit rung and the one with its light out.
	var pairs int
	for _, line := range m.ladderLines(96, 4) {
		for _, run := range strings.Split(line, "\x1b[m") {
			// A cell drawn with both a foreground and a background is a lit
			// rung over an unlit one, or the other way about.
			if strings.Contains(run, "38;2;") && strings.Contains(run, "48;2;") {
				pairs++
			}
		}
	}
	if pairs == 0 {
		t.Error("no cell in the strip carries a lamp and a gap, so the bars are one flat colour")
	}

	// And on a screen the gaps are left empty, as they were.
	var rows int
	for _, line := range m.ladderLines(96, 20) {
		if strings.TrimSpace(ansiOff(line)) == "" {
			rows++
		}
	}
	if rows > 0 {
		t.Logf("%d rows are empty on the tall picture", rows)
	}
	for _, line := range m.ladderLines(96, 20) {
		if strings.Contains(line, "48;2;") {
			t.Error("the tall ladder painted a background, so its gaps are no longer empty")
			break
		}
	}
}
