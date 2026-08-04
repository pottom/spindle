package player

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zmb3/spotify/v2"
)

const (
	// maxCoverPixels is the largest artwork worth fetching: past this the extra
	// detail is invisible in a terminal and only costs bandwidth.
	maxCoverPixels = 640

	// pageLimit is how much of a browsing list one request asks for. Fifty is
	// the most the Web API hands back for the user's playlists and for a
	// playlist's items, so anything smaller would only mean more round trips
	// for the same list.
	pageLimit = 50

	// searchLimit is the same for a search, and it is not fifty.
	//
	// Measured 2026-08-04 against a live account: /v1/search answers a limit of
	// ten and refuses eleven with 400 "Invalid limit", whatever is being
	// searched for, while every other list still takes fifty. It is not
	// documented anywhere; asking for fifty simply makes every search fail.
	searchLimit = 10

	// searchWindow is as far into a search as Spotify will go: offset 900 with
	// a limit of ten answers, and offset 1000 is refused. Paging has to stop
	// there rather than walk into a 400 at the end of a long list.
	searchWindow = 1000
)

// Spotify drives playback through the Spotify Web API.
type Spotify struct {
	client *spotify.Client

	// http is the same authenticated client the library is built on, and base
	// is where the API lives, kept for the few answers the library cannot read.
	// See PlaylistsPage. A field rather than a constant so a test can point it
	// at a stub, which is what the library's own option does for the rest.
	http *http.Client
	base string

	// followed remembers which cursor reaches which offset of the followed
	// artists, the one list the Web API refuses to page by offset. See
	// FollowedArtists. Browsing runs in tea.Cmd goroutines, hence the lock.
	mu       sync.Mutex
	followed map[int]string
}

// NewSpotify builds the backend around an authenticated HTTP client. The
// transport is wrapped so a 429 becomes a typed error with its Retry-After
// intact; the Spotify client would otherwise discard the header.
func NewSpotify(httpClient *http.Client, opts ...spotify.ClientOption) *Spotify {
	wrapped := *httpClient
	wrapped.Transport = &rateLimiter{base: httpClient.Transport}
	return &Spotify{
		client: spotify.New(&wrapped, opts...),
		http:   &wrapped,
		base:   "https://api.spotify.com/v1/",
	}
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

func (s *Spotify) SearchPage(ctx context.Context, query string, offset int) (Page[Track], error) {
	res, err := s.Search(ctx, query, SearchTracks, offset)
	return res.Tracks, err
}

// Search matches a query against one kind, or against all of them when the kind
// is empty. Spotify answers every kind asked for in a single request, so a
// fresh query learns what else matched for the price of the tracks alone.
func (s *Spotify) Search(ctx context.Context, query string, kind SearchKind, offset int) (Results, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Results{}, nil
	}

	var want spotify.SearchType
	for _, k := range searchTypes(kind) {
		want |= k
	}

	start := max(offset, 0)
	if start >= searchWindow {
		return Results{}, nil
	}

	res, err := s.client.Search(ctx, query, want, spotify.Limit(searchLimit), spotify.Offset(start))
	if err != nil {
		return Results{}, fmt.Errorf("search: %w", err)
	}
	if res == nil {
		return Results{}, nil
	}

	// Spotify's own "next" link keeps pointing past the window it will answer,
	// so the end of the window is where paging stops, not where it is told to.
	next := start + searchLimit
	more := next < searchWindow

	out := Results{}
	if res.Tracks != nil {
		items := make([]Track, 0, len(res.Tracks.Tracks))
		for i := range res.Tracks.Tracks {
			items = append(items, trackFromFull(&res.Tracks.Tracks[i]))
		}
		out.Tracks = Page[Track]{Items: items, More: more && res.Tracks.Next != "", Next: next}
	}
	if res.Albums != nil {
		items := make([]Album, 0, len(res.Albums.Albums))
		for i := range res.Albums.Albums {
			items = append(items, albumFromSimple(&res.Albums.Albums[i]))
		}
		out.Albums = Page[Album]{Items: items, More: more && res.Albums.Next != "", Next: next}
	}
	if res.Artists != nil {
		items := make([]Artist, 0, len(res.Artists.Artists))
		for i := range res.Artists.Artists {
			items = append(items, artistFromFull(&res.Artists.Artists[i]))
		}
		out.Artists = Page[Artist]{Items: items, More: more && res.Artists.Next != "", Next: next}
	}
	if res.Playlists != nil {
		items := make([]Playlist, 0, len(res.Playlists.Playlists))
		for i := range res.Playlists.Playlists {
			items = append(items, playlistFromSimple(&res.Playlists.Playlists[i]))
		}
		out.Playlists = Page[Playlist]{Items: items, More: more && res.Playlists.Next != "", Next: next}
	}
	return out, nil
}

