package ui

import (
	"math"
	"sort"
	"strings"
	"testing"
	"time"
)

// summon puts him on screen the way the key does, which is the only way he
// comes on at all now — without walking his expressions on, which is the key's
// other job and nothing to do with being there. See faceUp and faceShow.
func summon(m *Model, bar int64) {
	m.words.starts = bar
	m.face.shown = time.Now()
}

// faceDots draws a face into a plain field, for looking at and for measuring.
func faceDots(t *testing.T, w, h int, look faceLook) []string {
	t.Helper()

	p, ok := faceLayout(w, h)
	if !ok {
		return nil
	}
	grid := make([][]byte, h)
	for y := range grid {
		grid[y] = []byte(strings.Repeat(".", w))
	}
	p.draw(look, func(x, y int, _ facePart) {
		if x >= 0 && y >= 0 && x < w && y < h {
			grid[y][x] = '#'
		}
	})

	out := make([]string, h)
	for i, r := range grid {
		out[i] = string(r)
	}
	return out
}

// The face is geometry rather than a bitmap, so it is drawn at whatever size
// the terminal gives it — and where there is not room for a part, the part goes
// rather than being squashed.
func TestTheFaceIsDrawnAtAnySize(t *testing.T) {
	for _, size := range [][2]int{{72, 22}, {90, 33}, {100, 50}, {135, 65}} {
		w, h := size[0], size[1]
		p, ok := faceLayout(w, h)
		if !ok {
			t.Fatalf("%dx%d: no face", w, h)
		}
		if p.stroke < 2 {
			t.Errorf("%dx%d: a stroke of %d dots is thinner than the type it stands in for", w, h, p.stroke)
		}

		dots := faceDots(t, w, h, faceLook{})
		var lit int
		for _, r := range dots {
			lit += strings.Count(r, "#")
		}
		t.Logf("%3dx%-3d stroke %d, brows %v, %d dots lit", w, h, p.stroke, p.browsToo, lit)

		if lit == 0 {
			t.Fatalf("%dx%d: nothing was drawn", w, h)
		}
		// Nothing may fall outside the box it was given.
		for y, r := range dots {
			if len(r) != w {
				t.Fatalf("row %d is %d wide, want %d", y, len(r), w)
			}
		}
	}

	// Under the floor there is no face at all, and the marks are better than
	// what a face would be at that size.
	if _, ok := faceLayout(60, faceLeast-1); ok {
		t.Error("a face was laid out in fewer rows than it can be read in")
	}
	// And between the two floors it goes on without brows.
	p, ok := faceLayout(72, 22)
	if !ok || p.browsToo {
		t.Errorf("at 72x22 the face has brows (%v), want the lid to carry it alone", p.browsToo)
	}
}

// A blink closes the eye. Not by sliding something over it — the shape stays an
// eye all the way down to the line it ends as.
func TestTheEyeClosesToALine(t *testing.T) {
	const w, h = 100, 50

	p, ok := faceLayout(w, h)
	if !ok {
		t.Fatal("no face")
	}

	// The eye alone: he has hands and legs beside it now, and they are not it.
	tall := func(shut float32) int {
		top, bottom := 1<<30, -1
		p.draw(faceLook{lid: [2]float32{shut, shut}}, func(x, y int, part facePart) {
			if part != facePartEye || x > w/2 {
				return
			}
			top, bottom = min(top, y), max(bottom, y)
		})
		return bottom - top + 1
	}

	open, half, shut := tall(0), tall(0.5), tall(1)
	t.Logf("the eye stands %d dots open, %d half shut and %d closed", open, half, shut)

	if !(shut < half && half < open) {
		t.Errorf("the eye went %d → %d → %d, want it closing", open, half, shut)
	}
	if shut > open/4 {
		t.Errorf("a shut eye is %d dots deep against %d open, want it closed to a line", shut, open)
	}
}

