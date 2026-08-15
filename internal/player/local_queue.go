package player

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// localQueue is what the daemon's /player/queue endpoint answers with: the
// upcoming tracks, in order, named.
type localQueue struct {
	Current *localQueueTrack  `json:"current"`
	Tracks  []localQueueTrack `json:"tracks"`
}

type localQueueTrack struct {
	URI    string `json:"uri"`
	Queued bool   `json:"queued"`

	Name        string   `json:"name"`
	ArtistNames []string `json:"artist_names"`
	AlbumName   string   `json:"album_name"`
	AlbumCover  string   `json:"album_cover_url"`
	Duration    int64    `json:"duration"`
	ReleaseDate string   `json:"release_date"`
	TrackNumber int      `json:"track_number"`
	DiscNumber  int      `json:"disc_number"`
	TotalTracks int      `json:"total_tracks"`
	AlbumType   string   `json:"album_type"`
	Popularity  int      `json:"popularity"`
	Tempo       float64  `json:"tempo"`
}

func (t localQueueTrack) toTrack() Track {
	popularity := t.Popularity
	return Track{
		ID:          trackIDFromURI(t.URI),
		Title:       t.Name,
		Artists:     t.ArtistNames,
		Album:       t.AlbumName,
		CoverURL:    t.AlbumCover,
		Duration:    time.Duration(t.Duration) * time.Millisecond,
		Queued:      t.Queued,
		Released:    releaseDate(t.ReleaseDate),
		TrackNumber: t.TrackNumber,
		DiscNumber:  t.DiscNumber,
		TotalTracks: t.TotalTracks,
		AlbumType:   t.AlbumType,
		Popularity:  &popularity,
		Tempo:       t.Tempo,
	}
}

// Queue is what is playing and what comes next, both as the device holds them.
//
// The Web API is deliberately not consulted. It answers with the server's copy
// of the queue, which lags behind anything edited here — a track removed and
// then played past would come back — and it identifies the same recordings
// under ids of its own that the device will not answer to.
func (l *Local) Queue(ctx context.Context) (Queue, error) {
	if l.idle() {
		return l.web.Queue(ctx)
	}

	return l.queueTracks(ctx)
}

// QueueIsLocal reports whether the queue is coming from the daemon rather than
// from Spotify.
//
// It is asked by whoever decides how often to ask for the queue. The daemon
// answers over localhost and costs nothing worth counting; the Web API is
// somebody's quota, and a caller that thinks it is talking to the daemon while
// it is talking to Spotify can spend that quota in a minute.
func (l *Local) QueueIsLocal() bool { return !l.idle() }

// queueTracks asks the daemon what is playing and what is coming.
func (l *Local) queueTracks(ctx context.Context) (Queue, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/player/queue", nil)
	if err != nil {
		return Queue{}, fmt.Errorf("build queue request: %w", err)
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return Queue{}, fmt.Errorf("fetch daemon queue: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return Queue{}, fmt.Errorf("fetch daemon queue: unexpected status %s", resp.Status)
	}

	var q localQueue
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return Queue{}, fmt.Errorf("decode daemon queue: %w", err)
	}

	out := Queue{Upcoming: make([]Track, 0, len(q.Tracks))}
	if q.Current != nil {
		current := q.Current.toTrack()
		out.Current = &current
	}
	for _, t := range q.Tracks {
		out.Upcoming = append(out.Upcoming, t.toTrack())
	}
	return out, nil
}

// releaseDate turns the daemon's protobuf rendering of a release date —
// "year:1987 month:11 day:21", with the later fields missing when the label
// never recorded them — into what the rest of spindle speaks.
func releaseDate(s string) string {
	var out string
	for _, part := range strings.Fields(s) {
		name, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		switch name {
		case "year":
			out = value
		case "month", "day":
			if out == "" {
				return ""
			}
			out += "-" + value
		}
	}
	return out
}

