package ui

import "math"

// The chase: what walks across the screen while nobody is singing.
//
// The faces put up through a solo are drawn from type, which is why there are a
// dozen of them and why they stand still. This one is not — it is drawn out of
// the dots directly, so it can walk, and it eats its way across the screen from
// one side to the other on a line of pellets.
//
// It is one of the set: a bar of a song with no words gets a face or gets this,
// decided by the bar's own timestamp, so a record plays the same way twice and
// the same solo is not the same joke every time.

const (
	// chaseSpeed is how far it travels in a frame, in dots. About twenty-five a
	// second, which crosses an ordinary terminal in the length of a solo.
	chaseSpeed = 0.85

	// chaseSize is how tall it is, as a share of the picture. Big enough to be
	// the thing on the screen, small enough that the pellets have somewhere to
	// be.
	chaseSize = 0.22

	// The mouth. chaseBite is how wide it opens at its widest, in radians, and
	// chaseChomp how fast it works when nothing is playing — the loud parts of
	// the music open it further, so it chews to the beat.
	chaseBite  = 1.0
	chaseChomp = 0.34

	// chasePellets is how far apart the pellets are, in dots, and chaseEaten how
	// close the mouth has to be to take one.
	chasePellets = 14
	chaseEaten   = 6
)

// chaseState is where it has got to.
type chaseState struct {
	// on says the chase is what the screen is showing.
	on bool

	// at is how far across it is, in dots, and back says it came in from the
	// right and is walking the other way.
	at   float32
	back bool

	// mouth is where the chewing has got to, and eaten how many pellets are
	// gone from the near end of the line.
	mouth float32
	eaten int
}

// chaseFor decides whether a bar with no words gets the chase rather than a
// face, and which way it walks. One bar in three, so it stays a surprise.
func chaseFor(at int64) (chase, back bool) {
	h := uint64(at)*0x9e3779b97f4a7c15 + 0x94d049bb133111eb
	h ^= h >> 31
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	return h%3 == 0, h&(1<<40) != 0
}

// chaseFlow walks it on by a frame.
func (m *Model) chaseFlow(w, rows int) {
	dotsX := w * dotsPerCellX
	if dotsX <= 0 {
		return
	}

	// The mouth works faster the louder it is: a solo that is going somewhere
	// gets chewed harder than a quiet one.
	var loud float32
	if len(m.scope.bands) > 0 {
		for _, v := range m.scope.bands {
			loud = max(loud, v)
		}
	}
	m.chase.mouth += chaseChomp * (0.6 + loud)

	m.chase.at += chaseSpeed
	if m.chase.at > float32(dotsX)+float32(rows*dotsPerCellY) {
		// Off the far side: round it comes again, the other way about.
		m.chase.at = 0
		m.chase.back = !m.chase.back
		m.chase.eaten = 0
	}

	// Whatever it has walked past is eaten.
	m.chase.eaten = max(m.chase.eaten, int((m.chase.at+chaseEaten)/chasePellets))
}

// chaseLines draws it, w cells across and rows deep.
func (m Model) chaseLines(w, rows int) []string {
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

	light := func(x, y int, step int8) {
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

	radius := float32(dotsY) * chaseSize
	middle := float32(dotsY) / 2

	// Where it is, and which way it faces. Walking back is the same walk with
	// the screen read the other way round.
	head := m.chase.at + radius
	if m.chase.back {
		head = float32(dotsX) - head
	}

	// The pellets it has not reached yet.
	for i := m.chase.eaten + 1; ; i++ {
		x := float32(i * chasePellets)
		if x > float32(dotsX) {
			break
		}
		if m.chase.back {
			x = float32(dotsX) - x
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				light(int(x)+dx, int(middle)+dy, int8(levels-3))
			}
		}
	}

	// The mouth: open and shut, wider the louder it is.
	gape := chaseBite * float64(0.5+0.5*math.Sin(float64(m.chase.mouth)))

	for y := -int(radius); y <= int(radius); y++ {
		for x := -int(radius); x <= int(radius); x++ {
			d := math.Hypot(float64(x), float64(y))
			if d > float64(radius) {
				continue
			}

			// The wedge it has bitten out, on the side it is walking towards.
			angle := math.Atan2(float64(y), float64(x))
			if m.chase.back {
				angle = math.Atan2(float64(y), -float64(x))
			}
			if math.Abs(angle) < gape {
				continue
			}

			// Brightest at the rim, so it reads as a body rather than a blob.
			step := int8(min(int(d/float64(radius)*float64(levels))+1, levels-1))
			light(int(head)+x, int(middle)+y, step)
		}
	}

	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}
