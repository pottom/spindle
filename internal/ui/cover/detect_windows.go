package cover

import "os"

// askTerminal is not implemented on Windows. The query itself would be easy
// enough to write, but reading the reply needs a console-specific wait that
// cannot be exercised or tested from here, and a probe that hangs on stdin would
// swallow the user's first keystroke.
//
// Halfblock is therefore the automatic choice on Windows. Terminals that do
// speak the protocol there can still be driven with --cover=kitty.
func askTerminal(_, _ *os.File) []byte {
	return nil
}
