package ui

import (
	"math"
	"time"
)

// The row of marks, keeping time by leaning rather than by jumping.
//
// The marks have always ridden their own part of the spectrum: the bass at the
// left, the cymbals at the right, each rising by what its share of the sound is
// doing. When the beat arrived, the obvious thing was to put that rise on it —
// the whole row leaving the ground together on every beat.
//
// It was the wrong thing, and the key that turns keeping time off is what
// proved it: watched against each other on the same record, the row that simply
// answered the sound looked better than the row that landed on the beat. Held
// to the beat, the height of a mark is answering two things at once — how loud
// its band is *and* where the beat is — and neither reads, so what is left is a
// row twitching.
//
// So the two are given a dimension each. Up and down is the sound's, all of it,
// as it always was. The beat gets the lean: the row sways from one side to the
// other, all the way over on each beat, so what keeps time is a movement of its
// own rather than a share of somebody else's. Two things moving in two
// directions read as one body doing two things; the same two fighting over one
// direction read as noise.

const (
	// wordsSwayMost is how far the row leans at the end of a sway, sideways over
	// its own height. A shade under what a mark leans when it is dealt a lean of
	// its own: this one is happening all the time, and a row permanently at the
	// angle a type designer calls italic is a row that has fallen over.
	wordsSwayMost = 0.16
)

// wordsSway is how far the row is leaning this frame, and whether it is swaying
// at all — which it is not with no beat to keep, with the key turned off, or
// when what is on screen is a line of words rather than a bar of marks.
//
// One sway takes two beats: all the way over on one, all the way back on the
// next. A whole swing on every beat was tried first and it is too quick to read
// as a body — a crowd swaying goes over and back over a bar, not twice a beat.
// Taken from a cosine of the beat rather than from the pulse everything else
// here uses, because a sway has no strike in it: it is at its furthest exactly
// on the beat and moving fastest between two, which is what a pendulum does and
// what a hit does not.
func (m Model) wordsSway() (float32, bool) {
	if !m.words.beats || m.words.telling {
		return 0, false
	}

	// How long the bar has been up, on the clock everything else on this screen
	// reads. The picture is asked for a little before the bar sounds, so the age
	// can still be negative when the row first goes up.
	age := max(time.Duration(m.wordsClock()-m.words.starts)*time.Millisecond, 0)

	beats, ok := m.beatsIn(age)
	if !ok {
		return 0, false
	}
	phase, ok := m.beatPhase()
	if !ok {
		return 0, false
	}

	// Where the sway has got to, counted in beats and carried through the one
	// being played: a whole number on every beat, and everything between them in
	// between. Nothing here jumps at a beat — the cosine is already where it
	// needs to be when the count moves on.
	at := float64(beats) + float64(phase)
	return wordsSwayMost * float32(math.Cos(math.Pi*at)), true
}
