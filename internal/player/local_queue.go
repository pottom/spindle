package player

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// localQueue is what the daemon's /player/queue endpoint answers with. It
// carries no metadata: the daemon knows the order and the origin of each track,
// the Web API knows what they are called.
type localQueue struct {
	Tracks []localQueueTrack `json:"tracks"`
}

type localQueueTrack struct {
	URI    string `json:"uri"`
	Queued bool   `json:"queued"`
}

// Queue is the upcoming tracks, each marked with whether it was queued by hand.
//
// The order and the marks come from the daemon, the titles from the Web API.
// Pairing up two independently fetched lists was tried first and cannot be made
// to work: Spotify relinks tracks to their local equivalents, so the same song
// is a different id on each side. Asking for metadata by the daemon's own ids
// sidesteps the question entirely.
func (l *Local) Queue(ctx context.Context) ([]Track, error) {
	if l.idle() {
		return l.web.Queue(ctx)
	}

	origins, err := l.queueOrigins(ctx)
	if err != nil {
		// Falling back loses the marks, and with them the ability to edit — but
		// a queue nobody can reorder still beats no queue at all.
		return l.web.Queue(ctx)
	}
	if len(origins) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(origins))
	queued := make(map[string]bool, len(origins))
	for _, o := range origins {
		id := trackIDFromURI(o.URI)
		ids = append(ids, id)
		queued[id] = o.Queued
	}

	tracks, err := l.web.TracksByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tracks {
		tracks[i].Queued = queued[tracks[i].ID]
	}
	return tracks, nil
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

// queueOrigins asks the daemon which upcoming tracks were queued by hand.
func (l *Local) queueOrigins(ctx context.Context) ([]localQueueTrack, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/player/queue", nil)
	if err != nil {
		return nil, fmt.Errorf("build queue request: %w", err)
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch daemon queue: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch daemon queue: unexpected status %s", resp.Status)
	}

	var q localQueue
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("decode daemon queue: %w", err)
	}
	return q.Tracks, nil
}
