package ui

import "math"

// Sparks: what the trace throws off when it moves.
//
// The waveform is a beam, and a beam that jumps carries something with it. When
// a column's swing rises sharply — a snare, a pluck, the front of a note — a
// bead leaves the crest at the speed the crest gained, arcs away from the centre
// line and is pulled back to it. It lands where the trace will be, so the eye
// follows the hit twice: once as the wave, once as what came off it.
//
// The same arithmetic as the water on the big screen, mirrored about the line
// the trace rests on rather than about the middle of the picture, and thrown by
// a swing rather than by a band.

const (
	// What decides a throw is how much louder the music just got, not how much
	// taller the trace did: the trace is scaled to the recent loudness, so a
	// snare that doubles the signal draws a wave the same height as the one
	// before it. Measured on a live stream, the normalised swing of a column
	// barely moves through a hit — the loudness underneath it is what moves.
	//
	// sparkIdle is the chance a crest sheds something on an ordinary frame, and
	// sparkHit how much a rise in loudness multiplies that. Together: a steady
	// passage glitters faintly along its peaks, a hit throws a handful.
	sparkIdle = 0.06
	sparkHit  = 5.0

	// sparkThrow is how hard a throw goes, per root of the half-height it is
	// drawn in — so the arc is the same fraction of the picture in the strip
	// under the artwork as on the whole screen.
	sparkThrow = 0.55

	// sparkLift is how hard an ordinary spark leaves, and sparkHeave how much a
	// rise in loudness adds to it. The first is set by the arithmetic of the
	// arc: a bead rises by its speed squared over twice the gravity, and in the
	// strip under the artwork — eight dot rows from the line to the edge — this
	// carries it about three of them, which is far enough to be a spark and near
	// enough to still belong to the trace.
	sparkLift  = 0.7
	sparkHeave = 1.5

	// sparkGravity is what pulls a bead back to the line, in dot rows a frame.
	sparkGravity = 0.2

	// sparkDim is what one keeps of its light each frame. Shorter-lived than
	// the water on the big screen: this is a spark off a beam, not a drop.
	sparkDim = 0.93

	// sparkSpray is the share of the columns that throw when they jump. All of
	// them would be a second waveform drawn above the first.
	sparkSpray = 0.45

	// sparkMost is how many can be in the air at once.
	sparkMost = 512
)

// spark is one bead: which cell column it left, how far it is from the centre
// line, which side of it, how fast it is going and how brightly it burns.
type spark struct {
	col    int
	side   int8
	at     float32
	speed  float32
	bright float32
}

// throwSparks advances what is in the air and lets the trace throw more.
//
// It is a step of a simulation rather than a drawing, so it happens in the
// update loop; View stays a pure function of what it leaves behind.
func (m *Model) throwSparks(w, rows int) {
	if w <= 0 || rows <= 0 {
		m.scope.sparks = nil
		return
	}

	half := float32(rows*dotsPerCellY) / 2

	kept := m.scope.sparks[:0]
	for _, s := range m.scope.sparks {
		s.speed -= sparkGravity
		s.at += s.speed
		s.bright *= sparkDim

		// Gone when it falls back onto the line, leaves the picture, or has
		// nothing left to see.
		if s.at > 0 && s.at < half && s.bright > 0.08 && s.col < w {
			kept = append(kept, s)
		}
	}
	m.scope.sparks = kept

	// How much louder it just got, as a share of where the level stands.
	rise := max(m.scope.envelope-m.scope.wasLoud, 0) / max(m.scope.envelope, scopeFloor)
	m.scope.wasLoud = m.scope.envelope

	throw := sparkThrow * float32(math.Sqrt(float64(half)))
	for c, now := range m.scopeCrest(w) {
		if len(m.scope.sparks) >= sparkMost {
			break
		}

		// The tallest crests throw the most: squared, so the peaks of a wave
		// spray and the flat of it does not.
		swing := abs32(now)
		chance := sparkSpray * swing * swing * (sparkIdle + rise*sparkHit)
		if m.scope.roll() > chance {
			continue
		}

		side := int8(1)
		if now < 0 {
			side = -1
		}
		m.scope.sparks = append(m.scope.sparks, spark{
			col:    c,
			side:   side,
			at:     swing * half * scopeDeflection,
			speed:  throw * (sparkLift + rise*sparkHeave),
			bright: min(swing+rise, 1),
		})
	}
}

// scopeCrest is the furthest the trace swings under each cell, with the sign of
// which way it went — the crest a spark would leave from.
func (m Model) scopeCrest(w int) []float32 {
	dotsX := w * dotsPerCellX
	start := m.scopeTrigger(dotsX)

	out := make([]float32, w)
	for x := range dotsX {
		y := m.scopeSample(start, x, dotsX)
		if at := x / dotsPerCellX; abs32(y) > abs32(out[at]) {
			out[at] = y
		}
	}
	return out
}

// sparkGrid draws what is in the air: which dots the beads light, and how
// brightly, for a picture w cells across and rows deep.
func (m Model) sparkGrid(w, rows, levels int) ([]uint8, []int8) {
	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}

	dotsY := rows * dotsPerCellY
	middle := dotsY / 2

	for _, s := range m.scope.sparks {
		y := middle - int(s.at)*int(s.side)
		if y < 0 || y >= dotsY || s.col >= w {
			continue
		}

		// Two dots wide, like the trace it came off, so a bead reads as a bead
		// rather than as a speck of dust on the screen.
		cell := (y/dotsPerCellY)*w + s.col
		for x := range dotsPerCellX {
			grid[cell] |= 1 << brailleBit[x][y%dotsPerCellY]
		}
		if step := int8(min(int(s.bright*float32(levels)), levels-1)); step > paint[cell] {
			paint[cell] = step
		}
	}
	return grid, paint
}

// roll is a random number in 0..1 from a generator the model carries, so the
// same music throws the same sparks twice.
func (s *scopeState) roll() float32 {
	if s.seed == 0 {
		s.seed = 0x2545f491
	}
	s.seed ^= s.seed << 13
	s.seed ^= s.seed >> 17
	s.seed ^= s.seed << 5
	return float32(s.seed>>8) / float32(1<<24)
}
