package ui

import (
	"testing"
	"time"
)

// A frame is due a frame after the last one, not a frame after the work.
//
// The command that fetches a frame is issued at the end of an update, so a whole
// frame's sleep started from after the fetch, the update and the draw rather
// than from where the last frame began. Measured on the running interface: 28.1
// frames a second against the 30 it is set to, with nothing reported late
// because each one was only a couple of milliseconds over.
func TestFramesAreDueOnTheGridAndNotAfterTheWork(t *testing.T) {
	scopePace.due = time.Time{}

	at := time.Now()
	first := scopeDue(at)
	if got := first.Sub(at); got != scopeInterval {
		t.Fatalf("the first frame was due in %s, not %s", got, scopeInterval)
	}

	// The frame arrived, and the work after it took a few milliseconds. The next
	// one is still due on the grid.
	work := 4 * time.Millisecond
	second := scopeDue(first.Add(work))
	if got := second.Sub(first); got != scopeInterval {
		t.Errorf("after %s of work the next frame was due in %s, not %s", work, got, scopeInterval)
	}

	// Away for a while — stopped, off screen, or the machine busy with something
	// else. The grid starts again rather than firing off everything it missed.
	away := second.Add(5 * time.Second)
	third := scopeDue(away)
	if got := third.Sub(away); got != scopeInterval {
		t.Errorf("coming back after five seconds the frame was due in %s, not %s", got, scopeInterval)
	}
}
