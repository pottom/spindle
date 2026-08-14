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

// The bounds a real cell falls inside. A cell is a letter box: taller than it is
// wide, by somewhere between a third again and three times.
//
// Because a number can be reported and still be nonsense. Measured over ssh from
// a Windows client: 5 × 19 px, which is a cell nearly four times taller than it
// is wide and no font on earth. The pixel fields of the kernel's window size come
// from whatever the ssh client said, and a client that guesses badly makes every
// picture drawn from them the wrong shape — while Measured says the figure was
// reported rather than assumed, so nothing downstream doubts it.
const (
	cellLeast    = 3
	cellTallest  = 3.0
	cellSquarest = 1.3
)

// plausible reports whether a cell size could be a terminal's.
func (c CellSize) plausible() bool {
	if c.Width < cellLeast || c.Height < cellLeast*2 {
		return false
	}
	ratio := float64(c.Height) / float64(c.Width)
	return ratio >= cellSquarest && ratio <= cellTallest
}

// DetectCellSize asks the terminal for its pixel geometry. Terminals are free to
// report nothing, in which case the 2:1 assumption stands — and free to report
// something impossible, in which case it stands as well.
func DetectCellSize(f *os.File) CellSize {
	if f == nil {
		return defaultCellSize
	}
	if got := detectCellSize(f); got.plausible() {
		return got
	}
	return defaultCellSize
}
