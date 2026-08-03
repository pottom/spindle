package cover

import (
	"os"

	"golang.org/x/sys/unix"
)

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

// DetectCellSize asks the tty for its pixel geometry via TIOCGWINSZ. Terminals
// are free to report zeroes, in which case the 2:1 assumption stands.
func DetectCellSize(f *os.File) CellSize {
	if f == nil {
		return defaultCellSize
	}
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return defaultCellSize
	}
	return CellSize{
		Width:    int(ws.Xpixel) / int(ws.Col),
		Height:   int(ws.Ypixel) / int(ws.Row),
		Measured: true,
	}
}
