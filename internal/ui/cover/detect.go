package cover

import (
	"bytes"
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

	state, err := term.MakeRaw(in.Fd())
	if err != nil {
		return false
	}
	defer term.Restore(in.Fd(), state) //nolint:errcheck // nothing useful to do here

	if _, err := out.WriteString(graphicsQuery); err != nil {
		return false
	}

	deadline := time.Now().Add(detectTimeout)
	if err := in.SetReadDeadline(deadline); err != nil {
		return false
	}
	defer in.SetReadDeadline(time.Time{}) //nolint:errcheck // best effort

	var reply []byte
	buf := make([]byte, 256)
	for time.Now().Before(deadline) {
		n, err := in.Read(buf)
		reply = append(reply, buf[:n]...)

		// The device attributes reply ends in "c"; once it has arrived we have
		// seen everything the terminal is going to say.
		if bytes.ContainsRune(reply, 'c') || err != nil {
			break
		}
	}
	return bytes.Contains(reply, []byte("\x1b_G"))
}
