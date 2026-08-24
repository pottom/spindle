package daemon

import "testing"

// A track title is somebody else's text, and will eventually carry the two
// characters AppleScript reads as its own.
//
// It lives in a file named for the platform because the function it tests does:
// appleQuote is only built on macOS, and this test sitting in notify_test.go
// meant the whole of internal/daemon failed to compile on Linux — every test in
// the package unrunnable on the machine most likely to need them.
func TestQuotingATitleThatFightsBack(t *testing.T) {
	got := appleQuote(`Say "Hello" \ Goodbye`)
	if want := `"Say \"Hello\" \\ Goodbye"`; got != want {
		t.Errorf("appleQuote() = %s, want %s", got, want)
	}
}
