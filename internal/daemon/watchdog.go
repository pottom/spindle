package daemon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// The watchdog, which exists because a network that comes back does not bring
// the device back with it.
//
// Measured, on a laptop whose wifi dropped for a couple of minutes: the session
// lost the access point ("did not receive last pong ack from accesspoint, 174s
// passed"), and the player's goroutine stopped answering anything at all.
// go-librespot's own API answered 503 to every request from then on — which is
// our guard doing its job, saying "stuck" rather than hanging — and it never
// recovered. The net came back; the device did not. It had to be killed by hand,
// and it would not even let go of its lock.
//
// The daemon is one loop. A goroutine wedged in a read cannot be unwedged from
// outside, and nothing inside it is going to notice, so the only honest answer
// is to end the process and let a fresh one take the device. That is what this
// does: it asks the daemon's own API how it is, and when the answer has been
// "stuck" for long enough that no outage explains it, it gives up.
//
// Who starts the next one is the interface's business — see the reviving in
// cmd/spindle. A device that is gone can be replaced; a device that is present
// and deaf cannot.

// The three are variables rather than constants so that a test can watch a
// wedge happen in a moment rather than in a minute.
var (
	// watchEvery is how often the daemon is asked how it is. Rare enough to be
	// free, often enough that a wedge is noticed inside a minute.
	watchEvery = 10 * time.Second

	// watchPatience is how long an answer is waited for. The API answers in a
	// fifth of a millisecond when the loop is running and not at all when it is
	// not, so anything in between is generous.
	watchPatience = 3 * time.Second

	// wedgeAfter is how long everything has to be failing before the device is
	// declared lost.
	//
	// A minute. A dropped connection recovers inside a few seconds or it does
	// not recover: librespot reconnects on its own where it can, and the case
	// this is for is the one where the loop that would do the reconnecting is
	// itself stuck. Long enough that a passing outage is never called a wedge,
	// short enough that somebody who put a record on and walked off comes back
	// to a working device rather than a dead one.
	wedgeAfter = 60 * time.Second
)

// ErrWedged reports that the playback loop stopped answering and did not come
// back. The process is expected to end: nothing inside it can mend this.
var ErrWedged = errors.New("the playback device stopped answering")

// watch asks the daemon's own API how it is, and returns when it has been
// unanswerable for wedgeAfter. It returns nil if ctx ends first.
//
// It waits for the first good answer before it starts counting: a daemon that
// has not finished starting up is not a daemon that has stopped.
func watch(ctx context.Context, port int, logf func(string, ...any)) error {
	client := &http.Client{Timeout: watchPatience}
	url := fmt.Sprintf("http://%s:%d/status", DefaultHost, port)

	var stuckSince time.Time
	started := false

	tick := time.NewTicker(watchEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}

		if answering(ctx, client, url) {
			if !stuckSince.IsZero() {
				logf("the device is answering again")
			}
			started, stuckSince = true, time.Time{}
			continue
		}
		if !started {
			continue // it has not come up yet, so it cannot have stopped
		}
		if stuckSince.IsZero() {
			stuckSince = time.Now()
			logf("the device has stopped answering; giving it %s", wedgeAfter)
			continue
		}
		if time.Since(stuckSince) >= wedgeAfter {
			return ErrWedged
		}
	}
}

// answering reports whether the API took a request and answered it. A 503 is
// our own guard saying the loop would not take it, which is exactly the state
// this is watching for.
func answering(ctx context.Context, client *http.Client, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close() //nolint:errcheck // the status alone is the answer
	return resp.StatusCode == http.StatusOK
}
