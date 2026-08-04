package ui

const (
	// barsPeakFall is how much of its height a peak marker gives up each frame.
	// Slow enough to hang where a hit reached, fast enough not to look stuck.
	barsPeakFall = 0.012

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
		s.peaks[i] = max(v, s.peaks[i]-barsPeakFall)
	}
}

// barsLines draws the spectrum: one column of braille dots per band, lowest
// frequency on the left, as every analyser reads.
//
// The bands arrive spaced by octave and scaled in decibels, so what is drawn is
// already what the ear hears as even; the drawing only has to place it.
func (m Model) barsLines(w int) []string {
	if w <= 0 || len(m.scope.bands) == 0 || len(m.styles.ScopeCore) == 0 {
		return nil
	}

	dotsY := scopeRows * dotsPerCellY
	grid := make([]uint8, w*scopeRows)
	loud := make([]float32, w)

	// Each band gets as even a share of the width as it can, with a blank
	// column between: a solid block reads as one shape, not as a spectrum.
	n := len(m.scope.bands)
	for b := range n {
		from := b * w / n
		to := max((b+1)*w/n-barsGap, from+1)

		height := int(float64(m.scope.bands[b]) * float64(dotsY))
		peak := int(float64(m.scope.peaks[b]) * float64(dotsY))

		for c := from; c < to && c < w; c++ {
			loud[c] = m.scope.bands[b]
			for y := range height {
				dy := dotsY - 1 - y
				grid[(dy/dotsPerCellY)*w+c] |= 1 << brailleBit[0][dy%dotsPerCellY]
				grid[(dy/dotsPerCellY)*w+c] |= 1 << brailleBit[1][dy%dotsPerCellY]
			}
			if peak <= height {
				continue
			}
			// The marker sits one dot thick at the height reached, which is
			// what separates it from the bar it fell from.
			dy := min(max(dotsY-peak, 0), dotsY-1)
			grid[(dy/dotsPerCellY)*w+c] |= 1 << brailleBit[0][dy%dotsPerCellY]
			grid[(dy/dotsPerCellY)*w+c] |= 1 << brailleBit[1][dy%dotsPerCellY]
			loud[c] = 1
		}
	}
	return m.scopeDraw(w, grid, loud)
}
