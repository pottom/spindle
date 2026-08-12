package cover

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	_ "image/jpeg"
	_ "image/png"

	"github.com/pottom/spindle/internal/xdg"
)

const (
	// decodedCacheSize is the number of decoded images held in memory.
	decodedCacheSize = 10

	// maxCoverBytes caps a download; album art is well under this.
	maxCoverBytes = 4 << 20
)

// Loader runs the artwork pipeline: disk cache, download, decode and render.
// Every method is safe to call from a tea.Cmd goroutine.
type Loader struct {
	renderer Renderer

	// graphics is what the terminal was found to be able to do. See SetGraphics.
	graphics Graphics
	client   *http.Client
	dir      string

	mu      sync.Mutex
	decoded *imageLRU

	// seq numbers the loads, so a renderer can tell an overtaken one from the
	// newest. Requests are started from tea.Cmd goroutines and finish in any
	// order.
	seq atomic.Uint64
}

// NewLoader prepares the pipeline. A cache directory that cannot be created is
// not fatal: the loader simply downloads every time.
// SetGraphics keeps what the terminal was found to be able to do, so the
// interface can say why the cover is drawn the way it is rather than leaving
// somebody to wonder whether this is as good as it gets.
func (l *Loader) SetGraphics(g Graphics) { l.graphics = g }

// Graphics is what the terminal was found to be able to do.
func (l *Loader) Graphics() Graphics { return l.graphics }

func NewLoader(r Renderer, client *http.Client) *Loader {
	dir := ""
	if base, err := xdg.CacheDir(); err == nil {
		dir = filepath.Join(base, "covers")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			dir = ""
		}
	}
	return &Loader{
		renderer: r,
		client:   client,
		dir:      dir,
		decoded:  newImageLRU(decodedCacheSize),
	}
}

// Art is a rendered cover together with what the UI needs to colour itself after
// the artwork.
type Art struct {
	// Cells is the rendered artwork, ready to embed in a view.
	Cells string

	// Accent is the colour the cover reads as; HasAccent is false for artwork
	// with no usable colour, such as a black and white sleeve.
	Accent    color.RGBA
	HasAccent bool
}

// Renderer reports which backend is in use, for FINDINGS.md and diagnostics.
func (l *Loader) Renderer() Renderer { return l.renderer }

// Load returns artwork ready to embed in a view. It blocks on I/O and must never
// be called from Update.
func (l *Loader) Load(ctx context.Context, url string, wCells, hCells, slot int) (Art, error) {
	if url == "" {
		return Art{}, fmt.Errorf("load cover: no artwork url")
	}

	// Claimed before the download, not after: the order requests were made in is
	// what says which answer is the current one.
	seq := l.seq.Add(1)

	img, err := l.image(ctx, url)
	if err != nil {
		return Art{}, err
	}
	cells, err := l.renderer.Render(img, wCells, hCells, seq, slot)
	if err != nil {
		return Art{}, fmt.Errorf("load cover: %w", err)
	}

	accent, ok := DominantColor(img)
	return Art{Cells: cells, Accent: accent, HasAccent: ok}, nil
}

// image returns the decoded cover, consulting the in-memory cache, then the disk
// cache, and only then the network.
func (l *Loader) image(ctx context.Context, url string) (image.Image, error) {
	key := cacheKey(url)

	l.mu.Lock()
	img, ok := l.decoded.get(key)
	l.mu.Unlock()
	if ok {
		return img, nil
	}

	// Some artwork is drawn rather than downloaded. It goes into the same cache
	// as the rest: the drawing is cheap but it is not free, and a cursor resting
	// on the row would redraw it on every load.
	if img, ok := generated(url); ok {
		l.mu.Lock()
		l.decoded.put(key, img)
		l.mu.Unlock()
		return img, nil
	}

	data, err := l.bytes(ctx, url, key)
	if err != nil {
		return nil, err
	}

	img, _, err = image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode cover: %w", err)
	}

	l.mu.Lock()
	l.decoded.put(key, img)
	l.mu.Unlock()
	return img, nil
}

// bytes returns the raw cover file, downloading it once per album.
func (l *Loader) bytes(ctx context.Context, url, key string) ([]byte, error) {
	path := ""
	if l.dir != "" {
		path = filepath.Join(l.dir, key)
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build cover request: %w", err)
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch cover: unexpected status %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxCoverBytes))
	if err != nil {
		return nil, fmt.Errorf("read cover: %w", err)
	}

	if path != "" {
		// A cache write failure is not worth surfacing; the image is in hand.
		_ = os.WriteFile(path, data, 0o644)
	}
	return data, nil
}

// cacheKey names the on-disk file. Spotify cover URLs are content-addressed, so
// hashing the URL identifies the artwork just as well as an album id would.
func cacheKey(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:16])
}
