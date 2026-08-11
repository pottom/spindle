package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// spillLine is a line of five words, each 16 dots wide, sitting 24 dots tall.
func spillLine() (Model, int, int) {
	const dotsX, top, tall, words = 96, 40, 24, 5
	at := make([]int16, dotsX)
	for x := range dotsX {
		if x%(dotsX/words) >= dotsX/words-3 {
			at[x] = -1
			continue
		}
		at[x] = int16(x * words / dotsX)
	}
	var m Model
	m.words.wasWhere = msg.WordLayout{
		Count: words, DotsX: dotsX, At: at,
		Tops: []int{top}, Bottoms: []int{top + tall - 1},
	}
	return m, top, tall
}

// spillAt is where the line has got to and how lit it is, part way through.
func spillAt(m Model, top, tall int, gone float32) (lo, hi, lit, burn int) {
	const dotsX, dotsY, levels = 96, 200, 32
	lo, hi = dotsY, -1
	for y := top; y < top+tall; y++ {
		for x := range dotsX {
			if m.words.wasWhere.WordAt(x, y) < 0 {
				continue
			}
			_, to, b, ok := m.wordsSpill(x, y, gone, levels)
			if !ok || to < 0 || to >= dotsY {
				continue
			}
			lo, hi = min(lo, to), max(hi, to)
			lit++
			burn += int(b)
		}
	}
	return lo, hi, lit, burn
}

// A line that lets go falls, and goes dark as it does.
//
// Both halves of that are the test, because the first attempt had each on its
// own and neither together. It faded as the square of the time, so the light was
// out before the screen's gravity had carried anything anywhere: measured, the
// line had sagged five dot rows — a fifth of its own height — by the time it was
// too faint to see. A fall nobody can see is not a fall.
func TestALineThatLetsGoFallsWhileItCanStillBeSeen(t *testing.T) {
	m, top, tall := spillLine()

	// Half way, it is on its way down and still lit enough to watch.
	_, half, _, halfBurn := spillAt(m, top, tall, 0.5)
	_, _, lit, startBurn := spillAt(m, top, tall, 0)
	if half <= top+tall {
		t.Errorf("half way through, the line has not left its place: bottom %d, was %d", half, top+tall-1)
	}
	if halfBurn*4 < startBurn {
		t.Errorf("half way through, only a quarter of the light is left: %d of %d", halfBurn, startBurn)
	}
	if lit == 0 {
		t.Fatal("nothing was drawn at all")
	}

	// And by the time the light is nearly gone it has fallen most of its own
	// height clear of where it stood, under the same pull the water and the
	// volume's lamps fall under. Measured, 22 rows of the 24 it is tall.
	_, deep, _, _ := spillAt(m, top, tall, 0.85)
	if fell := deep - (top + tall - 1); fell*4 < tall*3 {
		t.Errorf("by the end the line has fallen %d rows, less than three quarters of its %d height",
			fell, tall)
	}
}

// It goes out, rather than being cut off still lit.
func TestALineThatLetsGoIsDarkBeforeItIsDropped(t *testing.T) {
	m, top, tall := spillLine()
	_, _, _, startBurn := spillAt(m, top, tall, 0)
	_, _, _, endBurn := spillAt(m, top, tall, 0.97)
	if endBurn*8 > startBurn {
		t.Errorf("at the end it still carries %d of %d — it will vanish rather than fade", endBurn, startBurn)
	}
}

// It never brightens on the way out. A leaving picture that flares reads as
// something arriving; wordsPop flares on purpose and this must not.
func TestALineThatLetsGoOnlyEverDims(t *testing.T) {
	m, top, tall := spillLine()
	was := 1 << 30
	for step := range 20 {
		_, _, _, burn := spillAt(m, top, tall, float32(step)/20)
		if burn > was {
			t.Errorf("at %.2f the line was brighter than the frame before: %d after %d",
				float32(step)/20, burn, was)
		}
		was = burn
	}
}

// The fall is one way out among the ways out, not the new usual one.
func TestTheFallIsDealtAboutOneLineInEight(t *testing.T) {
	lines := []string{"jaj de jo", "minden reggel ugyanaz", "hazafele", "nem tudom", "ez az"}
	fell, all := 0, 0
	for _, text := range lines {
		for starts := int64(0); starts < 400; starts += 7 {
			all++
			if wordsSpillsNow(text, starts) {
				fell++
			}
		}
	}
	if share := float64(fell) / float64(all); share < 0.07 || share > 0.19 {
		t.Errorf("the fall came up %d times in %d (%.0f%%), want about one in eight", fell, all, share*100)
	}
	// And the same line at the same moment always goes the same way.
	for range 5 {
		if !wordsSpillsNow("jaj de jo", 42) == wordsSpillsNow("jaj de jo", 42) {
			t.Fatal("the deal is not the same twice")
		}
	}
}

// A line that falls is given longer to do it, and nothing else is.
func TestOnlyTheFallIsGivenLonger(t *testing.T) {
	if wordsLeavingFor(wordsSpilling) <= wordsLeavingFor(wordsWiping) {
		t.Error("the fall is not given longer than the others")
	}
	for _, move := range []wordsMove{wordsDrifting, wordsRising, wordsFalling, wordsBursting,
		wordsSpreading, wordsWiping, wordsWipingBack, wordsBlurring, wordsPopping} {
		if got := wordsLeavingFor(move); got != wordsLeaving {
			t.Errorf("move %d was given %v rather than the usual %v", move, got, wordsLeaving)
		}
	}
}

// Ink appears below where the line stood, on the screen rather than in the
// arithmetic.
//
// Worth drawing the frames for, because everything measured on wordsSpill alone
// can be right while nothing reaches the screen: when this was driven by hand
// from a key, the line came back whole and stood in front of the copy falling
// away from it, so all that showed was a little dust coming off something
// sitting still.
func TestTheLineIsSeenToFall(t *testing.T) {
	const w, rows = 90, 30

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	img, layout, ok := wordsImage([]string{"jaj de jo"}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the line")
	}
	grain := cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.have, m.words.was = grain, grain
	m.words.where, m.words.wasWhere = layout, layout
	m.words.cellsX, m.words.cellsY, m.words.text = w, rows, "jaj de jo"
	m.words.leave = wordsSpilling

	foot := layout.Bottoms[0] / dotsPerCellY
	// The line that is arriving is left finished and still, so that everything
	// that moves below the foot is the copy falling away and nothing else. Part
	// way through its gathering it is scattered over the whole picture, which is
	// ink below the line that has nothing to do with the fall.
	m.words.since = time.Now().Add(-5 * time.Second)

	below := func(at time.Duration) int {
		m.words.went = time.Now().Add(-at)
		n := 0
		for i, line := range m.wordsLines(w, rows) {
			if i > foot {
				n += len(strings.TrimSpace(ansiOff(line)))
			}
		}
		return n
	}

	// The meter's band sits under the type and is always there, so what is
	// measured is the growth over it rather than ink appearing from nothing.
	was := below(0)
	for _, at := range []time.Duration{100, 300, 600} {
		n := below(at * time.Millisecond)
		if n <= was {
			t.Errorf("%v after letting go, %d cells below the line — no more than the %d before",
				at*time.Millisecond, n, was)
		}
		was = n
	}
}
