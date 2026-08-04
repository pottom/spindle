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

	// barsSlack is how many cells a whole number of equal bars may leave over
	// before it is worth drawing fewer, wider ones. A cell or two at each end
	// is not a margin anybody sees; a hand's width is.
	barsSlack = 4
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
	// The gaps are the same width as well, so what will not divide is left as a
	// margin, split between the two ends.
	n := len(m.scope.bands)
	pitch, count := barsFit(w, n)
	left := max((w-pitch*count)/2, 0)

	for b := range count {
		from := left + b*pitch
		to := max(from+pitch-barsGap, from+1)

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

// barsFit chooses how the bars tile the width: how many cells one takes, and
// how many there are.
//
// A bar for every band, if that fills the width near enough. It usually does
// not divide, and where the cells left over amount to something — at sixty-odd
// bands on a wide screen it was a hand's width of nothing at each end, in the
// one place that is meant to be full — fewer and wider bars are drawn instead,
// at whatever pitch wastes least. Folding two bands into one column costs a
// little detail nobody was reading; an empty margin costs the picture its
// width, and uneven gaps between the bars cost it its regularity, which is what
// makes a meter read as a meter.
//
// Never more bars than bands: a column nothing was measured for would have to be
// drawn from a neighbour, and an invented band is worse than a missing one.
func barsFit(w, n int) (pitch, count int) {
	pitch, count = max(w/n, 1), min(n, w)
	waste := w - pitch*count
	if waste <= barsSlack {
		return pitch, count
	}

	// The first pitch that fits well enough wins, and it is the one with the
	// most bars: a wider bar is always a shorter list of them, so chasing the
	// last cell of the width would keep costing bands after it had stopped
	// buying anything.
	for p := pitch + 1; p <= w; p++ {
		k := min(n, w/p)
		if k < 1 {
			break
		}
		if over := w - p*k; over < waste {
			pitch, count, waste = p, k, over
		}
		if waste <= barsSlack {
			break
		}
	}
	return pitch, count
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
