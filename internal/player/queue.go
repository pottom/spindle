package player

import "context"

// QueueEditor is implemented by backends that can rewrite the queue, not just
// append to it. Spotify's Web API cannot: it offers no way to remove or reorder
// what is waiting, so only the local daemon can do this.
//
// It is kept out of Player on purpose. Widening the interface would force every
// backend to pretend it can do this, and the UI would lose the one honest way
// to ask.
type QueueEditor interface {
	// SetQueue replaces the hand-queued tracks, in order. Tracks that came from
	// the context — the rest of an album or playlist — are left alone, since
	// they are not the queue's to move.
	SetQueue(ctx context.Context, trackIDs []string) error
}

// trackURI is how Spotify names a track everywhere except the Web API, which
// prefers the bare id.
func trackURI(id string) string { return "spotify:track:" + id }
