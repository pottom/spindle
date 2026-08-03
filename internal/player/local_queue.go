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
// Neither source has both halves. The Web API returns titles, artists and covers
// but will not say where a track came from; the daemon knows exactly that but
// holds only ids. Asking each for what it already has costs one extra local
// request and no extra Web API quota — looking the metadata up by the daemon's
// ids would have been tidier, but the batch endpoint is closed to applications
// registered since late 2024.
//
// The two lists agree on order, so they are matched by position and checked by
// id as they go. Where a track was relinked to a locally licensed copy the ids
// differ, which is what OriginID is for; anything still unmatched ends the run,
// because a mark on the wrong row would offer an edit that silently does
// nothing.
func (l *Local) Queue(ctx context.Context) (Queue, error) {
	q, err := l.web.Queue(ctx)
	if err != nil || len(q.Upcoming) == 0 || l.idle() {
		return q, err
	}

	origins, err := l.queueOrigins(ctx)
	if err != nil {
		// Without the marks the queue is read-only, which the UI works out for
		// itself. A queue nobody can reorder still beats no queue at all.
		return q, nil
	}

	for i := range q.Upcoming {
		if i >= len(origins) || !sameTrack(origins[i].URI, q.Upcoming[i]) {
			break
		}
		q.Upcoming[i].Queued = origins[i].Queued
	}
	return q, nil
}

// sameTrack reports whether a device's track uri names the track the Web API
// described, allowing for relinking.
func sameTrack(uri string, t Track) bool {
	id := trackIDFromURI(uri)
	return id == t.ID || (t.OriginID != "" && id == t.OriginID)
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