// He never comes on by himself.
//
// He was dealt from the bar — one in three, then capped at twice a record — and
// the trouble with that was never how often it fired but who decided. This
// screen goes up in a room with people in it, and whoever is running that room
// knows when a figure walking on is the thing and when it is an interruption.
// So he is on a key, beside the record's name, which went the same way.
func TestHeNeverComesOnByHimself(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 6 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// Every bar of a long instrumental, walked through as the screen would, and
	// every moment inside each of them.
	for bar := range int64(12) {
		starts := bar * wordsSpell.Milliseconds()
		m.words.starts = starts

		for _, into := range []time.Duration{0, faceEnters, faceEnters + time.Second, 20 * time.Second} {
			m.setProgress(time.Duration(starts)*time.Millisecond + into)
			m.face.came = time.Now().Add(-time.Minute) // the visit before is over
			m.faceFlow()

			if m.faceUp() {
				t.Fatalf("bar %d, %s in: he came on with nobody asking", starts, into)
			}
		}
	}
	t.Log("twelve bars of a record, four moments in each: he stayed away from all of them")

	// And the key still brings him, there and then.
	m.faceShow()
	if !m.faceUp() {
		t.Error("the key was pressed and nobody came")
	}
}

// He is not what a bar of music looks like — the marks are that. He turns up in
// the middle of one, does his thing and goes, and the marks have the bar back.
func TestHeVisitsABarRatherThanTakingIt(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// Any bar, and he is sent for: there is no other way he comes on.
	var bar int64 = 21_000
	m.words.starts = bar

	m.setProgress(time.Duration(bar) * time.Millisecond)
	if m.faceUp() {
		t.Error("he is there before anybody asked, want the marks")
	}

	m.faceShow()
	if !m.faceUp() {
		t.Error("he was sent for and did not come")
	}

	// And he goes again: a visit is a few seconds, not the rest of the bar.
	m.face.shown = time.Now().Add(-faceShows - time.Second)
	m.face.came = time.Now().Add(-time.Minute)
	if m.faceUp() {
		t.Error("he is still there long after his turn, want the marks back")
	}

	// Never over a line that is being sung.
	m.words.beats, m.words.text = false, "a line of the song"
	m.words.starts = bar
	m.setProgress(time.Duration(bar)*time.Millisecond + faceEnters + time.Second)
	if m.faceUp() {
		t.Error("he walked on over a line of the song")
	}
}

// And he always does something. Somebody who turns up, stands there and leaves
// again is not a turn, he is a glitch.
func TestHeAlwaysDoesSomething(t *testing.T) {
	seen := map[faceDoing]int{}
	for bar := range int64(60) {
		gag := faceGagFor(bar*7_000, 0)
		if gag == faceStill || gag == faceBlinking {
			t.Fatalf("a visit was given %d, which is nothing to watch", gag)
		}
		seen[gag]++
	}
	t.Logf("sixty visits: %v", seen)
	if len(seen) < int(faceDoings-faceWinking) {
		t.Errorf("only %d of the %d turns ever come up", len(seen), faceDoings-faceWinking)
	}
}

// And every one of them shows on his face. A turn that is dealt and looks
// exactly like standing there is a turn nobody will ever see.
func TestEveryTurnShows(t *testing.T) {
	m := scopeModel(160, 46)
	m.words.beats = true
	m.ps.Duration = 4 * time.Minute

	// The middle of a visit, where his hands are his own: on the way in he is
	// waving with both of them, and everything would be measured against that.
	m.setProgress(faceEnters + m.faceStay()/2)

	still := m.faceNow()
	for doing := faceWinking; doing < faceDoings; doing++ {
		long := faceDoingFor(doing)
		if long <= 0 {
			t.Errorf("the turn %d runs for no time at all", doing)
			continue
		}

		moved := map[string]bool{}
		for _, at := range []float64{0.3, 0.5, 0.75} {
			m.face.doing = doing
			m.face.since = time.Now().Add(-time.Duration(at * float64(long)))
			look := m.faceNow()

			for part, changed := range map[string]bool{
				"the lids":  look.lid != still.lid,
				"the brows": look.brow != still.brow,
				"the mouth": look.mouth != still.mouth,
				"the eyes":  look.look != still.look,
				"the hands": look.hold != still.hold,
			} {
				if changed {
					moved[part] = true
				}
			}
		}
		keys := make([]string, 0, len(moved))
		for part := range moved {
			keys = append(keys, part)
		}
		sort.Strings(keys)
		t.Logf("turn %d runs %s and moves %v", doing, long, keys)
		if len(moved) == 0 {
			t.Errorf("the turn %d changes nothing on him over its whole %s", doing, long)
		}
	}
}

