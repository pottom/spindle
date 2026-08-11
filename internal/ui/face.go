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

	// The nose: where it hangs between the eyes, and how far its hook turns at
	// the bottom as a share of the gap it lives in.
	faceNoseTop  = 0.44
	faceNoseLow  = 0.70
	faceNoseHook = 0.30

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
	//
	// Low enough to fit inside a drawn figure's head: what the generator leaves
	// hollow in the robot is sixteen rows at the size he is usually drawn, and
	// a face at that size is two lids and a mouth, which is enough to blink.
	faceLeast = 12

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

	// stride is where his legs are in their step, -1 to 1, and nought while he
	// stands; facing is which way he is going, which he keeps facing after he
	// has stopped.
	stride, facing float64
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
	facePartNose
	facePartLip
	facePartHand
	facePartLeg
	facePartBody
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
	p.nose(look, light)
	p.mouth(look, light)
	p.hands(look, p.reach, light)
	p.legs(look, light)
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

// nose draws the nose: a stroke down the middle with a hook at the foot of it.
//
// Drawn the way somebody sketching a face would draw one — in profile, on a
// face that is otherwise front on. Two nostrils would be two specks, and a
// triangle would be a shape rather than a stroke; the hook is what says nose
// at a glance and at this size.
func (p faceParts) nose(look faceLook, light func(int, int, facePart)) {
	// It lives in the gap between the eyes, which is the only room there is.
	gap := p.eyes[1].x - (p.eyes[0].x + p.eyes[0].w)
	if gap < p.stroke*4 {
		return
	}

	top := float64(p.h) * faceNoseTop
	low := float64(p.h) * faceNoseLow
	x := float64(p.w) / 2

	// It lifts a shade as the mouth opens, the way a face does when it is
	// pulling one.
	top -= float64(look.mouth) * float64(p.stroke)
	low -= float64(look.mouth) * float64(p.stroke)

	p.curve(func(t float64) (float64, float64) {
		return x, top + (low-top)*t
	}, int(low-top), facePartNose, light)

	// The hook turns the way he is going. A nose drawn in profile is the one
	// part of him that says which way he is facing, so it had better agree with
	// his feet.
	way := look.facing
	if way == 0 {
		way = -1
	}
	hook := float64(gap) * faceNoseHook
	p.curve(func(t float64) (float64, float64) {
		return x + way*hook*t, low + hook*0.35*math.Sin(math.Pi*t*0.5)
	}, int(hook*2), facePartNose, light)
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

	// A double take: the eyes go, stay gone long enough for it to be a look
	// rather than a flick, and come back. The brows arrive with them, which is
	// the half of it that says he saw something.
	faceLookOut  = 170 * time.Millisecond
	faceLookHold = 300 * time.Millisecond
	faceLookBack = 240 * time.Millisecond

	// faceGrinHold is how long a grin is held, and faceGrinSquint how far the
	// lids come down with it — a grin is as much the eyes as the mouth, and a
	// wide mouth under open eyes is a shout.
	faceGrinHold   = 620 * time.Millisecond
	faceGrinSquint = 0.45

	// faceWaveFor is how long he waves at whoever is watching. The wave itself
	// is the arm swing that is already going; this only puts the hand up.
	faceWaveFor = 900 * time.Millisecond

	// faceEase is how fast the eyes follow the sound and the mouth follows the
	// loudness. Slow enough to be a glance rather than a twitch.
	faceEase = 0.14

	// faceMouthMost is how far the mouth opens on the loudest thing playing.
	faceMouthMost = 0.85

	// faceEnters is how far into a bar he comes on, and faceStayLeast to
	// faceStayMost how long he is there for — dealt from the bar, so one visit
	// is a walk-through and the next is a whole turn.
	faceEnters    = 2500 * time.Millisecond
	faceStayLeast = 4200 * time.Millisecond
	faceStayMost  = 10500 * time.Millisecond

	// faceStayMore is what the one this code draws itself gets on top of that.
	// See faceStayFor.
	faceStayMore = 1.5

	// faceShows is how long he stays when he is asked for by hand.
	faceShows = 7 * time.Second

	// faceGagRest is how long he leaves between two things, so a long stay is a
	// turn rather than a twitch, and faceGagBy how far through his stay he does
	// something anyway if the music has not given him a cue.
	faceGagRest = 600 * time.Millisecond
	faceGagBy   = 0.45

	// faceTurn is the rise in loudness he takes as his cue.
	faceTurn = 0.055
)

