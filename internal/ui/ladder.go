package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// The ladder: the spectrum as a stack of lamps.
//
// Everything else spindle draws is made of dots inside a cell, which is what
// gets a curve or a beam out of a terminal. This one wants the opposite — it is
// built from blocks with air between them, the way the meter on the front of an
// amplifier is, and the terminal gives that away for nothing: half a block lit
// with the other half left dark is a lamp with a gap under it, one to a row.
//
// The colour climbs rather than travelling across, because on a meter like this
// the colour is the reading. How high a bar has gone can be read off one bar
// alone, without comparing it to its neighbours — which is why every hi-fi
// built since the seventies has looked like this.

const (
	// ladderSegment is the lamp itself: the top half of a cell, leaving the
	// bottom half as the gap under the next one.
	ladderSegment = '▀'

	// ladderPeakLift is how much brighter the falling marker is drawn than the
	// rung it is resting above, in steps of the palette. Far enough to read as
	// a separate lamp rather than as the top of the bar.
	ladderPeakLift = 4
)

// ladderLines draws the segmented meter, w cells across and rows deep.
func (m Model) ladderLines(w, rows int) []string {
	if w <= 0 || rows <= 0 || len(m.scope.bands) == 0 || len(m.styles.Ladder) == 0 {
		return nil
	}

	// The colour of every cell, or -1 where no lamp is lit. One lamp to a row,
	// so the grid is the picture.
	lamp := make([]int8, w*rows)
	for i := range lamp {
		lamp[i] = -1
	}

	steps := len(m.styles.Ladder)

	// The same tiling as the other spectrum: equal bars, equal gaps, so the two
	// are the same meter drawn two ways rather than two different readings.
	n := len(m.scope.bands)
	pitch, count := barsFit(w, n)
	left := max((w-pitch*count)/2, 0)

	for b := range count {
		from := left + b*pitch
		to := max(from+pitch-barsGap, from+1)

		band, top := m.bandsAt(b, count)
		lit := int(float64(band) * float64(rows))
		peak := int(float64(top) * float64(rows))

		for c := from; c < to && c < w; c++ {
			for r := range lit {
				// Which rung this is, out of the whole climb rather than out of
				// this bar: a quiet bar is meant to be a couple of green lamps,
				// not a small copy of the whole ladder.
				step := min(r*steps/max(rows-1, 1), steps-1)
				lamp[(rows-1-r)*w+c] = int8(step)
			}

			if peak <= lit || top < barsPeakFloor {
				continue
			}
			// The marker rests one lamp above where the bar reached, drawn
			// hotter than the rung it sits over so it reads as a marker.
			at := min(max(rows-peak, 0), rows-1)
			step := min(peak*steps/max(rows-1, 1)+ladderPeakLift, steps-1)
			lamp[at*w+c] = int8(step)
		}
	}

	return m.ladderDraw(w, rows, lamp)
}

// ladderDraw turns the lamps into rows.
func (m Model) ladderDraw(w, rows int, lamp []int8) []string {
	lines := make([]string, rows)
	for r := range rows {
		var sb strings.Builder

		var run strings.Builder
		var style lipgloss.Style
		lit := false
		flush := func() {
			if run.Len() > 0 {
				sb.WriteString(style.Render(run.String()))
				run.Reset()
			}
		}

		for c := range w {
			step := lamp[r*w+c]
			if step < 0 {
				flush()
				lit = false
				sb.WriteByte(' ')
				continue
			}

			want := m.styles.Ladder[step]
			if !lit || want.GetForeground() != style.GetForeground() {
				flush()
				style, lit = want, true
			}
			run.WriteRune(ladderSegment)
		}
		flush()
		lines[r] = fit(sb.String(), w)
	}
	return lines
}
