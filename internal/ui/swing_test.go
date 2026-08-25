package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// swungStage is the big screen with a bar of marks up on it, which is what the
// step moves.
func swungStage(t *testing.T, step int) Model {
	t.Helper()

	m := stageWords("a")
	m.width, m.height = 100, 30
	m.resize()
	m.stage.swing = step
	m.setProgress(40 * time.Second)
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	if !m.words.beats {
		t.Fatalf("the picture up is %q, wanted a bar of marks", m.words.text)
	}
	return m
}

// swingRide is how far the marks are thrown either side of where they were set,
// in dots, over a sweep of the spectrum.
func swingRide(t *testing.T, step int, loud float64) (low, high int) {
	t.Helper()

	m := swungStage(t, step)
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

// The lean is not the throw. A shear that grows with the height stops looking
// like a row leaning into the beat and starts looking like a row falling over —
// reported from a real screen at the throw's own figures.
func TestTheLeanIsGivenLessThanTheThrow(t *testing.T) {
	for step := 2; step <= swingSteps; step++ {
		m := swungStage(t, step)

		lean, throw := m.stageLean(), m.stageSwing()
		if lean >= throw {
			t.Errorf("step %d leans by %v against a throw of %v", step, lean, throw)
		}
		if lean <= 1 {
			t.Errorf("step %d does not lean any further than the first step", step)
		}
	}

	// And the first step is neither: it is the picture as it was.
	m := swungStage(t, 1)
	if got := m.stageLean(); got != 1 {
		t.Errorf("the first step leans at %v, want the picture unchanged", got)
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

// The words are left out of it. A lyric is there to be read, and it was already
// given far less movement than a row of marks for exactly that reason: a line of
// a dozen words cannot throw itself about the way seven marks can without coming
// apart. Reported from a real screen — the words were right as they were.
func TestTheStepLeavesTheWordsAlone(t *testing.T) {
	for step := 1; step <= swingSteps; step++ {
		m := stageWords("a")
		m.width, m.height = 100, 30
		m.resize()
		m.stage.swing = step
		m.lyrics.forTrack, m.lyrics.missing = "a", false
		m.setProgress(40 * time.Second)
		if cmd := m.wordsGrind(); cmd != nil {
			m.wordsTake(cmd)
		}

		// Whatever is up, the moment it is not a bar of marks the step is not
		// in it.
		m.words.beats = false
		if got := m.stageSwing(); got != 1 {
			t.Errorf("step %d threw a line of words at %v", step, got)
		}
		if got := m.stageLean(); got != 1 {
			t.Errorf("step %d leaned a line of words at %v", step, got)
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
