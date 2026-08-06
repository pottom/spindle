package ui

import (
	"math"
	"sort"
)

// The figures: small line drawings that walk about the screen.
//
// They are not drawn by this code. They come from pictures somebody else drew,
// converted to dots by cmd/spindle-figures and written into figures_gen.go —
// so what happens at playback time is a lookup and a blit, and adding another
// figure is a folder of drawings and a ten line manifest.
//
// What the drawing does not carry is a face. A still picture cannot blink, and
// the one thing this screen is for is answering the music, so the head is left
// hollow by the generator and the face is drawn into it — the same eyes, brows
// and mouth that were there before, following the same sound.

// figureDrawing is one figure, at every size it was made for.
type figureDrawing struct {
	from    string
	licence string
	sizes   []figureSize
}

// figureSize is a figure at one height.
type figureSize struct {
	tall  int
	poses map[string]figurePose
}

// figurePose is one drawing: the dots, and where its head is.
type figurePose struct {
	wide, tall                 int
	headX, headY, headW, headH int
	rows                       []string
}

// figureFor is the drawing of a given name, and whether there is one.
func figureFor(name string) (figureDrawing, bool) {
	d, ok := figures[name]
	return d, ok
}

// at is the size of a figure closest to a wanted height, and the pose from it.
func (d figureDrawing) at(tall int, pose string) (figurePose, bool) {
	if len(d.sizes) == 0 {
		return figurePose{}, false
	}

	best := d.sizes[0]
	for _, size := range d.sizes[1:] {
		if abs(size.tall-tall) < abs(best.tall-tall) {
			best = size
		}
	}
	p, ok := best.poses[pose]
	return p, ok
}

// draw lights the pose's dots. The head is not drawn: it was cleared when the
// figure was made, and what goes in it is the caller's business.
func (p figurePose) draw(light func(x, y int)) {
	for y, row := range p.rows {
		for x, on := range row {
			if on == '#' {
				light(x, y)
			}
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// figureTall is how much of the screen a figure stands in, as a share of its
// height. Taller than the marks he stands in for: he is a whole person rather
// than a line of type, and the meters take what he leaves.
const figureTall = 0.55

// figureLines draws the figure who is on, with the face drawn into his head.
//
// The drawing gives the body, the arms, the legs and the walk; it cannot give
// an expression, because it is a still picture. So the head comes back hollow
// from the generator and the face goes in it — the same eyes and mouth that
// answer the music everywhere else on this screen.
func (m Model) figureLines(w, rows int) []string {
	who, ok := figureFor(m.faceWho())
	if !ok || w <= 0 || rows <= 0 || len(m.styles.Words) == 0 {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	levels, freqs := len(m.styles.Words[0]), len(m.styles.Words)

	pose, ok := who.at(int(figureTall*float64(dotsY)), m.figurePose())
	if !ok {
		return nil
	}

	// Where he stands: his own walk across the screen, and his feet on the foot
	// of the band the marks are set in, so the meter below him is the meter a
	// bar of notes leaves.
	room := (dotsX - pose.wide) / 2
	left := room + int(m.faceWalk()*float64(room+pose.wide))
	top := (dotsY-int(wordsMark*float64(dotsY)))/2 + int(wordsMark*float64(dotsY)) - pose.tall

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}

	var part [faceParts_]wordPaint
	for i := range part {
		part[i] = m.wordsBeatPaint(int(i), int(faceParts_), freqs, levels)
	}

	light := func(x, y int, at facePart) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if s := part[at]; s.level > paint[cell] {
			paint[cell], hue[cell] = s.level, s.hue
		}
	}

	pose.draw(func(x, y int) { light(x+left, y+top, facePartBody) })

	// And a face, if the figure left a hole for one. A figure drawn with a face
	// of its own does not want a second pair of eyes over it — what changes his
	// expression then is the pose, not something drawn on top.
	if p, ok := faceLayout(pose.headW, pose.headH); ok && pose.headW > 0 {
		p.reach = 0 // his own arms are in the drawing; he does not want two pairs
		p.draw(m.faceNow(), func(x, y int, at facePart) {
			light(x+left+pose.headX, y+top+pose.headY, at)
		})
	}

	if tall := max((dotsY-(top+pose.tall))/dotsPerCellY, 0); tall >= wordsBand {
		m.wordsUnder(grid, paint, hue, w, rows, tall, max(top-dotsPerCellY, 0))
	}
	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}

// figurePose is the drawing he is in this frame: walking through the cycle as
// he moves, and whatever he is doing when he has stopped.
func (m Model) figurePose() string {
	if _, moving := m.faceGoing(); moving {
		at := int(math.Abs(math.Sin(math.Pi*faceSteps*m.faceGone()))*4) % 4
		if m.faceWalk() > 0 {
			at += 4
		}
		return "walk" + string(rune('0'+at))
	}

	switch m.face.doing {
	case faceGaping:
		return "cheer"
	case faceBrowing:
		return "show"
	case faceWinking:
		return "talk"
	}
	return "idle"
}

// faceWho is which figure is on: one of the drawn ones, or nothing at all,
// which is the one this code draws itself.
//
// Dealt from the bar, so a record turns up the same twice, and so that the two
// kinds are both seen — the drawn figure has a body and a walk the geometry
// cannot match, and the geometry has a face and a pair of hands that answer the
// music in a way a still drawing never will.
func (m Model) faceWho() string {
	if len(figures) == 0 {
		return ""
	}

	// For now, always one of the drawn ones: the geometry is kept for the size
	// where no drawing fits and for the day another is wanted, but it is not
	// dealt while the figures are being looked at.
	h := uint64(m.words.starts)*0x94d049bb133111eb + 0xd6e8feb86659fd93
	h ^= h >> 30
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 27

	names := make([]string, 0, len(figures))
	for name := range figures {
		names = append(names, name)
	}
	sort.Strings(names)
	return names[int(h>>8)%len(names)]
}
