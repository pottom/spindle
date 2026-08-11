package ui

import (
	"time"
)

// How loud the room is, said on the big screen and then taken back.
//
// The screen has no furniture on it — no clock, no bar, no caption — so a
// reading that stayed would be the first thing to break that, and how loud the
// room is is only interesting while somebody is changing it. So it is there for
// a moment and then it is not.
//
// What it is: a stack of lamps just inside the left edge, lit from the floor up
// to the level. That is the ladder's idiom — this screen already draws a
// segmented meter and already reads one as a height — so the volume arrives in a
// shape the eye here has been taught.
//
// How it leaves is the other half, and the more important one. It does not fade
// out, which is what a caption does: the lamps let go of the wall and fall away,
// under the same gravity the water on this screen falls under. The column drains
// from the top down, so the last thing seen is the level going. Nothing new is
// invented for it and nothing is left behind.

const (
	// volumeShows is how long after the last change the column stays lit. Long
	// enough to read after the last press of a run, short enough that it is
	// gone before you have looked away.
	volumeShows = 1100 * time.Millisecond

	// volumeLampIs what one lamp is worth. The same step the keys move in, so a
	// press is always exactly one lamp — a meter that answers a key with less
	// than a lamp reads as a key that did nothing.
	volumeLampIs = volumeStep

	// volumeInset is how far in from the left edge the column stands, in dot
	// columns.
	//
	// Four, which is two cells in: one clear cell of dark between the lamps and
	// the outermost dots, which are the record's own progress. Touching them the
	// head would look like it had grown a tail it does not have; a cell apart it
	// plainly has not.
	volumeInset = 4

	// volumeWide is how many dots across a lamp is, and volumeTall how many
	// down. A lamp has to be a block rather than a dot or the column reads as a
	// dotted line, which on this screen means water.
	volumeWide = 2
	volumeTall = 3

	// volumeGap is the dark between two lamps. One dot: enough to count them,
	// little enough that the column reads as one object.
	volumeGap = 1

	// volumeDrift is how fast a let-go lamp leaves the wall, in dots a frame,
	// and volumeSpill the little upward kick it goes with. Tuned at 30 fps and
	// converted for the rate drawn at — see pace.go.
	//
	// It leaves rightward and rises before it falls, which is what anything
	// falling off a wall does and what every drop on this screen already does.
	volumeDrift = 0.55
	volumeSpill = 0.9

	// volumeDim is what is left of a falling lamp's light after a frame.
	volumeDim = 0.94
)

// volumeLamp is one lamp that has let go of the wall.
type volumeLamp struct {
	x, y   float32
	drift  float32
	speed  float32
	step   int8 // where it stood in the column, which is its colour
	bright float32
}

// volumeState is the column and whatever has fallen off it.
type volumeState struct {
	// was is the reading the column is drawn for, and at when it last changed.
	// The reading rather than the key, so the column answers a hand on another
	// machine as readily as this one — see stageEdgeFlow, which follows the
	// playhead for the same reason.
	was  int
	at   time.Time
	seen bool

	// spilt says the lamps have already been let go for this showing, so they
	// are not thrown again on every frame after it lapses.
	spilt bool

	falling []volumeLamp
}

// volumeFlow keeps the column and moves whatever has fallen off it.
func (m *Model) volumeFlow(rows int) {
	if m.ps == nil {
		return
	}

	// A change, from wherever it came.
	if !m.volume.seen {
		m.volume.was, m.volume.seen = m.ps.Volume, true
	} else if m.ps.Volume != m.volume.was {
		m.volume.was, m.volume.at, m.volume.spilt = m.ps.Volume, time.Now(), false
	}

	// The moment the showing lapses, the lamps let go — once.
	if !m.volume.spilt && !m.volume.at.IsZero() && time.Since(m.volume.at) >= volumeShows {
		m.volume.spilt = true
		m.volumeSpill(rows)
	}

	if len(m.volume.falling) == 0 {
		return
	}
	dotsY := rows * dotsPerCellY
	kept := m.volume.falling[:0]
	for _, l := range m.volume.falling {
		l.speed -= paceFall(stageGravity)
		l.x += paceSpeed(l.drift)
		l.y -= l.speed
		l.bright *= paceKeep(volumeDim)
		if l.y < float32(dotsY) && l.bright > 0.05 {
			kept = append(kept, l)
		}
	}
	m.volume.falling = kept
}

