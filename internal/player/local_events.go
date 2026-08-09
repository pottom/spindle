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

// dialTimeout bounds the attempt to open the stream, and silentFor how long an
// open one may say nothing before it is treated as gone.
//
// A daemon that has got itself stuck still holds its port: the connection is
// accepted and then nothing happens, so waiting on either without a deadline is
// waiting on a process that is never going to speak — and the interface goes on
// drawing the last thing it heard as though it were still true, which is the
// worst thing it could do. Nothing here is chatty enough to run into them: the
// daemon sends an event on every change, and this end asks for a fresh look on
// a timer regardless.
//
// Variables, so a test does not have to wait them out.
var (
	dialTimeout = 5 * time.Second
	silentFor   = 90 * time.Second
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
			// there is nothing useful to say about it — but it does have to be
			// noticed: what the daemon last said stops being what is happening
			// the moment it stops answering.
			l.lost()

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

	dial, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(dial, endpoint, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow() //nolint:errcheck // closing a dead socket says nothing

	for {
		// Bounded, so a stream that has gone quiet is noticed here rather than
		// waited on: see silentFor.
		hear, done := context.WithTimeout(ctx, silentFor)
		_, _, err := conn.Read(hear)
		done()
		if err != nil {
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
			// The look also answers whether the daemon is there at all, which
			// the event stream can be slow to notice: a socket held open by a
			// process that has gone away reads as connected until it is used.
			if err := l.refresh(ctx); err != nil && ctx.Err() == nil {
				l.lost()
			}
		}
	}
}
