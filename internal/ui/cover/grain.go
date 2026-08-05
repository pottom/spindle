package cover

import (
	"context"
	"image"
	"image/color"

	"golang.org/x/image/draw"
)

// Grain is a cover taken apart into what a visualiser can move: how bright
// every dot of it is, and what colour every cell of it is.
//
// The two are at different resolutions on purpose. A braille cell holds eight
// dots and one colour, so the shape of the picture has to live in the dots —
// which are only ever on or off — and its colour in the cell underneath them.
// That is also why this is worth doing at all: at eight dots to a cell a full
// screen is a few hundred thousand of them, which is enough of a picture to
// recognise a record by.
type Grain struct {
	DotsX, DotsY   int
	CellsX, CellsY int

	// Lum is how bright each dot is, 0..255, stretched so that the picture uses
	// the whole range whatever the cover was like.
	Lum []uint8

	// Cell is the colour of each cell, quantised: a photograph has a different
	// colour in every cell, and a different colour in every cell is a colour
	// code in every cell — which is what a terminal chokes on. Rounding them
	// together is what puts the runs back.
	Cell []color.RGBA
}

// grainLevels is how many steps each colour channel is rounded to. Four is
// sixty-four colours, which is few enough for whole rows of a sky or a wall to
// come out as one run and many enough that a cover still looks like itself.
const grainLevels = 4

// Grind samples a cover for the dot visualiser. It blocks on I/O the first time
// a cover is asked for and must never be called from Update.
func (l *Loader) Grind(ctx context.Context, url string, cellsX, cellsY, dotsPerCellX, dotsPerCellY int) (Grain, error) {
	img, err := l.image(ctx, url)
	if err != nil {
		return Grain{}, err
	}
	return Grind(img, cellsX, cellsY, dotsPerCellX, dotsPerCellY), nil
}

// Grind is the same, for a cover already decoded — which is what the tests
// have and what the drawn artwork is.
func Grind(img image.Image, cellsX, cellsY, perX, perY int) Grain {
	if cellsX <= 0 || cellsY <= 0 || perX <= 0 || perY <= 0 {
		return Grain{}
	}

	g := Grain{
		DotsX:  cellsX * perX,
		DotsY:  cellsY * perY,
		CellsX: cellsX,
		CellsY: cellsY,
	}

	// The cover is square and the screen is not, so it is fitted rather than
	// stretched: a record squashed to the shape of a terminal is not the record.
	src := fit(img, g.DotsX, g.DotsY)

	scaled := image.NewRGBA(image.Rect(0, 0, g.DotsX, g.DotsY))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, src.Bounds(), draw.Src, nil)

	g.Lum = make([]uint8, g.DotsX*g.DotsY)
	for y := range g.DotsY {
		for x := range g.DotsX {
			c := scaled.RGBAAt(x, y)
			// Rec. 601 luma: what the eye reads as brightness, rather than the
			// average of three channels it does not weigh equally.
			g.Lum[y*g.DotsX+x] = uint8((299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000)
		}
	}
	stretch(g.Lum)

	g.Cell = make([]color.RGBA, cellsX*cellsY)
	for cy := range cellsY {
		for cx := range cellsX {
			var r, gr, b, n int
			for y := cy * perY; y < (cy+1)*perY; y++ {
				for x := cx * perX; x < (cx+1)*perX; x++ {
					c := scaled.RGBAAt(x, y)
					r, gr, b, n = r+int(c.R), gr+int(c.G), b+int(c.B), n+1
				}
			}
			if n == 0 {
				continue
			}
			g.Cell[cy*cellsX+cx] = color.RGBA{
				R: round(r/n), G: round(gr/n), B: round(b/n), A: 255,
			}
		}
	}
	return g
}

// round takes a channel to one of grainLevels steps.
func round(v int) uint8 {
	step := 255 / (grainLevels - 1)
	return uint8(min((v+step/2)/step, grainLevels-1) * step)
}

// stretch pulls the picture's brightness out to the full range.
//
// Half the covers ever printed are dark, and a dark picture dithered against a
// fixed threshold is a black rectangle. The ends are taken from percentiles
// rather than from the extremes, so one white corner cannot decide the exposure
// of the whole sleeve.
func stretch(lum []uint8) {
	if len(lum) == 0 {
		return
	}

	var hist [256]int
	for _, v := range lum {
		hist[v]++
	}

	at := func(share float64) int {
		want, seen := int(float64(len(lum))*share), 0
		for v, n := range hist {
			if seen += n; seen >= want {
				return v
			}
		}
		return 255
	}

	lo, hi := at(0.04), at(0.96)
	if hi-lo < 16 {
		return // flat enough that stretching it would only amplify noise
	}

	span := float64(hi - lo)
	for i, v := range lum {
		lum[i] = uint8(min(max((float64(v)-float64(lo))/span, 0), 1) * 255)
	}
}

// fit crops a cover to the shape it is being drawn in, from the middle.
func fit(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	want := float64(w) / float64(h)
	have := float64(b.Dx()) / float64(b.Dy())

	switch {
	case have > want:
		// Too wide: take a middle column of it.
		keep := int(float64(b.Dy()) * want)
		off := (b.Dx() - keep) / 2
		return crop(img, image.Rect(b.Min.X+off, b.Min.Y, b.Min.X+off+keep, b.Max.Y))
	case have < want:
		keep := int(float64(b.Dx()) / want)
		off := (b.Dy() - keep) / 2
		return crop(img, image.Rect(b.Min.X, b.Min.Y+off, b.Max.X, b.Min.Y+off+keep))
	}
	return img
}

// crop is SubImage where the image has it, and a copy where it does not.
func crop(img image.Image, r image.Rectangle) image.Image {
	type subImager interface {
		SubImage(image.Rectangle) image.Image
	}
	if s, ok := img.(subImager); ok {
		return s.SubImage(r)
	}

	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	return out
}
