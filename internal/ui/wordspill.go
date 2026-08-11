package ui

import "time"

// The line lets go and falls.
//
// Every other way out of this screen is the way in, run backwards: a line that
// wiped on wipes off, one that rose sinks. That is the right default — it makes
// the pair read as one movement rather than two — but it means nothing on this
// screen ever simply drops, and one thing on this screen already does. The
// volume's lamps let go of their wall and fall away, and it is the best-looking
// exit here, because it is the only one that is not a rewind.
//
// So the words can go the same way, and they go it under the same gravity: the
// water falls under stageGravity, the lamps fall under stageGravity, and a line
// letting go falls under stageGravity. That is the whole reason this belongs
// rather than being a tenth effect — three unrelated things on one screen obey
// one pull, so the screen has a floor.
//
// Written against the clock rather than against the frame, unlike the water and
// the lamps: those integrate a speed every frame and so have to be converted for
// the rate being drawn at, while this is a closed form in how far through the
// leaving we are. It is right at any frame rate for free. See pace.go for what
// the other kind costs.

const (
	// wordsSpills is how long a line takes to fall away.
	//
	// More than twice the ordinary leaving, and it has to be, because gravity is
	// not negotiable: at 300ms the screen's own pull carries a dot nine rows,
	// which on type two dozen dots tall is a sag. Measured at 700ms the first
	// word falls 33 rows and the last 19 — a line height and more, which is a
	// fall.
	//
	// It is longer than the 420ms the next line takes to gather, so the two do
	// overlap. What stops that being two pictures is the light: by the time the
	// new line is whole the old one is under half lit and still dimming, and it
	// is below where the type sits. Leaving is the one thing on this screen that
	// is allowed to still be happening when the next thing starts.
	wordsSpills = 700 * time.Millisecond

	// wordsSpillHold is the share of the fall spent letting go, piece by piece
	// from the left.
	//
	// A fifth. At nothing the line drops as one slab, which reads as the picture
	// being lowered rather than as it coming apart. It was a third, following
	// wordsPop, and a third is too much here for a reason that does not apply
	// there: a bursting piece needs no room, while a falling one needs time, and
	// every share spent waiting is a share the last word does not get to fall
	// in. At a third the last word managed two rows before it went dark.
	wordsSpillHold = 0.20

	// wordsSpillKick is the little upward jump a piece gives as it goes, in dot
	// rows a second, and wordsSpillDrift how far sideways it wanders.
	//
	// Anything letting go of something rises before it falls, which is what the
	// lamps do and what every drop of the water does. The kick is small: it
	// peaks about a row up, which is enough to read as a release and not enough
	// to look like a throw.
	//
	// The drift is smaller still, and it was four times this. The lamps wander
	// because they come off a wall sideways; words come off nothing, and at
	// sixteen rows a second the line opened outward faster than it fell — which
	// is not a fall, it is wordsSpreading, and that already exists. Measured at
	// gone 0.6 the pieces had gone further sideways than down.
	wordsSpillKick  = 22
	wordsSpillDrift = 4

	// wordsSpillFade is how the light goes: straight, so what is left of a piece
	// is what is left of its fall.
	//
	// It was squared, on the reasoning that an exit should get out of the way.
	// Measured, that put the light out before the pull had done anything: the
	// mean burn was down to a ninth by the time the first word had fallen six
	// rows, so the whole movement happened in the dark and what reached the
	// screen was a line dimming in place. A fall nobody can see is not a fall.
	wordsSpillFade = 1
)

// wordsSpillPull is the screen's gravity in dot rows a second squared.
//
// stageGravity is dot rows a frame per frame at the rate everything was tuned
// at, so this is that, twice divided by the frame. Derived rather than written
// down: a gravity typed in again here would be one more number to keep in step
// with the water, and the whole point is that they are the same pull.
const wordsSpillPull = stageGravity * float32(time.Second/paceTuned) * float32(time.Second/paceTuned)

// wordsSpill is where a dot of the picture on its way out has got to, when the
// line is falling away.
//
// Per piece rather than per dot: a word holds together as it drops, the way a
// lamp does. Falling dot by dot the line would come apart into rain, and rain is
// what the water on this screen already is.
func (m Model) wordsSpill(x, y int, gone float32, levels int) (int, int, int8, bool) {
	where := m.words.wasWhere
	piece := where.WordAt(x, y)
	if piece < 0 || where.Count == 0 {
		return 0, 0, 0, false
	}

	// Whose turn it is to let go, and how long it has been falling. The starts
	// are spread over the first share of the leaving, so the last piece still
	// has the rest of it to fall in.
	start := float32(piece) / float32(max(where.Count-1, 1)) * wordsSpillHold
	if gone <= start {
		// Still standing, waiting its turn, and lit as it was.
		return x, y, int8(min(int(wordsAhead*float32(levels)), levels-1)), true
	}
	t := (gone - start) * float32(wordsSpills) / float32(time.Second)

	// Which way this piece leans as it goes: outward from the middle of the
	// line, so the line opens rather than slides.
	lean := float32(1)
	if cx, _ := where.Middle(piece); cx < where.DotsX/2 {
		lean = -1
	}

	// Up a little, then down under the screen's own pull.
	fall := -wordsSpillKick*t + wordsSpillPull*t*t/2
	drift := lean * wordsSpillDrift * t

	// And out of light on its own clock, which finishes before the falling does.
	left := 1 - (gone-start)/(1-start)
	if left <= 0 {
		return 0, 0, 0, false
	}
	for range wordsSpillFade - 1 {
		left *= left
	}
	// From exactly the brightness it was standing at, and down from there.
	// wordsPop doubles it, because a piece bursting flares; a piece letting go
	// does not, and doubling it here made the line brighten as it left —
	// measured, the mean burn went from 8 to 13 over the first fifth.
	burn := int8(float32(levels) * left * wordsAhead)
	if burn <= 0 {
		return 0, 0, 0, false
	}
	return x + int(drift), y + int(fall), burn, true
}

// wordsLeavingFor is how long a line takes to go, which depends on how it goes.
//
// Only the fall wants longer, and it wants it because gravity is not negotiable:
// the others are shaped by their own arithmetic and finish whenever they are
// told to.
func wordsLeavingFor(move wordsMove) time.Duration {
	if move == wordsSpilling {
		return wordsSpills
	}
	return wordsLeaving
}

// wordsSpillEvery is how often a line lets go rather than retracing its
// arrival, as one in this many.
//
// One in eight, which is once for each of the eight ways a line can arrive: the
// fall is one more way out among the ways out, not a new default. On a record
// whose lines change every three seconds that is a fall about every twenty-five,
// which is often enough to be judged and rare enough to still be a change when
// it comes.
const wordsSpillEvery = 8

// wordsSpillsNow is whether this line lets go instead of retracing.
//
// Dealt from the line that is leaving and from when it was sung, the same two
// things every other choice on this screen is dealt from — so a record plays the
// same way twice and a test can watch it do so. See wordsMoveFor, which stirs
// the two together for the same reason.
func wordsSpillsNow(text string, starts int64) bool {
	var h uint32 = 2166136261
	for _, r := range text {
		h = (h ^ uint32(r)) * 16777619
	}
	x := uint64(h)*0x9e3779b97f4a7c15 + uint64(starts)*0xd6e8feb86659fd93
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 29
	return x%wordsSpillEvery == 0
}