// And it is on a key, because a wink lasts a third of a second and happens once
// in a solo, which is no way to look at one on purpose.
func TestTheFaceCanBeAskedFor(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.words.beats, m.words.text = true, wordsNotes
	m.words.starts = 1

	if m.faceUp() {
		t.Fatal("the face was up before it was asked for")
	}

	m.faceShow()
	if !m.faceUp() {
		t.Fatal("the face was asked for and did not come")
	}

	// Pressing again brings on the next of them, doing the next thing, rather
	// than the same one over again.
	who, doing := map[string]bool{}, map[faceDoing]bool{}
	for range len(figureCast()) + 2 {
		was := m.faceWho()
		m.faceShow()
		if m.faceWho() == was {
			t.Errorf("pressing again brought %q back on", was)
		}
		who[m.faceWho()] = true
		doing[m.face.doing] = true
	}
	t.Logf("a run of presses brought on %d of them, doing %d different things", len(who), len(doing))

	if len(who) < len(figureCast()) {
		t.Errorf("%d of the %d ever came on", len(who), len(figureCast()))
	}
	if doing[faceStill] {
		t.Error("one of the presses showed nothing at all")
	}
}

// The face is drawn into the same picture as everything else on that screen —
// the whole terminal, no more and no less.
func TestTheFaceFillsTheScreen(t *testing.T) {
	const w, rows = 160, 46

	m := scopeModel(w, rows)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.words.beats, m.words.text = true, wordsNotes
	m.face.shown = time.Now()

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.6
	}
	m.scope.bands = bands

	art := m.faceLines(w, rows)
	if len(art) != rows {
		t.Fatalf("the face drew %d rows, want %d", len(art), rows)
	}
	for i, line := range art {
		if got := len([]rune(ansiOff(line))); got != w {
			t.Errorf("row %d is %d cells wide, want %d", i, got, w)
		}
	}

	var lit int
	for _, line := range art {
		for _, r := range ansiOff(line) {
			if r != ' ' {
				lit++
			}
		}
	}
	t.Logf("the face and its meters light %d cells of %d", lit, w*rows)
	if lit == 0 {
		t.Error("nothing was drawn")
	}
}

// The eye is three things, not one lump: an outline, an iris ring inside it and
// a pupil inside that, with daylight between them. Drawn any closer together
// the iris and the pupil run into each other and the eye has no pupil at all.
func TestTheEyeHasAPupilInIt(t *testing.T) {
	// The face as a 160x46 terminal sets it.
	dotsX, dotsY := 320, 184
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		t.Fatal("no face at the size a terminal gives it")
	}
	dots := faceDots(t, wide, high, faceLook{})
	eye := p.eyes[0]

	// Count the runs of lit dots across the middle of the eye: outline, iris,
	// pupil, iris, outline is five.
	row := dots[eye.y+eye.h/2][eye.x : eye.x+eye.w]
	var runs int
	for i := range row {
		if row[i] == '#' && (i == 0 || row[i-1] != '#') {
			runs++
		}
	}
	t.Logf("across the middle of the eye: %q — %d runs", row, runs)

	if runs < 5 {
		t.Errorf("the eye crosses in %d runs, want the outline, the iris and the pupil each on their own", runs)
	}

	// And the pupil goes where the eye is looking, without reaching the rim.
	left := faceDots(t, wide, high, faceLook{look: -1})
	right := faceDots(t, wide, high, faceLook{look: 1})
	// Where the pupil is: the middle run of the five, since the outline does
	// not move and would hide the answer if it were measured with them.
	at := func(dots []string) int {
		row := dots[eye.y+eye.h/2]
		var runs [][2]int
		for x := eye.x; x < eye.x+eye.w; x++ {
			if row[x] != '#' {
				continue
			}
			if n := len(runs); n > 0 && runs[n-1][1] == x-1 {
				runs[n-1][1] = x
				continue
			}
			runs = append(runs, [2]int{x, x})
		}
		if len(runs) < 3 {
			t.Fatalf("the eye crosses in %d runs, with no pupil to find", len(runs))
		}
		mid := runs[len(runs)/2]
		return (mid[0] + mid[1]) / 2
	}
	if at(left) >= at(right) {
		t.Error("the eyes do not follow the sound")
	}
}

