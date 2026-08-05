package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// remoteTimeout bounds a single call to the daemon. It is long for a loopback
// request on purpose — the daemon answers from a single request loop and may be
// mid-track-change — and still short enough that a prompt running status every
// second cannot be held up by a wedged one.
const remoteTimeout = 2 * time.Second

// errNoDaemon reports that nothing answered on the local API.
//
// It is kept apart from every other failure because a script can act on it: no
// daemon means start one, whereas a refused command means the daemon is there
// and unwilling, and the two deserve different exit codes. It says "answered"
// rather than "running" because a daemon wedged past the timeout is no more use
// than an absent one, and the two cannot be told apart from out here.
var errNoDaemon = errors.New("no daemon answered")

// remote is the command line's connection to the daemon's local API.
//
// It deliberately does not go through player.Local. That composition falls back
// to the Web API, which needs an OAuth token, a network round trip and, the
// first time, a browser — none of which a command in a shell script can afford
// to wait for. Here the daemon is the only source, and its absence is an answer
// rather than a reason to go looking elsewhere.
type remote struct {
	addr string
	http *http.Client
}

func newRemote(addr string) *remote {
	return &remote{addr: addr, http: &http.Client{Timeout: remoteTimeout}}
}

// events is where the daemon pushes to. It is the same address in a different
// scheme: the API and the stream are one server.
func (r *remote) events() string {
	return "ws" + strings.TrimPrefix(r.addr, "http") + "/events"
}

// get fetches a document from the daemon and returns it raw, so that --json can
// print exactly what the daemon said rather than a re-encoding of it.
func (r *remote) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.addr+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build %s: %w", path, err)
	}

	resp, err := r.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w at %s", errNoDaemon, r.addr)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", path, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return body, nil
}

// post sends a command. The daemon answers commands with an empty document, so
// there is nothing to hand back but success.
func (r *remote) post(ctx context.Context, path string, body any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return fmt.Errorf("encode %s: %w", path, err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.addr+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w at %s", errNoDaemon, r.addr)
	}
	defer resp.Body.Close() //nolint:errcheck // the answer carries nothing

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("call %s: unexpected status %s", path, resp.Status)
	}
	return nil
}
