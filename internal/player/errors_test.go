package player

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zmb3/spotify/v2"
)

func TestParseRetryAfter(t *testing.T) {
	cases := map[string]time.Duration{
		"3":     3 * time.Second,
		"1":     time.Second,
		"120":   120 * time.Second,
		"":      defaultRetryAfter,
		"soon":  defaultRetryAfter,
		"0":     defaultRetryAfter, // waiting zero would send us straight back in
		"-4":    defaultRetryAfter,
		"3.5":   defaultRetryAfter,
		"  3  ": defaultRetryAfter,
	}
	for header, want := range cases {
		if got := parseRetryAfter(header); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", header, got, want)
		}
	}
}

// The Spotify client decodes the error body and throws the Retry-After header
// away with it, so the transport has to catch a 429 before it gets there.
func TestRateLimitSurfacesRetryAfter(t *testing.T) {
	var calls int
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	err := s.Pause(context.Background())

	var limited *RateLimitedError
	if !errors.As(err, &limited) {
		t.Fatalf("err = %v, want a RateLimitedError", err)
	}
	if limited.RetryAfter != 7*time.Second {
		t.Errorf("RetryAfter = %v, want 7s", limited.RetryAfter)
	}
	if calls != 1 {
		t.Errorf("made %d requests, want 1 — a throttled call must not be retried behind our back", calls)
	}
}

func TestRateLimitOnRead(t *testing.T) {
	s := newStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := s.State(context.Background())
	var limited *RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter != 2*time.Second {
		t.Errorf("err = %v, want a RateLimitedError of 2s", err)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want error
	}{
		{"premium", spotify.Error{Status: http.StatusForbidden, Message: "Player command failed: Premium required"}, ErrPremiumRequired},
		{"device gone", spotify.Error{Status: http.StatusNotFound, Message: "Device not found"}, ErrNoActiveDevice},
		{"nothing at all", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classify("act", c.in); !errors.Is(got, c.want) {
				t.Errorf("classify = %v, want %v", got, c.want)
			}
		})
	}

	// A 403 that is not about the subscription must not be reported as one: a
	// paying listener pressing play too fast gets one of these, and being told
	// to buy what they already have hides what actually happened.
	refused := classify("play track", spotify.Error{
		Status:  http.StatusForbidden,
		Message: "Player command failed: Restriction violated",
	})
	if errors.Is(refused, ErrPremiumRequired) {
		t.Errorf("classify = %v, want the reason Spotify gave", refused)
	}
	if got := refused.Error(); !strings.Contains(got, "Restriction violated") || strings.Contains(got, "Player command failed") {
		t.Errorf("classify = %q, want Spotify's reason without its prefix", got)
	}

	// Anything unrecognised keeps its context and comes through intact.
	other := spotify.Error{Status: http.StatusInternalServerError, Message: "boom"}
	got := classify("resume playback", other)
	if errors.Is(got, ErrPremiumRequired) || errors.Is(got, ErrNoActiveDevice) {
		t.Errorf("classify mislabelled a server error as %v", got)
	}
	if !errors.As(got, &spotify.Error{}) {
		t.Errorf("classify lost the underlying error: %v", got)
	}
}
