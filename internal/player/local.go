package player

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Local drives the spindle daemon over its local API.
//
// It is a composition, not a replacement: the daemon owns playback on this
// machine and is the only thing that can push state or edit a queue, but it
// cannot search and knows nothing about playlists or other devices. Those keep
// coming from the Web API, behind the same interface, so the UI never learns
// there are two sources.
//
// When playback is somewhere else — a phone, say — the daemon has nothing to
// report and State falls through to the Web API. Push gives way to polling for
// as long as that lasts.
type Local struct {
	web  *Spotify
	addr string
	http *http.Client

	mu         sync.RWMutex
	snapshot   *localStatus
	snapshotAt time.Time

	changes chan struct{}
}

// NewLocal wraps a running daemon at addr, falling back to web for everything
// the daemon does not know.
func NewLocal(web *Spotify, addr string, client *http.Client) *Local {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &Local{
		web:  web,
		addr: addr,
		http: client,
		// Buffered by one and never blocked on: a reader that is busy gets one
		// wake-up covering everything it missed, which is all it needs.
		changes: make(chan struct{}, 1),
	}
}

// Changes implements Watcher.
func (l *Local) Changes() <-chan struct{} { return l.changes }

// State reports the daemon's own playback when it has any, and whatever the Web
// API says otherwise.
func (l *Local) State(ctx context.Context) (*State, error) {
	if st := l.localState(); st != nil {
		return st, nil
	}

	// Nothing playing here. Either it is playing elsewhere or nowhere, and only
	// the Web API can tell those apart.
	return l.web.State(ctx)
}

// localState returns the daemon's view, or nil when it is idle. The snapshot is
// kept current by the event stream, so this costs no network at all.
func (l *Local) localState() *State {
	l.mu.RLock()
	snapshot, taken := l.snapshot, l.snapshotAt
	l.mu.RUnlock()

	if snapshot == nil || snapshot.Stopped {
		return nil
	}

	st := snapshot.toState()

	// The daemon only speaks when something happens, so between events the
	// snapshot's position is frozen at whenever it was taken. Returning it
	// as-is makes the progress bar tick forward and then snap back on every
	// poll. Carry the clock forward instead.
	if st.Playing {
		st.Progress += time.Since(taken)
		if st.Duration > 0 && st.Progress > st.Duration {
			st.Progress = st.Duration
		}
	}
	return st
}

// refresh pulls a fresh status from the daemon.
func (l *Local) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/status", nil)
	if err != nil {
		return fmt.Errorf("build status request: %w", err)
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return fmt.Errorf("fetch daemon status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch daemon status: unexpected status %s", resp.Status)
	}

	var status localStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("decode daemon status: %w", err)
	}

	l.mu.Lock()
	l.snapshot, l.snapshotAt = &status, time.Now()
	l.mu.Unlock()

	l.notify()
	return nil
}

// notify wakes whoever is watching, without ever waiting for them.
func (l *Local) notify() {
	select {
	case l.changes <- struct{}{}:
	default:
	}
}

// post sends a command to the daemon.
func (l *Local) post(ctx context.Context, path string, body any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return fmt.Errorf("encode %s: %w", path, err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.addr+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.http.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("call %s: unexpected status %s", path, resp.Status)
	}
	return nil
}

// Playback control goes to the daemon when it is the one playing, and to the
// Web API when something else is.

func (l *Local) Play(ctx context.Context) error {
	if l.idle() {
		return l.web.Play(ctx)
	}
	return l.post(ctx, "/player/resume", nil)
}

func (l *Local) Pause(ctx context.Context) error {
	if l.idle() {
		return l.web.Pause(ctx)
	}
	return l.post(ctx, "/player/pause", nil)
}

func (l *Local) Next(ctx context.Context) error {
	if l.idle() {
		return l.web.Next(ctx)
	}
	return l.post(ctx, "/player/next", nil)
}

func (l *Local) Previous(ctx context.Context) error {
	if l.idle() {
		return l.web.Previous(ctx)
	}
	return l.post(ctx, "/player/prev", nil)
}

func (l *Local) Seek(ctx context.Context, pos time.Duration) error {
	if l.idle() {
		return l.web.Seek(ctx, pos)
	}
	return l.post(ctx, "/player/seek", map[string]any{"position": pos.Milliseconds()})
}

func (l *Local) SetVolume(ctx context.Context, pct int) error {
	if l.idle() {
		return l.web.SetVolume(ctx, pct)
	}

	l.mu.RLock()
	steps := 0
	if l.snapshot != nil {
		steps = l.snapshot.VolumeSteps
	}
	l.mu.RUnlock()

	if steps <= 0 {
		steps = 100
	}
	return l.post(ctx, "/player/volume", map[string]any{"volume": min(max(pct, 0), 100) * steps / 100})
}

func (l *Local) SetShuffle(ctx context.Context, on bool) error {
	if l.idle() {
		return l.web.SetShuffle(ctx, on)
	}
	return l.post(ctx, "/player/shuffle_context", map[string]any{"shuffle_context": on})
}

func (l *Local) SetRepeat(ctx context.Context, mode string) error {
	if l.idle() {
		return l.web.SetRepeat(ctx, mode)
	}

	// The daemon splits what spindle keeps as one mode, so both halves have to
	// be set — including the one being turned off.
	if err := l.post(ctx, "/player/repeat_track", map[string]any{"repeat_track": mode == RepeatTrack}); err != nil {
		return err
	}
	return l.post(ctx, "/player/repeat_context", map[string]any{"repeat_context": mode == RepeatContext})
}

// PlayTrack and PlayPlaylist go through the Web API even when the daemon is
// playing: it is the one that knows which device to aim at, and the daemon's
// own play endpoint cannot start something on another device.

func (l *Local) PlayTrack(ctx context.Context, trackID string) error {
	return l.web.PlayTrack(ctx, trackID)
}

func (l *Local) PlayPlaylist(ctx context.Context, playlistID string, offset int) error {
	return l.web.PlayPlaylist(ctx, playlistID, offset)
}

// Everything the daemon does not know about.

func (l *Local) Devices(ctx context.Context) ([]Device, error) { return l.web.Devices(ctx) }
func (l *Local) TransferTo(ctx context.Context, id string) error {
	return l.web.TransferTo(ctx, id)
}
func (l *Local) Queue(ctx context.Context) ([]Track, error) { return l.web.Queue(ctx) }
func (l *Local) Search(ctx context.Context, q string) ([]Track, error) {
	return l.web.Search(ctx, q)
}
func (l *Local) Playlists(ctx context.Context) ([]Playlist, error) { return l.web.Playlists(ctx) }
func (l *Local) PlaylistTracks(ctx context.Context, id string) ([]Track, error) {
	return l.web.PlaylistTracks(ctx, id)
}

// idle reports whether the daemon has nothing playing, in which case commands
// belong to whatever device does.
func (l *Local) idle() bool { return l.localState() == nil }

// eventsURL is where the daemon pushes from.
func (l *Local) eventsURL() (string, error) {
	u, err := url.Parse(l.addr)
	if err != nil {
		return "", fmt.Errorf("parse daemon address: %w", err)
	}
	u.Scheme = "ws"
	u.Path = "/events"
	return u.String(), nil
}
