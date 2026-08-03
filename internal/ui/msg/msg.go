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

// NoActiveDevice reports that nothing is playing anywhere. It is a normal
// condition with its own screen, not a failure, so it is its own message rather
// than an Error.
type NoActiveDevice struct{}

// DevicesFetched carries the list of devices playback could be moved to.
type DevicesFetched struct {
	Devices []player.Device
}

// QueueFetched carries what is lined up after the current track.
type QueueFetched struct {
	Tracks []player.Track
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

// StateChanged reports that the backend noticed something move without being
// asked. Only the local daemon can say this; the Web API has no push.
type StateChanged struct{}

// Refetch asks for a fresh player state now. It is sent on a delay after a
// track change, because Spotify needs a moment to catch up with itself.
type Refetch struct{}

// VolumeSettled fires once the volume keys have stopped moving, so a held key
// sends one request instead of twenty.
type VolumeSettled struct {
	Seq int
}

// ControlDone reports that a control call succeeded, which is what clears a
// standing complaint about the account not being able to control playback.
type ControlDone struct{}

// RateLimited reports that Spotify wants to be left alone for a while.
type RateLimited struct {
	RetryAfter time.Duration
}

// Error carries a failure the UI is expected to show rather than crash on.
type Error struct {
	Err error
}
