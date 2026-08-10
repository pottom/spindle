package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// crewModel is a big screen showing a row of marks over a record with a beat,
// with every part of the spectrum going equally hard — so that what the row
// does is the figure and nothing else.
func crewModel(t *testing.T) Model {
	t.Helper()

	m := scopeModel(120, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.words.beats, m.words.text = true, wordsNotes

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.8
	}
	m.scope.bands = bands

	m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
	m.scope.beatAt = time.Now()
	if !m.beatKeeping() {
		t.Fatal("a beat was given and the screen did not keep time")
	}
	return m
}

// crewAt puts the row a given number of beats into its bar and a given part of
// the way through the beat it is on. Whichever bar the caller chose is left
// alone: which bar it is decides what the row was dealt.
func crewAt(m Model, beats int, phase float64) Model {
	period := m.scope.beat.Period
	m.scope.beat.Since = time.Duration(phase * float64(period))
	m.scope.beatAt = time.Now()
	m.setProgress(time.Duration(m.words.starts)*time.Millisecond + time.Duration(beats)*period)
	return m
}

// crewBarDealt finds a bar whose first phrase was dealt a given figure. A test
// cannot order a figure any more than a listener can — they are dealt from the
// bar — so it goes looking for one.
func crewBarDealt(fig crewFigure) int64 {
	for starts := int64(0); starts < 10_000; starts += 10 {
		if crewFor(starts, 0) == fig {
			return starts
		}
	}
	return -1
}

// The row see-saws: half of it lands on the beat and half between two, so what
// is on the screen is a group doing something with each other rather than a row
// of meters agreeing.
func TestTheRowSeeSaws(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewAlternate)
	if m.words.starts < 0 {
		t.Fatal("no bar was dealt the alternating figure")
	}

	at := func(phase float64) []int { return crewAt(m, 0, phase).wordsRiding(6) }

	on, off := at(0), at(0.5)
	t.Logf("on the beat the row stands at %v, half a beat later at %v", on, off)

	// Up is a smaller number, so the ones that landed are the lower ones.
	for i := range on {
		if i%2 == 0 && on[i] >= off[i] {
			t.Errorf("mark %d is at %d on the beat and %d off it, want it up on the beat", i, on[i], off[i])
		}
		if i%2 == 1 && off[i] >= on[i] {
			t.Errorf("mark %d is at %d off the beat and %d on it, want it up off the beat", i, off[i], on[i])
		}
	}

	// And nobody stands still: what makes this a figure rather than half a row
	// switched off is that both halves are moving, at different moments.
	for i := range on {
		if on[i] == 0 && off[i] == 0 {
			t.Errorf("mark %d never left the ground", i)
		}
	}
}

// A figure is held for a phrase and then another is dealt: the row performs a
// number and then a different one, rather than changing every beat, which
// would be a twitch rather than a figure.
func TestTheFigureIsHeldForAPhrase(t *testing.T) {
	m := crewModel(t)

	// A bar that was dealt two different figures over its first two phrases, so
	// that holding one and changing to the other are both being watched.
	for starts := int64(0); starts < 10_000; starts += 10 {
		if crewFor(starts, 0) != crewFor(starts, 1) {
			m.words.starts = starts
			break
		}
	}
	t.Logf("the bar at %dms was dealt figures that change between its phrases", m.words.starts)

	var figs []crewFigure
	var sb strings.Builder
	for beat := range 3 * crewPhrase {
		c, ok := crewAt(m, beat, 0).crewNow()
		if !ok {
			t.Fatalf("beat %d: the row was not performing at all", beat)
		}
		figs = append(figs, c.fig)
		fmt.Fprintf(&sb, "%d ", c.fig)
	}
	t.Logf("over %d beats the row danced %s", len(figs), sb.String())

	for beat, fig := range figs {
		if want := figs[beat/crewPhrase*crewPhrase]; fig != want {
			t.Errorf("beat %d is dancing figure %d, want %d — the phrase it belongs to", beat, fig, want)
		}
	}
}

// The figures are dealt from the bar: a record shows both of them, a listener
// cannot call the next one, and the same bar twice dances the same.
func TestTheFiguresAreDealtFromTheBar(t *testing.T) {
	seen := map[crewFigure]int{}
	for starts := int64(0); starts < 2000; starts += 10 {
		for phrase := range 4 {
			seen[crewFor(starts, phrase)]++
		}
	}
	t.Logf("over 200 bars of four phrases the figures came up %v", seen)

	for fig := crewUnison; fig < crewFigures; fig++ {
		if seen[fig] == 0 {
			t.Errorf("figure %d was never dealt", fig)
		}
	}

	// And the deal is a deal, not a coin toss.
	for starts := int64(0); starts < 500; starts += 70 {
		for phrase := range 3 {
			if a, b := crewFor(starts, phrase), crewFor(starts, phrase); a != b {
				t.Fatalf("the bar at %dms danced %d and then %d", starts, a, b)
			}
		}
	}
}