// He has hands, and they are drawn out to the sides where there is room — a
// palm, a thumb and at most two fingers, out of the same stroke as the rest of
// him. Four fingers at this size is a comb.
func TestHeHasHands(t *testing.T) {
	dotsX, dotsY := 320, 184
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		t.Fatal("no face")
	}
	p.reach = (dotsX - wide) / 2

	hands := func(hold faceHold) int {
		var n int
		p.draw(faceLook{hold: [2]faceHold{hold, hold}}, func(x, y int, at facePart) {
			if at == facePartHand {
				n++
			}
		})
		return n
	}

	down := hands(faceHoldDown)
	t.Logf("hands down: %d dots; wave %d; thumb %d; one %d; up %d",
		down, hands(faceHoldWave), hands(faceHoldThumb), hands(faceHoldOne), hands(faceHoldUp))

	// They are always on him — hanging at his sides is a pose, not an absence.
	for _, hold := range []faceHold{faceHoldDown, faceHoldWave, faceHoldThumb, faceHoldOne, faceHoldUp} {
		if hands(hold) < 40 {
			t.Errorf("hold %d drew %d dots, want a hand", hold, hands(hold))
		}
	}

	// And where there is no room outside the face, there are no hands: a hand
	// drawn into the meter is a smudge.
	tight := p
	tight.reach = 4
	var n int
	tight.draw(faceLook{hold: [2]faceHold{faceHoldUp, faceHoldUp}}, func(_, _ int, at facePart) {
		if at == facePartHand {
			n++
		}
	})
	if n != 0 {
		t.Errorf("with no room outside him the hands still drew %d dots", n)
	}
}

// And when he goes out with both arms up, they go off as he does.
func TestHisHandsThrowTheWater(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.stage.drops = nil

	m.faceSparks(m.width, m.height)
	t.Logf("his hands threw %d drops", len(m.stage.drops))

	if len(m.stage.drops) != 2*faceSparkEach {
		t.Errorf("his hands threw %d drops, want %d", len(m.stage.drops), 2*faceSparkEach)
	}
	// Two handfuls, one from each side of him — of him, not of the screen,
	// since he may have walked halfway across it by the time he goes.
	p, at, _, ok := m.faceRoom(m.width, m.height)
	if !ok {
		t.Fatal("no room for him")
	}
	middle := at + p.w/2

	var left, right int
	for _, d := range m.stage.drops {
		if d.col < middle {
			left++
		} else {
			right++
		}
	}
	if left == 0 || right == 0 {
		t.Errorf("the drops came %d from the left and %d from the right, want both hands", left, right)
	}
	for _, d := range m.stage.drops {
		if d.speed <= 0 {
			t.Error("a drop was thrown downwards")
		}
	}
}

// The hands are always there. What they are doing is the music's: down at his
// sides through a quiet passage, up as it builds, and the fist opening into
// fingers on the way.
func TestHisArmsRideTheMusic(t *testing.T) {
	dotsX, dotsY := 320, 184
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		t.Fatal("no face")
	}
	p.reach = (dotsX - wide) / 2

	// How high the hands are drawn, and how much of them there is.
	at := func(lift float32) (int, int) {
		top, n := 1<<30, 0
		p.draw(faceLook{lift: lift}, func(_, y int, part facePart) {
			if part == facePartHand {
				top, n = min(top, y), n+1
			}
		})
		return top, n
	}

	quiet, quietDots := at(0)
	loud, loudDots := at(1)
	t.Logf("quiet: the hands reach row %d, %d dots; loud: row %d, %d dots", quiet, quietDots, loud, loudDots)

	if quietDots == 0 {
		t.Error("the hands are not drawn at all in a quiet passage, want them at his sides")
	}
	if loud >= quiet {
		t.Errorf("the hands reach row %d loud and %d quiet, want the music to lift them", loud, quiet)
	}
	if loudDots <= quietDots {
		t.Errorf("%d dots of hand loud against %d quiet, want the fist to open", loudDots, quietDots)
	}
}