// faceDoing is what the face has started and not yet finished.
type faceDoing int

const (
	faceStill faceDoing = iota
	faceBlinking
	faceWinking
	faceBrowing
	faceGaping
	faceLooking
	faceGrinning
	faceWaving
	faceSinging
	faceKissing
	faceStunned
	faceNodding
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

	// crumbled is how whole he was last frame, so only what has come away since
	// is handed to the water; act is the run of drawings a figure is in the
	// middle of and actAt when it started; turns is how many things he has done this visit, did whether he
	// has done any at all, rested when he may do the next, and wasLoud what a
	// cue in the music is measured against.
	act      string
	actAt    time.Time
	crumbled float64

	// sweptLow and sweptHigh are the stretch of the screen he has walked through
	// this visit, so a mark he has knocked over stays knocked over while he
	// wanders about, and only what has just gone is handed to the water.
	// sweptFrom is where he came in, which is the side everything he walks past
	// is measured from. See figureBroken.
	sweptLow, sweptHigh, sweptFrom int

	turns   int
	did     bool
	rested  time.Time
	wasLoud float32

	// came is when the face last arrived, which is what it is drawn on from,
	// bar is the moment of the bar it arrived for, on is which of them walked
	// on, and was says it was up last frame — which is how its going is noticed.
	came time.Time
	bar  int64
	on   string
	was  bool

	// picked is who the key has walked to, so pressing it again brings on the
	// next of them rather than the same one over again; shown is when the face
	// was last asked for by hand, and stepped which expression it walked to.
	picked string

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
		m.face.look += (want - m.face.look) * faceEaseAt
		m.face.mouth += (loud*faceMouthMost - m.face.mouth) * faceEaseAt

		// The arms answer the whole of it rather than one band, and slowly, so
		// they climb through a build rather than flapping on every beat.
		var all float32
		for _, v := range bands {
			all += v
		}
		m.face.lift += (min32(all/float32(len(bands))*faceLiftFrom, 1) - m.face.lift) * faceLiftEaseAt
	}

	// If he went out with both arms up, they go off as he does.
	up, was := m.faceUp(), m.face.was

	// Who is on, asked before the visit below is marked: from there on the
	// answer is this one, and asking again would be asking it of itself.
	who := m.faceWho()
	if was && !up && m.face.doing == faceGaping {
		m.faceSparks(m.width, m.height)
	}
	m.face.was = up

	// A new visit: a bar he was not on for, or a different bar from the one he
	// was on for. Both, because a bar at the very top of a record is stamped
	// nought, which is also what the field holds before he has ever been on.
	if up && (!was || m.words.starts != m.face.bar) {
		m.face.bar, m.face.came = m.words.starts, now

		// Who it is, kept for the whole visit rather than asked again later: the
		// key's window is shorter than a stay, and the answer changes when it
		// closes. See faceWho.
		m.face.on = who
		m.face.turns, m.face.did = 0, false
		m.face.rested, m.face.act = time.Time{}, ""
		m.face.crumbled = 1
		m.face.sweptLow, m.face.sweptHigh = figureUnswept, -figureUnswept

		// Measured from where the music is as he arrives, or the first frame
		// of every visit reads as a rise out of silence and he takes it as a
		// cue before he is even on.
		m.face.wasLoud = m.scope.envelope
	}

	if m.face.doing != faceStill {
		// However long the face's own gesture takes, he is not done until the
		// run of drawings that goes with it has played out.
		if now.Sub(m.face.since) > max(faceDoingFor(m.face.doing), figureActLong(m.face.act)) {
			m.face.doing, m.face.since = faceStill, now
			m.face.act = ""
			m.face.rested = now.Add(faceGagRest)
		}
		return
	}

	// Nothing while he is on his way on or off.
	gone := m.faceGone()
	if !up || gone < faceWalkIn || gone > 1-faceWalkOut {
		return
	}

	// What he does he does on the music. The cue is the same rise the meter
	// throws its water on, so his timing is the record's rather than a clock's,
	// and however long the bar gave him is how many he gets in.
	rise := max32(m.scope.envelope-m.face.wasLoud, 0) / max32(m.scope.envelope, scopeFloor)
	m.face.wasLoud = m.scope.envelope

	cue := rise > faceTurn && now.After(m.face.rested)

	// And if the music never gives him one, he does something anyway rather
	// than standing there: he came on to do a thing.
	if !cue && (m.face.did || gone < faceGagBy) {
		if now.After(m.face.due) {
			m.face.doing, m.face.since = faceBlinking, now
			m.face.due = now.Add(faceBlinkEvery + time.Duration(m.scope.roll()*float32(faceBlinkVary)))
		}
		return
	}

	m.face.doing, m.face.since = faceGagFor(m.words.starts, m.face.turns), now

	// And, if a drawn figure is on, the run of drawings that goes with it. The
	// face's own doing is for the geometry; his is a set of pictures.
	m.face.act, m.face.actAt = figureActFor(m.faceWho(), m.words.starts, m.face.turns), now

	m.face.turns, m.face.did = m.face.turns+1, true
}

