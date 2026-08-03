package ui

import "github.com/pottom/spindle/internal/ui/cover"

// Dimensions from SCREENS.md section 3. The table is authoritative; the ASCII
// drawings are not.
const (
	minWidth  = 64
	minHeight = 20

	maxFrameWidth = 100

	leftMargin  = 3
	rightMargin = 2
	columnGap   = 3
	minInfoCols = 32

	// artMargin is the blank line kept above and below the artwork.
	artMargin = 1
)

// layout is the geometry of one screen, derived purely from the terminal size
// and the pixel size of a cell.
type layout struct {
	interior   int // content width, capped so an ultrawide terminal stays readable
	artWidth   int // artwork area, in cells
	artHeight  int
	infoWidth  int // the column next to the artwork
	bodyHeight int // rows above the help bar
}

// computeLayout resolves the frame geometry for a terminal of w × h cells. It
// assumes the size already passed fitsMinimum.
func computeLayout(w, h, helpHeight int, hasBanner bool, cell cover.CellSize) layout {
	interior := min(w, maxFrameWidth)

	// A blank line and the help bar sit below the body; a banner takes one more.
	chrome := 1 + helpHeight
	if hasBanner {
		chrome++
	}
	bodyHeight := max(h-chrome, 0)

	artWidth, artHeight := artworkArea(interior, bodyHeight, cell)
	return layout{
		interior:   interior,
		artWidth:   artWidth,
		artHeight:  artHeight,
		infoWidth:  interior - leftMargin - artWidth - columnGap - rightMargin,
		bodyHeight: bodyHeight,
	}
}

// artworkArea gives the artwork as much room as the frame can spare while keeping
// it square on screen and leaving the information column its minimum width. Cells
// are taller than they are wide, so a square area needs roughly twice as many
// columns as rows.
// The artwork never takes more than half the width: past that it stops being a
// player and starts being a picture viewer with captions.
func artworkArea(interior, bodyHeight int, cell cover.CellSize) (width, height int) {
	maxWidth := max(min(interior-leftMargin-columnGap-minInfoCols-rightMargin, interior/2), 1)
	maxHeight := max(bodyHeight-2*artMargin-1, 1)

	height = max(min(maxHeight, maxWidth*cell.Width/cell.Height), 1)
	width = max(min(height*cell.Height/cell.Width, maxWidth), 1)
	return width, height
}

// fitsMinimum reports whether the player screen can be drawn at all.
func fitsMinimum(w, h int) bool {
	return w >= minWidth && h >= minHeight
}
