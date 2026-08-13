package ui

import (
	"math"
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// The word that is being sung shakes, and throws sparks off itself doing it.
//
// The fourth way of following the singer, and the loudest. The other three
// answer with light — see wordsync.go, where the reasoning for that is written
// out — and this one answers with movement, which needed an argument before it
// was allowed.
//
// The argument is that a tremble is not a height. The height of these dots
// belongs to the sound: a band is loud and the type stands taller for it, and it
// stays taller for as long as the band is loud. A shake is the opposite kind of
// thing — a couple of dots either way, twenty times a second, gone inside a
// fifth of one — and nobody reads it as "this word is louder". They read it as
// something happening to that word, which is exactly what is happening.
//
// The sparks needed the same argument and lost half of it. The line already
// sparks, on the beat, and two things using one idiom is how a picture stops
// meaning anything. So these are made to differ where it can be seen: the beat's
// dozen come off the underside of the whole line and fall, and these come out of
// one word in every direction at once and are gone in a third of the time. If
// they still read as the same thing when they are watched, this is the one to
// drop.

const (
	// syncShakeDots is how far the word trembles, in dots, at the moment the
	// voice reaches it.
	syncShakeDots = 2.5

	// syncShakeHz is how fast it trembles. Fast enough to be a buzz rather than
	// a wobble: below about fifteen the word reads as swaying, which is a thing
	// the dancers do and this is not.
	syncShakeHz = 22.0

	// syncShakeLife is how long the tremble lasts. A fifth of a second — the
	// length of a syllable at a walking tempo, so a word shakes for its own
	// moment and is still before the next one starts.
	syncShakeLife = 200 * time.Millisecond

	// syncBurstEach is how many sparks a word throws.
	//
	// Forty, against the beat's twelve for the whole line. It was eighteen and
	// measured: eight of them changed a cell and the rest landed on ink that was
	// already lit, which is the same thing that made the beat's sparks invisible
	// when they were first written. A burst has to be seen against a letter, so
	// it needs enough of them that some clear it.
	syncBurstEach = 40

	// syncBurstLife is how long they last. Half a beat at a walking tempo, and
	// long enough at the speed below to carry a spark a cell or two clear of the
	// word it left — which is the distance at which it stops being part of the
	// letter and starts being something that came off it.
	syncBurstLife = 350 * time.Millisecond

	// syncBurstSpeed is how fast they leave, in dots a second, and syncBurstPull
	// what drags them back down. They are thrown in every direction and then
	// fall, so the ones going up come back through the line they left.
	syncBurstSpeed = 46.0
	syncBurstPull  = 40.0
)

// wordsSyncShakes is the tremble each word is under, in dots, as a pair of
// offsets. Nil where nothing is shaking.
//
// A closed form in the time since the voice reached the word: a decaying sine,
// which is what a struck thing does. Nothing is remembered between frames, so
// this is safe to call from a view and comes out the same at any frame rate —
// the rule the rest of this screen is built on.
func (m Model) wordsSyncShakes(count int) [][2]int {
	if count <= 0 || m.wordsSyncEffect() != syncBurst {
		return nil
	}
	since := m.wordsSyncSince(count)
	if since == nil {
		return nil
	}

	out := make([][2]int, count)
	for w, t := range since {
		if t < 0 || t > float32(syncShakeLife)/float32(time.Second) {
			continue
		}
		fade := 1 - t/(float32(syncShakeLife)/float32(time.Second))
		swing := float64(syncShakeDots * fade * fade)
		// The two axes a quarter turn apart, so the word travels a small circle
		// rather than rocking along one line — a rock reads as a lean, and the
		// lean is the beat's.
		a := 2 * math.Pi * syncShakeHz * float64(t)
		out[w] = [2]int{int(math.Round(swing * math.Cos(a))), int(math.Round(swing * math.Sin(a) / 2))}
	}
	return out
}

// wordsSyncSince is how long ago the voice reached each word, in seconds, or -1
// for the words it has not reached.
func (m Model) wordsSyncSince(count int) []float32 {
	at, ok := m.wordsSyncAt()
	if !ok || count <= 0 {
		return nil
	}
	window := lyricsDefaultLine
	if m.words.ends > m.words.starts {
		window = time.Duration(m.words.ends-m.words.starts) * time.Millisecond
	}
	// How long one word lasts, which is the line's singing shared out. The
	// syllables are what share it on the player screen; here a line is a few
	// words wide on a screen the size of a wall, and the difference between the
	// two rulers is under a frame.
	each := float32(lyricsSung(window)/time.Duration(count)) / float32(time.Second)

	out := make([]float32, count)
	for w := range out {
		out[w] = (at - float32(w)) * each
		if out[w] < 0 {
			out[w] = -1
		}
	}
	return out
}

// wordsSyncSpans is the dot columns each word covers, taken from the map of
// which word is under which dot.
//
// Not from the layout's own Lefts and Rights: those are filled for a row of
// marks and left empty for a line of type, which is what this screen mostly
// shows. Reading them here indexed a slice of nothing at minus one, and the
// program went down in front of somebody watching it — the one place a picture
// must not fail is the one where it is the whole screen.
func wordsSyncSpans(where msg.WordLayout, count int) [][2]int {
	if count <= 0 || where.DotsX <= 0 || len(where.At) == 0 {
		return nil
	}
	out := make([][2]int, count)
	for i := range out {
		out[i] = [2]int{where.DotsX, -1}
	}
	for i, at := range where.At {
		w := int(at)
		if w < 0 || w >= count {
			continue
		}
		x := i % where.DotsX
		out[w][0], out[w][1] = min(out[w][0], x), max(out[w][1], x)
	}
	return out
}

// wordsSyncBurstDraw throws the sparks of whichever word the voice has just
// reached into the picture.
//
// Radial, out of the word's own ink, and falling: what leaves upwards comes back
// through the letters it came from, which is what makes it read as one thing
// breaking rather than as a shower being dropped on it.
func (m Model) wordsSyncBurstDraw(g cover.Grain, grid []uint8, paint []int8, w, rows, levels int) {
	if m.wordsSyncEffect() != syncBurst || levels <= 0 {
		return
	}
	count := m.words.where.Count
	since := m.wordsSyncSince(count)
	if since == nil {
		return
	}
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 || !wordsFits(g, dotsX, dotsY) {
		return
	}
	life := float32(syncBurstLife) / float32(time.Second)
	spans := wordsSyncSpans(m.words.where, count)
	if spans == nil {
		return
	}

	for word, t := range since {
		if t < 0 || t > life {
			continue
		}
		along := t / life
		burn := int8(float32(levels-1) * (1 - along))
		if burn <= 0 {
			continue
		}
		left, right := spans[word][0], spans[word][1]
		if right <= left {
			continue
		}

		for k := range syncBurstEach {
			roll := wordsSparkRoll(word*977+m.words.line, k)

			// Where it left from: a lit dot of this word, so the burst comes
			// out of the ink rather than out of the box around it. Any of them,
			// not the topmost — a burst that all leaves from the crown of a
			// letter is a fountain, and this is meant to be a thing coming
			// apart.
			x := left + int(roll%uint64(right-left))
			var lit []int
			for y := range dotsY {
				if g.Lum[y*dotsX+x] >= wordsLit {
					lit = append(lit, y)
				}
			}
			if len(lit) == 0 {
				continue
			}
			from := lit[int(roll>>32)%len(lit)]

			// Every direction at once, and then gravity. The speed varies so
			// that the edge of the burst is ragged rather than a ring.
			angle := 2 * math.Pi * float64(roll>>8%1000) / 1000
			speed := syncBurstSpeed * (0.55 + 0.45*float64(roll>>20%1000)/1000)
			dx := int(math.Round(speed * math.Cos(angle) * float64(t)))
			dy := int(math.Round(speed*math.Sin(angle)*float64(t) + syncBurstPull*float64(t)*float64(t)/2))

			at, to := x+dx, from+dy
			if at < 0 || at >= dotsX || to < 0 || to >= dotsY {
				continue
			}
			cell := (to/dotsPerCellY)*w + at/dotsPerCellX
			if cell < 0 || cell >= len(grid) {
				continue
			}
			grid[cell] |= 1 << brailleBit[at%dotsPerCellX][to%dotsPerCellY]
			if burn > paint[cell] {
				paint[cell] = burn
			}
		}
	}
}
