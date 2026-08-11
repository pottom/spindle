package ui

import (
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

// The hatch drops the line and brings it back.
//
// It is not in the help and it is not meant to be, so this is the only thing
// that says it still works. A hatch nobody tests is a hatch that has quietly
// stopped working by the time it is next wanted.
func TestTheHatchLetsTheLineGo(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	img, layout, ok := wordsImage([]string{"jaj de jo"}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.text = w, rows, "jaj de jo"
	m.words.where = layout
	m.words.since = time.Now().Add(-5 * time.Second)

	m.wordsLetGo()

	if m.words.leave != wordsSpilling {
		t.Errorf("the line is leaving as %d rather than falling", m.words.leave)
	}
	if m.words.was.DotsX == 0 {
		t.Error("nothing was given notice, so there is nothing to fall")
	}
	if m.words.have.DotsX == 0 {
		t.Error("the line did not come back")
	}
	if time.Since(m.words.went) > time.Second {
		t.Error("the leaving did not start now")
	}

	// And with no line up it does nothing rather than falling over.
	var empty Model
	empty.wordsLetGo()
}
