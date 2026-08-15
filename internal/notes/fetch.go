package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pottom/spindle/internal/build"
)

// Asking the databases, at the pace each of them asks to be asked at.
//
// The pace is per source and it is not negotiable: MusicBrainz allows about one
// request a second and answers a burst with a 503 — measured, and it is
// concurrency that trips it rather than rate, so every request through here is
// serialised behind its source's own lock. Wikipedia and the rest are far more
// generous, and say so by having their own, shorter, spacing.
//
// The user agent is not decoration either. MusicBrainz throttles unnamed clients
// harder as a group, and their documentation names the libraries it has had
// trouble with; a distinctive one is genuinely faster.

// agent is what spindle calls itself when it asks. The form is the one
// MusicBrainz asks for: a name, a version and somewhere to complain to.
var agent = "spindle/" + build.Version() + " ( https://github.com/pottom/spindle )"

// pace serialises requests to one host and keeps them apart.
type pace struct {
	every time.Duration

	mu   sync.Mutex
	next time.Time
}

// wait blocks until this source may be asked again, or the context gives up
// first.
func (p *pace) wait(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if until := time.Until(p.next); until > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(until):
		}
	}
	p.next = time.Now().Add(p.every)
	return nil
}

// client is the one every source shares. A timeout of its own, because the
// context bounds the whole walk and a single source hanging must not spend all
// of it.
var client = &http.Client{Timeout: 15 * time.Second}

// fetch asks for a JSON answer and decodes it into out.
//
// A 429 or a 503 is a refusal rather than a failure: the source asked to be left
// alone, and the honest thing is to give up on this answer rather than to try
// again inside the same walk. Retrying is what turns a busy minute into a
// blocked hour.
func fetch(ctx context.Context, p *pace, url string, out any) error {
	if err := p.wait(ctx); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("ask %s: %w", url, err)
	}
	req.Header.Set("User-Agent", agent)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ask %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// Honour what it asks for next time round: a source that has just said
		// no is not asked again for a while, whatever its usual pace is.
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusServiceUnavailable {
			p.mu.Lock()
			p.next = time.Now().Add(backOff)
			p.mu.Unlock()
		}
		return fmt.Errorf("ask %s: %s", url, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("read %s: %w", url, err)
	}
	return nil
}

// backOff is how long a source that has just refused is left alone for. Longer
// than any of their stated limits, because the cost of waiting is a panel that
// fills in a minute later and the cost of not waiting is being shut out.
const backOff = 30 * time.Second
