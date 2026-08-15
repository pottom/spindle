package player

import (
	"net/http"
	"strconv"
	"testing"
)

// The documented page size is not the authority. This client id is refused at
// fifty on more than one endpoint, undocumented, and which ones they are changes
// — so the size is found out rather than written down.
func TestAListThatIsRefusedIsAskedForLess(t *testing.T) {
	var asked []string

	s := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		asked = append(asked, limit)

		// A stub of the endpoint that will not answer more than ten, which is
		// what /v1/search does today and what an artist's albums began doing.
		if n, _ := strconv.Atoi(limit); n > 10 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"status":400,"message":"Invalid limit"}}`)) //nolint:errcheck // test server
			return
		}
		w.Write([]byte(`{"items":[]}`)) //nolint:errcheck // test server
	})

	if _, err := s.ArtistAlbums(t.Context(), "ar1", 0); err != nil {
		t.Fatalf("ArtistAlbums: %v", err)
	}
	if len(asked) < 2 {
		t.Fatalf("it asked once, for %v — want it to have tried smaller", asked)
	}
	if want := asked[len(asked)-1]; want > "10" && want != "10" {
		t.Errorf("it settled on a limit of %s, want one the endpoint answers", want)
	}

	// And it remembers: the next call does not start by being refused again.
	was := len(asked)
	if _, err := s.ArtistAlbums(t.Context(), "ar1", 0); err != nil {
		t.Fatalf("ArtistAlbums: %v", err)
	}
	if len(asked)-was != 1 {
		t.Errorf("the second call took %d requests, want one — what worked is remembered", len(asked)-was)
	}
}

// The offset of the next page is counted by the size that was accepted. Asked
// for at ten and paged by fifty, a list skips forty rows — and the rows it skips
// are the ones nobody notices are missing.
func TestPagingCountsByTheSizeThatWorked(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		if n, _ := strconv.Atoi(r.URL.Query().Get("limit")); n > 10 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"status":400,"message":"Invalid limit"}}`)) //nolint:errcheck // test server
			return
		}
		w.Write([]byte(`{"items":[],"next":"more"}`)) //nolint:errcheck // test server
	})

	page, err := s.ArtistAlbums(t.Context(), "ar1", 0)
	if err != nil {
		t.Fatalf("ArtistAlbums: %v", err)
	}
	if !page.More {
		t.Fatal("a page with a next link says there is no more")
	}
	if page.Next != 10 {
		t.Errorf("the next page starts at %d, want 10 — what the endpoint accepted", page.Next)
	}
}

// Only the size is retried. A rate limit, a missing permission or a network that
// is not there are not sizes, and asking again is how a busy minute becomes a
// blocked hour.
func TestOnlyASizeIsRetried(t *testing.T) {
	asked := 0
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		asked++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"status":429,"message":"Too many requests"}}`)) //nolint:errcheck // test server
	})

	if _, err := s.ArtistAlbums(t.Context(), "ar1", 0); err == nil {
		t.Fatal("a refusal that is not about size was swallowed")
	}
	if asked != 1 {
		t.Errorf("it asked %d times, want once", asked)
	}
}
