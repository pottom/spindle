package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// barsPeakDecay is what a marker keeps of its height each frame. A marker
	// falls by a share of where it is rather than by a fixed step: a fixed step
	// leaves every marker descending at one rate, and markers that started
	// together stay together — which drew a straight line clean across the
	// screen, over bands that were silent.
	//
	// Slow enough to hang where a hit reached for about a second and a half; at
	// a third of that it was gone before it registered as a marker at all.
	barsPeakDecay = 0.98

	// barsPeakFloor is the level below which a band has no marker at all.
	// Nothing is sounding there, so nothing reached anything.
	barsPeakFloor = 0.04

	// barsGap is the blank column between bars, so they read as separate bands
	// rather than as one filled shape.
	barsGap = 1
)

// adoptBands takes a spectrum frame and carries the peak markers with it.
func (s *scopeState) adoptBands(bands []float32) {
	if len(bands) == 0 {
		return
	}
	s.bands = bands

	if len(s.peaks) != len(bands) {
		s.peaks = make([]float32, len(bands))
	}
	for i, v := range bands {
		s.peaks[i] = max(v, s.peaks[i]*barsPeakDecay)
		if s.peaks[i] < barsPeakFloor {
			s.peaks[i] = 0
		}
	}
}

// barsLines draws the spectrum: one column of braille dots per band, lowest
// frequency on the left, as every analyser reads.
//
// The bands arrive spaced by octave and scaled in decibels, so what is drawn is
// already what the ear hears as even; the drawing only has to place it.
func (m Model) barsLines(w int) []string {
	if w <= 0 || len(m.scope.bands) == 0 || len(m.styles.Bars) == 0 {
		return nil
	}

	dotsY := scopeRows * dotsPerCellY
	grid := make([]uint8, w*scopeRows)

	// The colour of every cell, chosen from where it sits: which part of the
	// frequency range the column covers, and how high up the bar the row is.
	paint := make([]int8, w*scopeRows)
	for i := range paint {
		paint[i] = -1
	}

	// Every bar is the same width, with a blank column between: a solid block
	// reads as one shape, not as a spectrum, and a meter whose columns are not
	// the same size reads as a meter that is wrong.
	//
	// One bar per band, unless there are more bands than cells to draw them in.
	n := len(m.scope.bands)
	count := min(n, w)
	bar := max(w/count-barsGap, 1)

	for b := range count {
		from := barsAt(b, count, w, bar)
		to := from + bar

		band, top := m.bandsAt(b, count)
		height := int(float64(band) * float64(dotsY))
		peak := int(float64(top) * float64(dotsY))

		levels := len(m.styles.Bars[0])
		for c := from; c < to && c < w; c++ {
			for y := range height {
				dy := dotsY - 1 - y
				cell := (dy/dotsPerCellY)*w + c
				grid[cell] |= 1 << brailleBit[0][dy%dotsPerCellY]
				grid[cell] |= 1 << brailleBit[1][dy%dotsPerCellY]

				// How far up its own bar this row is, not how far up the screen:
				// a short bar still burns brightest at its tip.
				up := float64(y) / float64(max(height-1, 1))
				if l := int8(min(int(up*float64(levels)), levels-1)); l > paint[cell] {
					paint[cell] = l
				}
			}

			if peak <= height || top < barsPeakFloor {
				continue
			}
			// The marker sits one dot thick at the height reached, which is
			// what separates it from the bar it fell from, and always at the
			// hottest step so it reads as a marker rather than as more bar.
			dy := min(max(dotsY-peak, 0), dotsY-1)
			cell := (dy/dotsPerCellY)*w + c
			grid[cell] |= 1 << brailleBit[0][dy%dotsPerCellY]
			grid[cell] |= 1 << brailleBit[1][dy%dotsPerCellY]
			paint[cell] = int8(levels - 1)
		}
	}
	return m.barsDraw(w, grid, paint)
}

// barsAt is the first column of one bar: the first bar against the left edge,
// the last against the right, the rest spread evenly between them.
//
// Spread rather than stepped by a fixed pitch, because a whole number of equal
// bars rarely fills a width exactly, and the cells left over have to go
// somewhere. Stepping leaves them as a margin, which on a wide screen was a
// hand's width of nothing at each end, in the one place that is meant to be
// full. Here they go into the gaps instead, so a gap here and there is a cell
// wider than its neighbours — which in a row of bars nobody sees, while every
// bar is still exactly as wide as every other, which everybody does.
func barsAt(b, count, w, bar int) int {
	if count < 2 {
		return 0
	}
	return b * (w - bar) / (count - 1)
}

// bandsAt is the level and the peak marker for one bar. It covers one band and
// only more than one on a screen too narrow to give each its own column. The
// loudest of them wins rather than their average: a meter answers what reached
// this far, and averaging a hit with its quiet neighbour flattens exactly the
// moment worth seeing.
func (m Model) bandsAt(bar, count int) (band, peak float32) {
	n := len(m.scope.bands)
	from, to := bar*n/count, max((bar+1)*n/count, bar*n/count+1)

	for i := from; i < to && i < n; i++ {
		band = max(band, m.scope.bands[i])
		if i < len(m.scope.peaks) {
			peak = max(peak, m.scope.peaks[i])
		}
	}
	return band, peak
}

// barsDraw turns the dot grid into rows, colouring each cell from the palette:
// hue by where the column sits in the frequency range, strength by how high up
// its bar the row is.
func (m Model) barsDraw(w int, grid []uint8, paint []int8) []string {
	freqs := len(m.styles.Bars)

	lines := make([]string, scopeRows)
	for r := range scopeRows {
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
			at := r*w + c
			if grid[at] == 0 || paint[at] < 0 {
				flush()
				lit = false
				sb.WriteByte(' ')
				continue
			}

			f := min(c*freqs/w, freqs-1)
			want := m.styles.Bars[f][paint[at]]
			if !lit || want.GetForeground() != style.GetForeground() {
				flush()
				style, lit = want, true
			}
			run.WriteRune(rune(brailleBase + int(grid[at])))
		}
		flush()
		lines[r] = fit(sb.String(), w)
	}
	return lines
}