// SetQueue replaces the hand-queued tracks, leaving the context alone.
func (l *Local) SetQueue(ctx context.Context, trackIDs []string) error {
	uris := make([]string, 0, len(trackIDs))
	for _, id := range trackIDs {
		uris = append(uris, trackURI(id))
	}

	err := l.post(ctx, "/player/set_queue", struct {
		Uris []string `json:"uris"`
	}{Uris: uris})
	if err != nil {
		return err
	}

	// The daemon does not announce a queue change, so nothing would tell the UI
	// to look again.
	l.notify()
	return nil
}

// Reorder puts the named tracks at the head of what is coming. The daemon does
// the whole thing itself, for the same reason Drop does: a track that came from
// the context can only be moved by lifting it out and carrying everything it
// passed into the queue.
func (l *Local) Reorder(ctx context.Context, trackIDs []string) error {
	uris := make([]string, 0, len(trackIDs))
	for _, id := range trackIDs {
		uris = append(uris, trackURI(id))
	}

	err := l.post(ctx, "/player/reorder", struct {
		Uris []string `json:"uris"`
	}{Uris: uris})
	if err != nil {
		return err
	}

	// The daemon does not announce a queue change, so nothing would tell the UI
	// to look again.
	l.notify()
	return nil
}

// Drop removes an upcoming track. The daemon does the whole thing itself: a
// track from the context can only be skipped by moving the cursor onto it, and
// everything passed over on the way has to be carried into the queue first.
func (l *Local) Drop(ctx context.Context, trackID string) error {
	err := l.post(ctx, "/player/drop", struct {
		Uri string `json:"uri"`
	}{Uri: trackURI(trackID)})
	if err != nil {
		return err
	}

	// The daemon does not announce a queue change, so nothing would tell the UI
	// to look again.
	l.notify()
	return nil
}

// localContext is what the daemon's /player/context endpoint answers with: what
// a playlist, album or artist holds, named the same way the queue's tracks are.
type localContext struct {
	URI    string            `json:"uri"`
	Offset int               `json:"offset"`
	Tracks []localQueueTrack `json:"tracks"`
}

// PlaylistTracks asks the daemon what a playlist holds.
//
// The Web API cannot answer this. Measured against a live account, it refuses
// /playlists/{id}/tracks outright and /playlists/{id}/items for anything the
// user does not own — a client id registered after Spotify's clampdown is not
// allowed to read a playlist it did not create. The daemon is a player rather
// than an application and the protocol it speaks has no such rule, so it is
// asked first and the Web API kept only for when there is no daemon at all.
func (l *Local) PlaylistTracksPage(ctx context.Context, playlistID string, offset int) (Page[Track], error) {
	page, err := l.contextTracks(ctx, "spotify:playlist:"+playlistID, offset)
	if err == nil {
		return page, nil
	}
	return l.web.PlaylistTracksPage(ctx, playlistID, offset)
}

func (l *Local) contextTracks(ctx context.Context, uri string, offset int) (Page[Track], error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/player/context", nil)
	if err != nil {
		return Page[Track]{}, fmt.Errorf("build context request: %w", err)
	}
	start := max(offset, 0)
	q := req.URL.Query()
	q.Set("uri", uri)
	q.Set("offset", strconv.Itoa(start))
	q.Set("limit", strconv.Itoa(pageLimit))
	req.URL.RawQuery = q.Encode()

	resp, err := l.http.Do(req)
	if err != nil {
		return Page[Track]{}, fmt.Errorf("fetch %s: %w", uri, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return Page[Track]{}, fmt.Errorf("fetch %s: unexpected status %s", uri, resp.Status)
	}

	var page localContext
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return Page[Track]{}, fmt.Errorf("decode %s: %w", uri, err)
	}

	out := make([]Track, 0, len(page.Tracks))
	for _, t := range page.Tracks {
		// A context lists what it holds, not what was asked for by hand.
		track := t.toTrack()
		track.Queued = false
		out = append(out, track)
	}

	// The daemon reports neither a total nor a link onwards: it walks the
	// context until it has the tracks that were asked for and stops there. A
	// full answer is therefore the only sign that more exist, and the price is
	// one empty request at the end of a list whose length divides by the page
	// size.
	return Page[Track]{Items: out, More: len(out) == pageLimit, Next: start + pageLimit}, nil
}
