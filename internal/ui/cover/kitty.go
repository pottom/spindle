package cover

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/image/draw"
)

const (
	// placeholder is the character kitty reserves for virtual image placements.
	// Because it is ordinary text, Bubble Tea's line diff handles it correctly.
	placeholder = '\U0010EEEE'

	// imageIDMask keeps an id inside the 24 bits a placeholder cell can carry,
	// which is the whole address space available here.
	imageIDMask = 0xFFFFFF

	// chunkSize is the escape-sequence payload limit the protocol imposes.
	chunkSize = 4096

	// unmeasuredSupersample is the safety factor applied when the terminal would
	// not report its cell size. Transmitting at the assumed resolution would
	// leave the terminal upscaling a too-small image on a HiDPI display.
	unmeasuredSupersample = 2
)

// Kitty renders through the kitty graphics protocol in Unicode placeholder mode:
// the image is transmitted once, out of band, and the view emits a rectangle of
// placeholder characters that the terminal fills in.
type Kitty struct {
	Cell CellSize

	// imageID is this spindle's own. One id is reused for every cover, because
	// re-transmitting under the same id replaces the picture — that is exactly
	// the swap a track change wants, and it keeps the terminal's image store
	// from filling up.
	//
	// It cannot be the same in two spindles at once. The id belongs to the
	// terminal rather than to the program, and every cover is preceded by a
	// delete of the id's placements: a second spindle sharing the number would
	// take the first one's picture off the screen with every track change. The
	// process id is what the two of them cannot both have.
	//
	// One id per slot, so a screen drawing two pictures at once is drawing two
	// images and not one image twice.
	imageID [slots]int

	mu  sync.Mutex
	out io.Writer // the tty; written to from the pipeline, never from View

	// sent is the newest load that reached the terminal, per slot. Every cover
	// in a slot goes out under the same image id, so a load that was overtaken
	// must not be written after the one that replaced it: the terminal would
	// keep a picture of a size the screen has stopped drawing, and show a
	// corner of it.
	sent [slots]uint64
}

// slots is how many pictures may be kept apart at once.
//
// Four: what the cursor is resting on, what is playing, the next record's colour
// and the program's own picture while it waits for a device.
//
// It was two, and the two that came after went out under numbers this renderer
// refuses — so their Render failed, quietly, and only on terminals that draw
// pictures this way. What that looked like was a logo that never appeared and a
// colour that never crossed, on exactly the terminals good enough to show both.
// A slot is an integer here; running out of them costs nothing but saying so.
const slots = 4

func NewKitty(out io.Writer, cell CellSize) *Kitty {
	// Zero is not an id the protocol accepts, and a pid can be anything. The
	// slots take consecutive ids from there; two spindles a single number apart
	// would collide, which is why the first id is shifted rather than masked.
	base := (os.Getpid()*slots)&imageIDMask | 1
	k := &Kitty{Cell: cell, out: out}
	for i := range k.imageID {
		k.imageID[i] = (base + i) & imageIDMask
	}
	return k
}

func (k *Kitty) Name() string { return "kitty" }

// errStale reports a cover that finished after the one that replaced it. It is
// not a failure — there is simply nothing left to do with it.
var errStale = errors.New("cover load overtaken by a newer one")

// IsStale reports whether an error is that of an overtaken load, which the UI
// should drop rather than complain about.
func IsStale(err error) bool { return errors.Is(err, errStale) }

func (k *Kitty) Render(img image.Image, wCells, hCells int, seq uint64, slot int) (string, error) {
	if slot < 0 || slot >= slots {
		return "", fmt.Errorf("render kitty: no such picture slot %d", slot)
	}

	cols, rows, pxW, pxH := fitCells(img, wCells, hCells, k.Cell)
	if cols == 0 || rows == 0 {
		return "", fmt.Errorf("render kitty: image does not fit %dx%d cells", wCells, hCells)
	}

	if !k.Cell.Measured {
		pxW *= unmeasuredSupersample
		pxH *= unmeasuredSupersample
	}
	// Sending more pixels than the source has only inflates the payload.
	b := img.Bounds()
	pxW, pxH = min(pxW, b.Dx()), min(pxH, b.Dy())

	scaled := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return "", fmt.Errorf("encode cover as png: %w", err)
	}
	if err := k.transmit(buf.Bytes(), cols, rows, seq, slot); err != nil {
		return "", err
	}
	return placeholderGrid(k.imageID[slot], cols, rows), nil
}

// transmit uploads the image and creates a virtual placement under imageID.
// Responses are suppressed with q=2: anything the terminal echoed back would be
// read by Bubble Tea as keyboard input.
func (k *Kitty) transmit(data []byte, cols, rows int, seq uint64, slot int) error {
	id := k.imageID[slot]

	encoded := base64.StdEncoding.EncodeToString(data)

	var sb strings.Builder

	// Delete the previous image and its placements first. Re-transmitting under
	// the same id replaces the picture but not the placement it already has, so
	// without this the terminal keeps drawing into the old rectangle and only
	// the corner of the new cover that fits inside it is ever seen.
	fmt.Fprintf(&sb, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)

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
				id, cols, rows, more, chunk)
			first = false
			continue
		}
		fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if seq < k.sent[slot] {
		// Overtaken while it was being prepared. The screen is already drawing
		// the newer cover, and this one has nowhere to go.
		return errStale
	}
	if _, err := io.WriteString(k.out, sb.String()); err != nil {
		return fmt.Errorf("transmit cover: %w", err)
	}
	k.sent[slot] = seq
	return nil
}

// placeholderGrid builds the rectangle of placeholder cells. Each cell carries
// its row and column as combining diacritics, and the run's foreground colour
// carries the image id, so the terminal can reassemble the picture wherever the
// characters end up.
func placeholderGrid(imageID, cols, rows int) string {
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
