package player

import (
	"context"
	"errors"
	"time"
)

// ErrNoActiveDevice reports that nothing is currently playing anywhere. It is a
// normal condition, not a failure: the UI answers it with a device list.
var ErrNoActiveDevice = errors.New("no active playback device")

// Player is the only playback interface the UI is allowed to depend on.
type Player interface {
	State(ctx context.Context) (*State, error)
	Play(ctx context.Context) error
	Pause(ctx context.Context) error
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	Seek(ctx context.Context, pos time.Duration) error
	SetVolume(ctx context.Context, pct int) error
	SetShuffle(ctx context.Context, on bool) error
	SetRepeat(ctx context.Context, mode string) error
	Devices(ctx context.Context) ([]Device, error)
	TransferTo(ctx context.Context, deviceID string) error

	// AddToQueue puts a track at the end of the queue. Rewriting the queue is
	// not part of this interface: see QueueEditor.
	AddToQueue(ctx context.Context, trackID string) error

	// Queue is what is playing and what comes next. Knowing what comes next is
	// what lets the UI show a skip instantly instead of waiting for Spotify to
	// admit it happened.
	Queue(ctx context.Context) (Queue, error)

	// Browsing. Search matches tracks; an empty query yields no results rather
	// than everything.
	Search(ctx context.Context, query string) ([]Track, error)
	Playlists(ctx context.Context) ([]Playlist, error)
	PlaylistTracks(ctx context.Context, playlistID string) ([]Track, error)

	// PlayTrack starts one track on its own; PlayPlaylist starts a playlist at
	// the given position.
	PlayTrack(ctx context.Context, trackID string) error

	// PlayFrom jumps to a track already coming up, keeping the album or
	// playlist it belongs to. PlayTrack would drop everything behind it.
	PlayFrom(ctx context.Context, trackID string) error

	// PlayNow starts a track that is not in the list at all, and leaves the
	// list alone: what was queued still follows it. PlayTrack is the other
	// reading — one track and nothing else — and it throws the queue away,
	// which is not what someone who has spent a minute filling it means by
	// "play this one".
	PlayNow(ctx context.Context, trackID string) error
	PlayPlaylist(ctx context.Context, playlistID string, offset int) error
}
