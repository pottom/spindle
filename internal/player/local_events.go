package player

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

const (
	// reconnectDelay is how long to wait before dialling the daemon again. The
	// daemon is on loopback, so a failure means it is restarting rather than
	// that the network is unhappy; there is no point backing off far.
	reconnectDelay = time.Second

	// resyncEvery is a safety net. Every event the daemon sends is a hint that
	// something moved, not the state itself, so a missed one would go unnoticed
	// forever without an occasional unprompted look.
	resyncEvery = 30 * time.Second
)

// Watch keeps the local snapshot current until ctx is cancelled.
//
// It does not parse the events. Each one only says that something happened, and
// the daemon's /status answers what — which is one request against loopback and
// far less code than mirroring a state machine that lives somewhere else.
func (l *Local) Watch(ctx context.Context) {
	// Start with the truth, so the UI has something before the first event.
	_ = l.refresh(ctx)

	go l.resync(ctx)

	for ctx.Err() == nil {
		if err := l.listen(ctx); err != nil && ctx.Err() == nil {
			// A dropped connection is expected when the daemon restarts, and
			// there is nothing useful to say about it: the snapshot goes stale,
			// the periodic resync notices, and the next dial reconnects.
			select {
			case <-ctx.Done():
			case <-time.After(reconnectDelay):
			}
		}
	}
}

// listen holds one connection open, refreshing on every event it carries.
func (l *Local) listen(ctx context.Context) error {
	endpoint, err := l.eventsURL()
	if err != nil {
		return err
	}

	conn, _, err := websocket.Dial(ctx, endpoint, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow() //nolint:errcheck // closing a dead socket says nothing

	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return err
		}
		if err := l.refresh(ctx); err != nil && ctx.Err() == nil {
			return err
		}
	}
}

// resync looks without being prompted, so a missed event cannot leave the
// snapshot wrong indefinitely.
func (l *Local) resync(ctx context.Context) {
	ticker := time.NewTicker(resyncEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = l.refresh(ctx)
		}
	}
}
