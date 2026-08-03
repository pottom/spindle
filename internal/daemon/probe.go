package daemon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	// probeTimeout bounds a single liveness check. Anything slower than this on
	// the loopback interface is not a daemon that is going to be useful.
	probeTimeout = 500 * time.Millisecond

	// readyTimeout is how long a freshly started daemon is given to answer. It
	// has to authenticate and reach Spotify's access point first.
	readyTimeout = 20 * time.Second

	// readyPoll is how often it is asked during that wait.
	readyPoll = 200 * time.Millisecond
)

// ErrNotReady reports that no daemon answered in time.
var ErrNotReady = errors.New("the daemon did not come up")

// Addr is where the local API lives.
func Addr() string { return fmt.Sprintf("http://%s:%d", DefaultHost, DefaultPort) }

// Running reports whether a daemon is answering.
//
// It asks the API rather than looking for a process or a lock file, because
// those say a process exists, not that it works — and a spindle daemon has been
// seen holding both while answering nothing at all.
func Running() bool {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	return alive(ctx)
}

// WaitReady blocks until a daemon answers or the wait runs out.
func WaitReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	ticker := time.NewTicker(readyPoll)
	defer ticker.Stop()

	for {
		if alive(ctx) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ErrNotReady
		case <-ticker.C:
		}
	}
}

func alive(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, Addr()+"/status", nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
