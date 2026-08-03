package msg

import (
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// Tick fires once a second and drives the local progress clock.
type Tick struct {
	Time time.Time
}

// StateFetched carries a fresh snapshot from the backend.
type StateFetched struct {
	State *player.State
}

// CoverReady carries rendered artwork, tagged with the URL and cell size it was
// produced for, so a result that arrives after the track or the window has
// changed can be discarded.
type CoverReady struct {
	URL           string
	Width, Height int
	Art           cover.Art
}

// CoverFailed reports that artwork could not be produced. A missing cover is not
// important enough to interrupt use, so it carries no error for display.
type CoverFailed struct {
	URL           string
	Width, Height int
}

// Error carries a failure the UI is expected to show rather than crash on.
type Error struct {
	Err error
}
