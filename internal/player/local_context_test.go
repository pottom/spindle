package player

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// newDaemonStub builds a Local talking to a fake daemon, with a fake Web API
// behind it as the fallback, and keeps the query each of the two was asked with.
func newDaemonStub(t *testing.T, status int, body string) (*Local, *url.Values, *url.Values) {
	t.Helper()
	var daemonQuery, webQuery url.Values

	web := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		webQuery = r.URL.Query()
		w.Write([]byte(`{"items":[]}`)) //nolint:errcheck // test server
	})

	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		daemonQuery = r.URL.Query()
		w.WriteHeader(status)
		w.Write([]byte(body)) //nolint:errcheck // test server
	}))
	t.Cleanup(daemon.Close)

	return NewLocal(web, daemon.URL, daemon.Client()), &daemonQuery, &webQuery
}

// daemonContext is an answer from /player/context holding n tracks.
func daemonContext(n int) string {
	tracks := make([]string, 0, n)
	for i := range n {
		tracks = append(tracks, fmt.Sprintf(`{"uri":"spotify:track:t%d","name":"track %d"}`, i, i))
	}
	return `{"uri":"spotify:playlist:p1","offset":0,"tracks":[` + strings.Join(tracks, ",") + `]}`
}

// The daemon is the only thing that can list a playlist for this client id, and
// it answers a page at a time. Both halves of the page have to reach it, or it
// falls back on its own defaults and hands back the top of the list again.
func TestContextRequestCarriesThePage(t *testing.T) {
	l, daemon, _ := newDaemonStub(t, http.StatusOK, daemonContext(0))

	if _, err := l.PlaylistTracksPage(t.Context(), "p1", 120); err != nil {
		t.Fatal(err)
	}
	if got := daemon.Get("uri"); got != "spotify:playlist:p1" {
		t.Errorf("uri = %q, want the playlist's", got)
	}
	if daemon.Get("offset") != "120" {
		t.Errorf("offset = %q, want 120", daemon.Get("offset"))
	}
	if want := strconv.Itoa(pageLimit); daemon.Get("limit") != want {
		t.Errorf("limit = %q, want %q", daemon.Get("limit"), want)
	}
}

// The daemon reports no total and no link onwards, so a full answer is the only
// thing that says the playlist goes on past this page.
func TestContextInfersWhatFollowsFromAFullPage(t *testing.T) {
	cases := []struct {
		name  string
		count int
		want  bool
	}{
		{"a full page", pageLimit, true},
		{"a page that ran out", pageLimit - 1, false},
		{"nothing at all", 0, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, _, _ := newDaemonStub(t, http.StatusOK, daemonContext(c.count))

			page, err := l.PlaylistTracksPage(t.Context(), "p1", 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != c.count {
				t.Errorf("%d tracks, want %d", len(page.Items), c.count)
			}
			if page.More != c.want {
				t.Errorf("More = %v, want %v", page.More, c.want)
			}
			if page.More && page.Next != pageLimit {
				t.Errorf("Next = %d, want %d", page.Next, pageLimit)
			}
		})
	}
}

// With no daemon running there is only the Web API, which is refused for most
// playlists but is still the honest thing to try. The offset has to survive the
// fall-through: dropping it would answer a request for the third page with the
// first one.
func TestContextFallsBackToTheWebApiWithTheSameOffset(t *testing.T) {
	l, _, web := newDaemonStub(t, http.StatusInternalServerError, "")

	if _, err := l.PlaylistTracksPage(t.Context(), "p1", 100); err != nil {
		t.Fatal(err)
	}
	if web.Get("offset") != "100" {
		t.Errorf("the Web API was asked for offset %q, want 100", web.Get("offset"))
	}
}
