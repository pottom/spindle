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

	// crewLevel is where a mark stands when nothing is timing it: the height the
	// timed figures average out at over a beat.
	//
	// This is the one number that keeps the figures from giving themselves away.
	// A row that lands on the beat spends most of the beat coming down again, so
	// over a beat it averages a little over half its travel; a row that is not
	// keeping time at all used to sit at the whole of it. The movement was the
	// same either side of the change and the *height* was not, so what a change
	// of figure read as was the picture being handed to a different program.
	// Measured over a beat rather than reasoned: see TestEveryFigureStandsAsTall.
	crewLevel = 0.56

	// crewBowDip is how far the row sinks when it bows, in dots.
	//
	// Down, which nothing else on this screen does — every other movement here
	// is a rise off the line the marks are set on. That is the whole reason the
	// bow reads as a gesture rather than as another bounce: it goes the way
	// nothing else goes. Kept well under what a mark rides up, because the
	// meter's own band is underneath and a row that dips into it is a row that
	// has fallen over rather than bowed.
	crewBowDip = 6
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

	// crewCall is the same halves of a beat given to the two halves of the row
	// instead: the left lands on the beat and the right answers between two.
	// Where the see-saw is one thing rippling, this is two groups talking.
	crewCall

	// crewHuddle is the one figure that is not about when they land at all:
	// they lean in towards the middle of the row on the beat and straighten
	// between two. A body leaning is a different thing from a body jumping, and
	// the row has spent every other figure jumping.
	crewHuddle

	// crewBow sinks the whole row on the first beat of the phrase and dances the
	// rest of it. One gesture, once, which is what a bow is.
	crewBow

	// crewFree is nobody keeping time: every mark on its own part of the sound
	// and nothing shared. A phrase of it is what makes the next figure arriving
	// an event rather than more of the same — and it is exactly the picture the
	// row has when the key has turned keeping time off, so it costs nothing to
	// be sure it looks right.
	crewFree

	crewFigures
)

// crewDeal is the hat the figures are drawn from, and how many tickets each has
// in it.
//
// Not one each. The plain two are what the row does most of the time and the
// rest are departures from them: a bow every sixth phrase is a bow nobody
// notices, and a row that stops keeping time as often as it keeps it is a row
// with something wrong with it. Measured only in the sense that the shares are
// stated here rather than being whatever an equal draw happened to give.
var crewDeal = []crewFigure{
	crewUnison, crewUnison, crewUnison, crewUnison,
	crewAlternate, crewAlternate, crewAlternate, crewAlternate,
	crewCall, crewCall, crewCall,
	crewHuddle, crewHuddle, crewHuddle,
	crewBow, crewBow,
	crewFree, crewFree,
}

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
	return crewDeal[h%uint64(len(crewDeal))]
}

// crew is what the row is doing this frame: the figure, and how far into the
// bar and the phrase it has got.
type crew struct {
	fig  crewFigure
	beat int // beats since the bar went up
}

// inPhrase is which beat of the phrase this is, nought for the first.
func (c crew) inPhrase() int { return c.beat % crewPhrase }

// crewNow is what the row is performing, and whether it is performing at all —
// which it is not when there is no beat to keep or the key has turned keeping
// time off.
func (m Model) crewNow() (crew, bool) {
	if !m.words.beats {
		return crew{}, false
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
		return crew{}, false
	}
	return crew{fig: crewFor(m.words.starts, beats/crewPhrase), beat: beats}, true
}

// crewBehind is how far behind the beat a mark keeps time, as a share of one,
// for the figure the row is performing.
func crewBehind(fig crewFigure, mark, count int) float32 {
	switch fig {
	case crewAlternate:
		if mark%2 == 1 {
			return 0.5
		}
	case crewCall:
		if mark >= count/2 {
			return 0.5
		}
	}
	return 0
}

