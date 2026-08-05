package ui

import (
	"math"
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The record, turning.
//
// This is what the screen shows while nobody is singing: the sleeve as a label
// in the middle of a disc, the grooves around it catching the light as it goes
// round, and a wave running out along them on every beat.
//
// It is the one picture here that is about the thing rather than about the
// sound. The programme is named after the pin a record turns on, the artwork is
// already taken apart into dots for the words to be drawn beside, and the tempo
// is already measured — so the disc turns at a speed nothing had to invent, and
// stops being a screensaver.

const (
	// recordRPM is how fast it turns. Thirty-three and a third, because that is
	// what an album turns at, and because a number taken from the thing itself
	// is worth more than a number that looked about right.
	recordRPM = 33.333

	// recordSize is how much of the shorter side of the screen the disc takes,
	// and recordLabel how much of the disc is the sleeve in the middle. The
	// proportions are a twelve-inch record's: a four-inch label on it.
	recordSize  = 0.94
	recordLabel = 0.33

	// recordHole is the spindle hole, as a share of the label.
	recordHole = 0.07

	// recordGroove is how far apart the grooves are, in dots. Close enough to
	// read as a surface rather than as a target, far enough that a terminal can
	// tell two of them apart.
	recordGroove = 4

	// recordCut is how thick a groove is drawn, in dots.
	recordCut = 1.4

	// recordShine is how much of a groove's brightness comes from the light
	// running across the disc rather than from the groove being there at all.
	// This is what makes it look like a record tilted under a lamp.
	recordShine = 0.75

	// recordGlare is how wide the light is, in radians.
	recordGlare = 0.9

	// A beat sends a wave out along the grooves. recordWaveSpeed is how fast it
	// travels in dots a frame, recordWaveWide how thick it is, and recordWaveGo
	// what it keeps of itself each frame.
	recordWaveSpeed = 5
	recordWaveWide  = 16
	recordWaveGo    = 0.95

	// recordHit is the rise in loudness that starts one.
	recordHit = 0.05
)

// recordState is the disc between frames.
type recordState struct {
	// turned is how far round it has gone, in radians.
	turned float32

	// wave is how far out the last beat has travelled and lit how strongly.
	wave, lit float32

	wasLoud float32
}

// putOnTheRecord shows it now rather than waiting for a solo to ask.
func (m *Model) putOnTheRecord() {
	m.words.forced, m.words.until = true, time.Now().Add(wordsForced)
	m.words.drawn = true
}

// recordNow says whether the disc is what the screen is showing: a bar of the
// song with no words in it, or one asked for by hand.
func (m *Model) recordNow() bool {
	if m.words.forced {
		if time.Now().Before(m.words.until) {
			return true
		}
		m.words.forced, m.words.drawn = false, false
	}

	drawn := false
	if m.lyrics.synced && m.ps != nil && m.lyrics.forTrack == m.ps.TrackID {
		if at := m.lyricsAt(); at >= 0 && at < len(m.lyrics.lines) {
			drawn = wordsBeats(m.lyrics.lines[at].Words)
		}
	}

	m.words.drawn = drawn
	return drawn
}

// recordFlow turns it by a frame, and sends a wave out on a beat.
func (m *Model) recordFlow() {
	m.record.turned += float32(recordRPM / 60 * 2 * math.Pi / float64(time.Second/scopeInterval))
	if m.record.turned > 2*math.Pi {
		m.record.turned -= 2 * math.Pi
	}

	rise := max(m.scope.envelope-m.record.wasLoud, 0) / max(m.scope.envelope, scopeFloor)
	m.record.wasLoud = m.scope.envelope

	if rise > recordHit {
		m.record.wave, m.record.lit = 0, min(rise*4, 1.4)
	}
	m.record.wave += recordWaveSpeed
	m.record.lit *= recordWaveGo
}

// recordLines draws the disc, w cells across and rows deep.
func (m Model) recordLines(w, rows int) []string {
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

	cx, cy := float64(dotsX)/2, float64(dotsY)/2
	outer := math.Min(cx, cy) * recordSize
	label := outer * recordLabel
	hole := label * recordHole

	// The sleeve, as it was taken apart for the words. It is ground for the
	// whole screen with the cover in the middle of it, so the label reads from
	// that middle square.
	grain := m.grain.have
	cover := math.Min(float64(grain.DotsX), float64(grain.DotsY)) / 2

	turn := float64(m.record.turned)
	sin, cos := math.Sin(turn), math.Cos(turn)

	for y := range dotsY {
		for x := range dotsX {
			dx, dy := float64(x)-cx, float64(y)-cy
			d := math.Hypot(dx, dy)
			if d > outer || d < hole {
				continue
			}

			// Where this dot was before the record turned: the picture is not
			// spun, every dot asks where it came from. That is what keeps it
			// smooth instead of scattering the label a little differently every
			// frame.
			sx := dx*cos + dy*sin
			sy := -dx*sin + dy*cos

			var step int8
			switch {
			case d <= label:
				step = m.recordLabel(grain, sx, sy, label, cover, levels)
			default:
				step = m.recordGroove(d, math.Atan2(sy, sx), outer, levels)
			}
			if step < 0 {
				continue
			}

			cell := (y/dotsPerCellY)*w + x/dotsPerCellX
			grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
			if step > paint[cell] {
				paint[cell] = step
				hue[cell] = int8(min(int(d/outer*float64(freqs)), freqs-1))
			}
		}
	}

	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}

// recordLabel is the sleeve in the middle: the cover, cut round and turning
// with the disc. It returns -1 where the dot is not lit.
func (m Model) recordLabel(grain cover.Grain, sx, sy, label, cover float64, levels int) int8 {
	if grain.DotsX == 0 || cover <= 0 {
		// Nothing ground yet: a plain label rather than a hole in the disc.
		return int8(levels / 3)
	}

	// From the label's own radius onto the cover's.
	at := cover / label
	px := int(sx*at) + grain.DotsX/2
	py := int(sy*at) + grain.DotsY/2
	if px < 0 || py < 0 || px >= grain.DotsX || py >= grain.DotsY {
		return -1
	}

	lum := int(grain.Lum[py*grain.DotsX+px])
	if lum < grainLit+(bayer[py%4][px%4]-8)*grainSpread {
		return -1
	}
	return int8(levels - 1)
}

// recordGroove is the surface: a ring every few dots, lit by where the light
// falls on it and by whatever wave the last beat sent out.
func (m Model) recordGroove(d, angle, outer float64, levels int) int8 {
	// A ring every few dots, measured in whole distance rather than in whole
	// dots: rounding the radius first turns the circles into a starburst, which
	// is a beautiful accident and not a record.
	if math.Mod(d, recordGroove) > recordCut {
		return -1
	}

	// The light: a band across the disc, brightest where it points. A record
	// under a lamp is dark until it is turned, and this is that.
	glare := math.Abs(math.Mod(angle+3*math.Pi, 2*math.Pi) - math.Pi)
	shine := math.Max(1-glare/recordGlare, 0)
	shine *= shine

	// And the wave the beat sent out along the grooves.
	if m.record.lit > 0.02 {
		if off := math.Abs(d - float64(m.record.wave)); off < recordWaveWide {
			shine += float64(m.record.lit) * (1 - off/recordWaveWide)
		}
	}

	// Everything fades towards the rim, so the disc has a middle rather than
	// being a flat target.
	fade := 1 - d/outer*0.45

	v := (1-recordShine)*fade + recordShine*shine*fade
	return int8(min(max(int(v*float64(levels)), 0), levels-1))
}
