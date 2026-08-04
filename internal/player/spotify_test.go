package player

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/zmb3/spotify/v2"
)

// newStub builds a Spotify backend pointed at a fake API.
func newStub(t *testing.T, handler http.HandlerFunc) *Spotify {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s := NewSpotify(srv.Client(), spotify.WithBaseURL(srv.URL+"/"))
	// The few calls that read the answer directly have to reach the stub as
	// well, and they do not go through the client library's option.
	s.base = srv.URL + "/"
	return s
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

	page, err := s.SearchPage(context.Background(), "   ", 0)
	if err != nil || page.Items != nil {
		t.Errorf("Search(blank) = %v, %v; want nothing", page.Items, err)
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

	page, err := s.PlaylistTracksPage(context.Background(), "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	tracks := page.Items
	if len(tracks) != 1 || tracks[0].Title != "Heroes" {
		t.Errorf("tracks = %+v, want just Heroes", tracks)
	}
	if tracks[0].Duration != 361*time.Second {
		t.Errorf("duration = %v, want 6m01s", tracks[0].Duration)
	}
}

// newQueryStub answers everything with the same body and keeps the query string
// it was asked with, which is where an offset shows up.
func newQueryStub(t *testing.T, body string) (*Spotify, *url.Values) {
	t.Helper()
	var got url.Values

	s := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Write([]byte(body)) //nolint:errcheck // test server
	})
	return s, &got
}

// Every browsing list can be longer than one request, and asking for the first
// page and stopping is how a playlist of three hundred tracks became one of
// fifty. Each call has to carry the offset it was given and the page size the
// backend settled on.
func TestPagingSendsOffsetAndLimit(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		limit int
		call  func(*Spotify) error
	}{
		// Search is the one with a limit of its own: Spotify refuses more than
		// ten there, and answers fifty everywhere else.
		{"search", `{"tracks":{"items":[]}}`, searchLimit, func(s *Spotify) error {
			_, err := s.SearchPage(context.Background(), "bowie", 150)
			return err
		}},
		{"playlists", `{"items":[]}`, pageLimit, func(s *Spotify) error {
			_, err := s.PlaylistsPage(context.Background(), 150)
			return err
		}},
		{"playlist tracks", `{"items":[]}`, pageLimit, func(s *Spotify) error {
			_, err := s.PlaylistTracksPage(context.Background(), "p1", 150)
			return err
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, got := newQueryStub(t, c.body)
			if err := c.call(s); err != nil {
				t.Fatal(err)
			}
			if got.Get("offset") != "150" {
				t.Errorf("offset = %q, want 150", got.Get("offset"))
			}
			if want := strconv.Itoa(c.limit); got.Get("limit") != want {
				t.Errorf("limit = %q, want %q", got.Get("limit"), want)
			}
		})
	}
}

// Spotify answers a search only as far as offset 1000, and its own next link
// keeps pointing past that — so walking a long result list to the end would end
// in a 400 rather than in the end of the list.
func TestSearchStopsAtTheWindow(t *testing.T) {
	s, _ := newQueryStub(t, `{"tracks":{"items":[{"id":"t1","name":"Heroes","artists":[{"name":"Bowie"}],"album":{"name":"a"}}],"next":"https://api.spotify.com/v1/search?offset=1000"}}`)

	page, err := s.SearchPage(context.Background(), "bowie", searchWindow-searchLimit)
	if err != nil {
		t.Fatal(err)
	}
	if page.More {
		t.Error("the last page inside the window still claims there is more")
	}
}

// An offset below zero is the caller's arithmetic going wrong, not a request to
// read backwards; Spotify answers 400 to a negative one.
func TestPagingClampsNegativeOffset(t *testing.T) {
	s, got := newQueryStub(t, `{"items":[]}`)

	if _, err := s.PlaylistTracksPage(context.Background(), "p1", -20); err != nil {
		t.Fatal(err)
	}
	if got.Get("offset") != "0" {
		t.Errorf("offset = %q, want 0", got.Get("offset"))
	}
}

// Whether more follows is Spotify's own answer — the link to the next page —
// and not something inferred here.
func TestPageReportsWhatFollows(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"a next page", `{"items":[],"next":"https://api.spotify.com/v1/me/playlists?offset=50"}`, true},
		{"the end of the list", `{"items":[],"next":null}`, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newQueryStub(t, c.body)
			page, err := s.PlaylistsPage(context.Background(), 0)
			if err != nil {
				t.Fatal(err)
			}
			if page.More != c.want {
				t.Errorf("More = %v, want %v", page.More, c.want)
			}
		})
	}
}

// A page thinned out by podcast episodes is still a full page as far as the
// list is concerned. Counting the tracks that survived would end the scroll on
// whichever page happened to hold one, and would make the next request overlap
// this one for as long as the list lasted.
func TestShortPageStillHasMore(t *testing.T) {
	s, _ := newQueryStub(t, `{"next":"https://api.spotify.com/v1/playlists/p1/tracks?offset=50","items":[
	  {"track":{"type":"track","id":"t1","name":"Heroes","artists":[],"album":{"images":[]}}},
	  {"track":null}
	]}`)

	page, err := s.PlaylistTracksPage(context.Background(), "p1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%d tracks, want the one that is playable", len(page.Items))
	}
	if !page.More {
		t.Error("More = false on a page that was only short because an item was dropped")
	}
	if page.Next != pageLimit {
		t.Errorf("Next = %d, want %d — what was asked for, not what survived", page.Next, pageLimit)
	}
}

// Transferring must not pause what it moves: play=false would silence the music
// the user just asked to hear somewhere else.
func TestTransferKeepsPlaying(t *testing.T) {
	s, got := newRecorder(t)

	if err := s.TransferTo(context.Background(), "dev2"); err != nil {
		t.Fatal(err)
	}
	if got.body["play"] != true {
		t.Errorf("play = %v, want true", got.body["play"])
	}

	ids, ok := got.body["device_ids"].([]any)
	if !ok || len(ids) != 1 || ids[0] != "dev2" {
		t.Errorf("device_ids = %v, want [dev2]", got.body["device_ids"])
	}
}

// Spotify renamed a playlist's track count from "tracks" to "items", the same
// rename that moved its contents from /tracks to /items. A library still
// reading the old name reads every playlist as empty, so both are accepted.
func TestPlaylistCountUnderEitherName(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"the old name", `{"items":[{"id":"p1","name":"Humor","tracks":{"total":3}}]}`},
		{"the new name", `{"items":[{"id":"p1","name":"Humor","items":{"total":3}}]}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newQueryStub(t, c.body)
			page, err := s.PlaylistsPage(context.Background(), 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != 1 || page.Items[0].Tracks != 3 {
				t.Errorf("playlists = %+v, want one of three tracks", page.Items)
			}
		})
	}
}
