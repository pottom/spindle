package daemon

import (
	"fmt"
	"strconv"
	"time"
)

// MaxCrossfade is the longest overlap on offer. Spotify's own clients stop at
// twelve seconds, and past that a transition stops being a transition.
const MaxCrossfade = 12 * time.Second

// DefaultCrossfade is off. Gapless is what a record sounds like, and an overlap
// nobody asked for is an effect applied to somebody else's music.
const DefaultCrossfade = time.Duration(0)

// ParseCrossfade reads a configured overlap, in seconds.
func ParseCrossfade(s string) (time.Duration, error) {
	if s == "" || s == "off" {
		return DefaultCrossfade, nil
	}

	secs, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unknown crossfade %q: want seconds, or off", s)
	}
	d := time.Duration(secs * float64(time.Second))
	if d < 0 || d > MaxCrossfade {
		return 0, fmt.Errorf("crossfade %q is outside 0 to %s", s, MaxCrossfade)
	}
	return d, nil
}
