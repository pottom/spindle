package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
)

// A picture ground for one size is never walked over another.
//
// The interface crashed on a live terminal:
//
//	runtime error: index out of range [56880] with length 56880
//	internal/ui.(*Model).figureSweep figure.go:727
//
// figureSweep asked whether the grain it held was the right width and walked it
// over the screen's whole field of dots. The width is exactly what survives a
// change of height: the terminal had been resized while the figure was walking,
// so the picture in hand was 120 dot rows tall and the screen 244 deep, and the
// walk ran off the end of it on the first frame.
//
// Two of the six places that walk a grain asked only about the width. They ask
// wordsFits now, which is one rule in one place.
func TestAPictureIsNeverWalkedOverTheWrongSize(t *testing.T) {
	const w, rows = 120, 46
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	// Ground for half the height the screen has, which is what a resize leaves
	// in hand until the next picture arrives.
	img, layout, ok := wordsImage([]string{"jaj de jo"}, dotsX, dotsY/2)
	if !ok {
		t.Fatal("the face could not draw the line")
	}
	small := cover.Grind(grayToImage(img), w, rows/2, dotsPerCellX, dotsPerCellY)

	if wordsFits(small, dotsX, dotsY) {
		t.Fatal("a grain of the wrong height was taken for the right one")
	}
	if !wordsFits(small, dotsX, dotsY/2) {
		t.Fatal("a grain was refused for the size it was actually ground at")
	}

	m := scopeModel(160, 46)
	m.width, m.height = w, rows
	m.words.beats = true
	m.words.starts = 7_000
	m.words.have, m.words.was = small, small
	m.words.where, m.words.wasWhere = layout, layout
	m.words.since = time.Now().Add(-5 * time.Second)
	m.words.went = time.Now()
	m.words.leave = wordsSpilling

	// Every path that walks the picture, over a screen it was not ground for.
	// Before wordsFits, the first of these took the whole interface down.
	m.figureSweep(w, rows)
	m.figureThrough(w, rows, func(x, y, piece int, burn float32) {})
	m.wordsLines(w, rows)

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	m.drawLeaving(grid, paint, hue, w, rows, 0.5, 32)
	m.wordsSparkDraw(small, grid, paint, w, rows, 32)
}

// And a grain whose dots were lost keeps its own size but is still refused.
func TestAPictureWithLessThanItSaysIsRefused(t *testing.T) {
	g := cover.Grain{DotsX: 8, DotsY: 4, Lum: make([]uint8, 8*4)}
	if !wordsFits(g, 8, 4) {
		t.Error("a whole grain was refused")
	}
	g.Lum = g.Lum[:8*4-1]
	if wordsFits(g, 8, 4) {
		t.Error("a grain one dot short of what it claims was accepted")
	}
}
