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

// Queue is the upcoming tracks, each carrying the id the device knows it by and
// whether it sits in the part of the list whose order can be set outright.
//
// Neither source has both halves. The Web API returns titles, artists and covers
// but will not say where a track came from; the daemon knows exactly that, and
// is the only one whose ids the device will answer to — the same recording can
// carry a different id on each side, and telling the device the Web API's id
// makes it search its context, fail, and quietly rewind to the first track.
// Looking the metadata up by the daemon's ids instead would be tidier, but the
// batch endpoint is closed to applications registered since late 2024.
//
// The two lists describe the same sequence, so they are matched by position.
// That only holds if nothing moved between the two requests, which is what the
// second reading of the daemon's queue checks.
func (l *Local) Queue(ctx context.Context) (Queue, error) {
	if l.idle() {
		return l.web.Queue(ctx)
	}

	before, err := l.queueOrigins(ctx)
	if err != nil {
		return l.web.Queue(ctx)
	}

	q, err := l.web.Queue(ctx)
	if err != nil || len(q.Upcoming) == 0 {
		return q, err
	}

	after, err := l.queueOrigins(ctx)
	if err != nil || !sameOrigins(before, after) {
		// The track changed under us. Without the marks the queue is read-only,
		// which the UI works out for itself, and the next refresh is moments
		// away — better than acting on a list that has already moved on.
		return q, nil
	}

	if len(q.Upcoming) > len(before) {
		q.Upcoming = q.Upcoming[:len(before)]
	}
	for i := range q.Upcoming {
		q.Upcoming[i].Queued = before[i].Queued
		q.Upcoming[i].DeviceID = trackIDFromURI(before[i].URI)
	}
	return q, nil
}

// sameOrigins reports whether two readings of the queue describe the same one.
func sameOrigins(a, b []localQueueTrack) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

// queueOrigins asks the daemon which upcoming tracks are its queue rather than
// its context. The distinction is not shown — to the user there is one list —
// but only the queue's order can be written back wholesale.
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
