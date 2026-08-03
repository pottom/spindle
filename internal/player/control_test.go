package player

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// capture records what a control call actually put on the wire.
type capture struct {
	method string
	path   string
	query  url.Values
	body   map[string]any
}

func newRecorder(t *testing.T) (*Spotify, *capture) {
	t.Helper()
	got := &capture{}

	s := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.query = r.URL.Query()

		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			json.Unmarshal(raw, &got.body) //nolint:errcheck // absence is the assertion
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return s, got
}

func TestSeekSendsMilliseconds(t *testing.T) {
	s, got := newRecorder(t)

	if err := s.Seek(context.Background(), 2*time.Minute+14*time.Second); err != nil {
		t.Fatal(err)
	}
	if want := "134000"; got.query.Get("position_ms") != want {
		t.Errorf("position_ms = %q, want %q", got.query.Get("position_ms"), want)
	}
	if !strings.HasSuffix(got.path, "/seek") {
		t.Errorf("path = %q, want it to end in /seek", got.path)
	}
}

func TestSetVolumeClamps(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"-10", "0"},
		{"250", "100"},
	} {
		s, got := newRecorder(t)
		in := 250
		if c.in == "-10" {
			in = -10
		}
		if err := s.SetVolume(context.Background(), in); err != nil {
			t.Fatal(err)
		}
		if got.query.Get("volume_percent") != c.want {
			t.Errorf("SetVolume(%s) sent %q, want %q", c.in, got.query.Get("volume_percent"), c.want)
		}
	}
}

func TestSetRepeatNormalisesMode(t *testing.T) {
	s, got := newRecorder(t)

	// An unknown mode must not be forwarded: Spotify would reject it outright.
	if err := s.SetRepeat(context.Background(), "sideways"); err != nil {
		t.Fatal(err)
	}
	if got.query.Get("state") != RepeatOff {
		t.Errorf("state = %q, want %q", got.query.Get("state"), RepeatOff)
	}
}

func TestPlayTrackSendsBareURI(t *testing.T) {
	s, got := newRecorder(t)

	if err := s.PlayTrack(context.Background(), "trk1"); err != nil {
		t.Fatal(err)
	}

	uris, ok := got.body["uris"].([]any)
	if !ok || len(uris) != 1 || uris[0] != "spotify:track:trk1" {
		t.Errorf("body = %v, want a single track uri", got.body)
	}
	if _, ok := got.body["context_uri"]; ok {
		t.Error("a single track must not carry a context")
	}
}

// Starting a playlist at a position is what keeps the rest of it queued behind
// the track that was chosen.
func TestPlayPlaylistCarriesOffset(t *testing.T) {
	s, got := newRecorder(t)

	if err := s.PlayPlaylist(context.Background(), "pl1", 4); err != nil {
		t.Fatal(err)
	}
	if got.body["context_uri"] != "spotify:playlist:pl1" {
		t.Errorf("context_uri = %v, want spotify:playlist:pl1", got.body["context_uri"])
	}

	offset, ok := got.body["offset"].(map[string]any)
	if !ok || offset["position"] != float64(4) {
		t.Errorf("offset = %v, want position 4", got.body["offset"])
	}
}

func TestPlayPlaylistClampsNegativeOffset(t *testing.T) {
	s, got := newRecorder(t)

	if err := s.PlayPlaylist(context.Background(), "pl1", -3); err != nil {
		t.Fatal(err)
	}
	offset, ok := got.body["offset"].(map[string]any)
	if !ok || offset["position"] != float64(0) {
		t.Errorf("offset = %v, want position 0", got.body["offset"])
	}
}

func TestPremiumRequired(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"status":403,"message":"Player command failed: Premium required"}}`)) //nolint:errcheck // test server
	})

	if err := s.Play(context.Background()); err != ErrPremiumRequired {
		t.Errorf("err = %v, want ErrPremiumRequired", err)
	}
}

// A control call aimed at a device that has gone away comes back 404. That is
// the no-device path, not a generic failure.
func TestControlOnVanishedDevice(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"status":404,"message":"Device not found"}}`)) //nolint:errcheck // test server
	})

	if err := s.Next(context.Background()); err != ErrNoActiveDevice {
		t.Errorf("err = %v, want ErrNoActiveDevice", err)
	}
}
