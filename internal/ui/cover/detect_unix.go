//go:build unix

package cover

import (
	"bytes"
	"os"
	"time"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

// probeGraphics sends the query and reads whatever comes back.
//
// The reply is read with poll(2) rather than os.File deadlines: Go's poller
// refuses to take a terminal, so SetReadDeadline fails with "file type does not
// support deadline" and every detection would come back negative.
func probeGraphics(out, in *os.File) bool {
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
