package ui

const (
	// barsPeakDecay is what a marker keeps of its height each frame. A marker
	// falls by a share of where it is rather than by a fixed step: a fixed step
	// leaves every marker descending at one rate, and markers that started
	// together stay together — which drew a straight line clean across the
	// screen, over bands that were silent.
	barsPeakDecay = 0.93

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
			if peak <= height || m.scope.peaks[b] < barsPeakFloor {
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
