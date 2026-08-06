package ui

import (
	"strings"
	"testing"
	"time"
)

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

	tall := func(shut float32) int {
		dots := faceDots(t, w, h, faceLook{lid: [2]float32{shut, shut}})
		p, _ := faceLayout(w, h)
		top, bottom := -1, -1
		for y := p.eyes[0].y - p.stroke; y <= p.eyes[0].y+p.eyes[0].h+p.stroke && y < h; y++ {
			if y < 0 {
				continue
			}
			if strings.Contains(dots[y][:w/2], "#") {
				if top < 0 {
					top = y
				}
				bottom = y
			}
		}
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

// He is not what a bar of music looks like — the marks are that. He turns up in
// the middle of one, does his thing and goes, and the marks have the bar back.
func TestHeVisitsABarRatherThanTakingIt(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// A bar he was dealt.
	var bar int64 = -1
	for at := range int64(60) {
		if faceDealt(at * 7_000) {
			bar = at * 7_000
			break
		}
	}
	if bar < 0 {
		t.Fatal("no bar in sixty was dealt him")
	}
	m.words.starts = bar

	at := func(into time.Duration) bool {
		m.setProgress(time.Duration(bar)*time.Millisecond + into)
		return m.faceUp()
	}

	if at(0) {
		t.Error("he is there the moment the bar starts, want the marks first")
	}
	if !at(faceEnters + time.Second) {
		t.Error("he never turns up in a bar he was dealt")
	}
	if at(faceEnters + faceStays + time.Second) {
		t.Error("he is still there long after his turn, want the marks back")
	}

	// And the bars he was not dealt are the marks' own, all the way through.
	for _, other := range []int64{bar + 7_000, bar + 14_000} {
		if faceDealt(other) {
			continue
		}
		m.words.starts = other
		m.setProgress(time.Duration(other)*time.Millisecond + faceEnters + time.Second)
		if m.faceUp() {
			t.Error("he turned up in a bar he was not dealt")
		}
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
		gag := faceGagFor(bar * 7_000)
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

// And it is on a key, because a wink lasts a third of a second and happens once
// in a solo, which is no way to look at one on purpose.
func TestTheFaceCanBeAskedFor(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.words.beats, m.words.text = true, wordsNotes
	m.words.starts = 1 // a bar that is not dealt a face

	if faceDealt(m.words.starts) {
		t.Skip("that bar happens to be dealt a face anyway")
	}
	if m.faceUp() {
		t.Fatal("the face was up before it was asked for")
	}

	m.faceShow()
	if !m.faceUp() {
		t.Fatal("the face was asked for and did not come")
	}

	// Pressing again walks the expressions rather than putting it away.
	seen := map[faceDoing]bool{}
	for range int(faceDoings) + 1 {
		m.faceShow()
		seen[m.face.doing] = true
	}
	t.Logf("the key walked through %d expressions", len(seen))
	if len(seen) < int(faceDoings)-1 {
		t.Errorf("the key showed %d of the %d expressions", len(seen), faceDoings-1)
	}
	if seen[faceStill] {
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

// He always comes in from a side and leaves by one. That is the whole shape of
// him: somebody walks on, does something and walks off.
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

	// Across one visit: off the side he came from, in the middle for his turn,
	// and off a side again.
	m.words.starts = 0
	where := func(t float64) int {
		into := faceEnters + time.Duration(t*float64(faceStays))
		m.setProgress(time.Duration(m.words.starts)*time.Millisecond + into)
		_, at, _, _ := m.faceRoom(m.width, m.height)
		return at
	}

	on, middle, off := where(0), where(0.5), where(1)
	p, _, _, _ := m.faceRoom(m.width, m.height)
	room := (m.width*dotsPerCellX - p.w) / 2
	t.Logf("he walks %d → %d → %d, standing at %d", on, middle, off, room)

	if middle != room {
		t.Errorf("he does his turn at %d, want the middle at %d", middle, room)
	}
	for _, at := range []int{on, off} {
		if at+p.w > 0 && at < m.width*dotsPerCellX {
			t.Errorf("he is at %d at one end of his visit, want him off the screen", at)
		}
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
	oneL, oneR := feet(faceLook{stride: 1, going: 1})
	twoL, twoR := feet(faceLook{stride: -1, going: 1})
	t.Logf("mid stride: %d and %d; the other foot: %d and %d", oneL, oneR, twoL, twoR)

	if oneL >= oneR {
		t.Errorf("one way through the step his feet are at %d and %d, want the left one up", oneL, oneR)
	}
	if twoR >= twoL {
		t.Errorf("the other way they are at %d and %d, want the right one up", twoL, twoR)
	}
}
