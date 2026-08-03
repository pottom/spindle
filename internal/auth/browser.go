package auth

import (
	"os/exec"
	"runtime"
)

// openBrowser asks the desktop to open a URL. It reports whether the command
// could be started at all — over SSH xdg-open fails silently, which is exactly
// why the caller prints the URL regardless.
func openBrowser(url string) bool {
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
