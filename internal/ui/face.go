package ui

import "math"

// The faces: what goes up while nobody is singing.
//
// They were set in type to begin with — a colon and a bracket, blown up to two
// hundred dots — and that is exactly what it looked like. These are drawn
// instead, out of the same dots as everything else on the screen: a head, eyes
// that blink, a mouth that answers the music. A record's solo is worth more than
// a piece of punctuation.

// faceKind is which one is up.
type faceKind int

const (
	faceSmile faceKind = iota
	faceWink
	faceKiss
	faceCheer
	faceCool
	faceKinds
)

const (
	// faceSize is how much of the picture's height the head takes.
	faceSize = 0.34

	// faceEye is the size of an eye, faceEyeApart how far apart they sit and
	// faceEyeUp how far above the middle — all as shares of the head's radius.
	faceEye      = 0.17
	faceEyeApart = 0.42
	faceEyeUp    = 0.28

	// faceLine is how thick a drawn stroke is, in dots.
	faceLine = 2.2

	// A blink is rare and quick: faceBlinkEvery is how many frames apart they
	// are on average and faceBlinkFor how many the eye stays shut.
	faceBlinkEvery = 150
	faceBlinkFor   = 5

	// faceMouth is how far the mouth opens with the music, as a share of the
	// head, on top of the smile it always has.
	faceMouth = 0.5
)

// faceState is what the drawn face carries between frames.
type faceState struct {
	kind faceKind

	// tick counts frames, for the blink and for anything else that moves on
	// its own rather than with the music.
	tick int

	// blink is how many frames of the eyes being shut are left.
	blink int

	// open is how far the mouth is open, following the music.
	open float32
}

// faceFor picks which face a bar with no words gets, from the bar's own
// timestamp, so a record plays the same way twice.
func faceFor(at int64) faceKind {
	h := uint64(at)*2654435761 + 2246822519
	h ^= h >> 29
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 32
	return faceKind(h % uint64(faceKinds))
}

// faceFlow moves it on by a frame: the eyes blink now and again, and the mouth
// follows how loud the music is.
func (m *Model) faceFlow() {
	m.face.tick++

	if m.face.blink > 0 {
		m.face.blink--
	} else if m.face.tick%faceBlinkEvery == 0 {
		m.face.blink = faceBlinkFor
	}

	var loud float32
	for _, v := range m.scope.bands {
		loud = max(loud, v)
	}
	// Quick to open and slower to close, the way a mouth is.
	rate := float32(0.2)
	if loud > m.face.open {
		rate = 0.5
	}
	m.face.open += (loud - m.face.open) * rate
}

// faceLines draws it, w cells across and rows deep.
func (m Model) faceLines(w, rows int) []string {
	if w <= 0 || rows <= 0 || len(m.styles.Words) == 0 {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	levels, freqs := len(m.styles.Words[0]), len(m.styles.Words)

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}

	ink := func(x, y int, step int8) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if step > paint[cell] {
			paint[cell] = step
			hue[cell] = int8(min(x/dotsPerCellX*freqs/w, freqs-1))
		}
	}

	p := &pen{ink: ink, hot: int8(levels - 1), warm: int8(levels - 3)}

	r := float64(dotsY) * faceSize
	cx, cy := float64(dotsX)/2, float64(dotsY)/2
	if m.face.kind == faceCheer {
		// The cheering one is a whole figure, so its head sits higher.
		cy = float64(dotsY) * 0.34
		r *= 0.62
	}

	m.faceDraw(p, cx, cy, r)
	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}

