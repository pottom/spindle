package cover

import (
	"image"
	"image/color"
	"math"
	"strings"
)

// Generated artwork is for the lists that have none.
//
// Liked songs is the one that matters: Spotify draws a tile for it in its own
// clients and serves that tile to nobody, so a third party either invents one
// or shows an empty box. The first saved track's sleeve was the other reading
// and it is worse — it says the collection is that record, and it changes every
// time a song is saved.
//
// A generated cover goes through the same pipeline as a downloaded one: it is
// named by a url that no server would answer, and the loader draws it instead of
// fetching it. Everything above the loader keeps passing a string around and
// none of it has to know.
const (
	// generatedScheme marks a cover this package draws rather than downloads.
	generatedScheme = "spindle-art:"

	// LikedURL is the saved tracks' cover.
	LikedURL = generatedScheme + "liked"

	// NoneURL stands in for a thing with no artwork of its own. A wall of covers
	// with a hole in it reads as a picture that failed to load; a drawn one says
	// there was never a picture to load.
	NoneURL = generatedScheme + "none"

	// generatedSize is how large the drawing is made. The renderers resize to
	// the cells they are given, and this is comfortably above the largest
	// artwork a terminal asks for while staying quick to draw.
	generatedSize = 512
)

// generated draws the artwork a url names, and reports whether it named one.
func generated(url string) (image.Image, bool) {
	if !strings.HasPrefix(url, generatedScheme) {
		return nil, false
	}
	switch url {
	case LikedURL:
		return likedArt(generatedSize), true
	case NoneURL:
		return noneArt(generatedSize), true
	default:
		return nil, false
	}
}

// The saved tracks' cover: a heart on a gradient.
//
// The colours are the one place in spindle where an accent is not taken from a
// sleeve, because there is no sleeve to take it from. They run violet to a deep
// blue, which is far enough from any accent a record supplies that the screen
// reads as having changed subject rather than as having got the colour wrong.
var (
	likedFrom = color.RGBA{R: 0x8B, G: 0x3F, B: 0xD6, A: 0xFF}
	likedTo   = color.RGBA{R: 0x24, G: 0x2A, B: 0x8C, A: 0xFF}
	likedInk  = color.RGBA{R: 0xF6, G: 0xF2, B: 0xFF, A: 0xFF}
)

// likedArt draws the cover: a gradient corner to corner, and a heart across the
// middle of it.
//
// The heart is the classic curve, (x²+y²−1)³ ≤ x²y³, sampled rather than
// stroked: a terminal shows this a few dozen cells wide, and an edge that is
// aliased at full size is a jagged line by the time it gets there. Four samples
// a pixel are enough to make the curve read as smooth at every size a cover is
// drawn at.
func likedArt(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	span := float64(size - 1)

	for y := range size {
		for x := range size {
			// The gradient runs corner to corner, so neither edge of the square
			// is a flat band of one colour.
			t := (float64(x) + float64(y)) / (2 * span)
			back := mix(likedFrom, likedTo, t)

			if cover := heartCoverage(x, y, size); cover > 0 {
				back = mix(back, likedInk, cover)
			}
			img.SetRGBA(x, y, back)
		}
	}
	return img
}

const (
	// likedScale is how much of the square the heart is drawn across. Larger
	// numbers draw it smaller: it is the width of the curve's own coordinates
	// that the square is made to cover.
	likedScale = 3.6

	// likedLift centres it. The curve's origin sits between its lobes, well above
	// the middle of the shape it draws, so the drawing has to be pushed back
	// down by hand.
	//
	// Measured rather than guessed at, which is how it came to be wrong: at the
	// 0.42 it was set to by eye, the heart had seventy pixels of air over it and
	// twenty-eight under it in a square of 256. At this it has forty-nine and
	// forty-nine, against forty-seven either side.
	likedLift = 0.13
)

