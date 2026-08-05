package ui

// The waterfall: the spectrum, kept.
//
// The bars say what is sounding now and the trace says how it is shaped now;
// both forget everything a thirtieth of a second later. This holds the last few
// seconds of the spectrum on screen at once, frequency up the side and time
// across, which is the only one of the three that shows the music's structure
// rather than its instant: a hi-hat is a row of regular marks, a snare a column
// that crosses the whole picture, a bass line a band along the bottom that
// moves when the note does.
//
// It is drawn in the same dots as the trace, and it works out at sixteen rows
// of frequency and two slices of time to a cell.

const (
	// fallFloor is how loud a band has to be before its dot lights.
	//
	// A dot is on or off, so the threshold is the whole of the picture's
	// character, and it cannot be argued about: measured against seventy-five
	// seconds of recorded spectrum, a floor low enough to be generous lit
	// ninety-seven per cent of the dots, which is a wall rather than a picture.
	// Here about half of them light, the beat is legible in the pattern, and
	// the five steps of the palette come out evenly used.
	fallFloor = 0.65

	// fallHistory is how many slices are kept. Two to a cell, so this is a wide
	// screen's worth and a little over — enough that the picture is full on any
	// terminal, and bounded so a long track does not grow it without end.
	fallHistory = 512
)

// rememberFall keeps a slice of the spectrum, newest last. It is done as the
// frames arrive rather than while drawing, because View has to stay a pure
// function of the model and a history is state.
func (s *scopeState) rememberFall(bands []float32) {
	if len(bands) == 0 {
		return
	}

	keep := make([]float32, len(bands))
	copy(keep, bands)

	s.fall = append(s.fall, keep)
	if len(s.fall) > fallHistory {
		s.fall = s.fall[len(s.fall)-fallHistory:]
	}
}

// fallLines draws the waterfall across w cells, newest on the right.
func (m Model) fallLines(w int) []string {
	if w <= 0 || len(m.scope.fall) == 0 || len(m.styles.Bars) == 0 {
		return nil
	}

	dotsX, rows := w*dotsPerCellX, scopeRows*dotsPerCellY
	freqs, levels := len(m.styles.Bars), len(m.styles.Bars[0])

	grid := make([]uint8, w*scopeRows)
	paint := make([]int8, w*scopeRows)
	hue := make([]int8, w*scopeRows)
	for i := range paint {
		paint[i] = -1
	}

	// Time runs to the right, so the newest slice is at the last dot column and
	// the history is read backwards from there. A screen wider than the history
	// starts blank on the left and fills as it plays, which is what a waterfall
	// does when it is switched on.
	first := len(m.scope.fall) - dotsX

	for x := range dotsX {
		at := first + x
		if at < 0 {
			continue
		}
		slice := m.scope.fall[at]

		for r := range rows {
			level := fallLevel(slice, rows, r)
			if level <= fallFloor {
				continue
			}

			cell := (r/dotsPerCellY)*w + x/dotsPerCellX
			grid[cell] |= 1 << brailleBit[x%dotsPerCellX][r%dotsPerCellY]

			// The cell takes the strongest thing in it: eight dots share one
			// colour, and the loudest of them is what the eye is looking at.
			over := (level - fallFloor) / (1 - fallFloor)
			if step := int8(min(int(over*float32(levels)), levels-1)); step > paint[cell] {
				paint[cell] = step
			}
		}
	}

	// The hue runs up the screen the way it runs across the spectrum: the low
	// end of the range at the bottom, where the bass is drawn.
	for r := range scopeRows {
		f := int8(min((scopeRows-1-r)*freqs/scopeRows, freqs-1))
		for c := range w {
			hue[r*w+c] = f
		}
	}

	return m.drawCells(w, grid, paint, hue)
}

// fallLevel is how loud a dot row is: the strongest band it covers.
//
// Sixteen rows over the spectrum's bands, lowest frequency at the bottom. The
// strongest rather than the average, for the reason the analyser itself takes
// the loudest bin: what a row is asked is how loud this part of the range got,
// and an average of a hit and the silence beside it is neither.
func fallLevel(slice []float32, rows, row int) float32 {
	up := rows - 1 - row
	lo := up * len(slice) / rows
	hi := max((up+1)*len(slice)/rows, lo+1)

	var level float32
	for _, v := range slice[lo:min(hi, len(slice))] {
		level = max(level, v)
	}
	return level
}
