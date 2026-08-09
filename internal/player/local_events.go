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

// dialTimeout bounds the attempt to open the stream; pingEvery is how often an
// open one is asked whether it is still there, and pingWithin how long it has
// to answer.
//
// A daemon that has got itself stuck still holds its port: the connection is
// accepted and then nothing happens, so an unbounded dial waits on a process
// that is never going to speak — and the interface goes on drawing the last
// thing it heard as though it were still true, which is the worst thing it
// could do.
//
// The open connection is asked rather than timed. Silence on it is not a fault:
// the daemon speaks when something happens, and a track playing through with
// nothing changing says nothing for minutes at a time. Treating that as a
// daemon lost put the whole interface into its offline state — which is not
// only wrong but visibly wrong, because the spectrum stops being asked for at
// all, so the picture freezes until the next look proves the daemon was there
// the whole time.
//
// Variables, so a test does not have to wait them out.
var (
	dialTimeout = 5 * time.Second
	pingEvery   = 20 * time.Second
	pingWithin  = 5 * time.Second
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

	// Whatever happened while there was no stream, this end has not heard about
	// — and a reconnection that waits for the next event to find out is an
	// interface showing the old picture for as long as nothing happens.
	_ = l.refresh(ctx)

	// Asked, rather than timed. See pingEvery.
	beat, stop := context.WithCancel(ctx)
	defer stop()
	go func() {
		ticker := time.NewTicker(pingEvery)
		defer ticker.Stop()

		for {
			select {
			case <-beat.Done():
				return
			case <-ticker.C:
				ask, done := context.WithTimeout(beat, pingWithin)
				err := conn.Ping(ask)
				done()
				if err != nil {
					// The read is what reports it: closing here ends that read
					// with the reason, and one place decides what a dead stream
					// means.
					conn.CloseNow() //nolint:errcheck // it is already dead
					return
				}
			}
		}
	}()

	for {
		_, _, err := conn.Read(ctx)
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
