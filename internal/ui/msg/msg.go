package msg

import (
	"time"

	"github.com/pottom/spindle/internal/player"
)

// Tick fires once a second and drives the local progress clock.
type Tick struct {
	Time time.Time
}

// StateFetched carries a fresh snapshot from the backend.
type StateFetched struct {
	State *player.State
}

// CoverReady carries rendered artwork, tagged with the URL it came from so a
// result that arrives after the track has moved on can be discarded.
type CoverReady struct {
	URL string
	Art string
}

// CoverFailed reports that artwork could not be produced. A missing cover is not
// important enough to interrupt use, so it carries no error for display.
type CoverFailed struct {
	URL string
}

// Error carries a failure the UI is expected to show rather than crash on.
type Error struct {
	Err error
}