// The two halves of the row talk to each other: the left lands on the beat and
// the right answers between two. Where the see-saw ripples along the row, this
// splits it in two.
func TestTheTwoHalvesOfTheRowTalk(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewCall)
	if m.words.starts < 0 {
		t.Fatal("no bar was dealt the call and its answer")
	}

	on, off := crewAt(m, 0, 0).wordsRiding(6), crewAt(m, 0, 0.5).wordsRiding(6)
	t.Logf("on the beat the row stands at %v, half a beat later at %v", on, off)

	for i := range on {
		if i < 3 && on[i] >= off[i] {
			t.Errorf("mark %d of the left half is at %d on the beat and %d off it, want it up on the beat", i, on[i], off[i])
		}
		if i >= 3 && off[i] >= on[i] {
			t.Errorf("mark %d of the right half is at %d off the beat and %d on it, want it answering", i, off[i], on[i])
		}
	}
}

// The bow: the whole row sinks together on the first beat of the phrase, which
// is the one movement on this screen that goes downwards — and then it dances
// the rest of the phrase.
func TestTheRowBows(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewBow)
	if m.words.starts < 0 {
		t.Fatal("no bar was dealt the bow")
	}

	bowing := crewAt(m, 0, 0).wordsRiding(6)
	after := crewAt(m, 1, 0).wordsRiding(6)
	t.Logf("on the first beat of the phrase the row stands at %v, on the next at %v", bowing, after)

	// Up is a smaller number, so a row that has gone down stands at a larger
	// one. The dip is laid over the dance rather than replacing it — a group
	// that stops moving to bow has not bowed, it has sat down — so what is
	// asked of it is that the row is markedly lower, not that it is below the
	// line it dances on.
	for i := range bowing {
		if bowing[i] <= after[i] {
			t.Errorf("mark %d stands at %d bowing and %d a beat later, want it lower while it bows", i, bowing[i], after[i])
		}
	}

	// And it is one movement: the whole row goes down by the same amount.
	for i, at := range bowing {
		if at != bowing[0] {
			t.Errorf("mark %d bows to %d and mark 0 to %d, want the row bowing as one", i, at, bowing[0])
		}
	}
}

// The huddle leans the row instead of lifting it: the two halves lean at each
// other on the beat and straighten between two. A body leaning is a different
// thing from a body jumping, and every other figure jumps.
func TestTheRowHuddles(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewHuddle)
	if m.words.starts < 0 {
		t.Fatal("no bar was dealt the huddle")
	}

	// The layout the marks were set with, so the lean has words to hang on.
	const w, rows = 120, 44
	line := wordsMarks(w*dotsPerCellX, rows*dotsPerCellY)
	_, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatalf("the row %q could not be drawn", line)
	}
	m.words.where, m.words.text = layout, line
	count := layout.Count

	on, _ := crewAt(m, 0, 0).wordsTilting(count)
	between, _ := crewAt(m, 0, 0.5).wordsTilting(count)
	if on == nil {
		t.Fatal("the row huddled and nothing leaned")
	}
	t.Logf("%d marks lean %v on the beat and %v between two", count, on, between)

	if on[0]*on[count-1] >= 0 {
		t.Errorf("the ends lean %v and %v, want them leaning at each other", on[0], on[count-1])
	}
	for i := range on {
		if abs32(between[i]) > abs32(on[i]) {
			t.Errorf("mark %d leans %v between two beats and %v on one, want it straightening between", i, between[i], on[i])
		}
	}
}

// A phrase of nobody keeping time, which is what makes the next figure arriving
// an event. It is the same picture the row has with the key off, so it is worth
// nothing new to be sure it looks right.
func TestTheRowIsLetOffForAPhrase(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewFree)
	if m.words.starts < 0 {
		t.Fatal("no bar was dealt a free phrase")
	}

	free := crewAt(m, 0, 0.5).wordsRiding(6)

	// Nothing times it: over a whole beat the row stands where it stood at the
	// start of it, which is what none of the other figures do.
	for _, phase := range []float64{0, 0.25, 0.75} {
		if at := crewAt(m, 0, phase).wordsRiding(6); at[0] != free[0] {
			t.Errorf("a quarter of the way through the beat the free row moved from %d to %d", free[0], at[0])
		}
	}

	// And it stands at the height the timed figures average out at, rather than
	// at the top of the travel — which is what used to give a change of figure
	// away. See crewLevel.
	timed := crewAt(withBar(m, crewBarDealt(crewUnison)), 0, 0.5).wordsRiding(6)
	t.Logf("free the row stands at %v, and a timed figure halfway through its beat at %v", free, timed)
	if free[0] > timed[0] {
		t.Errorf("free the row stands at %d and a timed one at %d halfway through a beat, want them near each other", free[0], timed[0])
	}
}

// withBar is the same model over a different bar, which is what decides what it
// was dealt.
func withBar(m Model, starts int64) Model {
	m.words.starts = starts
	return m
}

