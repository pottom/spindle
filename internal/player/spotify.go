package player

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
)

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

// NewSpotify builds the backend around an authenticated HTTP client. The
// transport is wrapped so a 429 becomes a typed error with its Retry-After
// intact; the Spotify client would otherwise discard the header.
func NewSpotify(httpClient *http.Client, opts ...spotify.ClientOption) *Spotify {
	wrapped := *httpClient
	wrapped.Transport = &rateLimiter{base: httpClient.Transport}
	return &Spotify{client: spotify.New(&wrapped, opts...)}
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

// Queue returns the tracks lined up after the current one.
func (s *Spotify) Queue(ctx context.Context) (Queue, error) {
	q, err := s.client.GetQueue(ctx)
	if err != nil {
		return Queue{}, classify("fetch queue", err)
	}
	if q == nil {
		return Queue{}, nil
	}

	out := Queue{Upcoming: make([]Track, 0, len(q.Items))}
	if q.CurrentlyPlaying.ID != "" {
		current := trackFromFull(&q.CurrentlyPlaying)
		out.Current = &current
	}
	for i := range q.Items {
		out.Upcoming = append(out.Upcoming, trackFromFull(&q.Items[i]))
	}
	return out, nil
}

func (s *Spotify) AddToQueue(ctx context.Context, trackID string) error {
	if err := s.client.QueueSong(ctx, spotify.ID(trackID)); err != nil {
		return classify("add to queue", err)
	}
	return nil
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
			ID:          p.ID.String(),
			Name:        p.Name,
			Owner:       ownerName(p.Owner),
			CoverURL:    bestImage(p.Images),
			Tracks:      int(p.Tracks.Total),
			Description: plainText(p.Description),
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

func (s *Spotify) Play(ctx context.Context) error {
	return classify("resume playback", s.client.Play(ctx))
}

func (s *Spotify) Pause(ctx context.Context) error {
	return classify("pause playback", s.client.Pause(ctx))
}

func (s *Spotify) Next(ctx context.Context) error {
	return classify("skip to next track", s.client.Next(ctx))
}

func (s *Spotify) Previous(ctx context.Context) error {
	return classify("skip to previous track", s.client.Previous(ctx))
}

func (s *Spotify) Seek(ctx context.Context, pos time.Duration) error {
	return classify("seek", s.client.Seek(ctx, int(pos.Milliseconds())))
}

func (s *Spotify) SetVolume(ctx context.Context, pct int) error {
	return classify("set volume", s.client.Volume(ctx, min(max(pct, 0), 100)))
}

func (s *Spotify) SetShuffle(ctx context.Context, on bool) error {
	return classify("set shuffle", s.client.Shuffle(ctx, on))
}

func (s *Spotify) SetRepeat(ctx context.Context, mode string) error {
	return classify("set repeat", s.client.Repeat(ctx, repeatFromSpotify(mode)))
}

// PlayTrack starts one track on its own, with no surrounding context.
func (s *Spotify) PlayTrack(ctx context.Context, trackID string) error {
	uri := spotify.URI("spotify:track:" + trackID)
	return classify("play track", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		URIs: []spotify.URI{uri},
	}))
}

// PlayFrom jumps to a track inside whatever is playing. The context has to be
// named again: Spotify treats a bare track as a one-track context and forgets
// the rest.
func (s *Spotify) PlayFrom(ctx context.Context, trackID string) error {
	cur, err := s.client.PlayerState(ctx)
	if err != nil {
		return classify("play from queue", err)
	}
	if cur == nil || cur.PlaybackContext.URI == "" {
		// Nothing to preserve, so the plain track is the honest answer.
		return s.PlayTrack(ctx, trackID)
	}

	uri := cur.PlaybackContext.URI
	track := spotify.URI(trackURI(trackID))
	return classify("play from queue", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		PlaybackContext: &uri,
		PlaybackOffset:  &spotify.PlaybackOffset{URI: track},
	}))
}

// PlayPlaylist starts a playlist at the given position, so the rest of it stays
// queued behind the track that was chosen.
func (s *Spotify) PlayPlaylist(ctx context.Context, playlistID string, offset int) error {
	uri := spotify.URI("spotify:playlist:" + playlistID)
	pos := max(offset, 0)
	return classify("play playlist", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		PlaybackContext: &uri,
		PlaybackOffset:  &spotify.PlaybackOffset{Position: &pos},
	}))
}

// TransferTo moves playback to another device, keeping whatever was playing
// playing. Passing false here would silently pause the music the user just asked
// to hear somewhere else.
func (s *Spotify) TransferTo(ctx context.Context, deviceID string) error {
	return classify("transfer playback", s.client.TransferPlayback(ctx, spotify.ID(deviceID), true))
}
