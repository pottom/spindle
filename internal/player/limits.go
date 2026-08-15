package player

import (
	"errors"
	"strings"
	"sync"

	"github.com/zmb3/spotify/v2"
)

// How much of a list one request may ask for.
//
// The documentation says fifty and the documentation is not the authority. This
// client id is refused at fifty on the search endpoint — 400 "Invalid limit",
// measured, undocumented, and the number that works there is ten — and the same
// refusal turned up on an artist's albums a fortnight later. Which endpoints are
// restricted, and to what, is not something anybody outside Spotify can look up:
// it depends on the application, and it changes.
//
// So it is not written down here. Each list starts at the most anybody is
// allowed, and an endpoint that refuses is asked for half as much until it
// stops refusing. What worked is remembered for the run, so the cost of finding
// out is one refused request per endpoint per session — and the alternative,
// which the search once was, is a constant somebody has to notice is wrong and
// come back and edit.
//
// The number that worked is also the number the next offset is counted by. A
// page asked for at ten and paged by fifty skips forty rows, and the rows it
// skips are the ones nobody notices are missing.

// pageSteps are the sizes a list is asked for, largest first.
//
// Steps rather than halving. Halved from fifty the sizes go 25, 12, 6 — and an
// endpoint that answers ten would settle at six, giving up nearly half of every
// page for ever. These are the numbers Spotify's own endpoints are built around:
// fifty is the documented maximum for a list, ten is what this client id is held
// to on the search, and the rest are there so that a stranger restriction still
// lands somewhere sensible rather than at one.
var pageSteps = [...]int{50, 20, 10, 5, 1}

// mostPerPage is where every list starts: the most the Web API documents.
const mostPerPage = 50

// smaller is the next size down from n, and n itself once there is nowhere left
// to go.
func smaller(n int) int {
	for _, step := range pageSteps {
		if step < n {
			return step
		}
	}
	return n
}

// limits remembers what each list will actually hand back.
type limits struct {
	mu sync.Mutex
	at map[string]int
}

// per is what to ask this list for now.
func (l *limits) per(list string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	if n, ok := l.at[list]; ok {
		return n
	}
	return mostPerPage
}

// refused halves what this list is asked for, and says what to try next. One is
// the floor: an endpoint that will not answer a single item is refusing
// something other than the size.
func (l *limits) refused(list string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.at == nil {
		l.at = map[string]int{}
	}
	was, ok := l.at[list]
	if !ok {
		was = mostPerPage
	}
	l.at[list] = smaller(was)
	return l.at[list]
}

// asking runs a list request at whatever size this endpoint has been found to
// accept, and tries again smaller when it is refused for being too large.
//
// Only for that: any other failure is handed straight back. A rate limit, a
// missing scope or a network that is not there are not sizes, and retrying them
// is how a busy minute becomes a blocked hour.
func (l *limits) asking(list string, call func(limit int) error) error {
	for {
		limit := l.per(list)
		err := call(limit)
		if err == nil || !refusedForSize(err) || limit <= 1 {
			return err
		}
		l.refused(list)
	}
}

// refusedForSize reports whether an error is Spotify saying the page asked for
// was too big.
//
// By the message, because there is nothing else to go on: the status is a plain
// 400, which is also what a malformed id gives. The message is "Invalid limit"
// and has been for as long as anybody has looked.
func refusedForSize(err error) bool {
	var e spotify.Error
	if !errors.As(err, &e) {
		return false
	}
	return e.Status == 400 && strings.Contains(strings.ToLower(e.Message), "limit")
}
