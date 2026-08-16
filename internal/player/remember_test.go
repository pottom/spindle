package player

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A search asked twice is asked once. The words somebody types are typed again,
// a record backed out of is opened again, and every one of those used to be a
// request out of a daily quota for an answer that had not changed.
func TestAnAnswerAboutTheCatalogueIsKept(t *testing.T) {
	asked := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"tracks":{"items":[]}}`)
	}))
	defer server.Close()

	var out strings.Builder
	way := &gate{
		tally: heldTally(&out, time.Date(2026, 8, 17, 9, 0, 0, 0, time.Local)),
		kept:  newKept(),
	}

	body := func() string {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/search?q=jolene&type=track", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := way.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return string(got)
	}

	first, second := body(), body()
	if asked != 1 {
		t.Errorf("the same search went out %d times, want once", asked)
	}
	if first != second {
		t.Errorf("the kept answer reads %q, want %q", second, first)
	}
	if first == "" {
		t.Error("the caller was handed an empty body")
	}

	// And the log says what did not go out, so a quiet day is legible as one.
	if !strings.Contains(out.String(), "kept") {
		t.Errorf("the record does not say an answer was read back:\n%s", out.String())
	}
}

// Nothing about the account is kept. What is playing, what the volume is, which
// devices exist: those change while spindle runs, and an answer held for an hour
// is a wrong screen rather than a saving.
func TestNothingAboutTheAccountIsKept(t *testing.T) {
	mine := []string{"/v1/me/player", "/v1/me/player/devices", "/v1/me/tracks", "/v1/me/playlists"}
	for _, path := range mine {
		req, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com"+path, nil)
		if worthKeeping(req) {
			t.Errorf("%s would be read back from an hour ago", path)
		}
	}

	catalogue := []string{"/v1/search?q=x", "/v1/artists/1/albums", "/v1/albums/2/tracks", "/v1/tracks/3"}
	for _, path := range catalogue {
		req, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com"+path, nil)
		if !worthKeeping(req) {
			t.Errorf("%s is asked for afresh every time", path)
		}
	}

	// And nothing that changes anything, whatever it is about.
	req, _ := http.NewRequest(http.MethodPut, "https://api.spotify.com/v1/me/player/play", nil)
	if worthKeeping(req) {
		t.Error("a request that changes something was treated as an answer to keep")
	}
}

// An answer goes stale, and a refusal is never kept at all: holding a 429 would
// turn a minute of being rate limited into an hour of it.
func TestWhatIsNotWorthReadingBack(t *testing.T) {
	at := time.Date(2026, 8, 17, 9, 0, 0, 0, time.Local)
	k := newKept()
	k.now = func() time.Time { return at }

	req, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/albums/1/tracks", nil)

	k.write(req, http.StatusTooManyRequests, http.Header{}, []byte(`{}`))
	if _, ok := k.read(req); ok {
		t.Error("a refusal was kept")
	}

	k.write(req, http.StatusOK, http.Header{}, []byte(`{"items":[]}`))
	if _, ok := k.read(req); !ok {
		t.Fatal("an answer was not kept at all")
	}

	at = at.Add(keptFor + time.Minute)
	if _, ok := k.read(req); ok {
		t.Error("an answer from over an hour ago was read back")
	}
}

// The room is bounded: a long session opens some dozens of records, and the
// oldest goes rather than the newest never arriving.
func TestTheOldestGoesFirst(t *testing.T) {
	at := time.Date(2026, 8, 17, 9, 0, 0, 0, time.Local)
	k := newKept()
	k.now = func() time.Time { return at }

	first, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/albums/0/tracks", nil)
	k.write(first, http.StatusOK, http.Header{}, []byte(`{}`))

	for i := 1; i <= keptMost; i++ {
		at = at.Add(time.Second)
		req, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/albums/"+string(rune('a'+i%26))+string(rune('a'+i/26))+"/tracks", nil)
		k.write(req, http.StatusOK, http.Header{}, []byte(`{}`))
	}

	if len(k.at) > keptMost {
		t.Errorf("%d answers are held, want at most %d", len(k.at), keptMost)
	}
	if _, ok := k.read(first); ok {
		t.Error("the oldest answer outlived a session that opened hundreds")
	}
}
