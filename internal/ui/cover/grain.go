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

	// The whole sleeve, not a piece of it. A cover is square and a terminal is
	// not, so the picture is the largest one of its own shape that fits, set in
	// the middle with the screen dark either side of it. Cropping to the shape
	// of the window would fill it, and would also throw away whatever the
	// designer put in the corners.
	//
	// A braille dot happens to be about square — a cell is twice as tall as it
	// is wide and holds two dots across by four down — so a square picture is
	// as many dots one way as the other, and no aspect correction is needed
	// beyond counting them.
	pw, ph := inside(img.Bounds().Dx(), img.Bounds().Dy(), g.DotsX, g.DotsY)
	if pw <= 0 || ph <= 0 {
		return g
	}
	offX, offY := (g.DotsX-pw)/2, (g.DotsY-ph)/2

	scaled := image.NewRGBA(image.Rect(0, 0, pw, ph))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Src, nil)

	// The brightness is stretched over the picture alone: with the dark margin
	// counted in, every cover would come out as an overexposed stamp in the
	// middle of a black screen.
	lum := make([]uint8, pw*ph)
	for y := range ph {
		for x := range pw {
			c := scaled.RGBAAt(x, y)
			// Rec. 601 luma: what the eye reads as brightness, rather than the
			// average of three channels it does not weigh equally.
			lum[y*pw+x] = uint8((299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000)
		}
	}
	stretch(lum)

	g.Lum = make([]uint8, g.DotsX*g.DotsY)
	for y := range ph {
		copy(g.Lum[(y+offY)*g.DotsX+offX:], lum[y*pw:(y+1)*pw])
	}

	g.Cell = make([]color.RGBA, cellsX*cellsY)
	for cy := range cellsY {
		for cx := range cellsX {
			var r, gr, b, n int
			for y := cy * perY; y < (cy+1)*perY; y++ {
				for x := cx * perX; x < (cx+1)*perX; x++ {
					px, py := x-offX, y-offY
					if px < 0 || py < 0 || px >= pw || py >= ph {
						continue
					}
					c := scaled.RGBAAt(px, py)
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

// inside is the largest picture of the cover's own shape that fits in the space
// there is.
func inside(imgW, imgH, w, h int) (int, int) {
	if imgW <= 0 || imgH <= 0 {
		return 0, 0
	}
	if imgW*h > w*imgH {
		return w, max(w*imgH/imgW, 1) // the width runs out first
	}
	return max(h*imgW/imgH, 1), h
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


