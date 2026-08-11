package ui

import (
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The line sparks on the beat.
//
// The same movement as the fall in wordspill.go, held down to almost nothing:
// instead of the whole line letting go once, a dozen dots let go on each beat,
// drop a few rows and go out before the next one. The line itself does not move.
//
// It comes from watching the fall and seeing what it was nearly. A whole line
// coming down is an event, and an event can happen once a verse; the same
// arithmetic at a hundredth of the scale can happen four times a bar, and then
// it is not an event at all — it is the type answering the record, which is what
// everything else on this screen does.
//
// Under the same pull as the fall, the water and the volume's lamps, for the
// same reason: three unrelated things obeying one gravity is what makes them one
// picture rather than three effects.
//
// Only when the screen is keeping time. Loose, there is no beat to spark on and
// the sparks would be a rhythm the record has not got — see beatKeeping, and the
// b key, which is what puts the two side by side.

const (
	// wordsSparkEach is how many dots let go on a beat.
	//
	// A dozen, out of the twelve hundred or so a line of type lights. One in a
	// hundred: enough that something happens on every beat and few enough that
	// what happens is a glint rather than the line shedding.
	wordsSparkEach = 12

	// wordsSparkLife is how long a spark lasts, as a share of the beat it was
	// thrown on.
	//
	// Nearly all of it, and under one so the beat is always clear before the
	// next arrives — a spark still falling when the next dozen let go turns four
	// glints a bar into a steady drizzle, which is the water's job and not the
	// type's.
	//
	// It was a little over half, and at that it never left the cell it started
	// in: the screen's gravity moves a dot three rows in a fifth of a second,
	// and a braille cell is four rows deep. Measured, twelve sparks a beat added
	// nothing at all to the picture — every one of them landed on ink that was
	// already there. At this it clears the letter by a cell or two on a slow
	// record and one on a fast one.
	wordsSparkLife = 0.85

	// wordsSparkKick is the upward flick a spark leaves with, in dot rows a
	// second.
	//
	// Small, and a quarter of what it was, because a spark has a third of a
	// second to live and at twelve the flick took the first half of that: it
	// went up, came back, and went dark about where it started. Enough now that
	// it comes off the letter rather than being dropped by it, and not enough to
	// spend the life it has.
	wordsSparkKick = 3

	// wordsSparkLit is how bright a spark starts, as a share of the palette.
	//
	// All of it. It was a share of what the unsung words are drawn at, on the
	// reasoning that a piece coming off a letter should not outshine the letter
	// — and that made it invisible, because it does not land on empty screen.
	// The meter's band sits under the type and the sparks fall into it, so a
	// spark dimmer than what is already in that cell changes nothing at all:
	// measured, twelve a beat moved two cells of two hundred and thirty-nine.
	//
	// A spark is the brightest thing in the frame for as long as it lasts. That
	// is what the word means, and it is the only way one dot says anything on a
	// screen this busy.
	wordsSparkLit = 1.0
)

// wordsSparkDraw puts this beat's sparks into the picture.
//
// A closed form in the beat's phase, like everything else drawn from the beat:
// nothing is remembered between frames, so this is safe to call from a view and
// right at any frame rate.
func (m Model) wordsSparkDraw(g cover.Grain, grid []uint8, paint []int8, w, rows, levels int) {
	if !m.beatKeeping() || levels <= 0 {
		return
	}
	beat, ok := m.beatsRun()
	if !ok {
		return
	}
	phase, ok := m.beatPhase()
	if !ok || phase >= wordsSparkLife {
		return
	}
	period := m.scope.beat.Period
	if period <= 0 {
		return
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 || len(g.Lum) < dotsX*dotsY {
		return
	}

	// How far along this spark's own life it is, and how long that has been in
	// seconds — which is what the gravity is written in.
	along := phase / wordsSparkLife
	t := float32(phase) * float32(period) / float32(time.Second)

	fall := -wordsSparkKick*t + wordsSparkPull*t*t/2
	if fall < 0 {
		fall = 0
	}
	burn := int8(float32(levels-1) * (1 - along) * wordsSparkLit)
	if burn <= 0 {
		return
	}

	for k := range wordsSparkEach {
		// Where this one comes from, dealt afresh on every beat so the line
		// glints somewhere new each time rather than dripping from the same
		// dozen places.
		x := int(wordsSparkRoll(beat, k) % uint64(dotsX))

		// The underside of the type in that column: a lit dot with nothing lit
		// below it. That is where anything comes off a shape, and it keeps the
		// sparks under the line rather than inside it, where they would only
		// make the letters look moth-eaten.
		from := -1
		for y := dotsY - 1; y >= 0; y-- {
			if g.Lum[y*dotsX+x] >= wordsLit {
				from = y
				break
			}
		}
		if from < 0 {
			continue
		}

		to := from + int(fall)
		if to < 0 || to >= dotsY {
			continue
		}
		cell := (to/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][to%dotsPerCellY]
		if burn > paint[cell] {
			paint[cell] = burn
		}
	}
}

// wordsSparkPull is the screen's gravity in dot rows a second squared, the same
// number the falling line uses. Named again rather than shared so that either
// can be tuned without the other silently moving; they are meant to agree, and
// TestTheSparksFallUnderTheSamePullAsTheLine says so.
const wordsSparkPull = wordsSpillPull

// wordsSparkRoll is which column a spark comes from, dealt from the beat it is
// thrown on and which of the dozen it is.
func wordsSparkRoll(beat, k int) uint64 {
	x := uint64(beat)*0x9e3779b97f4a7c15 + uint64(k)*0xd6e8feb86659fd93
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 29
	return x
}
