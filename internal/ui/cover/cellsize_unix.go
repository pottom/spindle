//go:build unix

package cover

import (
	"os"

	"golang.org/x/sys/unix"
)

// detectCellSize reads the pixel geometry from TIOCGWINSZ. DESIGN.md 5.4, step 1.
func detectCellSize(f *os.File) CellSize {
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
