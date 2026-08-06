package ui

import (
	"math"
	"time"
)

// The face, in dots.
//
// It goes where the three marks go — the middle of the lyric screen, on a bar
// nobody is singing — and it is built the way everything else on that screen is
// built: strokes two or three dots wide, no fills and no dithering. A face made
// of blobs would be pasted onto this picture; a face made of the same stroke as
// the type belongs to it.
//
// There is no head. A ring around it would cost a third of the height and say
// nothing, and the features inside it would have to halve. What is drawn is
// what a face is read from: two eyes, two brows and a mouth, in space.
//
// Nothing here is a bitmap. Every part is worked out from the box it is given,
// so the same face is drawn on a terminal of any size, and the parts that are
// too small to survive at the bottom of that range are dropped rather than
// squashed. See faceParts.

const (
	// faceStroke is how thick a line of the face is, as a share of the box's
	// height. Measured against the type it stands in for: the stem of a ♪ is
	// two dots on this screen, and a face reads as shape rather than as
	// letterform, so it is drawn a shade bolder.
	faceStroke = 0.055

	// The bands the parts are set in, as shares of the height from the top.
	faceBrowTop = 0.03
	faceBrowLow = 0.13
	faceEyeTop  = 0.21
	faceEyeLow  = 0.68
	faceLipTop  = 0.79
	faceLipLow  = 0.99

	// faceEyeWide is how much of the width one eye takes, and faceEyeGap the
	// air between the two of them.
	faceEyeWide = 0.34
	faceEyeGap  = 0.20

	// faceIris is the ring inside the eye and facePupil the dot inside that, as
	// shares of the eye's half height. Far enough apart that there is daylight
	// between them: closer, and the two run together into a lump, which is an
	// eye with no pupil in it.
	faceIris  = 0.55
	facePupil = 0.19

	// faceLiftMost is how far a pupil may travel from the middle of its eye,
	// as a share of the room it has. Short of the rim: an eye whose pupil is
	// touching its own outline is a cartoon.
	faceLookMost = 0.38

	// faceLeast is the fewest dot rows a face can be set in. Under this the
	// eyes fall to four rows and the mouth to two, which is a colon and a
	// bracket — and the marks are better than that.
	faceLeast = 18

	// faceBrowLeast is the fewest rows that will hold brows as well. Under it
	// the brow and the lid meet, and the lid carries the expression alone.
	faceBrowLeast = 26
)

// faceLook is what the face is doing: how far each lid has come down, how far
// each brow has lifted, how far the mouth is open, and where the eyes are
// looking. Every one of them is 0..1 so that a frame is a set of numbers rather
// than a picture, and anything can be part way between two of them.
type faceLook struct {
	lid   [2]float32 // 0 open, 1 shut
	brow  [2]float32 // 0 level, 1 raised
	mouth float32    // 0 closed, 1 open
	look  float32    // -1 left, +1 right

	// hold is what each hand is doing, lift how far the music has raised the
	// arms when it is doing nothing in particular, and swing where the swing
	// of them has got to.
	hold  [2]faceHold
	lift  float32
	swing float64
}

// faceParts is where each part of the face goes in a box of the given size,
// worked out once and shared by the drawing and by whatever wants to know how
// tall the face stands.
type faceParts struct {
	w, h   int
	stroke int

	// eyes are the two eye boxes, brows the two brow boxes, lip the mouth's.
	eyes  [2]faceBox
	brows [2]faceBox
	lip   faceBox

	// browsToo says the box is deep enough to carry brows at all, and reach is
	// how much room there is outside it for the hands.
	browsToo bool
	reach    int
}

// faceBox is a part's own rectangle inside the face, in dots.
type faceBox struct{ x, y, w, h int }

func (b faceBox) middle() (int, int) { return b.x + b.w/2, b.y + b.h/2 }

// faceLayout works out where everything goes in a w by h box of dots.
func faceLayout(w, h int) (faceParts, bool) {
	if w < faceLeast*2 || h < faceLeast {
		return faceParts{}, false
	}

	p := faceParts{w: w, h: h, browsToo: h >= faceBrowLeast}
	p.stroke = max(int(faceStroke*float64(h)), 2)

	eyeW := int(faceEyeWide * float64(w))
	gap := int(faceEyeGap * float64(w))
	left := (w - 2*eyeW - gap) / 2

	eyeY, eyeH := int(faceEyeTop*float64(h)), int((faceEyeLow-faceEyeTop)*float64(h))
	browY, browH := int(faceBrowTop*float64(h)), int((faceBrowLow-faceBrowTop)*float64(h))
	lipY, lipH := int(faceLipTop*float64(h)), int((faceLipLow-faceLipTop)*float64(h))

	// With no room for brows the eyes take the space they would have had, which
	// is what keeps them large enough to blink in.
	if !p.browsToo {
		eyeY, eyeH = browY, eyeY+eyeH-browY
	}

	for i := range 2 {
		x := left + i*(eyeW+gap)
		p.eyes[i] = faceBox{x, eyeY, eyeW, eyeH}
		// A brow is shorter than the eye under it and sits over its outer half,
		// which is what gives a face a direction to look in.
		p.brows[i] = faceBox{x + eyeW/6, browY, eyeW * 3 / 4, browH}
	}

	lipW := int(0.52 * float64(w))
	p.lip = faceBox{(w - lipW) / 2, lipY, lipW, lipH}
	return p, true
}

