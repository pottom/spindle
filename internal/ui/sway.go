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
	wordsSwayMost = 0.31

	// How hard the low end is hitting decides how far the row sways, and these
	// are the numbers that read it.
	//
	// A beat is found over twelve seconds of listening, so it goes on being
	// reported through a passage where nobody is playing it — the drums drop out
	// for a bar and the row goes on swaying to a beat that is not being struck.
	// What tells the difference is not how loud the bass is, because the daemon
	// scales the spectrum to its own recent loudness and the low end reads 0.6
	// to 0.8 either way. It is how hard the low end *jumps*: measured over ninety
	// seconds of a record, the biggest rise in a second ran 0.30 to 0.37 where
	// the kick was playing and 0.09 to 0.12 where it was not.
	//
	// swayLow is how many of the bands count as the low end — the same five the
	// analyser itself gives the bass. swayFall is what the recent hitting keeps
	// each frame, about a third of a second to fall by half, so it survives
	// between two beats at any tempo worth swaying to. swaySettle is the same for
	// the hardest it has hit lately, which is what the hitting is measured
	// against — long, because it is the record's own scale. swayLeast is the
	// floor under that scale, so a record with no percussion at all does not
	// measure its own hush and call it a beat. swayGain brings what is left up
	// to something a row can dance to.
	//
	// Swept over the recordings rather than chosen: at these numbers the quiet
	// stretch of a phonk record measured 0.35 and the rest of it 0.83.
	swayLow    = 5
	swayFall   = 0.95
	swaySettle = 0.9995
	swayLeast  = 0.10
	swayGain   = 2.0
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

	// Where the sway has got to, counted in beats and carried through the one
	// being played: a whole number on every beat, and everything between them in
	// between. Nothing here jumps at a beat — the cosine is already where it
	// needs to be when the count moves on.
	at, ok := m.swayAt()
	if !ok {
		return 0, false
	}
	return wordsSwayMost * m.words.drive * float32(math.Cos(math.Pi*at)), true
}

// How a bar of marks sways: the same beat, and not the same way twice running.
//
// One way is right some of the time and wrong the rest, which is the argument
// for dealing it rather than choosing it. A row all leaning together is a crowd;
// a row leaning at itself is a conversation; a row where every other one goes
// the other way is a zigzag that turns over on each beat; and a lean that
// arrives a little later the further along the row it is rolls, which a body
// does and a rank does not.
//
// Nothing here is a mark being singled out — that was tried, and a lone thing
// moving on a still row reads as a mechanism. Whichever of these is dealt,
// every mark in the row is leaning.
type swayFigure int

const (
	swayTogether   swayFigure = iota // all of them the same way
	swayFacing                       // the two halves at each other
	swayAlternating                  // every other one the other way
	swayTrailing                     // later the further along the row
	swayFigures
)

// swayTrail is how much of a beat further behind each mark is than the one
// before it, when the row is dealt the rolling one.
//
// Small on purpose. Across eight marks this puts a third of a beat between the
// two ends — enough that the lean visibly travels, little enough that the row is
// still one body leaning rather than a queue of separate things.
const swayTrail = 0.045

// swayFor is the figure a bar of marks was dealt, mixed the way a visiting
// figure is: the same bar sways the same way twice, and nobody can call it.
func swayFor(starts int64) swayFigure {
	h := uint64(starts)*0x94d049bb133111eb + 0x9e3779b97f4a7c15
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	return swayFigure(h % uint64(swayFigures))
}

// wordsSwaying is how far each mark of the row leans this frame.
func (m Model) wordsSwaying(count int) ([]float32, bool) {
	lean, ok := m.wordsSway()
	if !ok || count <= 0 {
		return nil, false
	}

	out := make([]float32, count)
	fig := swayFor(m.words.starts)

	// The rolling one is not a scaling of the row's lean but a lean of its own
	// per mark, because what makes it roll is when each of them gets there.
	if fig == swayTrailing {
		at, ok := m.swayAt()
		if !ok {
			return nil, false
		}
		for i := range out {
			out[i] = wordsSwayMost * m.words.drive *
				float32(math.Cos(math.Pi*(at-float64(i)*swayTrail)))
		}
		return out, true
	}

	middle := float32(count-1) / 2
	for i := range out {
		switch fig {
		case swayFacing:
			// Nought in the middle of the row and one at either end, so the two
			// halves lean at each other and the mark between them barely moves.
			if middle > 0 {
				out[i] = lean * (middle - float32(i)) / middle
			}
		case swayAlternating:
			if i%2 == 1 {
				out[i] = -lean
				continue
			}
			out[i] = lean
		default:
			out[i] = lean
		}
	}
	return out, true
}

// swayAt is where the sway has got to, counted in beats and carried through the
// one being played.
func (m Model) swayAt() (float64, bool) {
	age := max(time.Duration(m.wordsClock()-m.words.starts)*time.Millisecond, 0)
	beats, ok := m.beatsIn(age)
	if !ok {
		return 0, false
	}
	phase, ok := m.beatPhase()
	if !ok {
		return 0, false
	}
	return float64(beats) + float64(phase), true
}

// swayFlow follows how hard the low end is hitting, a frame at a time, which is
// how far the row is allowed to sway. See the constants above.
func (m *Model) swayFlow() {
	bands := m.scope.bands
	if len(bands) < swayLow {
		m.words.drive, m.words.swayHit, m.words.swayCeil = 0, 0, 0
		m.words.swayHeard = false
		return
	}

	var low float32
	for _, v := range bands[:swayLow] {
		low += v
	}
	low /= swayLow

	// The first frame is not a hit, however loud it is. Without this the low end
	// rising from nothing to whatever it is reads as the hardest strike of the
	// record — and since what the strikes are measured against is the hardest
	// there has been, one frame of arithmetic held the sway down for the forty
	// seconds that scale takes to fall.
	if !m.words.swayHeard {
		m.words.swayHeard, m.words.swayWas = true, low
		return
	}

	hit := max(low-m.words.swayWas, 0)
	m.words.swayWas = low

	m.words.swayHit = max(hit, m.words.swayHit*swayFall)
	m.words.swayCeil = max(hit, m.words.swayCeil*swaySettle)

	m.words.drive = min(m.words.swayHit/max(m.words.swayCeil, swayLeast)*swayGain, 1)
}
