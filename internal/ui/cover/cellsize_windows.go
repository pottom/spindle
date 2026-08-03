package cover

import "os"

// detectCellSize has nothing to ask on Windows: the console API reports the
// window in cells and never in pixels, and no Windows terminal answers the
// CSI 14 t query either. The 2:1 assumption stands, and the kitty backend
// supersamples to compensate.
func detectCellSize(_ *os.File) CellSize {
	return defaultCellSize
}
