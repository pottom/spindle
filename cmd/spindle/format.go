package main

import (
	"fmt"
	"strings"
	"time"
)

// clock renders a duration the way a player does: minutes and seconds, with an
// hours field only when there is one, so a three-minute song does not read as
// 0:03:12.
func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	seconds := int(d / time.Second)
	if hours := seconds / 3600; hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// millis converts the daemon's millisecond counts, which is how every duration
// arrives from it.
func millis(ms int64) time.Duration { return time.Duration(ms) * time.Millisecond }

// joinArtists names everyone credited. A track with no artists at all is
// possible — a local file the daemon knows nothing about — and reads better
// empty than as a stray separator.
func joinArtists(names []string) string { return strings.Join(names, ", ") }
