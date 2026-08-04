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

	// Browsing, read from anywhere in the list. Search matches tracks; an empty
	// query yields no results rather than everything. None of the three is
	// short on a real account — a library runs to hundreds of playlists and a
	// playlist to thousands of tracks — and every backend answers with a page,
	// so a caller that never asks for the second one is showing a list that has
	// been cut without saying so.
	//
	// The offset counts items, not pages, and the page size is the backend's own
	// business — it is whatever its source hands over in one request. A caller
	// starts at zero, follows Page.Next while Page.More, and stops there.
	SearchPage(ctx context.Context, query string, offset int) (Page[Track], error)
	PlaylistsPage(ctx context.Context, offset int) (Page[Playlist], error)
	PlaylistTracksPage(ctx context.Context, playlistID string, offset int) (Page[Track], error)

	// The library, on the same terms: an offset counting items, Page.Next to
	// follow and Page.More to say whether to.
	LikedTracks(ctx context.Context, offset int) (Page[Track], error)
	SavedAlbums(ctx context.Context, offset int) (Page[Album], error)
	FollowedArtists(ctx context.Context, offset int) (Page[Artist], error)

	// What one album or one artist holds. An album's tracks can arrive without
	// the album on them — the Web API does not repeat its name and cover on
	// every row — and that costs nothing, because a caller that asked for one
	// album has both already.
	//
	// An artist's own recordings are what ArtistAlbums lists. What the artist is
	// best known for is a separate question, and one no backend is obliged to
	// answer: see ArtistTopTracks.
	AlbumTracks(ctx context.Context, albumID string, offset int) (Page[Track], error)
	ArtistAlbums(ctx context.Context, artistID string, offset int) (Page[Album], error)

	// RecentlyPlayed is the listening history, most recently played first, and
	// is the one list here that is not a page. Spotify keeps some fifty entries
	// and walks them by timestamp rather than by offset, so there is a limit to
	// ask for and nothing to scroll into. A track played twice appears twice:
	// that is what a history is, and folding the repeats together would be a
	// different list than the one asked for.
	RecentlyPlayed(ctx context.Context, limit int) ([]Track, error)

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