// He always comes in from a side and leaves by one, and wanders about the
// screen in between. That is the whole shape of him: somebody walks on, does
// something and walks off.
func TestHeWalksOnAndOff(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	var back, through int
	for bar := range int64(60) {
		m.words.starts = bar * 7_000
		if in, out := m.faceWays(); in == out {
			back++
		} else {
			through++
		}
	}
	t.Logf("of sixty visits, %d walk straight through and %d turn back", through, back)
	if back == 0 || through == 0 {
		t.Error("every visit went the same way, want both")
	}

	// Across one visit: off the side at the start and the end when that is how
	// he is coming and going, and standing where he means to when it is his
	// dots that are doing the arriving. See figureWarp.
	for bar := range int64(60) {
		m.words.starts = bar * 7_000
		on, off := m.faceAt(0), m.faceAt(1)

		if figureSliding(m.figureComesBy()) && math.Abs(on) < 0.95 {
			t.Errorf("he walks on but starts at %.2f, want him off the screen", on)
		}
		if !figureSliding(m.figureComesBy()) && math.Abs(on) > faceRoam+0.01 {
			t.Errorf("he arrives some other way but starts at %.2f, want him on the screen", on)
		}
		if figureSliding(m.figureGoesBy()) && math.Abs(off) < 0.95 {
			t.Errorf("he walks off but ends at %.2f, want him off the screen", off)
		}
	}
	m.words.starts = 0

	// And in between he is on it, and he does not stand on one spot: he came in
	// to wander about, not to be put there. A walk through a row of marks goes
	// further out than a wander, because it has to reach the ends of the row.
	most := faceRoam
	if m.figureCrosses() {
		most = faceCross + faceStep
	}
	seen := map[int]bool{}
	for step := range 40 {
		at := m.faceAt(faceWalkIn + (1-faceWalkIn-faceWalkOut)*float64(step)/39)
		if math.Abs(at) > most+0.01 {
			t.Errorf("he wandered to %.2f, want him inside %.2f of the middle", at, most)
		}
		seen[int(at*20)] = true
	}
	t.Logf("while he was here he stood in %d different places", len(seen))
	if len(seen) < 3 {
		t.Error("he stood on one spot the whole time")
	}
}

// He has legs, and they march. He is drawn face on, so a walk is not two legs
// swinging sideways — from the front that is a pair of scissors — it is one
// foot up while the other is down.
func TestHeHasLegsThatMarch(t *testing.T) {
	dotsX, dotsY := 320, 184
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		t.Fatal("no face")
	}
	p.reach = (dotsX - wide) / 2

	// How far down each leg reaches, taking the two halves of him apart.
	feet := func(look faceLook) (int, int) {
		left, right := -1, -1
		p.draw(look, func(x, y int, part facePart) {
			if part != facePartLeg {
				return
			}
			if x < p.w/2 {
				left = max(left, y)
			} else {
				right = max(right, y)
			}
		})
		return left, right
	}

	standL, standR := feet(faceLook{})
	t.Logf("standing, his feet are at %d and %d", standL, standR)
	if standL < 0 || standR < 0 {
		t.Fatal("he has no legs")
	}
	if standL != standR {
		t.Errorf("standing still his feet are at %d and %d, want him level", standL, standR)
	}
	if standL <= p.lip.y+p.lip.h {
		t.Error("his legs do not reach below him")
	}

	// One step, then the other.
	oneL, oneR := feet(faceLook{stride: 1, facing: 1})
	twoL, twoR := feet(faceLook{stride: -1, facing: 1})
	t.Logf("mid stride: %d and %d; the other foot: %d and %d", oneL, oneR, twoL, twoR)

	if oneL >= oneR {
		t.Errorf("one way through the step his feet are at %d and %d, want the left one up", oneL, oneR)
	}
	if twoR >= twoL {
		t.Errorf("the other way they are at %d and %d, want the right one up", twoL, twoR)
	}
}

// And a nose, drawn the way somebody sketching a face would draw one: in
// profile, on a face that is otherwise front on. It lives in the gap between
// the eyes, clear of the mouth, and it lifts a shade as the mouth opens.
func TestHeHasANose(t *testing.T) {
	dotsX, dotsY := 320, 184
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		t.Fatal("no face")
	}
	p.reach = (dotsX - wide) / 2

	// What is drawn in the gap between the eyes, which is the nose's own room.
	from, to := p.eyes[0].x+p.eyes[0].w, p.eyes[1].x
	nose := func(look faceLook) (int, int, int) {
		top, bottom, n := 1<<30, -1, 0
		p.draw(look, func(_, y int, part facePart) {
			if part != facePartNose {
				return
			}
			top, bottom, n = min(top, y), max(bottom, y), n+1
		})
		return top, bottom, n
	}

	top, bottom, n := nose(faceLook{})
	t.Logf("the nose runs rows %d..%d, %d dots, in a gap %d wide", top, bottom, n, to-from)

	if n == 0 {
		t.Fatal("he has no nose")
	}
	if bottom >= p.lip.y {
		t.Errorf("the nose reaches row %d and the mouth starts at %d, want daylight between them", bottom, p.lip.y)
	}
	if bottom-top < p.stroke*3 {
		t.Errorf("the nose is %d dots deep, want a stroke rather than a speck", bottom-top)
	}

	// It lifts as the mouth opens.
	openTop, _, _ := nose(faceLook{mouth: 1})
	if openTop >= top {
		t.Errorf("with the mouth open the nose starts at %d against %d shut, want it lifting", openTop, top)
	}

	// And where the eyes leave it no room, there is no nose.
	tight := p
	tight.eyes[0].w = tight.eyes[1].x - tight.eyes[0].x - tight.stroke
	_ = from
	_ = to
	var still int
	tight.nose(faceLook{}, func(_, _ int, _ facePart) { still++ })
	if still != 0 {
		t.Errorf("with no gap between the eyes the nose drew %d dots", still)
	}
}

