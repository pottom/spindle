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

// Error carries a failure the UI is expected to show rather than crash on.
type Error struct {
	Err error
}
