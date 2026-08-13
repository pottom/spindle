package cover

import (
	"image"

	"golang.org/x/image/draw"
)

// Grain is a picture taken apart into what a visualiser can move: how bright
// every dot of it is, at the resolution a braille cell gives.
//
// A cell holds eight dots and one colour, so the shape of the picture has to
// live in the dots — which are only ever on or off — and any colour belongs to
// the cell underneath them. That is also why this is worth doing at all: at
// eight dots to a cell a full screen is a few hundred thousand of them, which
// is enough of a picture to read a line of type off.
//
// It is in this package because it began as a way of drawing a sleeve. What it
// grinds now is the lyric screen's type, set as an image and taken apart the
// same way.
type Grain struct {
	DotsX, DotsY   int
	CellsX, CellsY int

	// Lum is how bright each dot is, 0..255, stretched so that the picture uses
	// the whole range whatever it was like to begin with.
	Lum []uint8
}

// Grind takes a decoded picture apart.
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

	// The brightness is stretched over the picture alone: with the dark margin
	// counted in, every cover would come out as an overexposed stamp in the
	// middle of a black screen.
	lum := make([]uint8, pw*ph)

	// A picture that is already the size it will be drawn at, and already grey,
	// is read straight.
	//
	// This is not the rare case: it is every line of every lyric. The words are
	// set at exactly the dots they will occupy, and sending them through the
	// scaler resampled a picture into itself — one RGBA copy to go in, another
	// to come out, and the kernel's working buffers between them. Measured at
	// 352 by 84 cells, a line change cost 9.1 ms and 12.4 MB, and 12 MB arriving
	// every two seconds is a collection every two seconds. The frames that went
	// missing went missing at changes, which is exactly when this ran.
	if grey, ok := img.(*image.Gray); ok && img.Bounds().Dx() == pw && img.Bounds().Dy() == ph {
		b := grey.Bounds()
		for y := range ph {
			row := grey.Pix[(y+b.Min.Y-b.Min.Y)*grey.Stride:]
			copy(lum[y*pw:(y+1)*pw], row[:pw])
		}
	} else {
		scaled := image.NewRGBA(image.Rect(0, 0, pw, ph))
		draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Src, nil)
		for y := range ph {
			for x := range pw {
				c := scaled.RGBAAt(x, y)
				// Rec. 601 luma: what the eye reads as brightness, rather than
				// the average of three channels it does not weigh equally.
				lum[y*pw+x] = uint8((299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000)
			}
		}
	}
	stretch(lum)

	g.Lum = make([]uint8, g.DotsX*g.DotsY)
	for y := range ph {
		copy(g.Lum[(y+offY)*g.DotsX+offX:], lum[y*pw:(y+1)*pw])
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
