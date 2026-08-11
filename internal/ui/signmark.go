package ui

import (
	"encoding/base64"
	"sync"
)

// What is written on the placard.
//
// Drawn, and scaled to whatever the blank turns out to be. The drawings are cut
// off one sheet by cmd/spindle-sheet and baked by cmd/spindle-signs; see
// signs_gen.go.
//
// # These were strokes, and the strokes were not good enough
//
// They were set here in code rather than drawn, for a reason that was sound and
// a conclusion that was wrong. The reason: the blank is 28 dots across at the
// smallest baked figure, and an arrowhead at that size is four dots — four dots
// either survive exactly or they are a smudge, so drawing them at the size they
// are shown at seemed the only way to be sure of them.
//
// What that produced, measured at 26x12, the blank less its inset:
//
//	repeat all   |⡏⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⠉⢽|
//	repeat one   |⡏⠉⠉⠉⠉⠉⡉⠉⠉⠉⠉⠉⢽|
//	repeat off   |⡏⠉⠉⠉⠁⠀⠀⠀⠉⠉⠉⠉⢽|
//
// Three rectangles. The arrowhead that says which way the loop goes was computed
// as an eighth of the height, which at twelve dots is one — so it was not there
// at all, and the three states differed by a hairline and a gap. The drawn ones
// have a head, a numeral and a stroke through, and read as three different
// things at the same size.
//
// The lesson is not that drawings beat strokes. It is that a symbol laid out in
// shares of its box loses whatever share rounds to nothing, and the small end is
// where every one of these lives.

// signDrawing is one placard's drawing, a bit to a pixel, row by row.
type signDrawing struct {
	wide, tall int
	bits       string
}

// signNames is which drawing each switch shows. The generator keys by file name
// so that adding a sign is adding a drawing; this is the only place the two
// vocabularies meet.
var signNames = map[signWhat]string{
	signShuffled:  "shuffled",
	signInOrder:   "in-order",
	signRepeatAll: "repeat-all",
	signRepeatOne: "repeat-one",
	signRepeatOff: "repeat-off",
}

// signCache holds each sign scaled to each box it has been asked for.
//
// A sign is drawn every frame for the three seconds it crosses, and the box does
// not change while it does — so this is the same work sixty times over, and the
// scaling is the only part of the walk that is not a table lookup.
var signCache sync.Map // [3]int{what, w, h} -> the dots

// signMark writes what the sign says into a box of the given size.
func signMark(what signWhat, x, y, w, h int, light func(x, y int)) {
	if w < 6 || h < 4 {
		return
	}
	dots, dw, dh := signScaled(what, w, h)
	if dots == nil {
		return
	}
	// Centred: the drawing keeps its shape, so one way or the other it is
	// narrower or shorter than the blank.
	ox, oy := x+(w-dw)/2, y+(h-dh)/2
	for dy := range dh {
		for dx := range dw {
			if dots[dy*dw+dx] {
				light(ox+dx, oy+dy)
			}
		}
	}
}

// signScaled is the drawing fitted inside a box, and the size it came to.
func signScaled(what signWhat, w, h int) ([]bool, int, int) {
	name, ok := signNames[what]
	if !ok {
		return nil, 0, 0
	}
	d, ok := signDrawings[name]
	if !ok || d.wide <= 0 || d.tall <= 0 {
		return nil, 0, 0
	}

	// Fitted to whichever way round it runs out of room first.
	dw, dh := w, h
	if d.wide*h > d.tall*w {
		dh = max(d.tall*w/d.wide, 1)
	} else {
		dw = max(d.wide*h/d.tall, 1)
	}

	key := [3]int{int(what), w, h}
	if got, ok := signCache.Load(key); ok {
		return got.([]bool), dw, dh
	}

	src, err := base64.StdEncoding.DecodeString(d.bits)
	if err != nil {
		return nil, 0, 0
	}
	ink := func(x, y int) bool {
		i := y*d.wide + x
		return i/8 < len(src) && src[i/8]&(1<<(i%8)) != 0
	}

	// Any ink in the patch lights the dot, the same rule the baking used. An
	// average would lose every one of these: they are strokes on white, so the
	// mean of any patch is white.
	out := make([]bool, dw*dh)
	for dy := range dh {
		for dx := range dw {
			px0, px1 := dx*d.wide/dw, (dx+1)*d.wide/dw
			py0, py1 := dy*d.tall/dh, (dy+1)*d.tall/dh
		patch:
			for py := py0; py < max(py1, py0+1); py++ {
				for px := px0; px < max(px1, px0+1); px++ {
					if ink(px, py) {
						out[dy*dw+dx] = true
						break patch
					}
				}
			}
		}
	}
	signCache.Store(key, out)
	return out, dw, dh
}
