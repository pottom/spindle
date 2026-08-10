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
		fig, ok := crewAt(m, beat, 0).crewNow()
		if !ok {
			t.Fatalf("beat %d: the row was not performing at all", beat)
		}
		figs = append(figs, fig)
		fmt.Fprintf(&sb, "%d ", fig)
	}
	t.Logf("over %d beats the row danced %s(0 is unison, 1 alternating)", len(figs), sb.String())

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
