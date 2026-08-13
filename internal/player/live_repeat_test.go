package player

import (
	"testing"
	"time"
)

// Whether repeat-one does anything, as opposed to whether the flag comes back.
//
// The flag test cannot answer this: it proves the daemon remembers what it was
// told, not that it acts on it at the moment that matters, which is the end of a
// track. So this runs a track to its end with the flag on and asks what is
// playing afterwards.
//
// It makes a sound. Only a few seconds of one, at whatever the volume already
// was — nothing here touches the volume, because a test that quietly changes it
// is worse than a test that makes a noise.
func TestLiveRepeatOneReplaysTheTrack(t *testing.T) {
	liveLocal(t) // skips unless SPINDLE_LIVE is set and a daemon is holding something

	was := saveWhatIsPlaying(t)
	t.Cleanup(func() { was.restore(t) })

	for _, c := range []struct {
		what  string
		mode  string
		again bool
	}{
		{"repeat one", RepeatTrack, true},
		{"repeat off", RepeatOff, false},
	} {
		t.Run(c.what, func(t *testing.T) {
			daemonPost(t, "/player/repeat_track", map[string]any{"repeat_track": c.mode == RepeatTrack})
			daemonPost(t, "/player/repeat_context", map[string]any{"repeat_context": false})

			before := runToTheEnd(t)
			after := nowPlaying(t)

			switch {
			case c.again && after.uri != before:
				t.Errorf("with repeat one, the track changed at the end: %s became %s", before, after.uri)
			case !c.again && after.uri == before:
				t.Errorf("with repeat off, the same track came round again: %s", before)
			}
			t.Logf("%s: %s at %dms", c.what, after.name, after.pos)
		})
	}
}

// runToTheEnd starts whatever is loaded, drops the needle a few seconds from the
// end, and waits for the track to run out. It returns the uri that was playing,
// so the caller can ask whether it is still the one playing afterwards.
//
// Seeking rather than waiting out a whole track: the question is only what
// happens at the boundary, and three minutes of waiting would answer it no
// better.
func runToTheEnd(t *testing.T) string {
	t.Helper()

	st := nowPlaying(t)
	if st.uri == "" || st.duration < 8000 {
		t.Skip("nothing loaded that is long enough to run to its end")
	}

	daemonPost(t, "/player/seek", map[string]any{"position": st.duration - 4000})
	daemonPost(t, "/player/resume", nil)

	// Long enough for the four seconds to play out and the next thing to load.
	time.Sleep(9 * time.Second)
	daemonPost(t, "/player/pause", nil)
	return st.uri
}

type sounding struct {
	uri, name string
	pos       int64
	duration  int64
}

func nowPlaying(t *testing.T) sounding {
	t.Helper()
	var st struct {
		Track *struct {
			URI      string `json:"uri"`
			Name     string `json:"name"`
			Position int64  `json:"position"`
			Duration int64  `json:"duration"`
		} `json:"track"`
	}
	daemonGet(t, "/status", &st)
	if st.Track == nil {
		return sounding{}
	}
	return sounding{st.Track.URI, st.Track.Name, st.Track.Position, st.Track.Duration}
}
