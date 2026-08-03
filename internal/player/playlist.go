package player

import "time"

// Playlist is a named collection of tracks, with enough metadata to list it
// without fetching its contents.
type Playlist struct {
	ID       string
	Name     string
	Owner    string
	CoverURL string
	Tracks   int
	Duration time.Duration
}
