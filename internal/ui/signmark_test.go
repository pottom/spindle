package ui

import "testing"

// signDots is one sign drawn into a box, as a field of dots.
func signDots(t *testing.T, what signWhat, w, h int) []bool {
	t.Helper()
	got := make([]bool, w*h)
	signMark(what, 0, 0, w, h, func(x, y int) {
		if x >= 0 && y >= 0 && x < w && y < h {
			got[y*w+x] = true
		}
	})
	return got
}

var signEvery = []signWhat{signShuffled, signInOrder, signRepeatAll, signRepeatOne, signRepeatOff}

var signSays = map[signWhat]string{
	signShuffled:  "shuffled",
	signInOrder:   "in order",
	signRepeatAll: "repeat all",
	signRepeatOne: "repeat one",
	signRepeatOff: "repeat off",
}

// The five signs say five different things at the size the smallest placard is.
//
// This is the test the strokes these replaced would have failed. They laid the
// mark out in shares of the box, and at twelve dots high the arrowhead came to a
// share that rounded to one — so repeat all, repeat one and repeat off reached
// the screen as three rectangles differing by a hairline and a gap. Measured on
// the running interface, the blank is 28x14 dots at the smallest baked figure
// and 45x23 at the largest, which is 26x12 and 43x21 inside the inset.
//
// Fifteen dots is the bar. Measured, nine of the ten pairs differ by more than a
// fifth of the whole box, and the tenth is repeat all against repeat one, which
// differs by 19 dots at the small size and 26 at the large: the two are the same
// loop and the numeral inside it is the entire message, as it is on every player
// that has ever drawn this. A bar high enough to fail that pair would be a bar
// against saying it the way everyone says it. Fifteen still fails what was here
// before, where the same pair differed by a nine-dot stroke.
func TestTheSignsAreTellableApart(t *testing.T) {
	const least = 15

	for _, box := range [][2]int{{26, 12}, {43, 21}} {
		w, h := box[0], box[1]

		drawn := map[signWhat][]bool{}
		for _, what := range signEvery {
			dots := signDots(t, what, w, h)
			lit := 0
			for _, on := range dots {
				if on {
					lit++
				}
			}
			// Not blank, and not a solid block either — the first bake read the
			// ink the wrong way round, because the cutter writes white on black
			// while the sheet it cut was black on white. Every sign came out
			// filled, which is a picture of nothing just as much as an empty box.
			if lit < w*h/10 || lit > w*h*3/4 {
				t.Errorf("%s at %dx%d: %d of %d dots lit", signSays[what], w, h, lit, w*h)
			}
			drawn[what] = dots
		}

		for i, a := range signEvery {
			for _, b := range signEvery[i+1:] {
				apart := 0
				for i, on := range drawn[a] {
					if on != drawn[b][i] {
						apart++
					}
				}
				if apart < least {
					t.Errorf("%s and %s at %dx%d differ by %d dots, want at least %d",
						signSays[a], signSays[b], w, h, apart, least)
				}
			}
		}
	}
}

// Every sign fits the blank it is given, at every size a placard comes in.
func TestASignStaysInsideItsPlacard(t *testing.T) {
	for _, box := range [][2]int{{26, 12}, {34, 17}, {43, 21}} {
		w, h := box[0], box[1]
		for _, what := range signEvery {
			out := 0
			signMark(what, 0, 0, w, h, func(x, y int) {
				if x < 0 || y < 0 || x >= w || y >= h {
					out++
				}
			})
			if out > 0 {
				t.Errorf("%s in %dx%d: %d dots outside the sign", signSays[what], w, h, out)
			}
		}
	}
}

// A placard too small to write on is left blank rather than smudged.
func TestATinySignIsLeftBlank(t *testing.T) {
	for _, box := range [][2]int{{5, 12}, {26, 3}, {0, 0}} {
		for _, what := range signEvery {
			signMark(what, 0, 0, box[0], box[1], func(x, y int) {
				t.Errorf("%s drew into a %dx%d sign", signSays[what], box[0], box[1])
			})
		}
	}
}
