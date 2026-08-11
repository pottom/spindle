package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// The same beat counted the other way is not a change of tempo.
//
// A beat finder cannot tell a beat from every other beat, and on one three
// minute record it flipped six times, calling 171 bpm 86 for a stretch and back
// again — measured from the running screen, 42 readings out of 337 at the wrong
// octave.
func TestABeatIsNotHalvedAndDoubledSixTimesAMinute(t *testing.T) {
	const fast = 350 * time.Millisecond // 171 bpm

	if got := beatSteady(fast, 2*fast); got != fast {
		t.Errorf("a reading at half the rate was taken: %s", got)
	}
	if got := beatSteady(2*fast, fast); got != 2*fast {
		t.Errorf("a reading at twice the rate was taken: %s", got)
	}
	// How it actually arrives: 700ms against a held 349.
	if got := beatSteady(349*time.Millisecond, 700*time.Millisecond); got != 349*time.Millisecond {
		t.Errorf("a near miss at half the rate was taken: %s", got)
	}

	// The wander inside one octave is the measurement doing its job — 348 to
	// 352 ms on that record — and is taken.
	for _, ms := range []int{348, 350, 352, 360} {
		fresh := time.Duration(ms) * time.Millisecond
		if got := beatSteady(fast, fresh); got != fresh {
			t.Errorf("a reading of %dms was refused, and it is the same beat", ms)
		}
	}
	// A real change of tempo is not a factor of two, and is taken.
	if got := beatSteady(fast, 500*time.Millisecond); got != 500*time.Millisecond {
		t.Errorf("a real change of tempo was refused: %s", got)
	}
	if got := beatSteady(0, fast); got != fast {
		t.Errorf("the first reading was refused: %s", got)
	}
}

// The count goes forwards and never jumps.
//
// Dividing the elapsed time by the newest period was the old way, and the
// newest period moves: the wander alone was worth six beats by the end of a
// three minute record, and an octave flip two hundred at once. A mark turns
// every two to six beats, so two hundred re-rolls every one of them in the same
// frame — which is what the row rearranging itself for no reason looked like.
func TestTheBeatCountOnlyGoesForwards(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.stage.loose = true
	m.ps = &player.State{TrackID: "one", Duration: 4 * time.Minute, Playing: true}

	// A period that wanders and then flips octave, the way the measured one did.
	periods := []time.Duration{350, 352, 348, 700, 351, 700, 349}
	var was int
	for _, p := range periods {
		m.scope.beat.Period, m.scope.beatAt = p*time.Millisecond, time.Now()
		// One beat's worth of frames: the phase climbing and wrapping once.
		for _, phase := range []float32{0.2, 0.5, 0.9, 0.05} {
			m.run.phase, m.run.forTrack = phase, "one"
			if phase < 0.1 {
				m.run.count++ // the wrap, as beatRunFlow finds it
			}
		}
		if m.run.count < was {
			t.Fatalf("the count went backwards, %d to %d", was, m.run.count)
		}
		if m.run.count-was > 1 {
			t.Errorf("one beat's worth of frames moved the count by %d", m.run.count-was)
		}
		was = m.run.count
	}
	t.Logf("seven periods including two octave flips: %d beats, one each", m.run.count)

	// And a new record starts at nought rather than carrying somebody else's.
	m.ps = &player.State{TrackID: "two", Duration: time.Minute, Playing: true}
	m.beatRunFlow()
	if m.run.count != 0 {
		t.Errorf("the next record started at %d beats", m.run.count)
	}
}

// The record shortens the turn: driven hard they turn nearly every beat.
//
// A count dealt once and held is a metronome however fast it is set. At a
// hundred and seventy the row was turning often and still reading as machinery,
// because each mark was turning on its own fixed number and nothing about that
// number was the music.
func TestTheDriveShortensTheTurn(t *testing.T) {
	const w, rows = 200, 44
	build := func(drive float32) []int {
		m := New(player.NewMock(), nil, defaultTestCell)
		m.width, m.height = w, rows
		m.stage.on, m.stage.loose = true, true
		m.ps = &player.State{TrackID: "one", Duration: 4 * time.Minute, Playing: true}
		m.words.beats, m.words.drive, m.words.leanAt = true, drive, 4242
		_, layout, ok := markPicture("band", w, rows, 4242)
		if !ok {
			t.Fatal("no row")
		}
		m.words.where = layout
		m.scope.beat.Period, m.scope.beatAt = 350*time.Millisecond, time.Now()
		m.run.forTrack = "one"

		// How many beats each mark goes without turning, read off the pattern.
		out := make([]int, layout.Count)
		for i := range out {
			var last, gaps, turns int
			was := false
			for beat := range 200 {
				m.run.count = beat
				got := m.wordsTurning(layout.Count)
				if len(got) <= i {
					continue
				}
				if got[i] != was {
					if turns > 0 {
						gaps += beat - last
					}
					turns, last, was = turns+1, beat, got[i]
				}
			}
			if turns > 1 {
				out[i] = gaps / (turns - 1)
			}
		}
		return out
	}

	lull, driven := build(0), build(1)
	t.Logf("at rest  the marks hold for %v beats", lull)
	t.Logf("driven   the marks hold for %v beats", driven)

	var sumL, sumD int
	for i := range lull {
		sumL += lull[i]
		sumD += driven[i]
	}
	if sumD >= sumL {
		t.Errorf("driving the record did not shorten the turn: %d beats against %d", sumD, sumL)
	}
	for i, n := range driven {
		if n > marksTurnLeast {
			t.Errorf("mark %d held for %d beats at full drive, and the floor is %d", i, n, marksTurnLeast)
		}
	}
}
