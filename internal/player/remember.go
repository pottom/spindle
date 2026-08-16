package player

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Keeping the answers that cannot go stale while you look at them.
//
// A record's tracks, an artist's albums, what a search matched: these are facts
// about a catalogue, not about the account, and they are the same answer an hour
// later. They were asked for again every time — typing a query, backing out of
// an artist and going in again, opening the same record twice — and each of
// those is a request out of a daily quota.
//
// Anything under /me is never kept. What is playing, what the volume is, which
// devices exist, which lists somebody has: those are the account, they change
// while spindle is running, and a stale answer to any of them is a bug on the
// screen rather than a saving. The user's own lists are written to disk with a
// version to check them against, which is a different bargain — see the ui
// package's list cache.
//
// spotify-player keeps the same things for the same hour, one layer up, in a
// cache per kind of thing. Down here it costs no caller anything and covers
// every read at once.

const (
	// keptFor is how long an answer is worth reading back. An hour, after
	// spotify-player, whose caches for contexts, searches and lyrics expire at
	// exactly that.
	keptFor = time.Hour

	// keptMost is how many answers are held. A long session opens some dozens of
	// records; past that the oldest goes.
	keptMost = 256

	// keptLargest is the biggest answer worth holding. A search is a few
	// kilobytes and a long record's tracks some tens; anything past this is one
	// unusual list holding the rest out.
	keptLargest = 1 << 20
)

// answer is a reply, kept whole. The body is bytes rather than a stream because
// a stream can only be read once and this one is read again and again.
type answer struct {
	status int
	header http.Header
	body   []byte
	at     time.Time
}

// kept is what has been answered already. The zero value keeps nothing, which is
// what a test wants unless it says otherwise.
type kept struct {
	mu sync.Mutex
	at map[string]answer

	// now is the clock, so a test can hold it still. Nil is time.Now.
	now func() time.Time
}

func newKept() *kept { return &kept{at: make(map[string]answer)} }

func (k *kept) clock() time.Time {
	if k == nil || k.now == nil {
		return time.Now()
	}
	return k.now()
}

// read returns the answer to this request, where there is one that has not gone
// stale.
func (k *kept) read(req *http.Request) (answer, bool) {
	if k == nil || k.at == nil {
		return answer{}, false
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	held, ok := k.at[keptKey(req)]
	if !ok {
		return answer{}, false
	}
	if k.clock().Sub(held.at) >= keptFor {
		delete(k.at, keptKey(req))
		return answer{}, false
	}
	return held, true
}

// write remembers an answer. Only the ones that answered: a refusal is about the
// moment rather than about the catalogue, and holding one would turn a minute of
// being rate limited into an hour of it.
func (k *kept) write(req *http.Request, status int, header http.Header, body []byte) {
	if k == nil || k.at == nil || status != http.StatusOK || len(body) > keptLargest {
		return
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	if len(k.at) >= keptMost {
		k.dropOldest()
	}
	k.at[keptKey(req)] = answer{
		status: status,
		header: header.Clone(),
		body:   body,
		at:     k.clock(),
	}
}

// dropOldest makes room. Called with the lock held.
func (k *kept) dropOldest() {
	var oldest string
	var at time.Time
	for key, held := range k.at {
		if oldest == "" || held.at.Before(at) {
			oldest, at = key, held.at
		}
	}
	delete(k.at, oldest)
}

// keptKey is the whole address, query and all: a search for one word and a
// search for another are different questions, and so are two pages of the same
// one.
func keptKey(req *http.Request) string {
	if req.URL == nil {
		return req.Method
	}
	return req.Method + " " + req.URL.String()
}

// worthKeeping reports whether an answer to this request can be read back later
// without lying. A catalogue is the same an hour from now; an account is not.
func worthKeeping(req *http.Request) bool {
	if req.Method != http.MethodGet || req.URL == nil {
		return false
	}

	path := strings.TrimPrefix(req.URL.Path, "/v1")
	for _, kind := range keptKinds {
		if strings.HasPrefix(path, kind) {
			return true
		}
	}
	return false
}

// keptKinds are the reads that are about the catalogue rather than about
// whoever is asking. Deliberately a list of what may be kept rather than of what
// may not: a Web API endpoint nobody here has thought about yet is asked for
// afresh, which is the answer that can only be too slow rather than wrong.
var keptKinds = []string{
	"/search",
	"/artists/",
	"/albums/",
	"/tracks/",
	"/shows/",
	"/episodes/",
	"/audio-features/",
	"/markets",
}

// reply turns a kept answer back into a response the client can read. A fresh
// reader every time, because the caller closes it.
func (a answer) reply(req *http.Request) *http.Response {
	return &http.Response{
		Status:        http.StatusText(a.status),
		StatusCode:    a.status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        a.header.Clone(),
		Body:          io.NopCloser(bytes.NewReader(a.body)),
		ContentLength: int64(len(a.body)),
		Request:       req,
	}
}

// hold reads a response body out so it can be kept, and puts an identical one
// back for the caller. A body is a stream that can be read once, and the caller
// has not read it yet.
func hold(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, keptLargest+1))
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