// faceGagFor is his nth thing this visit. He always does one — somebody who
// turns up, stands there and leaves again is not a turn, he is a glitch — and
// on a long stay, with a busy enough record, he does several.
func faceGagFor(starts int64, turn int) faceDoing {
	h := uint64(starts)*0xbf58476d1ce4e5b9 + 0x94d049bb133111eb + uint64(turn)*0x9e3779b97f4a7c15
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
	case faceLooking:
		return faceLookOut + faceLookHold + faceLookBack
	case faceGrinning:
		return 2*faceBlinkShut + faceGrinHold
	case faceWaving:
		return faceWaveFor
	case faceSinging:
		return faceSingFor
	case faceKissing:
		return faceKissFor
	case faceStunned:
		return faceBlinkShut + faceStunHold + faceBlinkOpen
	case faceNodding:
		return faceNodFor
	}
	return 0
}

// faceFacing is the way whoever is on is turned.
//
// He keeps facing the way he was going after he stops: somebody who turns to
// face front the moment he pulls up is a sprite being swapped, not a figure
// having a rest. Before he has taken a step it is the way he came in.
func (m Model) faceFacing() float64 {
	if way, _ := m.faceGoing(); way != 0 {
		return way
	}
	in, _ := m.faceWays()
	return -in
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

	// Which way he is going, and so which way he is looking and stepping.
	_, moving := m.faceGoing()
	look.facing = m.faceFacing()
	if moving {
		look.stride = math.Sin(2 * math.Pi * faceSteps * m.faceGone())
	}

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
	case faceLooking:
		// A double take. The eyes go, and the brows go with them on the way
		// back, which is the half of it that says he saw something.
		v := faceGlancing(since)
		look.look = min32(max32(look.look+v, -1), 1)
		if back := max32(-abs32(v)+1, 0); since > faceLookOut {
			look.brow = [2]float32{back, back}
		}
	case faceGrinning:
		// A grin is as much the eyes as the mouth: a wide mouth under open eyes
		// is a shout, and the same mouth under a squeeze is a grin.
		v := faceRising(since, 2*faceBlinkShut, faceGrinHold)
		look.mouth = max32(look.mouth, v*0.8)
		look.lid = [2]float32{v * faceGrinSquint, v * faceGrinSquint}
		look.brow = [2]float32{v * 0.35, v * 0.35}
	case faceWaving:
		// At whoever is watching, rather than at the room he is walking into.
		// The wave is the arm swing that is going anyway; this puts the hand up
		// and lifts the brows over it.
		v := faceRising(since, faceBlinkShut, faceWaveFor-faceBlinkShut-faceBlinkOpen)
		if v > 0.2 {
			look.hold[0] = faceHoldWave
		}
		look.brow[0] = max32(look.brow[0], v*0.5)

	case faceSinging:
		// Along with it, badly. The mouth is the record's already — this shuts
		// the eyes over it and lifts the brows, which is the difference between
		// a mouth that is open and somebody singing.
		//
		// Held open at the least, too: the quiet bars of a song are where a
		// singer holds a note, and a mouth that closes on them reads as somebody
		// who has stopped rather than somebody sustaining.
		v := faceRising(since, faceBlinkShut, faceSingFor-faceBlinkShut-faceBlinkOpen)
		look.mouth = max32(look.mouth, v*faceSingLeast)
		look.lid = [2]float32{v * faceSingShut, v * faceSingShut}
		look.brow = [2]float32{v * 0.5, v * 0.5}
		if v > 0.4 {
			look.hold[0] = faceHoldOne
		}

	case faceKissing:
		// Pursed, and thrown. The mouth goes small rather than wide, which is
		// the one shape nothing else here makes, and the hand comes up to it
		// and away — so the throw is the hand and not a heart nobody drew.
		v := faceRising(since, faceBlinkShut, faceKissFor-faceBlinkShut-faceBlinkOpen)
		look.mouth = min32(look.mouth, faceKissMouth)
		look.lid = [2]float32{v * 0.55, v * 0.55}
		look.brow = [2]float32{v * 0.4, v * 0.4}
		if v > 0.25 {
			look.hold[0] = faceHoldWave
		}

	case faceStunned:
		// What a drop deserves. Eyes wide open under raised brows, which is the
		// opposite of the grin: there the eyes squeeze and here they do not.
		v := faceRising(since, faceBlinkShut, faceStunHold)
		look.lid = [2]float32{0, 0}
		look.brow = [2]float32{v, v}
		look.mouth = max32(look.mouth, v*0.7)
		if v > 0.5 {
			look.hold = [2]faceHold{faceHoldUp, faceHoldUp}
		}

	case faceNodding:
		// Keeping time, with the one part of him that can: the eyes drop and
		// come back on the beat, and a finger goes up with them. Off the beat
		// this is a slow blink, which is what nodding to a record without a beat
		// looks like anyway.
		v := float32(math.Abs(math.Sin(math.Pi * faceNods * since.Seconds() / faceNodFor.Seconds())))
		if phase, ok := m.beatPhase(); ok {
			v = 1 - phase
		}
		look.lid = [2]float32{v * faceNodShut, v * faceNodShut}
		look.brow = [2]float32{v * 0.3, v * 0.3}
		look.hold[1] = faceHoldOne
	}
	return look
}

