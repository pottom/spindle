package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// The ladder: the spectrum as a stack of lamps.
//
// Everything else spindle draws is made of dots inside a cell, which is what
// gets a curve or a beam out of a terminal. This one wants the opposite — it is
// built from blocks with air between them, the way the meter on the front of an
// amplifier is, and the terminal gives that away for nothing: half a block is a
// lamp, and the other half of the cell is either the gap under it or the next
// lamp up.
//
// The colour climbs rather than travelling across, because on a meter like this
// the colour is the reading. How high a bar has gone can be read off one bar
// alone, without comparing it to its neighbours — which is why every hi-fi
// built since the seventies has looked like this.

const (
	// The halves of a cell, each of which can be a lamp of its own: the top one
	// lit alone, the bottom one lit alone, or both, in two colours.
	ladderTop    = '▀'
	ladderBottom = '▄'

	// ladderTall is the height, in rows, at which the ladder can afford to leave
	// the air under each lamp empty.
	//
	// The gap is the look — a stack of separate lamps rather than a bar — and an
	// empty one costs half the resolution. On a screen that is a good trade:
	// forty rows are forty lamps either way. In the strip under the artwork it
	// is a ruinous one, four rows becoming four lamps, which is not a spectrum
	// but four dashes moving about. So below this the gap is drawn rather than
	// left: every other rung is the same lamp with its light out, which stripes
	// the bar exactly as the empty gaps do and costs nothing, because an unlit
	// lamp still says where the lit ones stop.
	ladderTall = 8

	// ladderPeakLift is how much hotter the falling marker is drawn than the
	// rung it rests above, in steps of the palette. Far enough to read as a
	// separate lamp rather than as the top of the bar.
	ladderPeakLift = 4
)

// ladderLines draws the segmented meter, w cells across and rows deep.
func (m Model) ladderLines(w, rows int) []string {
	if w <= 0 || rows <= 0 || len(m.scope.bands) == 0 || len(m.styles.Ladder) == 0 {
		return nil
	}

	// One rung to a cell where there is room for the air under it, two where
	// there is not.
	perCell := 2
	if rows >= ladderTall {
		perCell = 1
	}
	high := rows * perCell

	// The colour of every rung, or -1 where no lamp is lit, and whether that
	// rung is one of the ones drawn with its light out.
	lamp := make([]int8, w*high)
	for i := range lamp {
		lamp[i] = -1
	}
	out := make([]bool, w*high)

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
		lit := int(float64(band) * float64(high))
		peak := int(float64(top) * float64(high))

		for c := from; c < to && c < w; c++ {
			for r := range lit {
				// Which rung this is out of the whole climb, rather than out of
				// this bar: a quiet band is meant to be a couple of lamps at the
				// cool end, not a small copy of the whole ladder.
				step := min(r*steps/max(high-1, 1), steps-1)
				at := (high - 1 - r) * w + c
				lamp[at] = int8(step)
				out[at] = perCell == 2 && r%2 == 1
			}

			if peak <= lit || top < barsPeakFloor {
				continue
			}
			// The marker rests one rung above where the bar reached, drawn
			// hotter than the rung it sits over so it reads as a marker.
			at := min(max(high-peak, 0), high-1)
			step := min(peak*steps/max(high-1, 1)+ladderPeakLift, steps-1)
			lamp[at*w+c] = int8(step)
			out[at*w+c] = false
		}
	}

	return m.ladderDraw(w, rows, perCell, lamp, out)
}

// ladderDraw turns the rungs into rows.
//
// A cell holds one rung or two. With one, the top half is the lamp and the
// bottom half is the air under it. With two, the halves are lamps of their own
// and are drawn in one cell as a foreground over a background — which is the
// only way a terminal will put two colours in one place.
func (m Model) ladderDraw(w, rows, perCell int, lamp []int8, out []bool) []string {
	colour := func(at int) color.Color {
		if out[at] {
			return m.styles.LadderOff[lamp[at]].GetForeground()
		}
		return m.styles.Ladder[lamp[at]].GetForeground()
	}

	lines := make([]string, rows)
	for r := range rows {
		var sb strings.Builder
		for c := range w {
			upper := (r*perCell)*w + c
			lower := -1
			if perCell == 2 {
				lower = (r*perCell+1)*w + c
			}

			switch {
			case lamp[upper] < 0 && (lower < 0 || lamp[lower] < 0):
				sb.WriteByte(' ')
			case lower < 0 || lamp[lower] < 0:
				sb.WriteString(lipgloss.NewStyle().Foreground(colour(upper)).Render(string(ladderTop)))
			case lamp[upper] < 0:
				sb.WriteString(lipgloss.NewStyle().Foreground(colour(lower)).Render(string(ladderBottom)))
			default:
				// Both there: the upper half over the lower one, which is the
				// only way a terminal will put two colours in one cell.
				style := lipgloss.NewStyle().
					Foreground(colour(upper)).
					Background(colour(lower))
				sb.WriteString(style.Render(string(ladderTop)))
			}
		}
		lines[r] = fit(sb.String(), w)
	}
	return lines
}
