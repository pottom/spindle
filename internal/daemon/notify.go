package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Notifications are the daemon's, not the interface's, because the daemon is
// what outlives the interface. Closing the window does not stop the music —
// that is the whole point of the split — and a track change nobody is looking
// at is exactly the one worth saying out loud.
//
// Off unless asked for: a notification every three minutes is somebody else's
// idea of useful, and the screen already says what is playing while it is open.

const (
	// notifyDelay is how long a track has to keep playing before it is
	// announced. Skipping through a list fires a change per press, and nobody
	// wants a stack of notifications for tracks they have already left.
	notifyDelay = 3 * time.Second

	// notifyReconnect is how long to wait before dialling the events stream
	// again. It is on loopback, so a failure means the API is still coming up.
	notifyReconnect = time.Second
)

// watchAndNotify announces each new track until ctx is cancelled. Everything
// here is best effort: a notification that cannot be posted is not worth a line
// in the log, let alone interrupting playback for.
func watchAndNotify(ctx context.Context, port int) {
	events := fmt.Sprintf("ws://%s:%d/events", DefaultHost, port)
	status := fmt.Sprintf("http://%s:%d/status", DefaultHost, port)

	var last string
	for ctx.Err() == nil {
		if err := listenAndNotify(ctx, events, status, &last); err != nil && ctx.Err() == nil {
			select {
			case <-ctx.Done():
			case <-time.After(notifyReconnect):
			}
		}
	}
}

// listenAndNotify holds one connection to the event stream open. The events
// only say that something moved; what moved comes from /status, which is one
// request against loopback and far less code than a second state machine.
func listenAndNotify(ctx context.Context, events, status string, last *string) error {
	conn, _, err := websocket.Dial(ctx, events, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow() //nolint:errcheck // closing a dead socket says nothing

	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return err
		}

		// Let a run of skips settle before saying anything, and ask again
		// afterwards: what is playing then is what the listener chose.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(notifyDelay):
		}

		now, err := nowPlaying(ctx, status)
		if err != nil || now.Uri == "" || now.Paused || now.Stopped {
			continue
		}
		if now.Uri == *last {
			continue
		}
		*last = now.Uri
		notify(now.Name, strings.Join(now.Artists, ", "))
	}
}

// post sends a command to the daemon's own API and drops whatever it answers:
// the caller is a key press, and there is nothing to tell it.
func post(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), postTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

// postTimeout bounds one of those. It is a request to a server on this machine.
const postTimeout = 3 * time.Second

// playing is as much of the daemon's own status as a notification needs.
type playing struct {
	Uri     string
	Name    string
	Artists []string
	Paused  bool
	Stopped bool
}

func nowPlaying(ctx context.Context, status string) (playing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, status, nil)
	if err != nil {
		return playing{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return playing{}, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	var out struct {
		Paused  bool `json:"paused"`
		Stopped bool `json:"stopped"`
		Track   *struct {
			Uri     string   `json:"uri"`
			Name    string   `json:"name"`
			Artists []string `json:"artist_names"`
		} `json:"track"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return playing{}, err
	}
	if out.Track == nil {
		return playing{Paused: out.Paused, Stopped: out.Stopped}, nil
	}
	return playing{
		Uri:     out.Track.Uri,
		Name:    out.Track.Name,
		Artists: out.Track.Artists,
		Paused:  out.Paused,
		Stopped: out.Stopped,
	}, nil
}