// faceDraw lights the face into a dot field. light is given a dot and which
// part of the face it belongs to, so the caller can give each part its own
// colour and its own place to bounce to.
type facePart int

const (
	facePartBrow facePart = iota
	facePartEye
	facePartLip
	facePartHand
	faceParts_
)

// draw lights the face.
//
// All of him at once. He is whole before he is on: what he does is walk in from
// the side, and a figure who assembles himself out of strokes while he walks is
// two entrances at the same time.
func (p faceParts) draw(look faceLook, light func(x, y int, part facePart)) {
	if p.browsToo {
		for i := range 2 {
			p.brow(i, look, light)
		}
	}
	for i := range 2 {
		p.eye(i, look, light)
	}
	p.mouth(look, light)
	p.hands(look, p.reach, light)
}

// faceShare is how far one part has come, given how far the whole face has: it
// waits its turn and then arrives over the rest of the time.
func faceShare(grow, from, to float64) float64 {
	if grow >= 1 {
		return 1
	}
	return min64(max64((grow-from)/(to-from), 0), 1)
}

// eye draws one eye: an almond outline, and inside it — while it is open
// enough to hold them — an iris ring and a pupil.
//
// The lid does not slide down over the eye like a shutter. It closes the way an
// eye closes: the top arc comes down to meet the bottom one, so the shape stays
// an eye all the way to the line it ends as.
func (p faceParts) eye(i int, look faceLook, light func(int, int, facePart)) {
	box := p.eyes[i]
	shut := look.lid[i]

	cx, cy := box.middle()
	rx, ry := float64(box.w)/2, float64(box.h)/2

	// The lid does not slide down over the eye like a shutter. It closes the
	// way an eye closes: the upper arc falls and the lower one rises a little
	// to meet it, so the shape stays an eye all the way down to the line it
	// ends as.
	// Both reaches close to nothing; what does not close is the line they meet
	// on, which sits below the middle where a shut eye's lashes sit.
	up := ry * float64(1-shut)
	down := ry * float64(1-shut)
	cy += int(ry * 0.22 * float64(shut))

	// The outline is walked as a curve rather than scanned column by column: a
	// column of dots per x comes out thick where the curve is flat and one dot
	// thick where it is steep, and a line of two weights does not read as drawn.
	p.curve(func(t float64) (float64, float64) {
		a := 2 * math.Pi * t
		c, s := math.Cos(a), math.Sin(a)
		reach := up
		if s > 0 {
			reach = down * 0.82 // the lower lid is the shallower of the two
		}
		return float64(cx) + rx*c, float64(cy) + math.Copysign(reach*faceAlmond(c), s)
	}, int(4*(rx+ry)), facePartEye, light)

	// What is inside only survives while the eye is open enough to hold it —
	// the iris first, then the pupil, so a lid coming down takes the eye apart
	// in the order an eye goes rather than all at once.
	open := up + down
	iris, pupil := faceIris*ry, facePupil*ry
	if open < pupil*2+float64(p.stroke) || pupil < 1 {
		return
	}

	// The pupil travels, and the iris travels with it, staying inside the
	// aperture the lids have left: a half shut eye looks through a slit rather
	// than over its own lid.
	lookX := float64(cx) + float64(look.look)*faceLookMost*(rx-iris-float64(p.stroke))
	lookY := float64(cy)

	// The iris is struck a shade finer than the eye it sits in, the way the
	// inside of a drawing is lighter than its outline.
	if open >= iris*2+float64(p.stroke)*2 && iris >= float64(p.stroke)*1.4 {
		fine := p
		fine.stroke = max(p.stroke-1, 2)
		fine.curve(func(t float64) (float64, float64) {
			a := 2 * math.Pi * t
			return lookX + iris*math.Cos(a), lookY + iris*math.Sin(a)
		}, int(8*iris), facePartEye, light)
	}

	p.disc(lookX, lookY, pupil, facePartEye, light)
}

// faceAlmond is the eye's own profile. An ellipse would give it round ends and
// a face made of two circles; a straight taper would give it corners. This is
// between them — full through the middle, and coming to a point.
func faceAlmond(c float64) float64 {
	return math.Pow(max64(1-c*c, 0), 0.8)
}

