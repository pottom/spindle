package player

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zmb3/spotify/v2"
)

// newStub builds a Spotify backend pointed at a fake API.
func newStub(t *testing.T, handler http.HandlerFunc) *Spotify {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewSpotify(spotify.New(srv.Client(), spotify.WithBaseURL(srv.URL+"/")))
}

// Spotify answers "nothing is playing" with 204 and an empty body, which the
// client turns into a zero value rather than an error. Left alone that would
// surface as a track with no name; it has to become ErrNoActiveDevice so the UI
// can show the device list instead.
func TestStateOnEmptyResponse(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	st, err := s.State(context.Background())
	if !errors.Is(err, ErrNoActiveDevice) {
		t.Fatalf("err = %v, want ErrNoActiveDevice", err)
	}
	if st != nil {
		t.Errorf("state = %+v, want nil", st)
	}
}

func TestStateMapsPlayback(t *testing.T) {
	const body = `{
	  "device": {"id":"dev1","is_active":true,"name":"MacBook Pro","type":"Computer","volume_percent":72},
	  "shuffle_state": true,
	  "repeat_state": "track",
	  "is_playing": true,
	  "progress_ms": 134000,
	  "item": {
	    "id": "trk1",
	    "name": "Bohemian Rhapsody",
	    "duration_ms": 355000,
	    "artists": [{"name":"Queen"},{"name":"David Bowie"}],
	    "album": {
	      "name": "A Night at the Opera",
	      "images": [
	        {"url":"https://img/1000","width":1000,"height":1000},
	        {"url":"https://img/640","width":640,"height":640},
	        {"url":"https://img/300","width":300,"height":300}
	      ]
	    }
	  }
	}`

	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body)) //nolint:errcheck // test server
	})

	st, err := s.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"track id", st.TrackID, "trk1"},
		{"title", st.Title, "Bohemian Rhapsody"},
		{"album", st.Album, "A Night at the Opera"},
		{"cover", st.CoverURL, "https://img/640"},
		{"progress", st.Progress, 134 * time.Second},
		{"duration", st.Duration, 355 * time.Second},
		{"playing", st.Playing, true},
		{"shuffle", st.Shuffle, true},
		{"repeat", st.Repeat, RepeatTrack},
		{"volume", st.Volume, 72},
		{"device id", st.DeviceID, "dev1"},
		{"device name", st.DeviceName, "MacBook Pro"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if len(st.Artists) != 2 || st.Artists[0] != "Queen" || st.Artists[1] != "David Bowie" {
		t.Errorf("artists = %v, want [Queen David Bowie]", st.Artists)
	}
}

// A speaker can be active with nothing loaded on it. That is a playable state,
// not an absent one.
func TestStateWithIdleDevice(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"device":{"id":"dev1","name":"Kitchen","type":"Speaker","volume_percent":30},"item":null}`)) //nolint:errcheck // test server
	})

	st, err := s.State(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want none", err)
	}
	if st.DeviceName != "Kitchen" || st.Title != "" || st.Duration != 0 {
		t.Errorf("state = %+v, want an idle device with no track", st)
	}
}

func TestDevicesLowercasesType(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"devices":[{"id":"d1","name":"iPhone","type":"Smartphone","is_active":false}]}`)) //nolint:errcheck // test server
	})

	devices, err := s.Devices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || devices[0].Type != "smartphone" {
		t.Errorf("devices = %+v, want one smartphone", devices)
	}
}

func TestSearchSkipsEmptyQuery(t *testing.T) {
	s := newStub(t, func(http.ResponseWriter, *http.Request) {
		t.Error("an empty query must not reach the API")
	})

	tracks, err := s.Search(context.Background(), "   ")
	if err != nil || tracks != nil {
		t.Errorf("Search(blank) = %v, %v; want nil, nil", tracks, err)
	}
}

// A playlist can hold podcast episodes, and a track unavailable in the market
// comes back as null. Both must be skipped rather than listed as blanks.
func TestPlaylistTracksSkipsWhatIsNotATrack(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"items":[
		  {"track":{"type":"track","id":"t1","name":"Heroes","duration_ms":361000,
		            "artists":[{"name":"David Bowie"}],
		            "album":{"name":"best of bowie","images":[]}}},
		  {"track":null},
		  {"track":{"type":"episode","id":"e1","name":"Some podcast"}}
		]}`)) //nolint:errcheck // test server
	})

	tracks, err := s.PlaylistTracks(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].Title != "Heroes" {
		t.Errorf("tracks = %+v, want just Heroes", tracks)
	}
	if tracks[0].Duration != 361*time.Second {
		t.Errorf("duration = %v, want 6m01s", tracks[0].Duration)
	}
}

func TestControlIsNotImplementedYet(t *testing.T) {
	s := NewSpotify(nil)
	ctx := context.Background()

	calls := map[string]error{
		"Play":       s.Play(ctx),
		"Pause":      s.Pause(ctx),
		"Next":       s.Next(ctx),
		"Previous":   s.Previous(ctx),
		"Seek":       s.Seek(ctx, time.Second),
		"SetVolume":  s.SetVolume(ctx, 50),
		"SetShuffle": s.SetShuffle(ctx, true),
		"SetRepeat":  s.SetRepeat(ctx, RepeatOff),
		"TransferTo": s.TransferTo(ctx, "d1"),
		"PlayTrack":  s.PlayTrack(ctx, "t1"),
	}
	for name, err := range calls {
		if !errors.Is(err, ErrNotImplemented) {
			t.Errorf("%s returned %v, want ErrNotImplemented", name, err)
		}
	}
}
