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
	p.draw(look, 1, func(x, y int, _ facePart) {
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

// The face takes the marks' place on some bars, and it can be asked for.
func TestTheFaceTakesTheMarksPlace(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords

	// A bar of marks, which is the only thing a face may stand in for.
	m.words.beats, m.words.text = true, wordsNotes

	var faces int
	for bar := range int64(30) {
		m.words.starts = bar * 7_000
		if m.faceUp() {
			faces++
		}
	}
	t.Logf("%d of thirty bars are given a face", faces)
	if faces == 0 || faces == 30 {
		t.Errorf("%d of thirty bars are given a face, want some of them", faces)
	}

	// A line of words never is.
	m.words.beats, m.words.text = false, "a line of the song"
	if m.faceUp() {
		t.Error("a face was put over a line that is being sung")
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

// The face is drawn on rather than pasted up: a stroke at a time, in the order
// somebody would draw one — the eyes, then the brows, then the mouth.
func TestTheFaceDrawsItselfOn(t *testing.T) {
	const w, h = 124, 62

	lit := func(grow float64) int {
		p, ok := faceLayout(w, h)
		if !ok {
			t.Fatal("no face")
		}
		var n int
		seen := map[int]bool{}
		p.draw(faceLook{}, grow, func(x, y int, _ facePart) {
			if x >= 0 && y >= 0 && x < w && y < h && !seen[y*w+x] {
				seen[y*w+x] = true
				n++
			}
		})
		return n
	}

	was := 0
	for _, grow := range []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1} {
		now := lit(grow)
		t.Logf("%.0f%% drawn: %d dots", grow*100, now)
		if now < was {
			t.Errorf("at %.0f%% the face is %d dots against %d before it, want it only arriving", grow*100, now, was)
		}
		was = now
	}

	if full, part := lit(1), lit(0.4); part >= full {
		t.Errorf("the face is %d dots part way in and %d finished, want it still coming", part, full)
	}
	if lit(0.02) == 0 {
		t.Error("nothing at all is drawn as the face starts")
	}
}

// A wink happens to a bar, not to a clock: some bars are given one and the same
// bar is given it again.
func TestSomeBarsWink(t *testing.T) {
	var winks int
	for bar := range int64(60) {
		if faceWinks(bar * 7_000) {
			winks++
		}
	}
	t.Logf("%d of sixty bars wink", winks)
	if winks == 0 || winks == 60 {
		t.Errorf("%d of sixty bars wink, want some of them", winks)
	}
	// And it is the bar that decides, not the moment it is asked: two bars a
	// second apart do not have to agree, and one bar always does with itself.
	var same int
	for bar := range int64(20) {
		if faceWinks(bar*7_000) == faceWinks(bar*7_000+1_000) {
			same++
		}
	}
	if same == 20 {
		t.Error("every neighbouring pair of bars answered alike, so nothing is being decided")
	}
}

// A face arrives the way everything else on this screen arrives — one of the
// ways a line of the song is dealt — and now and again by drawing itself on,
// which is its own.
func TestAFaceArrivesADifferentWay(t *testing.T) {
	m := scopeModel(160, 46)
	m.words.beats = true

	seen := map[wordsMove]int{}
	var drawn int
	for bar := range int64(60) {
		m.words.starts = bar * 7_000
		move, on := m.faceComing()
		if on {
			drawn++
			continue
		}
		seen[move]++
	}
	t.Logf("sixty bars: %d drawn on, and %d different ways in besides", drawn, len(seen))

	if drawn == 0 || drawn == 60 {
		t.Errorf("%d of sixty faces draw themselves on, want some of them", drawn)
	}
	if len(seen) < 4 {
		t.Errorf("the faces came in %d different ways, want the screen's whole vocabulary", len(seen))
	}

	// And a bar answers the same way twice, so a record plays the same twice.
	m.words.starts = 12_345
	one, oneOn := m.faceComing()
	two, twoOn := m.faceComing()
	if one != two || oneOn != twoOn {
		t.Error("one bar was dealt two arrivals")
	}
}

// And it leaves the way it came, handed to the same machinery that carries a
// line of the song off the screen.
func TestAFaceLeavesTheWayItCame(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.words.beats, m.words.text = true, wordsNotes

	// A bar with a face on it, and then one without.
	for bar := range int64(60) {
		m.words.starts = bar * 7_000
		if faceDealt(m.words.starts) {
			break
		}
	}
	if !m.faceUp() {
		t.Skip("no bar in the run was dealt a face")
	}
	m.faceFlow()

	for bar := range int64(60) {
		if at := bar * 7_000; !faceDealt(at) {
			m.words.starts = at
			break
		}
	}
	if m.faceUp() {
		t.Fatal("the next bar has a face as well")
	}

	m.faceFlow()
	if m.words.was.DotsX != m.width*dotsPerCellX {
		t.Fatalf("the face left nothing behind to carry off: %d dots wide", m.words.was.DotsX)
	}
	if time.Since(m.words.went) > time.Second {
		t.Error("the face was not given notice as it went")
	}

	var lit int
	for _, v := range m.words.was.Lum {
		if v > 0 {
			lit++
		}
	}
	t.Logf("the face left %d dots to be carried off", lit)
	if lit == 0 {
		t.Error("what was handed on is empty")
	}
}