// curve walks a closed path and stamps the face's stroke along it, so the line
// is the same weight wherever it is going.
func (p faceParts) curve(at func(float64) (float64, float64), steps int, part facePart, light func(int, int, facePart)) {
	steps = max(steps, 24)
	for i := range steps {
		x, y := at(float64(i) / float64(steps))
		p.stamp(x, y, part, light)
	}
}

// stamp is one dab of the stroke: a round of its own width, which is what makes
// a walked curve come out an even line rather than a string of beads.
func (p faceParts) stamp(cx, cy float64, part facePart, light func(int, int, facePart)) {
	r := float64(p.stroke) / 2
	for y := -p.stroke; y <= p.stroke; y++ {
		for x := -p.stroke; x <= p.stroke; x++ {
			if float64(x*x+y*y) <= r*r+0.35 {
				light(int(cx)+x, int(cy)+y, part)
			}
		}
	}
}

// disc fills a small round — the pupil, and nothing else on this screen.
func (p faceParts) disc(cx, cy, r float64, part facePart, light func(int, int, facePart)) {
	if r < 1 {
		return
	}
	for y := -int(r) - 1; y <= int(r)+1; y++ {
		for x := -int(r) - 1; x <= int(r)+1; x++ {
			if float64(x*x+y*y) <= r*r {
				light(int(cx)+x, int(cy)+y, part)
			}
		}
	}
}

// brow draws one brow: a tapered arc that lifts and arches together, because a
// brow that only rises reads as a bar being dragged about.
func (p faceParts) brow(i int, look faceLook, light func(int, int, facePart)) {
	box := p.brows[i]
	up := float64(look.brow[i])

	lift := float64(box.h) * 1.1 * up
	arch := float64(box.h) * 0.45 * (0.6 + up)

	steps := box.w * 3
	for step := range steps {
		t := float64(step) / float64(steps-1)

		// The outer end of a brow is the one away from the nose, and that is
		// where its peak sits; the inner end runs down and thin.
		out := 1 - t
		if i == 1 {
			out = t
		}
		x := int(float64(box.x) + t*float64(box.w-1))

		// One arc, highest over the outer half. A full sine gives two humps and
		// reads as a bug rather than as a brow.
		y := float64(box.y+box.h) - lift - arch*(0.30+0.70*math.Sin(math.Pi*out))

		// Tapered: the full stroke at the outer end, thinning toward the nose.
		thick := max(int(math.Round(float64(p.stroke)*(0.45+0.55*out))), 1)
		for d := range thick {
			light(x, int(y)+d, facePartBrow)
		}
	}
}

