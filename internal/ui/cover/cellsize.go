package cover

import "os"

// CellSize is the pixel size of one terminal cell. Aspect-correct scaling is
// impossible without it.
type CellSize struct {
	Width  int
	Height int

	// Measured distinguishes a figure the terminal reported from the assumed
	// fallback. When it is false, callers that care about resolution should
	// supersample rather than trust these numbers.
	Measured bool
}

// defaultCellSize is the 2:1 height-to-width ratio assumed when the terminal
// will not say. DESIGN.md 5.4, step 3.
var defaultCellSize = CellSize{Width: 10, Height: 20}

// DetectCellSize asks the terminal for its pixel geometry. Terminals are free to
// report nothing, in which case the 2:1 assumption stands.
func DetectCellSize(f *os.File) CellSize {
	if f == nil {
		return defaultCellSize
	}
	return detectCellSize(f)
}