// How long he stays is the bar's business, and so is what he does while he is
// there — but when he does it is the music's.
func TestHowLongHeStaysIsDealt(t *testing.T) {
	m := scopeModel(160, 46)
	m.words.beats = true

	seen := map[time.Duration]int{}
	var least, most time.Duration = time.Hour, 0
	var his, theirs time.Duration
	for bar := range int64(60) {
		m.words.starts = bar * 7_000
		stay := m.faceStay()
		seen[stay/time.Second]++
		least, most = min(least, stay), max(most, stay)
		if faceWhoFor(m.words.starts) == "" {
			his = max(his, stay)
		} else {
			theirs = max(theirs, stay)
		}
	}
	t.Logf("sixty visits run from %s to %s, over %d different lengths", least, most, len(seen))

	if len(seen) < 3 {
		t.Errorf("the visits came in %d lengths, want them dealt", len(seen))
	}
	if least < faceStayLeast || most > time.Duration(faceStayMore*float64(faceStayMost)) {
		t.Errorf("a visit ran %s..%s, want it inside %s..%s",
			least, most, faceStayLeast, time.Duration(faceStayMore*float64(faceStayMost)))
	}

	// The one this code draws itself is worth more of a stay than a drawing:
	// he has a face and a pair of hands that answer the music, and one turn is
	// not enough of him.
	t.Logf("the longest of his own visits was %s, the longest of theirs %s", his, theirs)
	if his <= theirs {
		t.Errorf("his longest visit was %s and a drawing's %s, want him the longer", his, theirs)
	}

	// And the same bar is the same visit twice, while the bar after it is its
	// own: the length is dealt from the moment, not taken from a clock.
	once := m.faceStayFor(12_345)
	if again := m.faceStayFor(12_345); again != once {
		t.Errorf("one bar was dealt %s and then %s", once, again)
	}
	var alike int
	for bar := range int64(20) {
		if m.faceStayFor(bar*7_000)/time.Second == m.faceStayFor(bar*7_000+3_000)/time.Second {
			alike++
		}
	}
	if alike == 20 {
		t.Error("every neighbouring pair of bars was dealt the same length")
	}
}

// He takes his cue from the music: the same rise the meter throws its water on.
// And if the record never gives him one, he does something anyway rather than
// standing there — he came on to do a thing.
func TestHeTakesHisCueFromTheMusic(t *testing.T) {
	at := func(gone float64, rise bool) faceDoing {
		m := scopeModel(160, 46)
		m.stage.on = true
		m.scope.modes[tabPlayer] = scopeWords
		m.ps.Duration = 4 * time.Minute
		m.words.beats, m.words.text = true, wordsNotes

		m.words.starts = 21_000
		m.setProgress(time.Duration(m.words.starts)*time.Millisecond + faceEnters)

		// On, and standing: somebody sent for him a moment ago.
		summon(&m, m.words.starts)
		m.scope.envelope = 0.4
		m.faceFlow()

		// Further into the visit. With the key it is the press that his visit is
		// measured from, so that is what moves.
		m.face.shown = time.Now().Add(-time.Duration(gone * float64(faceShows)))
		if rise {
			m.scope.envelope = 1
		}
		m.faceFlow()
		return m.face.doing
	}

	if got := at(0.3, false); got != faceStill {
		t.Errorf("early in a quiet passage he is already doing %d, want him waiting for a cue", got)
	}
	if got := at(0.3, true); got == faceStill || got == faceBlinking {
		t.Errorf("the music gave him a cue and he did %d, want him taking it", got)
	}
	if got := at(0.6, false); got == faceStill {
		t.Error("the record never gave him a cue and he did nothing at all")
	}

	// And what he does changes as he does more of them, so a long stay is a
	// turn rather than the same trick over and over.
	seen := map[faceDoing]bool{}
	for turn := range 6 {
		seen[faceGagFor(7_000, turn)] = true
	}
	t.Logf("over six turns he does %d different things", len(seen))
	if len(seen) < 2 {
		t.Error("he does the same thing every time in one visit")
	}
}

