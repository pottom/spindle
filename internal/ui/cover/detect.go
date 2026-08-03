package cover

import (
	"bytes"
	"os"
	"time"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
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
//
// The reply is read with poll(2) rather than os.File deadlines: Go's poller
// refuses to take a terminal, so SetReadDeadline fails with "file type does not
// support deadline" and every detection would come back negative.
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
	return bytes.Contains(readReply(int(in.Fd())), []byte("\x1b_G"))
}

// readReply collects whatever the terminal says until the device attributes
// response lands or the timeout expires.
func readReply(fd int) []byte {
	deadline := time.Now().Add(detectTimeout)
	buf := make([]byte, 256)
	var reply []byte

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return reply
		}

		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(remaining.Milliseconds())+1)
		if err == unix.EINTR {
			continue
		}
		if err != nil || n == 0 {
			return reply
		}

		n, err = unix.Read(fd, buf)
		if n > 0 {
			reply = append(reply, buf[:n]...)
		}
		if err != nil || endsDeviceAttributes(reply) {
			return reply
		}
	}
}

// endsDeviceAttributes reports whether the device attributes reply — a CSI
// sequence terminated by "c" — has arrived, which means the terminal has said
// everything it is going to say.
func endsDeviceAttributes(reply []byte) bool {
	csi := bytes.LastIndex(reply, []byte("\x1b["))
	return csi >= 0 && bytes.IndexByte(reply[csi:], 'c') > 0
}
