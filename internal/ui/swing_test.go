package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// swingRide is how far the marks are thrown either side of where they were set,
// in dots, over a sweep of the spectrum.
func swingRide(t *testing.T, step int, loud float64) (low, high int) {
	t.Helper()

	m := stageWords("a")
	m.width, m.height = 100, 30
	m.resize()
	m.stage.swing = step
	m.setProgress(40 * time.Second)
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	m.scope.beat.Loud = loud
	m.words.swellLow, m.words.swellHigh = -20, -8

	for f := range 12 {
		m.scope.bands = make([]float32, 8)
		for i := range m.scope.bands {
			m.scope.bands[i] = float32((f+i*3)%12) / 11
		}
		for _, v := range m.wordsRiding(7) {
			low, high = min(low, v), max(high, v)
		}
	}
	return low, high
}

// The screen answers the record in everything but one thing: how much movement
// is the right amount of movement. That is a judgement about a room and a
// screen, and the only way to make it is to see the same record at two sizes of
// it — so the big screen has three steps, and the first of them is exactly what
// it did before there was a key.
func TestTheBigScreenThrowsFurtherOnTheKey(t *testing.T) {
	var was int
	for _, step := range []int{1, 2, 3} {
		low, high := swingRide(t, step, -12)
		span := high - low
		if span <= was {
			t.Errorf("step %d throws the row %d dots, no further than the %d before it", step, span, was)
		}
		was = span
	}

	// And the first step is the picture as it was: whatever is being judged
	// against has to be the thing that was there.
	off := stageWords("a")
	off.width, off.height = 100, 30
	off.resize()
	off.stage.on = false
	if got := off.stageSwing(); got != 1 {
		t.Errorf("a screen that is not the big one swings at %v, want 1", got)
	}
	if got := swingAt[1]; got != 1 {
		t.Errorf("the first step is %v, want the picture unchanged", got)
	}
}

// However far the step asks for, the row stays on the screen. A mark is a third
// of the height on its own; thrown further than the room left over it is not a
// bigger movement, it is a missing one.
func TestTheRowIsNotThrownOffTheScreen(t *testing.T) {
	// At the loudest the record has been, which is where the travel is largest.
	low, high := swingRide(t, swingSteps, -8)

	m := stageWords("a")
	m.width, m.height = 100, 30
	m.resize()
	dots := m.height * dotsPerCellY

	if -low > dots/2 || high > dots/2 {
		t.Errorf("the row rode %d..%d dots of a screen %d tall", low, high, dots)
	}

	// And it is the room that stopped it rather than the step being small: at
	// the loudest, the top step asks for more than there is.
	loudest := wordsBounce * swellMost * swingAt[swingSteps]
	if float32(high-low) >= loudest {
		t.Errorf("the row was given all %v dots it asked for; nothing bounded it", loudest)
	}
}

// Shift and a digit is a different character on nearly every layout there is,
// and the one thing they agree on is the digit underneath.
func TestTheStepIsReadFromTheDigitNotTheCharacter(t *testing.T) {
	for step, text := range map[int]string{1: "'", 2: `"`, 3: "+"} {
		m := stageWords("a")
		m.stage.on = true

		// A Hungarian keyboard: shift and 1 is an apostrophe, and the base code
		// underneath it is the digit. See baseKey.
		k := tea.KeyPressMsg{Code: rune(text[0]), BaseCode: rune('0' + step), Mod: tea.ModShift, Text: text}
		if _, handled := m.stageKey(k); !handled {
			t.Fatalf("shift and %d was passed on rather than taken", step)
		}
		if m.stage.swing != step {
			t.Errorf("shift and %d (%q) set step %d", step, text, m.stage.swing)
		}
	}
}

// And it is the big screen's key and nowhere else's: the working screens keep
// their digits for the tabs.
func TestTheStepIsTheBigScreensAlone(t *testing.T) {
	m := stageWords("a")
	m.stage.on = false
	m.stage.swing = 2

	if got := m.stageSwing(); got != 1 {
		t.Errorf("a working screen swung at %v", got)
	}
	if _, handled := m.stageKey(tea.KeyPressMsg{Code: '2', Mod: tea.ModShift}); handled {
		t.Error("the key was taken on a screen that is not the big one")
	}
}
