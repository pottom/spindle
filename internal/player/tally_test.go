package player

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func heldTally(out *strings.Builder, at time.Time) *tally {
	return &tally{out: out, now: func() time.Time { return at }}
}

// Every request says what it was and how it ended, so that a day's quota can be
// accounted for by reading rather than by arithmetic.
func TestEveryRequestIsWrittenDown(t *testing.T) {
	var out strings.Builder
	tal := heldTally(&out, time.Date(2026, 8, 16, 22, 4, 5, 0, time.Local))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	limiter := &rateLimiter{tally: tal}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/me/player?market=HU", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := limiter.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	line := out.String()
	if !strings.Contains(line, "22:04:05") {
		t.Errorf("the request does not say when: %q", line)
	}
	if !strings.Contains(line, "/me/player") || !strings.Contains(line, "GET") {
		t.Errorf("the request does not say what was asked: %q", line)
	}
	if !strings.Contains(line, "200") {
		t.Errorf("the request does not say how it ended: %q", line)
	}

	// The query is not written down: it holds ids and search terms, and this
	// file is a record of load rather than of what anybody listened to.
	if strings.Contains(line, "market=HU") {
		t.Errorf("the query was written down: %q", line)
	}
}

// A refusal is the line worth having, so a request that never got an answer says
// why rather than going unrecorded.
func TestARefusalIsWrittenDownToo(t *testing.T) {
	var out strings.Builder
	tal := heldTally(&out, time.Date(2026, 8, 16, 22, 4, 5, 0, time.Local))

	limiter := &rateLimiter{base: brokenTransport{}, tally: tal}
	req, _ := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1/me/tracks", nil)
	if _, err := limiter.RoundTrip(req); err == nil {
		t.Fatal("a broken transport answered")
	}

	if got := out.String(); !strings.Contains(got, "failed") || !strings.Contains(got, "/me/tracks") {
		t.Errorf("a request that never got an answer went unrecorded: %q", got)
	}
}

// A player with nowhere to write still plays. The record is a convenience, and
// nothing about it may be able to stop a request.
func TestNowhereToWriteIsNotAFailure(t *testing.T) {
	var missing *tally
	missing.note("GET", "/me/player", "200")
	(&tally{}).note("GET", "/me/player", "200")
}

type brokenTransport struct{}

func (brokenTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no route to host")
}
