package cover

import (
	"os"
	"time"

	"github.com/charmbracelet/x/term"
)

// graphicsQuery asks whether the terminal speaks the kitty graphics protocol and
// immediately follows up with a device attributes request. Every terminal answers
// the second one, so a reply that contains only that is a definitive "no".
// DESIGN.md 5.3.
const graphicsQuery = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\\x1b[c"

// detectTimeout bounds the wait. The program must never block on this.
const detectTimeout = 200 * time.Millisecond

// SupportsKitty reports whether the terminal answered the graphics query. It must
// be called before Bubble Tea takes over the terminal.
func SupportsKitty(out, in *os.File) bool {
	if out == nil || in == nil || !term.IsTerminal(in.Fd()) {
		return false
	}
	return probeGraphics(out, in)
}