// volumeSpill lets every lit lamp go at once.
//
// Each takes its own drift out of where it stood, so the column comes apart
// rather than sliding off as one piece: the top lamps go furthest, which reads
// as the level draining downward even though every one of them is falling.
func (m *Model) volumeSpill(rows int) {
	lit := m.volumeLit()
	if lit == 0 {
		return
	}
	dotsY := rows * dotsPerCellY
	floor := float32(m.volumeFoot(rows))
	pitch := volumePitch(dotsY)
	for i := range lit {
		m.volume.falling = append(m.volume.falling, volumeLamp{
			x:      volumeInset,
			y:      floor - float32(i*pitch),
			drift:  volumeDrift * (0.6 + 0.9*float32(i)/float32(max(lit, 1))),
			speed:  volumeSpill,
			step:   int8(i),
			bright: 1,
		})
	}
}

// volumeFoot is the dot row the bottom lamp stands on.
//
// On the band the words are set in rather than on the floor of the screen. The
// floor is where the water is thrown from and where the meter's own columns
// stand, so a reading down there is a reading among four other things; the band
// across the middle is the one place this screen keeps for whatever it has to
// say, and it is empty at the sides.
//
// The whole column is placed rather than the lit part of it, so the foot stays
// put and only the top of the stack moves — a meter whose bottom slid about with
// the reading would be a meter you had to find before you could read.
func (m Model) volumeFoot(rows int) int {
	dotsY := rows * dotsPerCellY
	middle := dotsY / 2

	// Where the line or the row of marks is standing, when there is one. The
	// other pictures have no band, and the middle of the screen is where the
	// words would have been.
	if tops, bottoms := m.words.where.Tops, m.words.where.Bottoms; len(tops) > 0 && len(bottoms) > 0 {
		high, low := tops[0], bottoms[0]
		for _, t := range tops {
			high = min(high, t)
		}
		for _, b := range bottoms {
			low = max(low, b)
		}
		middle = (high + low) / 2
	}

	full := (100/volumeLampIs)*volumePitch(dotsY) + volumeTall
	foot := middle + full/2 - volumeTall
	return min(max(foot, full), dotsY-volumeTall)
}

// volumePitch is how far apart two lamps stand, in dots.
//
// A full column is twenty lamps, and twenty lamps at their own height and gap
// want eighty dots — which a twenty row terminal has exactly none to spare of.
// So the gap goes first and then the lamps close up, rather than the top of the
// column running off the screen: a meter you cannot see the top of is not a
// meter.
func volumePitch(dotsY int) int {
	full := 100 / volumeLampIs
	if want := full * (volumeTall + volumeGap); want <= dotsY-volumeTall {
		return volumeTall + volumeGap
	}
	return max((dotsY-volumeTall)/full, 1)
}

// volumeLit is how many lamps the reading lights.
func (m Model) volumeLit() int {
	if m.ps == nil {
		return 0
	}
	return min(max(m.ps.Volume, 0), 100) / volumeLampIs
}

// volumeShowing is whether the column itself is up.
func (m Model) volumeShowing() bool {
	return !m.volume.at.IsZero() && time.Since(m.volume.at) < volumeShows
}

// volumeDraw puts the column and whatever is falling off it into the picture.
//
// Into the same grid as everything else rather than over the top of it, so it is
// coloured by the record like the rest of the screen and a drop passing through
// it is a drop passing through it.
func (m Model) volumeDraw(w, rows int, grid []uint8, paint, hue []int8, levels, freqs int) {
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 || levels <= 0 {
		return
	}

	light := func(x, y int, step int8, band int) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if step > paint[cell] {
			paint[cell] = step
			hue[cell] = int8(min(band, freqs-1))
		}
	}

	// The record's own colour, all the way up.
	//
	// It climbed with the height first, the way the segmented meter's does, and
	// that is the meter's answer to a different question: there the colour is
	// the reading, because a lamp on its own has to say how high the stack has
	// got. Here the stack is right there to be looked at, so the colour saying
	// it again is one question with two movements. What it says instead is whose
	// screen this is — the middle of the palette, which is the artwork's own
	// accent.
	accent := (freqs - 1) / 2
	lamp := func(x, y float32, bright float32) {
		step := int8(min(int((0.55+0.45*float64(bright))*float64(levels-1)), levels-1))
		band := accent
		for dy := range volumeTall {
			for dx := range volumeWide {
				light(int(x)+dx, int(y)+dy, step, band)
			}
		}
	}

	if lit := m.volumeLit(); m.volumeShowing() && lit > 0 {
		floor := m.volumeFoot(rows)
		pitch := volumePitch(dotsY)
		for i := range lit {
			lamp(volumeInset, float32(floor-i*pitch), 1)
		}
	}
	for _, l := range m.volume.falling {
		lamp(l.x, l.y, l.bright)
	}
}
