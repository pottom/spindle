package ui

import (
	"math"
	"time"
)

// The colour of a line, moved by the harmony.
//
// Everything on this screen answers how loud something is, or when. Nothing
// answers what note is being played, and it is the one thing about a record a
// meter cannot reach: two neighbouring semitones fall in the same band until
// well above where a tune lives. The daemon now folds a longer window into the
// twelve pitch classes, and this is what that is for.
//
// # Why the whole line and not each word
//
// Because that was tried and measured and thrown out. Lighting each word by its
// own part of the sound put the brightest and dimmest word of a six word line
// two of the palette's six steps apart at the median, changing all the while —
// and nobody can read "the third word's band is loud". What they can read is the
// line, and only if it is lit like one. So the colour moves as a line: every
// word of it together, always.
//
// # Why the circle of fifths
//
// A key is not a number and the twelve are not a scale — C is no more "before" G
// than after it. What they have is a distance, and the distance music actually
// travels is the circle of fifths: C to G is one step, C to F# is the far side.
// Weighting the twelve by how loudly each is sounding and taking the direction
// they point in gives an angle that holds still while a passage stays where it
// is and swings when the record moves — which is exactly the behaviour a colour
// wants. A key estimate would jump between two names on a borderline chord; an
// angle slides.
//
// Measured on Emerald Princess, whose score was to hand: the analyser reads the
// printed key signature out of the finished mix, D minor over the opening and A
// major at the bar the score modulates. So there is something real to draw with.
const (
	// hueSettle is how fast the angle is followed. Slow: a colour that chases
	// every chord is a strobe, and what is wanted is where the record has been
	// sitting rather than what the last bar did.
	hueSettle = 0.004

	// hueLeast is how much of the twelve has to be sounding before the angle is
	// believed at all. Silence and noise both point nowhere in particular.
	hueLeast = 0.15

	// hueSpread is how far around the palette the harmony may carry a line, as
	// a share of the whole of it.
	//
	// Not the whole way round, and this is the part to argue with rather than
	// the idea: the palette is built around the artwork's own accent, and a
	// colour free to travel anywhere is a screen that has stopped belonging to
	// the record it is playing. A third of the way is enough that a modulation
	// is unmistakable and little enough that the cover still owns the screen.
	hueSpread = 0.34
)

// hueOrder is the twelve in fifths order, as places in the chroma vector:
// C G D A E B F# C# G# D# A# F.
var hueOrder = [12]int{0, 7, 2, 9, 4, 11, 6, 1, 8, 3, 10, 5}

type hueState struct {
	x, y  float32 // where the harmony points, eased
	heard bool
	at    time.Time

	// wave is when the colour last set off across the line. See hueWave.
	wave time.Time
}

// hueFlow takes this frame's twelve and eases the direction they point in.
func (m *Model) hueFlow(notes []float32) {
	if len(notes) < 12 {
		return
	}

	var sum float32
	for _, v := range notes {
		sum += v
	}
	if sum < hueLeast {
		return
	}

	// The direction the sounding notes point in, around the circle of fifths.
	var x, y float64
	for step, class := range hueOrder {
		a := 2 * math.Pi * float64(step) / 12
		w := float64(notes[class]) / float64(sum)
		x += w * math.Cos(a)
		y += w * math.Sin(a)
	}

	if !m.hue.heard {
		m.hue.x, m.hue.y, m.hue.heard = float32(x), float32(y), true
		return
	}
	m.hue.x += (float32(x) - m.hue.x) * hueSettle
	m.hue.y += (float32(y) - m.hue.y) * hueSettle
	m.hue.at = time.Now()
}

// hueTurn is how far round the palette the harmony has carried the colour, from
// nought to one, or nought before anything has been heard.
func (m Model) hueTurn() float32 {
	if !m.hue.heard {
		return 0
	}

	// The angle, brought into nought to one, and then let out over as much of
	// the palette as it is allowed.
	a := math.Atan2(float64(m.hue.y), float64(m.hue.x))
	if a < 0 {
		a += 2 * math.Pi
	}
	return float32(a/(2*math.Pi)) * hueSpread
}

// A wave of colour crossing a line that is already standing, and gone.
//
// What fires it is a join — the record turning over, see joins.go — because a
// sweep on a timer is the thing this screen has already thrown out twice: an
// effect that arrives because its turn has come is a caption nobody asked for.
// A record changing section is a moment somebody can point at, and this is what
// it looks like from the front.
//
// It moves the hue and not the brightness. A line is lit evenly, which was
// measured and settled — a wave that brightened one word at a time would be the
// same fault the per-word lighting was, only moving. The colour running through
// is a wave; the reading is untouched.
//
// And it puts everything back. Out to the far end and home again, and when it
// has passed the line is exactly what it was.
const (
	// hueSweep is how long the whole journey takes, out and back.
	hueSweep = 1600 * time.Millisecond

	// hueWaveWide is how much of the line the wave covers at once, as a share of
	// it. Narrow enough to read as something travelling rather than as the line
	// changing colour.
	hueWaveWide = 0.30

	// hueWaveMost is how far round the palette the crest carries a word.
	hueWaveMost = 0.5
)

// hueWaveAt starts the wave. Called when the record turns over.
func (m *Model) hueWaveAt() { m.hue.wave = time.Now() }

// hueWave is where the crest is along the line, from nought to one, and whether
// there is one at all.
func (m Model) hueWave() (float32, bool) {
	if m.hue.wave.IsZero() {
		return 0, false
	}
	gone := time.Since(m.hue.wave)
	if gone >= hueSweep {
		return 0, false
	}

	// Out and back, and the turn at the far end is a turn rather than a bounce:
	// a wave that stops dead and reverses reads as two waves.
	at := float32(gone) / float32(hueSweep)
	if at > 0.5 {
		at = 1 - at
	}
	return at * 2, true
}

// hueWaveOn is how much the wave has hold of a word at this point along a line.
func hueWaveOn(along, crest float32) float32 {
	d := along - crest
	if d < 0 {
		d = -d
	}
	if d >= hueWaveWide {
		return 0
	}
	// Raised cosine: no edge to it, so the wave has no front and no back for the
	// eye to catch.
	return 0.5 + 0.5*float32(math.Cos(math.Pi*float64(d/hueWaveWide)))
}
