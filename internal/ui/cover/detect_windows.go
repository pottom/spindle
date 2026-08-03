package cover

import "os"

// probeGraphics is not implemented on Windows. The query itself would be easy
// enough to write, but reading the reply needs a console-specific wait that
// cannot be exercised or tested from here, and a probe that hangs on stdin would
// swallow the user's first keystroke.
//
// Halfblock is therefore the automatic choice on Windows. Terminals that do
// speak the protocol there — WezTerm, for one — can still be driven with
// --cover=kitty.
func probeGraphics(_, _ *os.File) bool {
	return false
}
