package cover

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"
	"sync"

	"golang.org/x/image/draw"
)

const (
	// placeholder is the character kitty reserves for virtual image placements.
	// Because it is ordinary text, Bubble Tea's line diff handles it correctly.
	placeholder = '\U0010EEEE'

	// imageID is reused for every cover. Re-transmitting under the same id
	// replaces the image, which is exactly the swap wanted on a track change,
	// and it keeps the terminal's image store from filling up.
	imageID = 1

	// chunkSize is the escape-sequence payload limit the protocol imposes.
	chunkSize = 4096
)

// Kitty renders through the kitty graphics protocol in Unicode placeholder mode:
// the image is transmitted once, out of band, and the view emits a rectangle of
// placeholder characters that the terminal fills in.
type Kitty struct {
	Cell CellSize

	mu  sync.Mutex
	out io.Writer // the tty; written to from the pipeline, never from View
}

func NewKitty(out io.Writer, cell CellSize) *Kitty {
	return &Kitty{Cell: cell, out: out}
}

func (k *Kitty) Name() string { return "kitty" }

func (k *Kitty) Render(img image.Image, wCells, hCells int) (string, error) {
	cols, rows, pxW, pxH := fitCells(img, wCells, hCells, k.Cell)
	if cols == 0 || rows == 0 {
		return "", fmt.Errorf("render kitty: image does not fit %dx%d cells", wCells, hCells)
	}

	scaled := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return "", fmt.Errorf("encode cover as png: %w", err)
	}
	if err := k.transmit(buf.Bytes(), cols, rows); err != nil {
		return "", err
	}
	return placeholderGrid(cols, rows), nil
}

// transmit uploads the image and creates a virtual placement under imageID.
// Responses are suppressed with q=2: anything the terminal echoed back would be
// read by Bubble Tea as keyboard input.
func (k *Kitty) transmit(data []byte, cols, rows int) error {
	encoded := base64.StdEncoding.EncodeToString(data)

	var sb strings.Builder
	first := true
	for len(encoded) > 0 {
		chunk := encoded
		if len(chunk) > chunkSize {
			chunk = chunk[:chunkSize]
		}
		encoded = encoded[len(chunk):]

		more := 0
		if len(encoded) > 0 {
			more = 1
		}

		if first {
			fmt.Fprintf(&sb, "\x1b_Ga=T,q=2,f=100,t=d,i=%d,U=1,c=%d,r=%d,m=%d;%s\x1b\\",
				imageID, cols, rows, more, chunk)
			first = false
			continue
		}
		fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if _, err := io.WriteString(k.out, sb.String()); err != nil {
		return fmt.Errorf("transmit cover: %w", err)
	}
	return nil
}

// placeholderGrid builds the rectangle of placeholder cells. Each cell carries
// its row and column as combining diacritics, and the run's foreground colour
// carries the image id, so the terminal can reassemble the picture wherever the
// characters end up.
func placeholderGrid(cols, rows int) string {
	if rows > len(rowColumnDiacritics) || cols > len(rowColumnDiacritics) {
		return ""
	}

	// The id occupies 24 bits, laid out as an RGB colour.
	r, g, b := (imageID>>16)&0xFF, (imageID>>8)&0xFF, imageID&0xFF

	var sb strings.Builder
	for row := range rows {
		if row > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm", r, g, b)
		for col := range cols {
			sb.WriteRune(placeholder)
			sb.WriteRune(rowColumnDiacritics[row])
			sb.WriteRune(rowColumnDiacritics[col])
		}
		sb.WriteString("\x1b[39m")
	}
	return sb.String()
}
