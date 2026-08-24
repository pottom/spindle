// Package browser opens a URL in whatever the desktop uses to look at one.
//
// It is its own package because two parts of spindle need it and neither owns
// it: the Web API login, which runs in a terminal somebody is watching, and the
// daemon's own sign-in, which runs in a process nobody is watching at all.
package browser

import (
	"os/exec"
	"runtime"
)

// Open asks the desktop to open a URL. It reports whether the command could be
// started at all — over SSH xdg-open fails silently, which is exactly why every
// caller prints the URL regardless of the answer.
func Open(url string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return false
	}
	// The browser outlives us; reap the launcher so it does not linger.
	go cmd.Wait() //nolint:errcheck // the exit status says nothing useful
	return true
}
