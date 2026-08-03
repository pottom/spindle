package player

import (
	"testing"
	"time"
)

func TestLocalStatusToState(t *testing.T) {
	s := &localStatus{
		DeviceID: "dev1", DeviceName: "spindle",
		Paused: false, Stopped: false,
		Volume: 32, VolumeSteps: 64,
		RepeatContext: true, ShuffleCtx: true,
		Track: &localTrack{
			URI:         "spotify:track:trk1",
			Name:        "Heroes",
			ArtistNames: []string{"David Bowie"},
			AlbumName:   "best of bowie",
			AlbumCover:  "https://img/640",
			Position:    61000,
			Duration:    361000,
		},
	}

	st := s.toState()
	checks := []struct {
		name      string
		got, want any
	}{
		{"track id stripped of its uri", st.TrackID, "trk1"},
		{"title", st.Title, "Heroes"},
		{"album", st.Album, "best of bowie"},
		{"cover", st.CoverURL, "https://img/640"},
		{"progress", st.Progress, 61 * time.Second},
		{"duration", st.Duration, 361 * time.Second},
		{"playing", st.Playing, true},
		{"shuffle", st.Shuffle, true},
		{"repeat", st.Repeat, RepeatContext},
		{"volume rescaled to percent", st.Volume, 50},
		{"device", st.DeviceName, "spindle"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// The daemon counts volume in steps of its own choosing; the rest of spindle
// speaks percent.
func TestPercentOf(t *testing.T) {
	cases := []struct{ value, steps, want int }{
		{32, 64, 50},
		{0, 64, 0},
		{64, 64, 100},
		{100, 100, 100},
		{50, 0, 0},    // no scale to speak of
		{-5, 64, 0},   // clamped rather than negative
		{99, 64, 100}, // clamped rather than over
	}
	for _, c := range cases {
		if got := percentOf(c.value, c.steps); got != c.want {
			t.Errorf("percentOf(%d, %d) = %d, want %d", c.value, c.steps, got, c.want)
		}
	}
}

// The daemon splits repeat into two booleans. Repeating one song inside a
// repeating context is still repeating one song.
func TestRepeatFromLocal(t *testing.T) {
	cases := []struct {
		context, track bool
		want           string
	}{
		{false, false, RepeatOff},
		{true, false, RepeatContext},
		{false, true, RepeatTrack},
		{true, true, RepeatTrack},
	}
	for _, c := range cases {
		if got := repeatFromLocal(c.context, c.track); got != c.want {
			t.Errorf("repeatFromLocal(%v, %v) = %q, want %q", c.context, c.track, got, c.want)
		}
	}
}

func TestTrackIDFromURI(t *testing.T) {
	cases := map[string]string{
		"spotify:track:trk1":  "trk1",
		"spotify:episode:ep1": "ep1",
		"trk1":                "trk1",
		"":                    "",
	}
	for uri, want := range cases {
		if got := trackIDFromURI(uri); got != want {
			t.Errorf("trackIDFromURI(%q) = %q, want %q", uri, got, want)
		}
	}
}

// An idle daemon has nothing to say, and the caller has to fall through to the
// Web API rather than show an empty player.
func TestStoppedDaemonHasNoState(t *testing.T) {
	l := &Local{snapshot: &localStatus{Stopped: true, DeviceName: "spindle"}}
	if st := l.localState(); st != nil {
		t.Errorf("localState = %+v, want nil while stopped", st)
	}
	if !l.idle() {
		t.Error("idle = false, want true while stopped")
	}
}

func TestPlayingDaemonReportsState(t *testing.T) {
	l := &Local{snapshot: &localStatus{
		DeviceName: "spindle", VolumeSteps: 100, Volume: 70,
		Track: &localTrack{URI: "spotify:track:x", Name: "something"},
	}}

	st := l.localState()
	if st == nil {
		t.Fatal("localState = nil, want the daemon's own view")
	}
	if st.Title != "something" || st.Volume != 70 {
		t.Errorf("state = %+v, want the daemon's track and volume", st)
	}
	if l.idle() {
		t.Error("idle = true while playing")
	}
}

// A wake-up must never block the backend, and a reader that is busy should find
// one waiting rather than a queue of them.
func TestNotifyCoalesces(t *testing.T) {
	l := &Local{changes: make(chan struct{}, 1)}

	for range 5 {
		l.notify()
	}

	if len(l.changes) != 1 {
		t.Errorf("%d wake-ups queued, want 1", len(l.changes))
	}
	<-l.changes
	select {
	case <-l.changes:
		t.Error("a second wake-up was queued")
	default:
	}
}