// Every figure stands the row at the same height.
//
// This is the second complaint that was made of it in the room, and it is the
// sharper of the two: over one figure the marks were let a long way up and down
// and over the next only a little, so what you could see was not a group
// changing what it was doing but a program being changed. The movement is the
// figure's to decide; how high the row stands is not.
func TestEveryFigureStandsAsTall(t *testing.T) {
	m := crewModel(t)
	period := m.scope.beat.Period

	// Averaged over two beats, sampled as the screen draws them, with every
	// band going equally hard so that what is left is the figure.
	height := func(fig crewFigure) float32 {
		var sum float32
		var n int
		for f := range 60 {
			into := time.Duration(f) * (33 * time.Millisecond)
			m.scope.beat.Since = into % period
			m.scope.beatAt = time.Now()

			for _, at := range m.crewCalls(6, crew{fig: fig, beat: 1}) {
				sum += at
				n++
			}
		}
		return sum / float32(n)
	}

	tallest, shortest := height(crewUnison), height(crewUnison)
	for fig := crewUnison; fig < crewFigures; fig++ {
		at := height(fig)
		t.Logf("dancing figure %d the row stands %.2f dots up on average", fig, at)
		tallest, shortest = max(tallest, at), min(shortest, at)
	}

	// Within a tenth of each other. Not identical: the bow spends one beat of
	// its phrase going down on purpose, and a figure that could not do anything
	// of its own would not be a figure.
	t.Logf("the tallest figure stands %.2f and the shortest %.2f", tallest, shortest)
	if tallest > shortest*1.1 {
		t.Errorf("one figure stands the row at %.2f dots and another at %.2f, want them level", tallest, shortest)
	}
}

// The row moves like bodies rather than like a meter.
//
// This is the complaint that was made of it in the room, and it is measurable:
// what a picture looks like frame by frame is how far anything on it moves
// between two frames. Standing exactly where the arithmetic put it, a mark went
// from the floor to the top of its jump in a single frame, and the whole row
// shook rather than danced — the spectrum it rides moves thirty times a second
// and every wobble in it went straight onto the screen.
//
// So the row is measured here as it is watched: thirty frames a second of real
// music, through a change of figure, counting the biggest step any one mark
// takes between two frames.
func TestTheRowMovesLikeBodies(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewUnison)
	m.words.where.Count = 6

	const frames = 90 // three seconds
	period := m.scope.beat.Period

	var worst, called int
	was, wasCalled := make([]int, 6), make([]int, 6)
	for f := range frames {
		// A spectrum that shakes the way a real one does, so what is being
		// measured is whether the row passes that shake on.
		for i := range m.scope.bands {
			m.scope.bands[i] = 0.5 + 0.4*float32(((f*7+i*13)%11))/10
		}

		into := time.Duration(f) * (33 * time.Millisecond)
		m.scope.beat.Since = into % period
		m.scope.beatAt = time.Now()
		m.setProgress(time.Duration(m.words.starts)*time.Millisecond + into)

		// Where the figure is calling the row this frame, which is where every
		// mark used to stand outright, and where it has actually got to.
		c, ok := m.crewNow()
		if !ok {
			t.Fatalf("frame %d: the row was not performing", f)
		}
		calls := m.crewCalls(6, c)

		m.crewFlow(6)
		now := m.wordsRiding(6)

		if f > 0 {
			for i := range now {
				if step := abs(now[i] - was[i]); step > worst {
					worst = step
				}
				if step := abs(int(calls[i]) - wasCalled[i]); step > called {
					called = step
				}
			}
		}
		copy(was, now)
		for i := range calls {
			wasCalled[i] = int(calls[i])
		}
	}

	t.Logf("over %d frames of a shaking spectrum the figure called for steps of up to %d dots between two frames; the row took at most %d",
		frames, called, worst)

	// A body moving across a screen at thirty frames a second covers a few dots
	// a frame. Anything much past that is a jump cut with nothing in between,
	// which is what the row used to do on every beat.
	if worst > 6 {
		t.Errorf("a mark moved %d dots in one frame, want it moving rather than jumping", worst)
	}
}

// With keeping time turned off the row is exactly what it was before any of
// this: every mark on its own part of the sound, nobody performing anything.
func TestTheRowStopsPerformingWhenTheKeyIsOff(t *testing.T) {
	m := crewModel(t)
	m.words.starts = crewBarDealt(crewAlternate)
	m = crewAt(m, 2, 0.5)

	if _, ok := m.crewNow(); !ok {
		t.Fatal("the row was not performing, so turning it off proves nothing")
	}
	performing := m.wordsRiding(6)

	m.stage.loose = false
	if m.beatKeeping() {
		t.Fatal("the key was turned off and the screen kept time anyway")
	}
	off := m.wordsRiding(6)

	want := make([]int, 6)
	for i := range want {
		want[i] = -int(m.wordsBeatRide(i, 6) * wordsBounce)
	}
	t.Logf("performing the row stands at %v, with the key off at %v", performing, off)

	for i := range want {
		if off[i] != want[i] {
			t.Errorf("with the key off mark %d stands at %d, want the %d it stood at before", i, off[i], want[i])
		}
	}
}
