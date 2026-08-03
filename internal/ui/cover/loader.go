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
	client   *http.Client
	dir      string

	mu      sync.Mutex
	decoded *imageLRU
}

// NewLoader prepares the pipeline. A cache directory that cannot be created is
// not fatal: the loader simply downloads every time.
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
func (l *Loader) Load(ctx context.Context, url string, wCells, hCells int) (Art, error) {
	if url == "" {
		return Art{}, fmt.Errorf("load cover: no artwork url")
	}

	img, err := l.image(ctx, url)
	if err != nil {
		return Art{}, err
	}
	cells, err := l.renderer.Render(img, wCells, hCells)
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
