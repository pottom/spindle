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
