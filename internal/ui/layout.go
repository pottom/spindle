package ui

// Dimensions from SCREENS.md section 3. The table is authoritative; the ASCII
// drawings are not.
const (
	minWidth  = 64
	minHeight = 20

	maxFrameWidth = 100

	coverCells  = 20 // artwork is 20 × 10 cells
	coverRows   = 10
	leftMargin  = 2
	rightMargin = 2
	columnGap   = 3
	minInfoCols = 28

	// The artwork sits in a bordered box, so it claims one extra cell per side.
	coverBoxWidth  = coverCells + 2
	coverBoxHeight = coverRows + 2

	// Geometry of the "window too small" screen, SCREENS.md 4.7.
	tooSmallBoxWidth  = 28
	tooSmallBoxHeight = 8
	tooSmallMinWidth  = 26
)

// layout is the geometry of one frame, derived purely from the terminal size.
type layout struct {
	frameWidth int // including both border columns
	interior   int // frame width minus the borders
	infoWidth  int // the column next to the artwork
	bodyHeight int // rows available between the top border and the separator
}

// computeLayout resolves the frame geometry for a terminal of w × h cells. It
// assumes the size already passed fitsMinimum.
func computeLayout(w, h int, helpHeight int, hasBanner bool) layout {
	frameWidth := min(w, maxFrameWidth)
	interior := frameWidth - 2

	// Top border, separator above the help bar, help bar, bottom border. A
	// banner adds its own line plus the separator that divides it off.
	chrome := 3 + helpHeight
	if hasBanner {
		chrome += 2
	}

	return layout{
		frameWidth: frameWidth,
		interior:   interior,
		infoWidth:  interior - leftMargin - coverBoxWidth - columnGap - rightMargin,
		bodyHeight: max(h-chrome, 0),
	}
}

// fitsMinimum reports whether the player screen can be drawn at all.
func fitsMinimum(w, h int) bool {
	return w >= minWidth && h >= minHeight
}
