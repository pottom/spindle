package player

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// heldDaemon is a stand-in for the daemon, holding a track so the Local is not
// idle and the daemon path is the one taken rather than the Web API fallback.
func heldDaemon(t *testing.T) (*Local, *[]call) {
	t.Helper()
	var got []call

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var into map[string]any
		_ = json.Unmarshal(body, &into)
		got = append(got, call{r.URL.Path, into})
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	l := NewLocal(nil, srv.URL, srv.Client())
	l.snapshot = &localStatus{Track: &localTrack{URI: "spotify:track:x"}}
	l.snapshotAt, l.live = time.Now(), true
	if l.idle() {
		t.Fatal("the stand-in daemon reads as idle, so this would test the Web API instead")
	}
	return l, &got
}

type call struct {
	path string
	body map[string]any
}

// Repeat is one mode here and two booleans on the daemon, so every mode has to
// write both — including the half it is turning off. Leave that out and going
// from repeat-one to repeat-all leaves repeat-one set, which is a mode nobody
// asked for and which the interface would draw as repeat-all.
func TestSettingRepeatWritesBothHalves(t *testing.T) {
	for _, c := range []struct {
		mode           string
		track, context bool
	}{
		{RepeatOff, false, false},
		{RepeatContext, false, true},
		{RepeatTrack, true, false},
	} {
		t.Run(c.mode, func(t *testing.T) {
			l, got := heldDaemon(t)
			if err := l.SetRepeat(context.Background(), c.mode); err != nil {
				t.Fatalf("set repeat %s: %v", c.mode, err)
			}

			if len(*got) != 2 {
				t.Fatalf("made %d requests, want both halves written: %+v", len(*got), *got)
			}
			want := map[string]bool{
				"/player/repeat_track":   c.track,
				"/player/repeat_context": c.context,
			}
			for _, r := range *got {
				w, ok := want[r.path]
				if !ok {
					t.Errorf("unexpected request to %s", r.path)
					continue
				}
				key := map[string]string{
					"/player/repeat_track":   "repeat_track",
					"/player/repeat_context": "repeat_context",
				}[r.path]
				if r.body[key] != w {
					t.Errorf("%s sent %v, want %v", r.path, r.body[key], w)
				}
				delete(want, r.path)
			}
			for path := range want {
				t.Errorf("nothing was sent to %s", path)
			}
		})
	}
}

// An unknown mode must clear both rather than leave whatever was set. Nothing
// sends one today, but the argument is a string and the cost of being wrong is a
// player stuck repeating.
func TestAnUnknownRepeatModeClearsBoth(t *testing.T) {
	l, got := heldDaemon(t)
	if err := l.SetRepeat(context.Background(), "sideways"); err != nil {
		t.Fatalf("set repeat: %v", err)
	}
	for _, r := range *got {
		for _, v := range r.body {
			if v == true {
				t.Errorf("%s was left set for an unknown mode: %+v", r.path, r.body)
			}
		}
	}
}

func TestSettingShuffleSendsWhatWasAsked(t *testing.T) {
	for _, on := range []bool{true, false} {
		l, got := heldDaemon(t)
		if err := l.SetShuffle(context.Background(), on); err != nil {
			t.Fatalf("set shuffle %v: %v", on, err)
		}
		if len(*got) != 1 || (*got)[0].path != "/player/shuffle_context" {
			t.Fatalf("shuffle %v made %+v", on, *got)
		}
		if (*got)[0].body["shuffle_context"] != on {
			t.Errorf("shuffle %v sent %v", on, (*got)[0].body["shuffle_context"])
		}
	}
}

// Coming back the other way: the daemon's two booleans fold into one mode, and
// repeating one song inside a repeating context is still repeating one song.
func TestFoldingTheDaemonsTwoBooleansIntoOneMode(t *testing.T) {
	for _, c := range []struct {
		context, track bool
		want           string
	}{
		{false, false, RepeatOff},
		{true, false, RepeatContext},
		{false, true, RepeatTrack},
		{true, true, RepeatTrack},
	} {
		if got := repeatFromLocal(c.context, c.track); got != c.want {
			t.Errorf("context=%v track=%v folded to %q, want %q", c.context, c.track, got, c.want)
		}
	}
}

// And that the state the interface draws from carries both flags through.
func TestStateCarriesTheModesThrough(t *testing.T) {
	st := (&localStatus{ShuffleCtx: true, RepeatTrack: true, VolumeSteps: 100}).toState()
	if !st.Shuffle {
		t.Error("shuffle was dropped on the way to the interface")
	}
	if st.Repeat != RepeatTrack {
		t.Errorf("repeat came through as %q, want %q", st.Repeat, RepeatTrack)
	}
}
