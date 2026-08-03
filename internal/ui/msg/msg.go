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

// PlaylistsFetched carries the library listing.
type PlaylistsFetched struct {
	Playlists []player.Playlist
}

// PlaylistTracksFetched carries the contents of one playlist.
type PlaylistTracksFetched struct {
	PlaylistID string
	Tracks     []player.Track
}

// SearchResults carries the hits for a query. Seq identifies the query, so a
// slow search landing after a newer one can be discarded.
type SearchResults struct {
	Seq     int
	Tracks  []player.Track
	Query   string
	Matched bool
}

// CoverSettled fires once the browse cursor has stopped moving for long enough
// to be worth loading artwork for.
type CoverSettled struct {
	Seq int
}

// Error carries a failure the UI is expected to show rather than crash on.
type Error struct {
	Err error
}
