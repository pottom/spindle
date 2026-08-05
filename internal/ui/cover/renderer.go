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
	return cols, rows, cols * cell.Width, rows * cell.Height
}
