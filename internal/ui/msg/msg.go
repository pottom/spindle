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
	Slot          int
	Art           cover.Art
}

// WordsReady carries a line of the song set in dots, with what it was set from
// so that a picture for a line the song has already left can be dropped.
type WordsReady struct {
	Text           string
	CellsX, CellsY int
	Grain          cover.Grain

	// Words says which word each dot belongs to, so that the one being sung can
	// be told from the ones that have been and the ones still to come.
	Words WordLayout
}

// WordLayout is where the words of a line landed once they were set.
//
// It is worked out while the type is being drawn, because that is the only
// moment the widths are known: a proportional face gives no formula for how
// wide a word is, only a measurement.
type WordLayout struct {
	Count int

	// DotsX is the width the layout was made for, and At maps a dot to the word
	// under it — one row of DotsX entries for each line of type, holding the
	// index of the word there or -1 for the space between two of them.
	DotsX int
	At    []int16

	// Tops and Bottoms are the dot rows each line of type covers.
	Tops, Bottoms []int

	// Lefts and Rights are the dot columns each mark of a row covers, its own
	// ink and not the air beside it. Only a row of marks fills them: a word of
	// type has no use for them, and what they are for is turning one mark round
	// on the spot without it sliding sideways as it goes.
	Lefts, Rights []int
}

// Middle is the dot in the middle of a word, which is what a piece bursting
// apart flies out from.
func (l WordLayout) Middle(word int) (int, int) {
	var first, last, top, bottom = l.DotsX, -1, 1 << 30, -1
	for i, at := range l.At {
		if int(at) != word {
			continue
		}
		x, line := i%max(l.DotsX, 1), i/max(l.DotsX, 1)
		first, last = min(first, x), max(last, x)
		if line < len(l.Tops) && line < len(l.Bottoms) {
			top, bottom = min(top, l.Tops[line]), max(bottom, l.Bottoms[line])
		}
	}
	if last < 0 {
		return 0, 0
	}
	return (first + last) / 2, (top + bottom) / 2
}

// WordAt is the word under a dot, or -1 where there is none.
func (l WordLayout) WordAt(x, y int) int {
	for i, top := range l.Tops {
		if y < top || y > l.Bottoms[i] || x < 0 || x >= l.DotsX {
			continue
		}
		if at := i*l.DotsX + x; at < len(l.At) {
			return int(l.At[at])
		}
	}
	return -1
}

// CoverFailed reports that artwork could not be produced. A missing cover is not
// important enough to interrupt use, so it carries no error for display.
type CoverFailed struct {
	URL           string
	Width, Height int
	Slot          int
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

	// Beat is where the beats of what is playing are, as of this frame. The
	// zero value means none was found — the first seconds of every record, and
	// the whole of one that has no beat to find.
	Beat player.Beat
}

// QueueFetched carries the track playing and what is lined up after it.
type QueueFetched struct {
	Current []player.Track // at most one, so the zero value carries nothing
	Tracks  []player.Track
}

// LibraryFetched carries one page of one of the library's lists. Kind says
// which — it is the ui's own numbering, passed back untouched — and Offset says
// where the page starts, so a page that arrives out of order can be told from
// the first one and appended rather than replacing what is already read.
type LibraryFetched struct {
	Kind      int
	Playlists []player.Playlist
	Albums    []player.Album
	Artists   []player.Artist
	Tracks    []player.Track
	Offset    int
	More      bool
	Next      int
}

// OpenedFetched carries one page of whatever has been opened: a playlist's or
// an album's tracks, or an artist's records. ID says which, so a page that
// arrives after the reader has gone somewhere else can be dropped.
//
// One message for the three because they are read the same way — a page at a
// time, appended to what is there, with the backend saying where the next one
// starts. Only which field is filled differs.
type OpenedFetched struct {
	ID     string
	Tracks []player.Track
	Albums []player.Album
	Offset int
	More   bool
	Next   int
}

// SearchResults carries what a query matched. Seq identifies the query, so a
// slow search landing after a newer one can be discarded.
//
// Kind is what was asked for: empty for a fresh query, which fetches every kind
// at once, and one kind when a list is being read further into.
type SearchResults struct {
	Seq     int
	Query   string
	Kind    player.SearchKind
	Matched bool
	Offset  int
	Results player.Results
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