// His nose turns the way he is going. Drawn in profile, it is the one part of
// him that says which way he is facing, so it had better agree with his feet.
func TestHisNoseFollowsHim(t *testing.T) {
	dotsX, dotsY := 320, 184
	high := int(wordsMark * float64(dotsY))
	wide := min(int(faceWide*float64(high)), int(0.62*float64(dotsX)))

	p, ok := faceLayout(wide, high)
	if !ok {
		t.Fatal("no face")
	}

	// How far the nose reaches either side of the middle of him.
	nose := func(facing float64) (int, int) {
		from, to := 1<<30, -1
		p.draw(faceLook{facing: facing}, func(x, _ int, part facePart) {
			if part == facePartNose {
				from, to = min(from, x), max(to, x)
			}
		})
		return from, to
	}

	leftFrom, leftTo := nose(-1)
	rightFrom, rightTo := nose(1)
	t.Logf("facing left the nose runs %d..%d, facing right %d..%d, on a face %d wide",
		leftFrom, leftTo, rightFrom, rightTo, p.w)

	if leftFrom >= rightFrom || leftTo >= rightTo {
		t.Error("the nose does not turn with him")
	}
	if leftTo > p.w/2+p.stroke {
		t.Errorf("facing left the nose still reaches %d, past his middle at %d", leftTo, p.w/2)
	}
	if rightFrom < p.w/2-p.stroke {
		t.Errorf("facing right the nose still reaches back to %d, past his middle at %d", rightFrom, p.w/2)
	}

	// And it agrees with where he is actually going.
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes
	summon(&m, 21_000)

	var moved, agreed int
	for step := range 60 {
		gone := float64(step) / 59

		// How far through the visit he is, which with the key is measured from
		// the press rather than from the bar.
		m.face.shown = time.Now().Add(-time.Duration(gone * float64(faceShows)))

		was, is := m.faceAt(gone-0.01), m.faceAt(gone+0.01)
		if math.Abs(is-was) < 0.01 {
			continue
		}
		moved++
		if math.Signbit(is-was) == math.Signbit(m.faceNow().facing) {
			agreed++
		}
	}
	t.Logf("he was moving at %d of sixty moments, facing the right way at %d", moved, agreed)
	if moved == 0 {
		t.Fatal("he never moved")
	}
	if agreed < moved {
		t.Errorf("he faced the way he was going at %d of %d moments", agreed, moved)
	}
}

// A visit that has begun runs to its end, whatever the bar underneath it does.
//
// The bar is what deals him, and a record with no words of its own stamps a new
// one every half minute — so a figure who had just walked on was taken off
// mid-stride by a stamp he had not been dealt, and the marks gathered over the
// top of him. Which is a picture cutting to another picture, and looks like it.
func TestAVisitSurvivesTheBarChangingUnderIt(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 20 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// A bar, and him on it because somebody sent for him.
	var bar int64 = 21_000
	m.setProgress(time.Duration(bar)*time.Millisecond + faceEnters + time.Second)
	summon(&m, bar)
	m.faceFlow()

	if !m.faceUp() {
		t.Fatal("he never came on")
	}
	t.Logf("he is on, a second into a stay of %s", m.faceStayFor(bar))

	// And now the bar changes under him — which is what a wordless record does
	// every half minute, whether or not anybody has sent for a figure.
	m.words.starts = bar + wordsSpell.Milliseconds()

	if !m.faceUp() {
		t.Error("the bar changed under him and he vanished mid-visit")
	}

	// He goes when his own visit is over, not before.
	m.face.shown = time.Now().Add(-faceShows - time.Second)
	m.face.came = time.Now().Add(-m.faceStayFor(bar) - time.Second)
	if m.faceUp() {
		t.Error("he is still there long after his own stay ran out")
	}
}
