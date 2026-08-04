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

// LyricsFetched carries the words of a track, or none at all — a track without
// lyrics is the ordinary case rather than a failure. TrackID is what they
// belong to, so an answer that arrives after a skip can be thrown away.
type LyricsFetched struct {
	TrackID string
	Synced  bool
	Lines   []player.Lyric
}

// WaveformReady carries one frame of what is being heard, and paces the trace:
// the next frame is asked for when this one lands.
type WaveformReady struct {
	Samples []float32
	Bands   []float32
}

// QueueFetched carries the track playing and what is lined up after it.
type QueueFetched struct {
	Current []player.Track // at most one, so the zero value carries nothing
	Tracks  []player.Track
}

// PlaylistsFetched carries one page of the library listing. Offset says where
// the page starts, so a page that arrives out of order can be told from the
// first one and appended rather than replacing what is already read.
type PlaylistsFetched struct {
	Playlists []player.Playlist
	Offset    int
	More      bool
	Next      int
}

// PlaylistTracksFetched carries one page of a playlist's contents.
type PlaylistTracksFetched struct {
	PlaylistID string
	Tracks     []player.Track
	Offset     int
	More       bool
	Next       int
}

// SearchResults carries one page of hits for a query. Seq identifies the query,
// so a slow search landing after a newer one can be discarded.
type SearchResults struct {
	Seq     int
	Tracks  []player.Track
	Query   string
	Matched bool
	Offset  int
	More    bool
	Next    int
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

// OrderSettled fires once the queue's move keys have stopped, so a run of
// presses is sent as one edit rather than one per press.
type OrderSettled struct {
	Seq int
}

// PlayDone reports that a request to start something has been answered, one way
// or the other. It is what lets the next one go: two overlapping requests to
// play can be applied in either order, and the device would end up on the one
// asked for first.
//
// Result carries whatever the call itself produced — a success, an error, a
// rate limit — for the model to handle as it would any other control.
type PlayDone struct {
	Result any
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
