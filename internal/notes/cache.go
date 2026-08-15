package notes

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pottom/spindle/internal/xdg"
)

// Keeping the answers.
//
// Everything in here changes on the scale of years and is asked for on the scale
// of seconds: walking a list of records asks after the same handful of artists
// over and over. Without a cache the whole idea is impossible — one request a
// second against MusicBrainz cannot serve a screen — and with one the second
// visit to an artist costs nothing at all.
//
// Two layers, because they answer different questions. In memory, so that
// walking back up a list is free and so that an artist nobody knows anything
// about is not asked after twice. On disk, so that the same is true tomorrow.
//
// A miss is cached too. "Nobody has heard of this" is an answer, and the long
// tail of a real library is full of it — the alternative is a request per frame
// for every record nothing will ever be known about.

// Cached is a chain with the answers kept.
type Cached struct {
	chain *Chain
	dir   string

	mu   sync.Mutex
	held map[string]held
}

type held struct {
	Artist Artist    `json:"artist"`
	At     time.Time `json:"at"`
}

// NewCached wraps a chain. A cache directory that cannot be made is not an
// error: the memory half still works, and the program is no worse off than it
// was before there was a disk cache at all.
func NewCached(chain *Chain) *Cached {
	c := &Cached{chain: chain, held: map[string]held{}}
	if base, err := xdg.CacheDir(); err == nil {
		dir := filepath.Join(base, "notes")
		if os.MkdirAll(dir, 0o755) == nil {
			c.dir = dir
		}
	}
	return c
}

// Sources is how many databases are behind this, for a screen that wants to say
// that a key would add one.
func (c *Cached) Sources() int { return c.chain.Sources() }

// Artist answers from what is held, or asks and keeps what comes back.
func (c *Cached) Artist(ctx context.Context, k Key) (Artist, error) {
	name := cacheName(k)

	if a, ok := c.lookup(name); ok {
		return a, nil
	}

	a, err := c.chain.Artist(ctx, k)
	if err != nil {
		// The walk was given up on rather than finished. Keeping that would be
		// keeping the terminal's own impatience as a fact about an artist.
		return a, err
	}
	c.keep(name, a)
	return a, nil
}

// lookup answers from memory, then from disk.
func (c *Cached) lookup(name string) (Artist, bool) {
	c.mu.Lock()
	h, ok := c.held[name]
	c.mu.Unlock()
	if ok {
		return h.Artist, time.Since(h.At) < Fetched
	}
	if c.dir == "" {
		return Artist{}, false
	}

	data, err := os.ReadFile(filepath.Join(c.dir, name))
	if err != nil {
		return Artist{}, false
	}
	if err := json.Unmarshal(data, &h); err != nil || time.Since(h.At) >= Fetched {
		return Artist{}, false
	}

	c.mu.Lock()
	c.held[name] = h
	c.mu.Unlock()
	return h.Artist, true
}

// keep files an answer, including an empty one.
func (c *Cached) keep(name string, a Artist) {
	h := held{Artist: a, At: time.Now()}

	c.mu.Lock()
	c.held[name] = h
	c.mu.Unlock()

	if c.dir == "" {
		return
	}
	if data, err := json.Marshal(h); err == nil {
		_ = os.WriteFile(filepath.Join(c.dir, name), data, 0o600)
	}
}

// cacheName is what an answer is filed under: the Spotify id where there is one,
// because that is what spindle will ask with next time, and the name where there
// is not.
func cacheName(k Key) string {
	if k.SpotifyArtist != "" {
		return "artist-" + safe(k.SpotifyArtist) + ".json"
	}
	return "artist-name-" + safe(strings.ToLower(k.Name)) + ".json"
}

// safe keeps a key to what is certainly a filename on every platform spindle
// builds for.
func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('.')
		}
	}
	return b.String()
}
