//go:build darwin

package daemon

import (
	"context"
	"os/exec"
	"time"
)

// notify posts a desktop notification.
//
// Through osascript rather than a library: the frameworks that post
// notifications properly will only do it for a bundled application, and spindle
// is a binary somebody put on their PATH. The script bridge is what a terminal
// program is left with, it needs nothing installed, and the cost is that macOS
// attributes the notification to the scripting host rather than to spindle.
func notify(title, body string) {
	if title == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()

	script := "display notification " + appleQuote(body) +
		" with title " + appleQuote(title) + " subtitle \"spindle\""
	_ = exec.CommandContext(ctx, "osascript", "-e", script).Run()
}

// notifyTimeout keeps a wedged helper from piling up goroutines. Posting a
// notification takes milliseconds; anything longer has gone wrong.
const notifyTimeout = 5 * time.Second

// appleQuote wraps a string for AppleScript, where the only escapes are the
// quote and the backslash — and a track title is somebody else's text, which
// will eventually contain both.
func appleQuote(s string) string {
	var b []byte
	b = append(b, '"')
	for i := range len(s) {
		switch s[i] {
		case '"', '\\':
			b = append(b, '\\', s[i])
		default:
			b = append(b, s[i])
		}
	}
	return string(append(b, '"'))
}
