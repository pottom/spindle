package ui

// What is written on the placard.
//
// Drawn here rather than traced from a sheet, and that is the one place in this
// project where a picture is not a drawing. The reason is the size: the blank is
// 28 dots across at the smallest baked figure, and a traced drawing squeezed
// into that loses the very thing it has to say — an arrowhead is four dots, and
// four dots either survive exactly or they are a smudge. Set as strokes at the
// size they are drawn at, they are the same mark at every size.
//
// They are diagrams, besides. Everything drawn for this screen is a character
// with a body and a hand; these are two arrows and a loop. Asking the same sheet
// for both would get a worse version of each.

// signLine draws a straight run of dots from one point to another, thick dots
// across so it survives at the size a placard is.
func signLine(x0, y0, x1, y1 int, light func(x, y int)) {
	dx, dy := x1-x0, y1-y0
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		light(x0, y0)
		return
	}
	for i := 0; i <= steps; i++ {
		x := x0 + dx*i/steps
		y := y0 + dy*i/steps
		light(x, y)
	}
}

// signHead draws an arrowhead at the end of a run going right, as an open V.
//
// Small on purpose. Drawn a quarter of the box high it was three dots on a
// twelve dot sign, which is the whole gap between the two runs it belongs to —
// the two heads met in the middle and the sign read as a scribble. The head is
// there to say which way round the loop goes; on the two arrows the crossing
// says everything and the heads only take room, so they are left off.
func signHead(x, y, size int, light func(x, y int)) {
	for i := 1; i <= size; i++ {
		light(x-i, y-i)
		light(x-i, y+i)
	}
}

// signMark writes what the sign says into a box of the given size.
//
// Everything is set out as a share of the box rather than in dots, so the same
// mark comes out at every baked size of the figure.
func signMark(what signWhat, x, y, w, h int, light func(x, y int)) {
	if w < 6 || h < 4 {
		return
	}
	head := max(h/8, 1)

	// Two runs across the sign, one high and one low. Crossed, they are
	// shuffled; parallel, they are in order. Nothing else changes between the
	// two, so the one difference is the whole message — and it is the crossing
	// that carries it, which is why there are no arrowheads on these.
	arrows := func(cross bool) {
		hi, lo := y+h/5, y+h-1-h/5
		left, right := x, x+w-1
		if cross {
			signLine(left, hi, right, lo, light)
			signLine(left, lo, right, hi, light)
			return
		}
		signLine(left, hi, right, hi, light)
		signLine(left, lo, right, lo, light)
	}

	// A loop: a rectangle with an arrowhead on the top run, broken open when
	// there is nothing to repeat.
	loop := func(gap, one bool) {
		l, r := x, x+w-1
		t, b := y, y+h-1
		mid := (l + r) / 2

		// The top, in two runs so the gap can be left out of the middle of it.
		if gap {
			signLine(l, t, mid-w/6, t, light)
			signLine(mid+w/6, t, r, t, light)
		} else {
			signLine(l, t, r, t, light)
		}
		signLine(l, b, r, b, light)
		signLine(l, t, l, b, light)
		signLine(r, t, r, b, light)
		signHead(r, t+head, head, light)

		if one {
			// A stroke down the middle, with a foot on it, which is the most a
			// numeral can be at this size and still be one rather than an l.
			signLine(mid, t+h/4, mid, b-h/4, light)
			signLine(mid-1, b-h/4, mid+1, b-h/4, light)
		}
	}

	switch what {
	case signShuffled:
		arrows(true)
	case signInOrder:
		arrows(false)
	case signRepeatAll:
		loop(false, false)
	case signRepeatOne:
		loop(false, true)
	case signRepeatOff:
		loop(true, false)
	}
}
