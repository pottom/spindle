package player

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
)

// ErrPremiumRequired reports that the account cannot be controlled remotely.
// Reading playback state works on a free account; changing it does not.
var ErrPremiumRequired = errors.New("playback control requires Spotify Premium")

// ErrNotResponding is a command the daemon would not take.
//
// It runs its API and its playback session on one goroutine, so a session that
// has stopped moving leaves commands queued behind it; after ten seconds the
// daemon gives up on the caller's behalf rather than leaving it holding. Reads
// come back with what was last true instead — see the Warning header — but a
// command cannot be answered out of a cupboard: a pause that returned before the
// session took it would report that the music had stopped while it played on.
//
// Named because the press has to say something a person can read. It arrived as
// `call /player/next: unexpected status 503 Service Unavailable`.
var ErrNotResponding = errors.New("the player is not responding")

// RateLimitedError carries how long Spotify wants to be left alone.
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("rate limited for %s", e.RetryAfter.Round(time.Second))
}

// defaultRetryAfter is used when Spotify rate limits without saying for how long.
const defaultRetryAfter = 5 * time.Second

// rateLimiter turns a 429 into a typed error before the Spotify client can
// swallow the Retry-After header, which it discards while decoding the body.
// It is also where every request to the Web API can be counted, because it is
// the one place they all pass through. See tally.go.
type rateLimiter struct {
	base  http.RoundTripper
	tally *tally
}

func (r *rateLimiter) RoundTrip(req *http.Request) (*http.Response, error) {
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}

	resp, err := base.RoundTrip(req)
	r.tally.noteResponse(req, resp, err)

	if err != nil || resp.StatusCode != http.StatusTooManyRequests {
		return resp, err
	}

	retry := parseRetryAfter(resp.Header.Get("Retry-After"))
	resp.Body.Close()
	return nil, &RateLimitedError{RetryAfter: retry}
}

// parseRetryAfter reads the header Spotify sends with a 429. It is a whole
// number of seconds; anything else gets the default rather than zero, because a
// zero wait would send us straight back into the limit.
func parseRetryAfter(header string) time.Duration {
	secs, err := strconv.Atoi(header)
	if err != nil || secs <= 0 {
		return defaultRetryAfter
	}
	return time.Duration(secs) * time.Second
}

// classify turns an API failure into something the UI knows how to say. Anything
// it does not recognise is passed through with context attached.
func classify(action string, err error) error {
	if err == nil {
		return nil
	}

	// The rate limiter has already produced a typed error; url.Error wraps it.
	var limited *RateLimitedError
	if errors.As(err, &limited) {
		return limited
	}

	var apiErr spotify.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case http.StatusForbidden:
			// A 403 on a player call is usually not about the subscription at
			// all: Spotify answers one for a command it will not carry out in
			// the present state — pressing play faster than it can act on it
			// is the common way to see one — and it says which in the message.
			// Reporting them all as "Premium required" tells a paying listener
			// something false and hides what actually went wrong.
			if isPremiumRefusal(apiErr.Message) {
				return ErrPremiumRequired
			}
			return fmt.Errorf("%s: %s", action, refusal(apiErr.Message))
		case http.StatusNotFound:
			// Spotify answers a control call with 404 when the device it was
			// aimed at has gone away.
			return ErrNoActiveDevice
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}

// isPremiumRefusal reports whether a 403 was about the subscription. Spotify
// names that reason; everything else it refuses for is a different sentence.
func isPremiumRefusal(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "premium") || strings.Contains(m, "non-premium")
}

// refusal tidies Spotify's wording for a line the user has to read at a glance.
// "Player command failed: Restriction violated" is its own prefix twice over.
func refusal(message string) string {
	if message == "" {
		return "refused by Spotify"
	}
	if _, rest, ok := strings.Cut(message, "Player command failed: "); ok {
		return rest
	}
	return message
}