// searchTypes is what to ask Spotify for. An empty kind asks for everything,
// which is one request rather than four.
func searchTypes(kind SearchKind) []spotify.SearchType {
	switch kind {
	case SearchAlbums:
		return []spotify.SearchType{spotify.SearchTypeAlbum}
	case SearchArtists:
		return []spotify.SearchType{spotify.SearchTypeArtist}
	case SearchPlaylists:
		return []spotify.SearchType{spotify.SearchTypePlaylist}
	case SearchTracks:
		return []spotify.SearchType{spotify.SearchTypeTrack}
	default:
		return []spotify.SearchType{
			spotify.SearchTypeTrack, spotify.SearchTypeAlbum,
			spotify.SearchTypeArtist, spotify.SearchTypePlaylist,
		}
	}
}

// PlaylistsPage lists the account's playlists.
//
// Read from the answer directly rather than through the client library, for one
// field: how many tracks each playlist holds. Spotify used to report that as
// "tracks" and now reports it as "items", the same rename that moved a
// playlist's contents from /tracks to /items, and every library still looking
// for the old name reads every playlist as empty. Both names are accepted here,
// so the count survives whichever one arrives.
func (s *Spotify) PlaylistsPage(ctx context.Context, offset int) (Page[Playlist], error) {
	start := max(offset, 0)

	var page struct {
		Next  string `json:"next"`
		Items []struct {
			ID          string          `json:"id"`
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Images      []spotify.Image `json:"images"`
			Owner       struct {
				DisplayName string `json:"display_name"`
				ID          string `json:"id"`
			} `json:"owner"`
			Tracks struct {
				Total int `json:"total"`
			} `json:"tracks"`
			Contents struct {
				Total int `json:"total"`
			} `json:"items"`
		} `json:"items"`
	}
	url := fmt.Sprintf("%sme/playlists?limit=%d&offset=%d", s.base, pageLimit, start)
	if err := s.read(ctx, url, &page); err != nil {
		return Page[Playlist]{}, fmt.Errorf("fetch playlists: %w", err)
	}

	out := make([]Playlist, 0, len(page.Items))
	for _, item := range page.Items {
		out = append(out, Playlist{
			ID:          item.ID,
			Name:        item.Name,
			Owner:       cmp.Or(item.Owner.DisplayName, item.Owner.ID),
			CoverURL:    bestImage(item.Images),
			Tracks:      max(item.Tracks.Total, item.Contents.Total),
			Description: plainText(item.Description),
		})
	}
	return Page[Playlist]{Items: out, More: page.Next != "", Next: start + pageLimit}, nil
}

