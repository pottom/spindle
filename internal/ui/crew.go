package ui

import "time"

// The row of marks as a group, and what the group is doing.
//
// A bar nobody sings puts up a row of marks, and every one of them rides its own
// part of the sound — the bass at the left, the cymbals at the right — so the
// sound runs along the row. Keeping time, they all left the ground together.
// Together is a row of meters agreeing with each other; what makes a row of
// figures read as a group is that they are doing something *with* each other.
//
// So the row performs. What it performs is a figure held for a phrase and then
// changed for another, dealt from the bar the way a visiting figure is dealt —
// arbitrary to anybody watching, fixed to a test, and the same record twice.
//
// One thing this deliberately does not do: send anything travelling along the
// row. A single mark struck and walked from one end to the other was built and
// thrown out, and the reason is worth keeping written down — with one mark
// moving and the rest standing, the eye follows a dot and reads a mechanism.
// Everybody moves, all the time. The figure lives in who lands when, never in
// who is the only one moving.

const (
	// crewPhrase is how many beats a figure is held for. Four: a bar of music is
	// the shortest stretch that reads as a figure rather than as a twitch, and
	// long enough that the change to the next one is an event.
	crewPhrase = 4
)

// crewFigure is what the row is doing together.
type crewFigure int

const (
	// crewUnison is everybody on the beat, which is what the row did before
	// there were figures at all. It is in the deal rather than being the
	// absence of one: a group that never simply hits together has nothing for
	// the other figures to be a departure from.
	crewUnison crewFigure = iota

	// crewAlternate is every other mark on the beat and the rest on the
	// off-beat, so the row see-saws. The plainest figure there is and the one
	// that reads from across a room.
	crewAlternate

	crewFigures
)

// crewFor is the figure a phrase was dealt.
//
// From the bar and from which phrase of it this is, mixed the way faceWhoFor
// mixes: the same bar plays the same figures in the same order twice, and
// nobody watching can call the next one.
func crewFor(starts int64, phrase int) crewFigure {
	h := uint64(starts)*0x94d049bb133111eb + uint64(phrase+1)*0x9e3779b97f4a7c15
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	return crewFigure(h % uint64(crewFigures))
}

// crewNow is the figure the row is performing, and whether it is performing at
// all — which it is not when there is no beat to keep or the key has turned
// keeping time off.
func (m Model) crewNow() (crewFigure, bool) {
	if !m.words.beats {
		return 0, false
	}

	// How long the bar has been up, on the clock everything else on this screen
	// reads. The picture is asked for a little before the bar sounds, so the age
	// can still be negative when the row first goes up.
	age := max(time.Duration(m.wordsClock()-m.words.starts)*time.Millisecond, 0)

	// Counted a little past now, by what the pulse spends coming up to a beat,
	// so a figure changes between two movements rather than inside one.
	lead := time.Duration(beatRise * float32(m.scope.beat.Period))

	beats, ok := m.beatsIn(age, lead)
	if !ok {
		return 0, false
	}
	return crewFor(m.words.starts, beats/crewPhrase), true
}

// crewBehind is how far behind the beat a mark keeps time, as a share of one,
// for the figure the row is performing.
func crewBehind(fig crewFigure, mark int) float32 {
	switch fig {
	case crewUnison:
		return 0
	case crewAlternate:
		if mark%2 == 1 {
			return 0.5
		}
	}
	return 0
}

// crewRiding is how far each mark of the row is lifted this frame, with the row
// performing a figure.
//
// What a mark rides is still its own part of the spectrum — that is what puts
// the sound in the row and it is not the figure's to take away. The figure says
// only *when* each of them lands.
func (m Model) crewRiding(count int, fig crewFigure) []int {
	out := make([]int, count)
	for i := range out {
		pulse := beatFloor + (1-beatFloor)*m.beatPulseAt(crewBehind(fig, i))
		out[i] = -int(m.wordsBeatRide(i, count) * wordsBounce * pulse)
	}
	return out
}
