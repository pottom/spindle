package player

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// Whether shuffle does anything, as opposed to whether the flag comes back.
//
// The two are not the same question and the flag test cannot answer this one:
// shuffle_context shuffles the context — a playlist or an album — and leaves
// tracks queued by hand where they are. Measured 2026-08-13, toggling it against
// a hand-built queue of five changed nothing at all, which looks exactly like a
// broken feature and is not one.
//
// So this loads a real playlist and compares the order either way. It loads it
// paused, because what is being tested is the order of what comes next and none
// of that needs a sound.
func TestLiveShuffleReordersTheContext(t *testing.T) {
	l := liveLocal(t)

	was := saveWhatIsPlaying(t)
	t.Cleanup(func() { was.restore(t) })

	uri := aPlaylistWorthShuffling(t, l)

	// Off first, so the plain order is the one to compare against.
	daemonPost(t, "/player/shuffle_context", map[string]any{"shuffle_context": false})
	daemonPost(t, "/player/play", map[string]any{"uri": uri, "paused": true})
	settle()
	plain := queueNames(t)

	if len(plain) < 5 {
		t.Skipf("only %d tracks came up, too few to tell an order from a shuffle", len(plain))
	}

	daemonPost(t, "/player/shuffle_context", map[string]any{"shuffle_context": true})
	// Shuffling rebuilds what comes next, so the context is reloaded to pick it
	// up rather than reading a queue built before the flag changed.
	daemonPost(t, "/player/play", map[string]any{"uri": uri, "paused": true})
	settle()
	shuffled := queueNames(t)

	t.Logf("plain:    %v", plain)
	t.Logf("shuffled: %v", shuffled)

	if sameOrder(plain, shuffled) {
		t.Errorf("shuffle changed nothing: %d tracks came up in the same order both ways", len(plain))
	}

}

func sameOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// aPlaylistWorthShuffling is one of the account's own playlists with enough in it
// that two orderings coming out the same would be a coincidence rather than a
// likelihood.
func aPlaylistWorthShuffling(t *testing.T, l *Local) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	page, err := l.web.PlaylistsPage(ctx, 0)
	if err != nil {
		t.Skipf("cannot list playlists: %v", err)
	}
	for _, p := range page.Items {
		if p.Tracks >= 10 {
			t.Logf("shuffling %q, %d tracks", p.Name, p.Tracks)
			return "spotify:playlist:" + p.ID
		}
	}
	t.Skip("no playlist here has ten tracks in it")
	return ""
}

// settle gives the daemon a moment to load the context and rebuild what comes
// next. It is a wait rather than a poll because there is no event that says "the
// queue you are about to read is the new one".
func settle() { time.Sleep(2500 * time.Millisecond) }

// queueTrack is one entry of the daemon's queue. The endpoint answers an object
// with the playing track under "current" and what follows under "tracks", not a
// bare list — worth writing down, because reading it as a list fails in a way
// that reads like the queue being empty.
type queueTrack struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type queueAnswer struct {
	Current *queueTrack  `json:"current"`
	Tracks  []queueTrack `json:"tracks"`
}

// queueNames is the playing track and what follows it, in order. The playing
// track is included because shuffling a context changes where it starts, and a
// comparison that dropped it would miss that.
func queueNames(t *testing.T) []string {
	t.Helper()
	var q queueAnswer
	daemonGet(t, "/player/queue", &q)

	var names []string
	if q.Current != nil {
		names = append(names, q.Current.Name)
	}
	for _, x := range q.Tracks {
		names = append(names, x.Name)
	}
	return names
}

// playing is enough of the daemon's state to put back what a test disturbed.
// Loading a context replaces both what is playing and what was queued by hand,
// and neither belongs to the test.
type playing struct {
	uri    string
	pos    int64
	paused bool
	queue  []string

	// The two modes belong here for the same reason the queue does: a test that
	// turns shuffle on to see what it does and walks away has changed how the
	// next hour of listening sounds. Left out of the first version of this, and
	// it left shuffle on.
	shuffle     bool
	repeatCtx   bool
	repeatTrack bool
}

func saveWhatIsPlaying(t *testing.T) playing {
	t.Helper()
	var st struct {
		Paused        bool `json:"paused"`
		ShuffleCtx    bool `json:"shuffle_context"`
		RepeatContext bool `json:"repeat_context"`
		RepeatTrack   bool `json:"repeat_track"`
		Track         *struct {
			URI      string `json:"uri"`
			Position int64  `json:"position"`
		} `json:"track"`
	}
	daemonGet(t, "/status", &st)

	var q queueAnswer
	daemonGet(t, "/player/queue", &q)

	p := playing{
		paused:      st.Paused,
		shuffle:     st.ShuffleCtx,
		repeatCtx:   st.RepeatContext,
		repeatTrack: st.RepeatTrack,
	}
	if st.Track != nil {
		p.uri, p.pos = st.Track.URI, st.Track.Position
	}
	for _, x := range q.Tracks {
		p.queue = append(p.queue, x.URI)
	}
	return p
}

func (p playing) restore(t *testing.T) {
	t.Helper()
	daemonPost(t, "/player/shuffle_context", map[string]any{"shuffle_context": p.shuffle})
	daemonPost(t, "/player/repeat_context", map[string]any{"repeat_context": p.repeatCtx})
	daemonPost(t, "/player/repeat_track", map[string]any{"repeat_track": p.repeatTrack})
	if p.uri == "" {
		return
	}
	daemonPost(t, "/player/play", map[string]any{
		"uri": p.uri, "paused": p.paused, "position": p.pos,
	})
	settle()
	if len(p.queue) > 0 {
		daemonPost(t, "/player/set_queue", map[string]any{"uris": p.queue})
	}
}

func daemonPost(t *testing.T, path string, body any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	resp, err := http.Post("http://127.0.0.1:"+daemonPortForTest()+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		t.Fatalf("post %s: %s %s", path, resp.Status, bytes.TrimSpace(msg))
	}
}

func daemonGet(t *testing.T, path string, into any) {
	t.Helper()
	resp, err := http.Get("http://127.0.0.1:" + daemonPortForTest() + path)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %s: %s", path, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

}
