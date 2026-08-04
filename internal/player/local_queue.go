package player

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