// faceGlancing is where a double take has got to: away, held, and back.
func faceGlancing(since time.Duration) float32 {
	switch {
	case since < faceLookOut:
		return -float32(since) / float32(faceLookOut)
	case since < faceLookOut+faceLookHold:
		return -1
	case since < faceLookOut+faceLookHold+faceLookBack:
		return -1 + float32(since-faceLookOut-faceLookHold)/float32(faceLookBack)
	}
	return 0
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

	// A drawn figure, if this bar was dealt one. See figure.go.
	if art := m.figureLines(w, rows); art != nil {
		return art
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
	if tall, head := m.wordsBandNow(w, rows); tall >= wordsBand {
		m.wordsUnder(grid, paint, hue, w, rows, tall, head)
	}
	return m.drawCellsIn(w, rows, grid, paint, hue, m.styles.Words, m.styles.WordsSeq)
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
				speed:  paceSpeed(faceSparkThrow * (0.6 + m.scope.roll())),
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
	if _, moving := m.faceGoing(); moving {
		top += int(faceBob * math.Abs(math.Sin(2*math.Pi*faceSteps*m.faceGone())))
	}
	return p, left, top, true
}

// faceWalk is where he is across the screen: -1 is off one side, +1 is off the
// other, and in between is on it.
//
// He comes on from a side, wanders about while he is here, and goes off by a
// side. Standing on one spot for the whole of a visit made him a picture that
// had been put there; moving between two or three of them makes him somebody
// who came in.
func (m Model) faceWalk() float64 { return m.faceAt(m.faceGone()) }

// faceAt is where he is at a given point of his visit.
func (m Model) faceAt(t float64) float64 {
	in, out := m.faceWays()

	switch {
	case t < faceWalkIn:
		// On from the side, as far as the first place he stops — unless he is
		// arriving some other way, in which case he is already where he means
		// to be and it is his dots that are doing the moving. See figureWarp.
		if !figureSliding(m.figureComesBy()) {
			return m.faceSpot(0)
		}
		return in + (m.faceSpot(0)-in)*faceEased(t/faceWalkIn)
	case t > 1-faceWalkOut:
		last := m.faceSpot(faceStops - 1)
		if !figureSliding(m.figureGoesBy()) {
			return last
		}
		return last + (out-last)*faceEased((t-(1-faceWalkOut))/faceWalkOut)
	}

	// And in between, from one spot to the next, standing about at each of them
	// before he moves on.
	u := (t - faceWalkIn) / (1 - faceWalkIn - faceWalkOut) * float64(faceStops-1)
	leg := min(int(u), faceStops-2)

	from, to := m.faceSpot(leg), m.faceSpot(leg+1)
	if f := u - float64(leg); f > facePause {
		return from + (to-from)*faceEased((f-facePause)/(1-facePause))
	}
	return from
}

// faceSpot is the nth place he stops at, dealt from the bar so that one visit
// is a pace across and the next is a shuffle on the spot.
func (m Model) faceSpot(at int) float64 {
	h := uint64(m.words.starts)*0xd6e8feb86659fd93 + uint64(at+1)*0x9e3779b97f4a7c15
	h ^= h >> 31
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 29

	// Walking through a row of marks, his stops are spread across it rather
	// than dealt anywhere: he came on to walk through the row, and three stops
	// dealt to the same half of the screen leave the other half standing. Where
	// each of them falls is still his own.
	if m.figureCrosses() {
		in, out := m.faceWays()
		f := float64(at) / float64(faceStops-1)
		return (in+(out-in)*f)*faceCross + (float64(h%1001)/1000-0.5)*faceStep
	}

	// Otherwise somewhere on the screen, never so far out that he is half off
	// it.
	return (float64(h%2001)/1000 - 1) * faceRoam
}

// faceGoing is which way he is moving and whether he is moving at all.
//
// Taken from where he was a moment ago and where he will be a moment from now,
// rather than stored: one path decides where he is, and everything that has to
// agree with it — his feet, his nose — reads it off the same place.
func (m Model) faceGoing() (float64, bool) {
	t := m.faceGone()
	d := m.faceAt(min64(t+faceLook_, 1)) - m.faceAt(max64(t-faceLook_, 0))
	if math.Abs(d) < faceStill_ {
		return 0, false
	}
	return math.Copysign(1, d), math.Abs(d) > faceStill_*2
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

	// Back out the way he came — unless there is a row of marks up for him to
	// walk through, in which case he crosses it. Turning round halfway leaves
	// the far end of the row standing, and a bar with one mark still up at the
	// end of it reads as a figure who could not finish the job rather than as a
	// figure who changed his mind. See figureBroken.
	if h%3 == 0 && !m.figureCrosses() {
		return in, in
	}
	return in, -in
}

// faceStay is how long this visit lasts. Dealt from the bar, so one is a walk
// through and the next is a whole turn — and so that the same bar is the same
// visit twice.
func (m Model) faceStay() time.Duration {
	if !m.face.shown.IsZero() && time.Since(m.face.shown) < faceShows {
		return faceShows
	}
	return m.faceStayFor(m.words.starts)
}

// faceStayFor is the same for a bar that is not the one playing.
//
// The one this code draws itself stays longer than the drawings do. A drawing
// walks on, holds a pose and goes, and there is only so long a still picture is
// worth looking at; he has a face that answers every rise in the music, a pair
// of hands and a dozen things to do with them, and cutting him off after one of
// them is throwing away the part of him that is not a sprite.
func (m Model) faceStayFor(starts int64) time.Duration {
	h := uint64(starts)*0x9e3779b97f4a7c15 + 0x2545f4914f6cdd1d
	h ^= h >> 29
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 32

	stay := faceStayLeast + time.Duration(h%uint64(faceStayMost-faceStayLeast))
	if faceWhoFor(starts) == "" {
		stay = time.Duration(float64(stay) * faceStayMore)
	}
	return stay
}

// faceGone is how far through his visit he is, 0 to 1.
func (m Model) faceGone() float64 {
	if !m.face.shown.IsZero() && time.Since(m.face.shown) < faceShows {
		return float64(time.Since(m.face.shown)) / float64(faceShows)
	}
	into := m.wordsClock() - m.words.starts - faceEnters.Milliseconds()
	return min64(max64(float64(into)/float64(m.faceStay().Milliseconds()), 0), 1)
}

// faceUp reports that the face is on screen now.
//
// Only when he has been sent for, and at no other time. He used to be dealt from
// the bar — one bar in three, then capped at twice a record — and the trouble
// with that is not how often it fired but who decided. This screen is put up in
// a room with people in it, and whoever is running that room knows when a figure
// walking on is the thing and when it is an interruption; the arithmetic never
// will. So he is on a key, beside the record's name, which went the same way for
// the same reason. See faceShow and soloTelling.
func (m Model) faceUp() bool {
	if !m.face.shown.IsZero() && time.Since(m.face.shown) < faceShows {
		return true
	}
	if !m.words.beats {
		return false
	}

	// A visit that has begun runs to its end, whatever the bar underneath it
	// does. On a record with no words of its own a new bar is stamped every half
	// minute — so a figure who had just walked on was taken off mid-stride, and
	// the marks gathered over the top of him. He came on; he leaves the way he
	// came.
	return m.face.was && !m.face.came.IsZero() && time.Since(m.face.came) < m.faceStayFor(m.face.bar)
}

// faceShow puts the face up on demand, and walks through what it can do a press
// at a time — which is the only way to look at a wink on purpose, since it
// happens once in a solo and lasts a third of a second.
func (m *Model) faceShow() {
	now := time.Now()

	// The next of them, every time. Somebody pressing the key twice is somebody
	// who wants to see somebody else, not the same one again.
	cast := figureCast()
	at := 0
	for i, who := range cast {
		if who == m.face.picked {
			at = i + 1
			break
		}
	}
	m.face.picked = cast[at%len(cast)]

	// And the next thing for them to do, so a run of presses is a run of turns
	// rather than the same one over and over.
	m.face.stepped = (m.face.stepped + 1) % faceDoings
	if m.face.stepped == faceStill {
		m.face.stepped = faceBlinking
	}
	m.face.doing, m.face.since = m.face.stepped, now
	m.face.act, m.face.actAt = figureActFor(m.face.picked, now.UnixMilli(), int(m.face.stepped)), now

	// And he is the one on, there and then. A visit already running outranks
	// everything else for as long as it lasts — that is what stops a figure
	// being swapped mid-stride — so without this the key only ever worked
	// between visits, and pressing it during one handed back whoever the bar
	// had dealt. Asked for by hand is the one thing that may interrupt.
	m.face.on, m.face.was = m.face.picked, true

	m.face.came, m.face.bar = now, 0
	m.face.turns, m.face.did, m.face.rested = 0, false, time.Time{}
	m.face.crumbled = 1
	m.face.sweptLow, m.face.sweptHigh = figureUnswept, -figureUnswept
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
	faceWalkIn  = 0.18
	faceWalkOut = 0.18

	// faceStops is how many places he stops at while he is here, faceRoam how
	// far from the middle those are as a share of the room, and facePause how
	// much of the way between two of them he spends standing at the first.
	faceStops = 3
	faceRoam  = 0.5
	facePause = 0.45

	// faceStep is how far a stop on a walk across is allowed to fall either side
	// of where the crossing puts it, so that walking through a row of marks is
	// still a walk rather than a machine indexing along it.
	faceStep = 0.16

	// faceCross is how far out the stops of a walk across go, against faceRoam
	// for a wander: he is going from one end of the row to the other, and a
	// crossing that turns back at the halfway mark on both sides is a pace.
	faceCross = 0.85

	// faceLook_ is how far either side of now his own path is read to tell
	// which way he is going, and faceStill_ how little of a move counts as
	// standing about.
	faceLook_  = 0.03
	faceStill_ = 0.004

	// faceBob is how far a figure rises and falls as it goes when its own
	// drawing does not say, in dots, and faceSteps how many steps that is over
	// a visit.
	faceBob   = 3
	faceSteps = 7

	// faceOpens is how far up the music has to bring a hand before the fist
	// opens into fingers.
	faceOpens = 0.45

	// faceLiftFrom is how much of the range the arms take as fully up: the mean
	// of the bands rarely comes near one, and arms that never leave his sides
	// are arms nobody put there.
	faceLiftFrom = 2.2

	// faceSingFor is how long a bar of singing along lasts, faceSingShut how far
	// the eyes close over it, and faceSingLeast the mouth it keeps open however
	// quiet the record goes.
	faceSingFor   = 2600 * time.Millisecond
	faceSingShut  = 0.75
	faceSingLeast = 0.55

	// faceKissFor is how long a kiss takes, and faceKissMouth how small the
	// mouth goes for it. Small rather than wide: it is the only shape on this
	// face that closes towards the middle.
	faceKissFor   = 1400 * time.Millisecond
	faceKissMouth = 0.18

	// faceStunHold is how long the eyes stay wide.
	faceStunHold = 900 * time.Millisecond

	// faceNodFor is how long he keeps time for, and faceNods how many nods that
	// is where there is no beat to take them from. faceNodShut is how far the
	// eyes drop on each.
	faceNodFor  = 3200 * time.Millisecond
	faceNods    = 6
	faceNodShut = 0.6

	// faceLiftEase is how fast the arms follow the music. Slower than the mouth
	// and the eyes: arms answer a passage, not a beat.
	faceLiftEase = 0.05
)

// faceHold is what the hands are doing.
type faceHold int

const (
	faceHoldDown  faceHold = iota // by his sides, out of the way
	faceHoldWave                  // hello, and goodbye
	faceHoldThumb                 // that was good
	faceHoldOne                   // wait for it
	faceHoldUp                    // both arms up, which is the whole joke
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

// The legs.
//
// He walks, and until now he walked by sliding. Two short legs under him are
// what turn that into walking: they swing against each other, the near one
// forward as the far one goes back, and the bob he already had becomes the
// thing that goes with them rather than a wobble on its own.
//
// Short on purpose. He is a head with limbs on it — there is no body, and legs
// long enough to want one would ask where it went.
const (
	// faceLeg is how far down he stands, as a share of his own height, and
	// faceFoot how long a foot is as a share of the leg.
	faceLeg  = 0.30
	faceFoot = 0.42

	// faceLift is how much of a leg comes up as its foot does, and faceStand
	// how far apart the two of them stand.
	faceLift  = 0.42
	faceStand = 0.13
)

// legs draws the pair, hanging from the bottom of him.
func (p faceParts) legs(look faceLook, light func(int, int, facePart)) {
	leg := faceLeg * float64(p.h)
	if leg < float64(p.stroke)*2 {
		return
	}

	// Under the mouth, a little in from the sides of him — where a small figure
	// drawn by hand would put them.
	hipY := float64(p.lip.y + p.lip.h)
	for side := range 2 {
		dir := 1.0
		if side == 0 {
			dir = -1
		}
		hipX := float64(p.w)/2 + dir*float64(p.w)*0.11

		// He is drawn face on, so a walk is not two legs swinging sideways —
		// from the front that is a pair of scissors. It is one foot up while
		// the other is down, which is what a small figure marching looks like
		// and what reads at this size.
		up := max64(look.stride, 0)
		if side == 1 {
			up = max64(-look.stride, 0)
		}

		turn := dir * faceStand
		sin, cos := math.Sin(turn), math.Cos(turn)
		long := leg * (1 - faceLift*up)
		footX, footY := hipX+long*sin, hipY+long*cos

		p.curve(func(t float64) (float64, float64) {
			return hipX + (footX-hipX)*t, hipY + (footY-hipY)*t
		}, int(long), facePartLeg, light)

		// The foot points the way he is going, or outwards while he stands.
		toe := dir
		if look.stride != 0 && look.facing != 0 {
			toe = look.facing
		}
		p.curve(func(t float64) (float64, float64) {
			return footX + toe*faceFoot*leg*t, footY
		}, int(faceFoot*leg), facePartLeg, light)
	}
}
