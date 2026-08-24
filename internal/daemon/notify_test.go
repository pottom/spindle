package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// What is playing is read from the daemon's own answer, and the answer says
// nothing at all when nothing is loaded.
func TestNowPlayingReadsTheStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"paused":  false,
			"stopped": false,
			"track": map[string]any{
				"uri":          "spotify:track:t1",
				"name":         "Heroes",
				"artist_names": []string{"David Bowie"},
			},
		})
	}))
	defer srv.Close()

	got, err := nowPlaying(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("nowPlaying: %v", err)
	}
	if got.Uri != "spotify:track:t1" || got.Name != "Heroes" || len(got.Artists) != 1 {
		t.Errorf("nowPlaying() = %+v, want the track from the status", got)
	}
}

func TestNowPlayingWithNothingLoaded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"stopped": true, "track": nil})
	}))
	defer srv.Close()

	got, err := nowPlaying(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("nowPlaying: %v", err)
	}
	if got.Uri != "" || !got.Stopped {
		t.Errorf("nowPlaying() = %+v, want nothing playing", got)
	}
}
