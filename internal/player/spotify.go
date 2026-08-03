package player

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
)

// ErrNotImplemented marks the parts of the interface the live backend has not
// grown yet. It is a placeholder with a deadline, not an abstraction.
var ErrNotImplemented = errors.New("not implemented for the Spotify backend")

const (
	// maxCoverPixels is the largest artwork worth fetching: past this the extra
	// detail is invisible in a terminal and only costs bandwidth.
	maxCoverPixels = 640

	// searchLimit and listLimit bound the pages we ask for. One screenful is
	// generous; the lists are for finding something, not for reading end to end.
	searchLimit = 50
	listLimit   = 50
)

// Spotify drives playback through the Spotify Web API.
type Spotify struct {
	client *spotify.Client
}

// NewSpotify wraps an authenticated client.
func NewSpotify(client *spotify.Client) *Spotify {
	return &Spotify{client: client}
}

// State reports what the active device is doing. Spotify answers 204 when
// nothing is playing anywhere, which the client turns into a zero value rather
// than an error — so an empty answer is translated here into ErrNoActiveDevice.
// This is the normal entry path, not a failure.
func (s *Spotify) State(ctx context.Context) (*State, error) {
	ps, err := s.client.PlayerState(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch player state: %w", err)
	}
	if ps == nil || ps.Device.ID == "" {
		return nil, ErrNoActiveDevice
	}

	st := &State{
		Playing:    ps.Playing,
		Shuffle:    ps.ShuffleState,
		Repeat:     repeatFromSpotify(ps.RepeatState),
		Volume:     int(ps.Device.Volume),
		DeviceID:   ps.Device.ID.String(),
		DeviceName: ps.Device.Name,
		Progress:   time.Duration(ps.Progress) * time.Millisecond,
	}

	// A device can be active with nothing loaded on it — an idle speaker, say.
	if ps.Item != nil {
		st.TrackID = ps.Item.ID.String()
		st.Title = ps.Item.Name
		st.Artists = artistNames(ps.Item.Artists)
		st.Album = ps.Item.Album.Name
		st.CoverURL = bestImage(ps.Item.Album.Images)
		st.Duration = time.Duration(ps.Item.Duration) * time.Millisecond
	}
	return st, nil
}

func (s *Spotify) Devices(ctx context.Context) ([]Device, error) {
	devices, err := s.client.PlayerDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch devices: %w", err)
	}

	out := make([]Device, 0, len(devices))
	for _, d := range devices {
		out = append(out, Device{
			ID:     d.ID.String(),
			Name:   d.Name,
			Type:   strings.ToLower(d.Type),
			Active: d.Active,
		})
	}
	return out, nil
}

func (s *Spotify) Search(ctx context.Context, query string) ([]Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	res, err := s.client.Search(ctx, query, spotify.SearchTypeTrack, spotify.Limit(searchLimit))
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	if res == nil || res.Tracks == nil {
		return nil, nil
	}

	out := make([]Track, 0, len(res.Tracks.Tracks))
	for i := range res.Tracks.Tracks {
		out = append(out, trackFromFull(&res.Tracks.Tracks[i]))
	}
	return out, nil
}

func (s *Spotify) Playlists(ctx context.Context) ([]Playlist, error) {
	page, err := s.client.CurrentUsersPlaylists(ctx, spotify.Limit(listLimit))
	if err != nil {
		return nil, fmt.Errorf("fetch playlists: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	out := make([]Playlist, 0, len(page.Playlists))
	for _, p := range page.Playlists {
		out = append(out, Playlist{
			ID:       p.ID.String(),
			Name:     p.Name,
			Owner:    ownerName(p.Owner),
			CoverURL: bestImage(p.Images),
			Tracks:   int(p.Tracks.Total),
			// Spotify does not report a playlist's total duration, and adding it
			// up would mean fetching every track. The UI omits what is zero.
		})
	}
	return out, nil
}

func (s *Spotify) PlaylistTracks(ctx context.Context, playlistID string) ([]Track, error) {
	page, err := s.client.GetPlaylistItems(ctx, spotify.ID(playlistID), spotify.Limit(listLimit))
	if err != nil {
		return nil, fmt.Errorf("fetch playlist tracks: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	out := make([]Track, 0, len(page.Items))
	for _, item := range page.Items {
		// Podcast episodes and tracks unavailable in this market come back empty.
		if item.Track.Track == nil {
			continue
		}
		out = append(out, trackFromFull(item.Track.Track))
	}
	return out, nil
}

// Playback control arrives in M3.

func (s *Spotify) Play(context.Context) error                { return ErrNotImplemented }
func (s *Spotify) Pause(context.Context) error               { return ErrNotImplemented }
func (s *Spotify) Next(context.Context) error                { return ErrNotImplemented }
func (s *Spotify) Previous(context.Context) error            { return ErrNotImplemented }
func (s *Spotify) Seek(context.Context, time.Duration) error { return ErrNotImplemented }
func (s *Spotify) SetVolume(context.Context, int) error      { return ErrNotImplemented }
func (s *Spotify) SetShuffle(context.Context, bool) error    { return ErrNotImplemented }
func (s *Spotify) SetRepeat(context.Context, string) error   { return ErrNotImplemented }
func (s *Spotify) TransferTo(context.Context, string) error  { return ErrNotImplemented }
func (s *Spotify) PlayTrack(context.Context, string) error   { return ErrNotImplemented }

func (s *Spotify) PlayPlaylist(context.Context, string, int) error { return ErrNotImplemented }