// crewCalls is where the figure asks each mark to stand this frame, in dots,
// up being positive — which is the opposite way round from how the picture is
// drawn, and the right way round for thinking about dancers.
//
// What a mark rides is still its own part of the spectrum — that is what puts
// the sound in the row, and it is not the figure's to take away. The figure
// says when each of them lands, and in one case which way.
func (m Model) crewCalls(count int, c crew) []float32 {
	out := make([]float32, count)

	// The bow: the whole row sinks together on the first beat of the phrase.
	// Laid over whatever the row is dancing rather than replacing it, so that
	// the dancers go on dancing while they go down — a group that stops moving
	// to bow has not bowed, it has sat down.
	var dip float32
	if c.fig == crewBow && c.inPhrase() == 0 {
		dip = crewBowDip * m.beatPulse()
	}

	for i := range out {
		ride := m.wordsBeatRide(i, count)

		// Free, and the huddle, which leans instead of landing: neither has
		// anything to say about when a mark rises, so both stand at the height
		// the others average out at. That last part is what stops a change of
		// figure being visible as a change of figure — see crewLevel.
		level := float32(crewLevel)
		if c.fig != crewFree && c.fig != crewHuddle {
			level = beatFloor + (1-beatFloor)*m.beatPulseAt(crewBehind(c.fig, i, count))
		}
		out[i] = ride*wordsBounce*level - dip
	}
	return out
}

const (
	// crewRise and crewFall are how much of the way a mark travels towards
	// where the figure is calling it, each frame, going up and coming back.
	//
	// This is what a body has and a meter does not: mass. Without it every mark
	// stands exactly where the arithmetic says it should this frame — and since
	// what feeds that arithmetic is a spectrum that moves at thirty frames a
	// second, the row shook rather than danced, and a change of figure was a
	// jump rather than a movement into it. With it, what a dancer is asked to do
	// is a place to get to.
	//
	// Up fast and down slow, because that is what a jump is: the push is quick
	// and the fall is gravity. At thirty frames a second the rise is most of the
	// way there in three frames — a tenth of a second, which still reads as a
	// hit — and the fall takes about a third of a second, which is most of a
	// beat at a hundred and twenty.
	crewRise = 0.45
	crewFall = 0.12
)

// crewFlow moves the row towards where the figure is calling it, a frame at a
// time. It is the only part of this that is not worked out fresh every frame,
// and the reason is the whole point: a dancer who is somewhere has to have been
// on the way there.
func (m *Model) crewFlow(count int) {
	c, ok := m.crewNow()
	if !ok || count <= 0 {
		m.words.standing = nil
		return
	}

	want := m.crewCalls(count, c)
	if len(m.words.standing) != count {
		// A row that has just changed size starts where it is called rather
		// than climbing from the floor, which would read as the whole row
		// growing out of the ground every time the terminal is resized.
		m.words.standing = want
		return
	}

	for i, at := range want {
		ease := float32(crewFall)
		if at > m.words.standing[i] {
			ease = crewRise
		}
		m.words.standing[i] += (at - m.words.standing[i]) * ease
	}
}

// crewRiding is how far each mark of the row is lifted this frame: where it has
// got to, rather than where it was last called.
func (m Model) crewRiding(count int, c crew) []int {
	standing := m.words.standing
	if len(standing) != count {
		standing = m.crewCalls(count, c)
	}

	out := make([]int, count)
	for i, at := range standing {
		out[i] = -int(at)
	}
	return out
}

// crewLeaning is how far each mark leans while the row huddles: in towards the
// middle on the beat, upright between two.
//
// The two halves lean opposite ways, which is what makes it a huddle rather
// than a row of italics — and the mark nearest the middle leans least, because
// it has least distance to lean across.
func (m Model) crewLeaning(count int, lean float32) []float32 {
	out := make([]float32, count)
	if count < 2 {
		return out
	}

	pulse := m.beatPulse()
	middle := float32(count-1) / 2
	for i := range out {
		// Nought in the middle of the row, one at either end, and negative on
		// the side that has to lean the other way.
		side := (middle - float32(i)) / middle
		out[i] = lean * side * pulse
	}
	return out
}
