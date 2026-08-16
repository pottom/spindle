package player

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pottom/spindle/internal/xdg"
)

// What the Web API was asked, and when.
//
// A day's quota went somewhere while nobody was listening, and the honest answer
// took an afternoon of reading dials in the source and arithmetic that could
// only ever be a guess. Every request already passes through one place on its
// way out, so it can say so on the way, and the next time the question comes up
// it is a `sort | uniq -c` rather than an argument.
//
//	sort -k3 ~/.local/state/spindle/api.log | uniq -cf 2 | sort -rn | head
//
// One line per request, path only — no query, because the query holds ids and
// search terms and this file is not a record of what anybody listened to.

const (
	// tallyStamp is the time on a line. The date is on a line of its own, as in
	// the daemon's log, and for the same reason: it is the same for pages.
	tallyStamp    = "15:04:05"
	tallyDayStamp = "2006-01-02, Monday"

	// tallyMost is where the file is started over. A quiet day is some
	// thousands of lines, so this holds a fortnight and cannot grow unwatched.
	tallyMost = 4 << 20
)

// tally writes down what was asked. The zero value writes nowhere, which is what
// a test wants and what a build with no state directory falls back to.
type tally struct {
	mu  sync.Mutex
	out io.Writer
	day string

	// now is the clock, so a test can hold it still. Nil is time.Now.
	now func() time.Time
}

// openTally opens the file beside the daemon's log. A failure is not one: the
// player works without a record of itself, and refusing to start over a log
// nobody asked for would be absurd.
func openTally() *tally {
	dir, err := xdg.StateDir()
	if err != nil {
		return &tally{}
	}

	path := filepath.Join(dir, "api.log")
	if info, err := os.Stat(path); err == nil && info.Size() > tallyMost {
		os.Remove(path)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return &tally{}
	}
	return &tally{out: f}
}

func (t *tally) at() time.Time {
	if t.now == nil {
		return time.Now()
	}
	return t.now()
}

// note writes one request down. status is what came back, or the reason nothing
// did.
func (t *tally) note(method, path, status string) {
	if t == nil || t.out == nil {
		return
	}

	at := t.at()

	t.mu.Lock()
	defer t.mu.Unlock()

	if day := at.Format(tallyDayStamp); day != t.day {
		t.day = day
		fmt.Fprintf(t.out, "──── %s\n", day)
	}
	fmt.Fprintf(t.out, "%s %-6s %-40s %s\n", at.Format(tallyStamp), method, path, status)
}

// noteResponse says how a request ended, in the words the reader needs: a status
// for an answer, and the failure itself for anything that never got one.
func (t *tally) noteResponse(req *http.Request, resp *http.Response, err error) {
	if t == nil || t.out == nil {
		return
	}

	status := "—"
	switch {
	case err != nil:
		status = "failed: " + err.Error()
	case resp != nil:
		status = fmt.Sprintf("%d", resp.StatusCode)
	}
	t.note(req.Method, apiPath(req), status)
}

// apiPath is the endpoint without the version prefix and without the query, so
// that the same call from two places reads as one line and no id or search term
// is written down. Ids inside the path are kept — /playlists/{id}/tracks is a
// different question from /me/tracks, and the id is how a reader tells which
// list was being walked.
func apiPath(req *http.Request) string {
	if req.URL == nil {
		return "?"
	}
	return strings.TrimPrefix(req.URL.Path, "/v1")
}