// heartCoverage is how much of one pixel the heart covers, from none to all.
func heartCoverage(px, py, size int) float64 {
	const samples = 2

	var inside int
	for sy := range samples {
		for sx := range samples {
			// The sample sits in the middle of its quarter of the pixel.
			fx := (float64(px) + (float64(sx)+0.5)/samples) / float64(size)
			fy := (float64(py) + (float64(sy)+0.5)/samples) / float64(size)

			// Into the curve's own coordinates: the middle of the square is the
			// origin, y counts upwards, and the scale leaves the heart a margin
			// on every side. It is nudged down a little because the curve's own
			// centre of mass sits above its origin.
			x := (fx - 0.5) * likedScale
			y := (0.5-fy)*likedScale + likedLift

			if inHeart(x, y) {
				inside++
			}
		}
	}
	return float64(inside) / float64(samples*samples)
}

// inHeart reports whether a point is inside the heart curve.
func inHeart(x, y float64) bool {
	d := x*x + y*y - 1
	return d*d*d-x*x*y*y*y <= 0
}

// mix blends towards b by t, on a straight line through the colours.
func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Min(math.Max(t, 0), 1)
	lerp := func(from, to uint8) uint8 {
		return uint8(math.Round(float64(from) + (float64(to)-float64(from))*t))
	}
	return color.RGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: 0xFF}
}

// The stand-in for a thing with no artwork: a pair of beamed notes, drawn as an
// outline, on a flat dark square.
//
// Flat and grey on purpose. Every other cover on the wall is a photograph and
// carries a colour the whole program then wears; this one has to say "nothing
// here" without becoming the thing the eye goes to, and without handing the
// program an accent that came from us rather than from a record.
var (
	noneBack = color.RGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF}
	noneInk  = color.RGBA{R: 0xB3, G: 0xB3, B: 0xB3, A: 0xFF}
)

// The note, in the square's own coordinates: 0 to 1 across and down.
const (
	// Centred on the square by weight rather than by extent, which is what the
	// eye does. The drawing's outermost points are the left head and the right
	// stem, and putting those two in the middle left the beam and both stems —
	// nearly all of the ink — sitting to the right of centre: measured, the
	// centre of mass was at 0.544 of the width. It is at 0.500 here.
	noteLeft   = 0.343 // where the left stem stands
	noteRight  = 0.723 // and the right one
	noteTop    = 0.25  // the top of the beam
	noteFoot   = 0.65  // where the stems end, at the middle of the heads
	noteStroke = 0.05  // how thick a stem is
	noteBeam   = 0.09  // and the beam, which is heavier
	noteHead   = 0.10  // the outer radius of a head
)

// noneArt draws it.
func noneArt(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			back := noneBack
			if cover := noteCoverage(x, y, size); cover > 0 {
				back = mix(back, noneInk, cover)
			}
			img.SetRGBA(x, y, back)
		}
	}
	return img
}

// noteCoverage is how much of one pixel the note covers, sampled rather than
// stroked: a terminal shows this a few dozen cells wide, and an edge that is
// aliased at full size is a staircase by the time it gets there.
func noteCoverage(px, py, size int) float64 {
	const samples = 3

	var inside int
	for sy := range samples {
		for sx := range samples {
			fx := (float64(px) + (float64(sx)+0.5)/samples) / float64(size)
			fy := (float64(py) + (float64(sy)+0.5)/samples) / float64(size)
			if inNote(fx, fy) {
				inside++
			}
		}
	}
	return float64(inside) / float64(samples*samples)
}

// inNote reports whether a point is on the drawing: the beam across the top, a
// stem hanging from either end of it, and a ring at the foot of each stem.
func inNote(x, y float64) bool {
	switch {
	case y >= noteTop && y <= noteTop+noteBeam && x >= noteLeft && x <= noteRight:
		return true
	case y >= noteTop && y <= noteFoot && x >= noteLeft && x <= noteLeft+noteStroke:
		return true
	case y >= noteTop && y <= noteFoot && x >= noteRight-noteStroke && x <= noteRight:
		return true
	}
	// The heads hang to the left of their stems, the way they are written.
	return inRing(x, y, noteLeft+noteStroke-noteHead, noteFoot) ||
		inRing(x, y, noteRight-noteHead, noteFoot)
}

// inRing reports whether a point is inside the drawn edge of a head.
func inRing(x, y, cx, cy float64) bool {
	dx, dy := x-cx, y-cy
	d := math.Sqrt(dx*dx + dy*dy)
	return d <= noteHead && d >= noteHead-noteStroke
}