// read fetches one answer from the Web API into out. It is for the few places
// the client library cannot be asked; everything else goes through it.
func (s *Spotify) read(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return classify("read", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return classify("read", fmt.Errorf("unexpected status %s", resp.Status))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *Spotify) PlaylistTracksPage(ctx context.Context, playlistID string, offset int) (Page[Track], error) {
	start := max(offset, 0)
	page, err := s.client.GetPlaylistItems(ctx, spotify.ID(playlistID),
		spotify.Limit(pageLimit), spotify.Offset(start))
	if err != nil {
		return Page[Track]{}, fmt.Errorf("fetch playlist tracks: %w", err)
	}
	if page == nil {
		return Page[Track]{}, nil
	}

	out := make([]Track, 0, len(page.Items))
	for _, item := range page.Items {
		// Podcast episodes and tracks unavailable in this market come back empty.
		if item.Track.Track == nil {
			continue
		}
		out = append(out, trackFromFull(item.Track.Track))
	}

	// What follows is taken from Spotify's own link to the next page rather than
	// from the length of this one: the items dropped just above would otherwise
	// make a full page look like the last one, and they are why the next offset
	// counts what was asked for instead of what survived.
	return Page[Track]{Items: out, More: page.Next != "", Next: start + pageLimit}, nil
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
// PlayNow is PlayTrack against the Web API: it has no way to start a track
// without replacing what was playing, queue and all. The local daemon does.
func (s *Spotify) PlayNow(ctx context.Context, trackID string) error {
	return s.PlayTrack(ctx, trackID)
}

func (s *Spotify) PlayTrack(ctx context.Context, trackID string) error {
	return s.PlayTrackOn(ctx, trackID, "")
}

// PlayTrackOn is PlayTrack aimed at one device. Naming the device is not a
// nicety: measured against a live account, a play call with no device answers
// 404 "No active device found" whenever Spotify does not already consider one
// active — which is most of the time, including while our own daemon is playing.
func (s *Spotify) PlayTrackOn(ctx context.Context, trackID, deviceID string) error {
	uri := spotify.URI("spotify:track:" + trackID)
	return classify("play track", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		DeviceID: deviceRef(deviceID),
		URIs:     []spotify.URI{uri},
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
// PlayContext starts a whole album, artist or playlist. Naming the device
// matters here as everywhere else: a play call without one is refused with 404
// whenever Spotify does not already believe a device is active.
func (s *Spotify) PlayContext(ctx context.Context, uri string) error {
	return s.PlayContextOn(ctx, uri, "")
}

func (s *Spotify) PlayContextOn(ctx context.Context, uri, deviceID string) error {
	context := spotify.URI(uri)
	return classify("play", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		DeviceID:        deviceRef(deviceID),
		PlaybackContext: &context,
	}))
}

func (s *Spotify) PlayContextAt(ctx context.Context, uri string, offset int) error {
	return s.PlayContextAtOn(ctx, uri, offset, "")
}

// PlayContextAtOn is PlayContextAt aimed at one device. See PlayTrackOn for why
// the device has to be named.
func (s *Spotify) PlayContextAtOn(ctx context.Context, uri string, offset int, deviceID string) error {
	context := spotify.URI(uri)
	pos := max(offset, 0)
	return classify("play", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		DeviceID:        deviceRef(deviceID),
		PlaybackContext: &context,
		PlaybackOffset:  &spotify.PlaybackOffset{Position: &pos},
	}))
}

func (s *Spotify) PlayPlaylist(ctx context.Context, playlistID string, offset int) error {
	return s.PlayPlaylistOn(ctx, playlistID, offset, "")
}

// PlayPlaylistOn is PlayPlaylist aimed at one device. See PlayTrackOn for why
// the device has to be named.
func (s *Spotify) PlayPlaylistOn(ctx context.Context, playlistID string, offset int, deviceID string) error {
	uri := spotify.URI("spotify:playlist:" + playlistID)
	pos := max(offset, 0)
	return classify("play playlist", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		DeviceID:        deviceRef(deviceID),
		PlaybackContext: &uri,
		PlaybackOffset:  &spotify.PlaybackOffset{Position: &pos},
	}))
}

// playTracksMost is how many tracks may be named in one play call. Spotify
// takes a few hundred and refuses more; a list longer than this is played from
// its head and read on from there.
const playTracksMost = 300

func (s *Spotify) PlayTracks(ctx context.Context, trackIDs []string, offset int) error {
	return s.PlayTracksOn(ctx, trackIDs, offset, "")
}

// PlayTracksOn is PlayTracks aimed at one device. See PlayTrackOn for why the
// device has to be named.
//
// The offset is applied by dropping what comes before it rather than by asking
// Spotify for a position: the list is only as long as the cap allows, and
// starting it at the chosen track is what makes the rest follow.
func (s *Spotify) PlayTracksOn(ctx context.Context, trackIDs []string, offset int, deviceID string) error {
	if from := max(offset, 0); from < len(trackIDs) {
		trackIDs = trackIDs[from:]
	}
	if len(trackIDs) == 0 {
		return nil
	}
	if len(trackIDs) > playTracksMost {
		trackIDs = trackIDs[:playTracksMost]
	}

	uris := make([]spotify.URI, 0, len(trackIDs))
	for _, id := range trackIDs {
		uris = append(uris, spotify.URI(trackURI(id)))
	}
	return classify("play tracks", s.client.PlayOpt(ctx, &spotify.PlayOptions{
		DeviceID: deviceRef(deviceID),
		URIs:     uris,
	}))
}

// deviceRef is the id in the shape PlayOptions wants, and nil for "wherever
// Spotify thinks playback is", which is what a spindle without a daemon means.
func deviceRef(deviceID string) *spotify.ID {
	if deviceID == "" {
		return nil
	}
	id := spotify.ID(deviceID)
	return &id
}

// TransferTo moves playback to another device, keeping whatever was playing
// playing. Passing false here would silently pause the music the user just asked
// to hear somewhere else.
func (s *Spotify) TransferTo(ctx context.Context, deviceID string) error {
	return classify("transfer playback", s.client.TransferPlayback(ctx, spotify.ID(deviceID), true))
}
