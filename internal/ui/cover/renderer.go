package cover

import "image"

// Renderer turns a decoded image into something embeddable in a Bubble Tea view.
type Renderer interface {
	// Render fits img into a wCells × hCells area and returns a string
	// embeddable in View().
	//
	// seq rises with every request. A renderer that writes to the terminal
	// itself needs it: two loads can be in flight at once, and if the older one
	// finishes last the terminal is left holding a picture of a size the screen
	// is no longer drawing.
	//
	// slot says which picture on the screen this is. A screen showing two at
	// once — what the cursor is on beside what is playing — needs them kept
	// apart: a renderer that holds one image in the terminal would have the two
	// replacing each other.
	Render(img image.Image, wCells, hCells int, seq uint64, slot int) (string, error)
	Name() string
}

// fitCells works out how many cells the image should actually occupy inside a
// wCells × hCells box once its aspect ratio is respected, along with the pixel
// size that corresponds to.
// FitCells is that arithmetic for whoever has to lay out around a picture: how
// many cells across and down one of the given shape will really be drawn in.
//
// The layout has to know exactly, not nearly. A wall of covers puts its marks
// against the edges of the pictures, so a caller that guessed a cell wider than
// the renderer draws would put every one of them out of true.
func FitCells(imgW, imgH, wCells, hCells int, cell CellSize) (cols, rows int) {
	cols, rows, _, _ = fitCells(image.NewGray(image.Rect(0, 0, imgW, imgH)), wCells, hCells, cell)
	return cols, rows
}

func fitCells(img image.Image, wCells, hCells int, cell CellSize) (cols, rows, pxW, pxH int) {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 || wCells <= 0 || hCells <= 0 {
		return 0, 0, 0, 0
	}

	boxW := wCells * cell.Width
	boxH := hCells * cell.Height

	scale := min(float64(boxW)/float64(sw), float64(boxH)/float64(sh))
	pxW = max(int(float64(sw)*scale), 1)
	pxH = max(int(float64(sh)*scale), 1)

	cols = min(max(pxW/cell.Width, 1), wCells)
	rows = min(max(pxH/cell.Height, 1), hCells)

	// And then the rectangle is squared off against the picture's own shape.
	//
	// Flooring each side on its own leaves a rectangle of cells whose shape is
	// not the picture's — up to a cell out in either direction. That does not
	// matter to a renderer that draws the picture itself, cell by cell, and it
	// matters a great deal to a terminal that is handed the rectangle and scales
	// the picture into it: whatever is left over is a band down one side or along
	// the foot, in a place nothing in this program can see or account for.
	//
	// So one side is kept and the other is taken to the nearest whole cell that
	// matches the picture. Whichever side gives a rectangle nearer the picture's
	// shape is the one that is kept.
	if byRows := nearest(rows*cell.Height*sw, sh*cell.Width, wCells); byRows > 0 {
		if wrongness(cols, rows, sw, sh, cell) > wrongness(byRows, rows, sw, sh, cell) {
			cols = byRows
		}
	}
	if byCols := nearest(cols*cell.Width*sh, sw*cell.Height, hCells); byCols > 0 {
		if wrongness(cols, rows, sw, sh, cell) > wrongness(cols, byCols, sw, sh, cell) {
			rows = byCols
		}
	}
	return cols, rows, cols * cell.Width, rows * cell.Height
}

// nearest is a/b rounded to the nearest whole number, capped, and at least one.
func nearest(a, b, most int) int {
	if b == 0 {
		return 0
	}
	return min(max((a+b/2)/b, 1), most)
}

// wrongness is how far a rectangle of cells is from the picture's own shape, in
// pixels of the longer side.
func wrongness(cols, rows, sw, sh int, cell CellSize) int {
	across, down := cols*cell.Width, rows*cell.Height
	out := across*sh - down*sw
	if out < 0 {
		out = -out
	}
	return out / max(sw, sh)
}