// mouth draws the mouth: a level bar whose last fifth turns up, opening into a
// slot as the sound does.
//
// Level on purpose. A bowl is what makes a smiley, and a smiley is a shape with
// no craft in it; a straight mouth with one corner lifted is a face that is
// thinking about something.
func (p faceParts) mouth(look faceLook, light func(int, int, facePart)) {
	box := p.lip
	open := float64(box.h-p.stroke) * float64(look.mouth)
	turn := float64(box.h) * 0.22

	lip := func(t float64) (float64, float64) {
		var curl float64
		if t > 0.86 {
			curl = -math.Pow((t-0.86)/0.14, 1.6) * turn
		}
		return float64(box.x) + t*float64(box.w-1), float64(box.y) + curl
	}

	steps := box.w * 2
	for step := range steps {
		t := float64(step) / float64(steps-1)
		x, y := lip(t)
		p.stamp(x, y, facePartLip, light)

		if open < float64(p.stroke) {
			continue
		}
		// The lower lip, held apart by however far the mouth is open and
		// rounded at the ends, so the gap is a mouth and not a letterbox.
		round := math.Sqrt(max64(1-math.Pow(2*t-1, 2), 0))
		p.stamp(x, y+open*round, facePartLip, light)
	}
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// What the face does when it is left alone.
const (
	// faceBlinkEvery is how long between blinks, and faceBlinkVary how much of
	// that is left to chance. A face that blinks on a metronome is a machine.
	faceBlinkEvery = 4500 * time.Millisecond
	faceBlinkVary  = 2500 * time.Millisecond

	// faceBlinkShut is how long a lid takes to fall and faceBlinkOpen how long
	// to lift. Up is slower than down, which is what an eye does.
	faceBlinkShut = 90 * time.Millisecond
	faceBlinkOpen = 150 * time.Millisecond
	faceBlinkHold = 40 * time.Millisecond

	// faceWinkHold is how long a wink stays shut — long enough to be read as a
	// wink rather than as a blink that went wrong.
	faceWinkHold = 180 * time.Millisecond

	// faceBrowHold is how long raised brows stay up.
	faceBrowHold = 700 * time.Millisecond

	// faceEase is how fast the eyes follow the sound and the mouth follows the
	// loudness. Slow enough to be a glance rather than a twitch.
	faceEase = 0.14

	// faceMouthMost is how far the mouth opens on the loudest thing playing.
	faceMouthMost = 0.85

	// faceEnters is how far into a bar he comes on, and faceStays how long he
	// is there for. Long enough to arrive, do the one thing he came to do and
	// leave; short enough that what you remember is the thing rather than him.
	faceEnters = 2500 * time.Millisecond
	faceStays  = 3800 * time.Millisecond

	// faceShows is how long he stays when he is asked for by hand.
	faceShows = 5 * time.Second

	// faceGagAfter is how long after he arrives he does the thing he came for —
	// long enough that he has been read as a face first.
	faceGagAfter = 1100 * time.Millisecond

)

// faceDoing is what the face has started and not yet finished.
type faceDoing int

const (
	faceStill faceDoing = iota
	faceBlinking
	faceWinking
	faceBrowing
	faceGaping
	faceDoings
)

// faceState is what the face carries between frames.
//
// The expression is not a stored picture but a moment: what it started, when it
// started, and how far the sound has moved the parts that follow it. Every
// frame works the look out again from those, so a frame that arrives late lands
// where it should rather than where the last one left off.
type faceState struct {
	doing faceDoing
	since time.Time
	due   time.Time // when the next blink falls

	// look, mouth and lift follow the sound rather than a clock.
	look, mouth, lift float32

	// gag is the thing he came to do and gagAt when he does it.
	gag   faceDoing
	gagAt time.Time

	// came is when the face last arrived, which is what it is drawn on from,
	// bar is the moment of the bar it arrived for, and was says it was up last
	// frame — which is how its going is noticed.
	came time.Time
	bar  int64
	was  bool

	// shown is when the face was last asked for by hand, and stepped which
	// expression the key has walked to.
	shown   time.Time
	stepped faceDoing
}

// faceFlow moves the face on by a frame: it follows the sound, and it starts
// whatever it is time to start.
func (m *Model) faceFlow() {
	now := time.Now()
	if m.face.due.IsZero() {
		m.face.due = now.Add(faceBlinkEvery)
	}

	// The eyes drift toward whichever part of the range is loudest, and the
	// mouth opens on how loud that is. This is the thing that makes a face read
	// as alive rather than as a graphic that blinks on a timer.
	if bands := m.scope.bands; len(bands) > 0 {
		var loud float32
		at := 0
		for i, v := range bands {
			if v > loud {
				loud, at = v, i
			}
		}
		want := (float32(at)/float32(max(len(bands)-1, 1)))*2 - 1
		m.face.look += (want - m.face.look) * faceEase
		m.face.mouth += (loud*faceMouthMost - m.face.mouth) * faceEase

		// The arms answer the whole of it rather than one band, and slowly, so
		// they climb through a build rather than flapping on every beat.
		var all float32
		for _, v := range bands {
			all += v
		}
		m.face.lift += (min32(all/float32(len(bands))*faceLiftFrom, 1) - m.face.lift) * faceLiftEase
	}

	// The face going down is handed to the machinery that carries a line off
	// the screen, so it leaves the way it came and fades as it goes rather than
	// being switched off.
	// If he went out with both arms up, they go off as he does.
	up := m.faceUp()
	if m.face.was && !up && m.face.gag == faceGaping {
		m.faceSparks(m.width, m.height)
	}
	m.face.was = up

	// A new bar is a new face: it draws itself on, and whether it will wink is
	// settled there and then from the bar's own moment, so a record does the
	// same thing twice.
	if up && m.words.starts != m.face.bar {
		m.face.bar, m.face.came = m.words.starts, now
		m.face.gag, m.face.gagAt = faceGagFor(m.words.starts), now.Add(faceGagAfter)
	}

	if m.face.doing != faceStill {
		if now.Sub(m.face.since) > faceDoingFor(m.face.doing) {
			m.face.doing, m.face.since = faceStill, now
		}
		return
	}

	// Nothing starts while he is still walking on.
	if m.faceGone() < faceWalkIn {
		return
	}

	// The thing he came to do, once, a moment after he has arrived.
	if !m.face.gagAt.IsZero() && now.After(m.face.gagAt) {
		m.face.doing, m.face.since = m.face.gag, now
		m.face.gagAt = time.Time{}
		return
	}

	if now.After(m.face.due) {
		m.face.doing, m.face.since = faceBlinking, now
		m.face.due = now.Add(faceBlinkEvery + time.Duration(m.scope.roll()*float32(faceBlinkVary)))
	}
}

// faceGagFor is what he came to do. He always does one: somebody who turns up,
// stands there and leaves again is not a turn, he is a glitch.
func faceGagFor(starts int64) faceDoing {
	h := uint64(starts)*0xbf58476d1ce4e5b9 + 0x94d049bb133111eb
	h ^= h >> 31
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 29

	// Anything but standing still and blinking, which he does anyway.
	return faceWinking + faceDoing(h%uint64(faceDoings-faceWinking))
}

// faceDoingFor is how long each thing the face does takes, end to end.
func faceDoingFor(doing faceDoing) time.Duration {
	switch doing {
	case faceBlinking:
		return faceBlinkShut + faceBlinkHold + faceBlinkOpen
	case faceWinking:
		return faceBlinkShut + faceWinkHold + faceBlinkOpen
	case faceBrowing:
		return 2*faceBlinkShut + faceBrowHold
	case faceGaping:
		return faceBlinkShut + faceBrowHold
	}
	return 0
}

// faceNow is the look the face wears this frame.
func (m Model) faceNow() faceLook {
	look := faceLook{look: m.face.look, mouth: m.face.mouth}

	// He waves himself on. After that the hands are the music's until whatever
	// he came to do wants them.
	if m.faceGone() < faceWalkIn {
		look.hold = [2]faceHold{faceHoldWave, faceHoldWave}
	}
	look.lift = m.face.lift
	look.swing = faceSwing * float64(0.35+0.65*m.face.lift) *
		math.Sin(2*math.Pi*faceWaves*time.Since(m.face.came).Seconds())

	since := time.Since(m.face.since)
	switch m.face.doing {
	case faceBlinking:
		v := faceShutting(since)
		look.lid = [2]float32{v, v}
	case faceWinking:
		// One lid, the brow over it lifting with it, the eyes glancing away
		// from the side that closed — and a finger up on the same side, which
		// is what a wink is for.
		v := faceShutting(since - (faceWinkHold-faceBlinkHold)/2)
		look.lid[1] = v
		look.brow[1] = v
		look.look = min32(look.look-0.5*v, 1)
		if v > 0 {
			look.hold[1] = faceHoldOne
		}
	case faceBrowing:
		v := faceRising(since, 2*faceBlinkShut, faceBrowHold)
		look.brow = [2]float32{v, v}
		if v > 0.4 {
			look.hold = [2]faceHold{faceHoldThumb, faceHoldThumb}
		}
	case faceGaping:
		// Both arms up, which is the whole joke.
		v := faceRising(since, faceBlinkShut, faceBrowHold)
		look.mouth = max32(look.mouth, v)
		look.brow = [2]float32{v * 0.6, v * 0.6}
		if v > 0.3 {
			look.hold = [2]faceHold{faceHoldUp, faceHoldUp}
		}
	}
	return look
}

// faceShutting is how far a lid has come down, given how long ago it started.
func faceShutting(since time.Duration) float32 {
	switch {
	case since < 0:
		return 0
	case since < faceBlinkShut:
		return float32(since) / float32(faceBlinkShut)
	case since < faceBlinkShut+faceBlinkHold:
		return 1
	case since < faceBlinkShut+faceBlinkHold+faceBlinkOpen:
		return 1 - float32(since-faceBlinkShut-faceBlinkHold)/float32(faceBlinkOpen)
	}
	return 0
}

// faceRising is the same for something that goes up, holds and comes down.
func faceRising(since, climb, hold time.Duration) float32 {
	switch {
	case since < climb:
		return float32(since) / float32(climb)
	case since < climb+hold:
		return 1
	case since < 2*climb+hold:
		return 1 - float32(since-climb-hold)/float32(climb)
	}
	return 0
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// faceRide is how far a part of the face rides its own share of the sound, in
// dots. Well under what a note is given: a face has to hold together while it
// moves, and a brow that leaves its eye behind is not a brow.
const faceRide = 3

// faceWide is how much wider than tall the face is set. The marks it stands in
// for run 2.6 : 1; a face is one object rather than three, so it is drawn
// tighter than the space it is allowed.
const faceWide = 2.0

// faceLines draws the face into the same picture the words are drawn into: the
// meter standing on the floor and hanging from the ceiling, the water crossing
// between them, and the face in the middle where a line would be set.
//
// It returns nil when the face is not what is on, or when there is no room for
// one — in which case whatever else holds the slot holds it.
func (m Model) faceLines(w, rows int) []string {
	if !m.faceUp() || w <= 0 || rows <= 0 || len(m.styles.Words) == 0 {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	levels, freqs := len(m.styles.Words[0]), len(m.styles.Words)

	p, left, top, ok := m.faceRoom(w, rows)
	if !ok {
		return nil
	}

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}

	// Each part answers its own share of the sound, exactly as the words of a
	// line do: the brows the bass, the eyes the middle, the mouth the top.
	var part [faceParts_]wordPaint
	for i := range part {
		part[i] = m.wordsBeatPaint(int(i), int(faceParts_), freqs, levels)
	}

	// Each part rides its own share of the sound, the way the words of a line
	// do — the brows the bass, the eyes the middle, the mouth the top. Small:
	// the notes are given six dots and a face wants two or three, or the parts
	// come apart from each other.
	var ride [faceParts_]int
	for i := range ride {
		ride[i] = -int(m.wordsBeatRide(int(i), int(faceParts_)) * faceRide)
	}

	p.draw(m.faceNow(), func(x, y int, at facePart) {
		x, y = x+left, y+top+ride[at]
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if s := part[at]; s.level > paint[cell] {
			paint[cell], hue[cell] = s.level, s.hue
		}
	})

	// The meter takes what the face left, the same way it takes what a line of
	// words leaves.
	tall := max((dotsY-(top+p.h))/dotsPerCellY, 0)
	if tall >= wordsBand {
		m.wordsUnder(grid, paint, hue, w, rows, tall, max(top-dotsPerCellY, 0))
	}
	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}

// faceSparks throws the water off his fingertips as he goes.
//
// It is the one thing he leaves behind. The drops belong to the meter and cross
// the whole screen already, so this is not a new picture — it is his hands
// handing something to the picture that was there before him.
func (m *Model) faceSparks(w, rows int) {
	p, left, top, ok := m.faceRoom(w, rows)
	if !ok {
		return
	}
	dotsY := rows * dotsPerCellY

	for side := range 2 {
		dir := 1.0
		if side == 0 {
			dir = -1
		}
		arm, mitt := faceArm*float64(p.h), faceMitt*float64(p.h)
		turn := faceHoldTurn(faceHoldUp)

		sx := float64(left) + float64(p.w)*0.5 + dir*float64(p.w)*0.5
		sy := float64(top) + float64(p.eyes[side].y+p.eyes[side].h/2)
		tipX := sx + dir*(arm+mitt*2.2)*math.Sin(turn)
		tipY := sy + (arm+mitt*2.2)*math.Cos(turn)

		for range faceSparkEach {
			if len(m.stage.drops) >= stageDrops {
				return
			}
			m.stage.drops = append(m.stage.drops, stageDrop{
				col:    int(tipX + float64(dir)*float64(m.scope.roll())*mitt),
				at:     float32(dotsY-1) - float32(tipY),
				speed:  faceSparkThrow * (0.6 + m.scope.roll()),
				bright: 0.7 + 0.3*m.scope.roll(),
			})
		}
	}
}

// faceSparkEach is how many drops leave each hand, and faceSparkThrow how hard.
const (
	faceSparkEach  = 14
	faceSparkThrow = 5.0
)

// faceRoom is where the face sits on a screen of this size: its parts, and the
// corner they are drawn from.
func (m Model) faceRoom(w, rows int) (faceParts, int, int, bool) {
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	// The same band of the screen the marks are set in, so the meters above and
	// below stand exactly where they stand for a bar of notes.
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		return faceParts{}, 0, 0, false
	}

	// What is left either side of him is where his hands go.
	room := (dotsX - wide) / 2
	p.reach = room

	// And where he is: off one side, across to the middle, and off the other.
	walk := m.faceWalk()
	left := room + int(walk*float64(room+wide))

	// He bobs as he goes. A couple of dots: enough that he is walking rather
	// than being slid across, little enough that the meters do not notice.
	top := (dotsY - high) / 2
	if walk != 0 {
		top += int(faceBob * math.Abs(math.Sin(2*math.Pi*faceSteps*m.faceGone())))
	}
	return p, left, top, true
}

// faceWalk is where he is across the screen: -1 is off the side he came in
// from, 0 is where he stops to do his turn, and +1 is off the side he leaves by.
//
// He always comes in from a side and always leaves by one. It is the whole
// shape of the thing — somebody walks on, does something, walks off — and a
// figure who instead materialised out of a scatter of dots in the middle of the
// screen was four different entrances competing with one joke.
func (m Model) faceWalk() float64 {
	in, out := m.faceWays()
	t := m.faceGone()

	switch {
	case t < faceWalkIn:
		return in * (1 - faceEased(t/faceWalkIn))
	case t > 1-faceWalkOut:
		return out * faceEased((t-(1-faceWalkOut))/faceWalkOut)
	}
	return 0
}

// faceEased is a movement that sets off and pulls up rather than running at one
// speed from end to end.
func faceEased(t float64) float64 {
	t = min64(max64(t, 0), 1)
	return t * t * (3 - 2*t)
}

// faceWays is which side he comes on from and which side he leaves by. Two
// visits in three he carries on the way he was going; the third he thinks
// better of it and goes back out the way he came.
func (m Model) faceWays() (float64, float64) {
	h := uint64(m.words.starts)*0x2545f4914f6cdd1d + 0x9e3779b97f4a7c15
	h ^= h >> 32
	h *= 0xd6e8feb86659fd93
	h ^= h >> 32

	in := -1.0
	if h&(1<<40) != 0 {
		in = 1
	}
	if h%3 == 0 {
		return in, in // back out the way he came
	}
	return in, -in
}

// faceGone is how far through his visit he is, 0 to 1.
func (m Model) faceGone() float64 {
	if !m.face.shown.IsZero() && time.Since(m.face.shown) < faceShows {
		return float64(time.Since(m.face.shown)) / float64(faceShows)
	}
	into := m.wordsClock() - m.words.starts - faceEnters.Milliseconds()
	return min64(max64(float64(into)/float64(faceStays.Milliseconds()), 0), 1)
}

// faceUp reports that the face is on screen now.
//
// He is not what a bar of music looks like — the marks are that. He is somebody
// who turns up in the middle of one, does a thing, and goes again, and the
// marks have the bar back afterwards. So this is a window inside a bar rather
// than the whole of it: a few seconds, a while after the bar started, on the
// bars that were dealt him.
func (m Model) faceUp() bool {
	if !m.face.shown.IsZero() && time.Since(m.face.shown) < faceShows {
		return true
	}
	if !m.words.beats || !faceDealt(m.words.starts) {
		return false
	}

	since := m.wordsClock() - m.words.starts - faceEnters.Milliseconds()
	return since >= 0 && since < faceStays.Milliseconds()
}

// faceDealt is whether the bar starting at a given moment gets a face rather
// than the notes. One in three: often enough to be a thing the screen does,
// seldom enough that the notes are still what a bar of music looks like.
func faceDealt(starts int64) bool {
	h := uint64(starts) * 0x9e3779b97f4a7c15
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 29
	return h%3 == 0
}

// faceShow puts the face up on demand, and walks through what it can do a press
// at a time — which is the only way to look at a wink on purpose, since it
// happens once in a solo and lasts a third of a second.
func (m *Model) faceShow() {
	now := time.Now()
	if !m.face.shown.IsZero() && time.Since(m.face.shown) < faceShows {
		m.face.stepped = (m.face.stepped + 1) % faceDoings
		if m.face.stepped == faceStill {
			m.face.stepped = faceBlinking
		}
		m.face.doing, m.face.since = m.face.stepped, now
	} else {
		m.face.stepped = faceStill
		m.face.came = now // asked for, and so drawn on from nothing
		m.face.bar = 0
		m.face.gag, m.face.gagAt = faceGagFor(now.UnixMilli()), now.Add(faceGagAfter)
	}
	m.face.shown = now
}

// The hands.
//
// There is room for them: the face takes 124 dots of a 320 dot screen, which
// leaves ninety either side — wider than one of his own eyes. What there is not
// room for is a cartoon glove. These are drawn out of the same stroke as the
// rest of him: a palm, a thumb, and at most two fingers, because four fingers
// at this size is a comb.
//
// The arm is not drawn out to a shoulder. It is a short stem that comes in from
// off the picture and turns about a point, so waving is one angle changing
// rather than a set of hand-drawn frames.
const (
	// faceArm is the stem's length and faceMitt the palm's radius, both as
	// shares of the face's own height.
	faceArm  = 0.34
	faceMitt = 0.15

	// faceSwing is how far the arms swing either way, in radians, and faceWaves
	// how many times they go over and back in a second.
	faceSwing = 0.30
	faceWaves = 2.4

	// faceWalkIn and faceWalkOut are the shares of a visit spent coming on and
	// going off; the rest of it he stands where he stopped and does his turn.
	faceWalkIn  = 0.26
	faceWalkOut = 0.26

	// faceBob is how far he rises and falls as he walks, in dots, and faceSteps
	// how many steps that is over a visit.
	faceBob   = 2.5
	faceSteps = 7

	// faceOpens is how far up the music has to bring a hand before the fist
	// opens into fingers.
	faceOpens = 0.45

	// faceLiftFrom is how much of the range the arms take as fully up: the mean
	// of the bands rarely comes near one, and arms that never leave his sides
	// are arms nobody put there.
	faceLiftFrom = 2.2

	// faceLiftEase is how fast the arms follow the music. Slower than the mouth
	// and the eyes: arms answer a passage, not a beat.
	faceLiftEase = 0.05
)

// faceHold is what the hands are doing.
type faceHold int

const (
	faceHoldDown faceHold = iota // by his sides, out of the way
	faceHoldWave                 // hello, and goodbye
	faceHoldThumb                // that was good
	faceHoldOne                  // wait for it
	faceHoldUp                   // both arms up, which is the whole joke
)

// hands draws the pair, out to the sides of the face.
//
// reach is how much room there is outside the face's own box, which is what
// decides whether they are drawn at all: on a narrow terminal the meters and
// the screen's edge are already there, and a hand drawn into them is a smudge.
func (p faceParts) hands(look faceLook, reach int, light func(int, int, facePart)) {
	arm := faceArm * float64(p.h)
	mitt := faceMitt * float64(p.h)
	if reach < int(arm+2*mitt) || mitt < float64(p.stroke) {
		return
	}

	for side := range 2 {
		p.hand(side, look, side, arm, mitt, light)
	}
}

// hand draws one of them, from the shoulder out.
//
// The hands are always there. What they are doing is the music's: with nothing
// else going on the arms ride the loudness, hanging at his sides through a
// quiet passage and coming up as it builds, and the fist opens into fingers on
// the way up. A gesture — a thumb, a finger, both arms over his head — takes
// them off the music for as long as it lasts.
func (p faceParts) hand(side int, look faceLook, index int, arm, mitt float64, light func(int, int, facePart)) {
	hold := look.hold[index]

	// Which way is out. The two are mirrors of each other, so one set of
	// arithmetic draws both.
	dir := 1.0
	if side == 0 {
		dir = -1
	}

	// The shoulder sits just outside the face, level with the eyes.
	sx := float64(p.w)*0.5 + dir*float64(p.w)*0.5
	sy := float64(p.eyes[side].y + p.eyes[side].h/2)

	// How far round from hanging straight down. Up is a bigger angle; the swing
	// goes about wherever they are held.
	turn := faceHoldTurn(hold) + look.swing
	if hold == faceHoldDown {
		// Nothing in particular: the music has them.
		turn = faceHoldTurn(faceHoldDown) +
			float64(look.lift)*(faceHoldTurn(faceHoldUp)-faceHoldTurn(faceHoldDown)) + look.swing
	}
	sin, cos := math.Sin(turn), math.Cos(turn)

	// Where the wrist and the palm sit along it. Nought is straight down, which
	// is where an arm is when nothing is being done with it.
	wx, wy := sx+dir*arm*sin, sy+arm*cos
	px, py := sx+dir*(arm+mitt*0.9)*sin, sy+(arm+mitt*0.9)*cos

	// The stem, drawn out from the shoulder as far as it has arrived.
	p.curve(func(t float64) (float64, float64) {
		return sx + (wx-sx)*t, sy + (wy-sy)*t
	}, int(arm), facePartHand, light)

	// The palm: a round of its own, open rather than filled, so it is a hand
	// and not a bat.
	fine := p
	fine.stroke = max(p.stroke-1, 2)
	fine.curve(func(t float64) (float64, float64) {
		a := 2 * math.Pi * t
		return px + mitt*math.Cos(a), py + mitt*0.85*math.Sin(a)
	}, int(8*mitt), facePartHand, light)

	// The thumb, off the inner edge — pointing up when that is the whole point
	// of the gesture, and tucked along the palm otherwise.
	thumb := turn - dir*0.9
	if hold == faceHoldThumb {
		thumb = math.Pi
	}
	p.stem(px, py, mitt*1.15, thumb, dir, light)

	// And the fingers, along the arm. A hand at his side is a loose fist; it
	// opens as the music brings it up.
	if hold == faceHoldDown && look.lift > faceOpens {
		open := faceShare(float64(look.lift), faceOpens, 1)
		p.stem(px+dir*mitt*0.45*cos, py-mitt*0.45*sin, mitt*1.35*open, turn, dir, light)
		p.stem(px-dir*mitt*0.45*cos, py+mitt*0.45*sin, mitt*1.35*open, turn, dir, light)
	}

	switch hold {
	case faceHoldOne:
		p.stem(px, py, mitt*1.5, turn, dir, light)
	case faceHoldWave, faceHoldUp:
		p.stem(px+dir*mitt*0.45*cos, py-mitt*0.45*sin, mitt*1.35, turn, dir, light)
		p.stem(px-dir*mitt*0.45*cos, py+mitt*0.45*sin, mitt*1.35, turn, dir, light)
	}
}

// stem draws a finger or a thumb: a short stroke out from a point.
func (p faceParts) stem(x, y, long, turn, dir float64, light func(int, int, facePart)) {
	sin, cos := math.Sin(turn), math.Cos(turn)
	p.curve(func(t float64) (float64, float64) {
		return x + dir*long*sin*t, y + long*cos*t
	}, int(long*2), facePartHand, light)
}

// faceHoldTurn is how far round from hanging straight down each hold is held,
// in radians: nought is by his side, pi is straight over his head.
func faceHoldTurn(hold faceHold) float64 {
	switch hold {
	case faceHoldWave:
		return 2.5
	case faceHoldThumb:
		return 0.7
	case faceHoldOne:
		return 2.3
	case faceHoldUp:
		return 2.9
	}
	return 0.35
}
