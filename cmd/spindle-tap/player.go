package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pottom/spindle/internal/daemon"
)

// Everything this tool needs from the daemon: what is playing, where it has got
// to, the words, and the two commands that put a record on and take it off.

// pollEvery is how often the position is refreshed. Half a second is far below
// anything the two clocks can drift by and far above what the daemon minds
// being asked: its own measurements put a status answer at a fifth of a
// millisecond.
const pollEvery = 500 * time.Millisecond

// slowPoll is the round trip beyond which an answer is not worth anchoring on.
//
// The daemon is one loop — its API and its playback session share a goroutine —
// so a stream that stalls takes the status with it. Measured here: answers come
// back in a fifth of a millisecond, and then one in a couple of hundred takes a
// second and a half. Anchoring on that one would stamp a position from a second
// ago with the time now, and every tap until the next poll would be that much
// out. So a slow answer is dropped and the previous anchor carries on: it is a
// clock, and it was right when it was set.
const slowPoll = 50 * time.Millisecond

// breakJump is how far the position may differ from what the anchor predicted
// before the pass is treated as broken by a seek.
const breakJump = 400 * time.Millisecond

var base = fmt.Sprintf("http://%s:%d", daemon.DefaultHost, daemon.DefaultPort)

// client always has a deadline. A daemon whose playback loop has wedged takes
// requests and never answers them, and the first version of this tool went to
// it from the loop that reads the keyboard, with Go's default client, which
// waits for ever. The screen froze with the music stopped and no key doing
// anything — which is exactly the moment somebody needs a key to work.
var client = &http.Client{Timeout: 2 * time.Second}

type status struct {
	Paused  bool `json:"paused"`
	Stopped bool `json:"stopped"`
	Tempo   float64
	Track   struct {
		URI      string   `json:"uri"`
		Name     string   `json:"name"`
		Artists  []string `json:"artist_names"`
		Position int64    `json:"position"`
		Duration int64    `json:"duration"`
	} `json:"track"`
}

type lyrics struct {
	Synced bool `json:"synced"`
	Lines  []struct {
		At    int64  `json:"at"`
		Words string `json:"words"`
	} `json:"lines"`
}

// words of a line, or nothing where there is no such line.
func (ly lyrics) words(i int) []string {
	if i < 0 || i >= len(ly.Lines) {
		return nil
	}
	return fields(ly.Lines[i].Words)
}

// lineAt is the line being sung at a position — the last one to have started.
func (ly lyrics) lineAt(at time.Duration) int {
	i := -1
	for n, l := range ly.Lines {
		if time.Duration(l.At)*time.Millisecond > at {
			break
		}
		i = n
	}
	return i
}

// anchor is a position the daemon reported and the local time it was read at.
// A tap between two polls is that position plus the clock since.
type anchor struct {
	at   time.Duration
	when time.Time
	ok   bool
}

func (a anchor) now() (time.Duration, bool) {
	if !a.ok {
		return 0, false
	}
	return a.at + time.Since(a.when), true
}

// refresh takes a new reading, and says whether the position jumped — which
// means somebody seeked, and the taps around it cannot be trusted. The tempo
// rides along because it comes from the same answer, and because it is only
// measured once the record has been heard for a few seconds.
func (a anchor) refresh() (out anchor, jumped bool, tempo float64) {
	asked := time.Now()
	st, err := get[status](base + "/status")
	if err != nil || st.Stopped {
		a.ok = false
		return a, false, 0
	}
	rtt := time.Since(asked)
	if reanchor.Swap(false) {
		a.ok = false // the jump was asked for; take the new position as it stands
	}
	if rtt > slowPoll && a.ok {
		return a, false, st.Tempo // a stalled answer says nothing about where the record is
	}
	// The position was read somewhere inside the round trip; half of it is the
	// honest guess at how old the number is by the time it lands.
	pos := time.Duration(st.Track.Position)*time.Millisecond + rtt/2
	if was, ok := a.now(); ok && !st.Paused {
		if d := was - pos; d > breakJump || d < -breakJump {
			jumped = true
		}
	}
	return anchor{at: pos, when: time.Now(), ok: !st.Paused}, jumped, st.Tempo
}

// play starts a track from the top. The daemon is asked rather than the Web
// API: a play with no context behind it runs out and stops dead — see
// player.Local.PlayTrack, where that was measured.
func play(uri string, from time.Duration) error {
	return post("/player/play", map[string]any{
		"uri": uri, "paused": false, "position": from.Milliseconds(),
	})
}

// seek moves the record by hand. It also tells the poller to forget where it
// thought the record was: a jump asked for is not the seek this is guarding
// against, and calling it one would say the pass was broken when it was not.
func seek(by time.Duration) error {
	reanchor.Store(true)
	return post("/player/seek", map[string]any{"position": by.Milliseconds(), "relative": true})
}

// reanchor is set when a jump is expected, and cleared by the poller as it
// takes the next reading.
var reanchor atomic.Bool

func playPause() error { return post("/player/playpause", nil) }

func pause() error { return post("/player/pause", nil) }

func wordsFor(uri string) (lyrics, error) {
	return get[lyrics](base + "/player/lyrics?uri=" + uri)
}

func get[T any](url string) (T, error) {
	var out T
	resp, err := client.Get(url) //nolint:noctx // loopback, and a spike
	if err != nil {
		return out, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("answered %s", resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

func post(path string, body any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	resp, err := client.Post(base+path, "application/json", &buf) //nolint:noctx // loopback, and a spike
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to read
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", path, resp.Status)
	}
	return nil
}