// faceDraw puts one on the page.
func (m Model) faceDraw(p *pen, cx, cy, r float64) {
	shut := m.face.blink > 0

	switch m.face.kind {
	case faceCheer:
		// A figure with its arms up: the one the screen has always liked best.
		p.ring(cx, cy, r)
		p.line(cx, cy+r, cx, cy+r*2.6)                 // the body
		p.line(cx, cy+r*1.4, cx-r*1.5, cy-r*0.4)       // the arms
		p.line(cx, cy+r*1.4, cx+r*1.5, cy-r*0.4)       //
		p.line(cx, cy+r*2.6, cx-r*1.1, cy+r*3.8)       // the legs
		p.line(cx, cy+r*2.6, cx+r*1.1, cy+r*3.8)       //
		p.dot(cx-r*faceEyeApart, cy-r*faceEyeUp, r*faceEye*0.8)
		p.dot(cx+r*faceEyeApart, cy-r*faceEyeUp, r*faceEye*0.8)
		p.arc(cx, cy+r*0.05, r*0.55, 0.35, math.Pi-0.35)
		return

	case faceWink:
		p.ring(cx, cy, r)
		// One eye shut, drawn as the line a shut eye is.
		p.line(cx-r*faceEyeApart-r*faceEye, cy-r*faceEyeUp, cx-r*faceEyeApart+r*faceEye, cy-r*faceEyeUp)
		p.dot(cx+r*faceEyeApart, cy-r*faceEyeUp, r*faceEye)
		// And a mouth pulled up on the winking side.
		p.arc(cx, cy+r*0.08, r*0.55, 0.15, math.Pi-0.55)
		return

	case faceKiss:
		p.ring(cx, cy, r)
		p.eyes(cx, cy, r, shut)
		// Pursed: a small ring rather than a smile, and a heart leaving it.
		p.ring(cx-r*0.1, cy+r*0.42, r*0.16)
		p.heart(cx+r*0.55, cy+r*0.3-float64(m.face.tick%40)*r*0.02, r*0.2)
		return

	case faceCool:
		p.ring(cx, cy, r)
		// Dark glasses: two filled lenses and the bridge between them.
		p.disc(cx-r*faceEyeApart, cy-r*faceEyeUp, r*0.3)
		p.disc(cx+r*faceEyeApart, cy-r*faceEyeUp, r*0.3)
		p.line(cx-r*0.15, cy-r*faceEyeUp, cx+r*0.15, cy-r*faceEyeUp)
		p.arc(cx, cy+r*0.05, r*0.5, 0.3, math.Pi-0.3)
		return
	}

	// The plain one, whose mouth opens with the music.
	p.ring(cx, cy, r)
	p.eyes(cx, cy, r, shut)

	open := float64(m.face.open) * faceMouth
	if open < 0.12 {
		p.arc(cx, cy+r*0.05, r*0.55, 0.25, math.Pi-0.25)
		return
	}
	// Singing along: an open mouth, taller the louder it is.
	p.oval(cx, cy+r*0.4, r*0.34, r*(0.18+open*0.5))
}

// pen draws shapes out of dots. Everything here is a circle, an arc or a line,
// which between them are a face.
type pen struct {
	ink        func(x, y int, step int8)
	hot, warm  int8
}

func (p *pen) at(x, y float64, step int8) { p.ink(int(math.Round(x)), int(math.Round(y)), step) }

// ring is a circle drawn as an outline.
func (p *pen) ring(cx, cy, r float64) {
	steps := int(2 * math.Pi * r * 2)
	for i := range steps {
		a := 2 * math.Pi * float64(i) / float64(steps)
		p.thick(cx+math.Cos(a)*r, cy+math.Sin(a)*r, p.hot)
	}
}

// arc is part of one, from one angle to another, measured downwards — which is
// what makes a smile.
func (p *pen) arc(cx, cy, r, from, to float64) {
	steps := int(r * 4)
	for i := range steps + 1 {
		a := from + (to-from)*float64(i)/float64(steps)
		p.thick(cx+math.Cos(a)*r, cy+math.Sin(a)*r, p.hot)
	}
}

// disc is a circle filled in.
func (p *pen) disc(cx, cy, r float64) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if math.Hypot(x, y) <= r {
				p.at(cx+x, cy+y, p.hot)
			}
		}
	}
}

// oval is a disc that is not round: a mouth open to sing.
func (p *pen) oval(cx, cy, rx, ry float64) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			if x*x/(rx*rx)+y*y/(ry*ry) <= 1 {
				p.at(cx+x, cy+y, p.hot)
			}
		}
	}
}

func (p *pen) dot(cx, cy, r float64) { p.disc(cx, cy, r) }

// eyes are the pair of them, open or shut.
func (p *pen) eyes(cx, cy, r float64, shut bool) {
	for _, side := range []float64{-1, 1} {
		x, y := cx+side*r*faceEyeApart, cy-r*faceEyeUp
		if shut {
			p.line(x-r*faceEye, y, x+r*faceEye, y)
			continue
		}
		p.dot(x, y, r*faceEye)
	}
}

// line is a straight stroke between two points.
func (p *pen) line(x0, y0, x1, y1 float64) {
	steps := int(math.Hypot(x1-x0, y1-y0) * 2)
	for i := range steps + 1 {
		t := float64(i) / float64(max(steps, 1))
		p.thick(x0+(x1-x0)*t, y0+(y1-y0)*t, p.hot)
	}
}

// heart is exactly what it sounds like, and is only ever thrown by one of them.
func (p *pen) heart(cx, cy, r float64) {
	steps := int(r * 12)
	for i := range steps {
		t := 2 * math.Pi * float64(i) / float64(steps)
		x := 16 * math.Pow(math.Sin(t), 3)
		y := -(13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t))
		p.thick(cx+x*r/16, cy+y*r/16, p.warm)
	}
}

// thick lays a stroke down a few dots wide, so a line reads as a line rather
// than as a row of specks.
func (p *pen) thick(x, y float64, step int8) {
	for dy := -faceLine / 2; dy <= faceLine/2; dy++ {
		for dx := -faceLine / 2; dx <= faceLine/2; dx++ {
			p.at(x+dx, y+dy, step)
		}
	}
}
