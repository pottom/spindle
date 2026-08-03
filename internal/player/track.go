package player

import "time"

// Track is one entry of a search result or a playlist. It is deliberately
// narrower than State: a list needs identity and labels, not playback flags.
type Track struct {
	ID       string
	Title    string
	Artists  []string
	Album    string
	CoverURL string
	Duration time.Duration
}
