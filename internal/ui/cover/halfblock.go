package cover

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync/atomic"

	"golang.org/x/image/draw"
)

// halfBlock is the upper half block: its foreground paints the top pixel of the
// cell and its background the bottom one, so one cell carries two pixel rows.
const halfBlock = "▀"

// Halfblock renders with 24-bit colour and the ▀ character. It needs nothing
// from the terminal beyond truecolour, so it is the universal fallback.
type Halfblock struct {
	Cell CellSize

	// behind is what a picture that is see-through is seen against.
	//
	// A cell can only be one colour, so a picture with transparency in it has to
	// be composited before it can be drawn at all — there is no way to say "the
	// terminal's own background" in an SGR sequence. Where the terminal has said
	// what its background is, that is the right answer and the transparency
	// disappears; where it has not, black is the assumption, which is what this
	// did everywhere before it could ask.
	//
	// Held atomically because the terminal answers the background query some
	// time after the renderer is made, and pictures are loaded off the main
	// goroutine.
	behind atomic.Pointer[color.RGBA]
}

func NewHalfblock(cell CellSize) *Halfblock {
	return &Halfblock{Cell: cell}
}

// SetBehind says what a see-through picture is seen against. See Halfblock.behind.
func (h *Halfblock) SetBehind(c color.Color) {
	if c == nil {
		return
	}
	r, g, b, _ := c.RGBA()
	h.behind.Store(&color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255})
}

// over lays a premultiplied pixel on the backdrop.
func (h *Halfblock) over(c color.RGBA) color.RGBA {
	if c.A == 255 {
		return c
	}
	var back color.RGBA
	if p := h.behind.Load(); p != nil {
		back = *p
	}
	gap := uint32(255 - c.A)
	return color.RGBA{
		R: uint8(uint32(c.R) + uint32(back.R)*gap/255),
		G: uint8(uint32(c.G) + uint32(back.G)*gap/255),
		B: uint8(uint32(c.B) + uint32(back.B)*gap/255),
		A: 255,
	}
}

func (h *Halfblock) Name() string { return "halfblock" }

func (h *Halfblock) Render(img image.Image, wCells, hCells int, _ uint64, _ int) (string, error) {
	cols, rows, _, _ := fitCells(img, wCells, hCells, h.Cell)
	if cols == 0 || rows == 0 {
		return "", fmt.Errorf("render halfblock: image does not fit %dx%d cells", wCells, hCells)
	}

	// One cell is one pixel wide and two pixel rows tall.
	scaled := image.NewRGBA(image.Rect(0, 0, cols, rows*2))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Src, nil)

	var sb strings.Builder
	sb.Grow(cols * rows * 24)
	for y := range rows {
		if y > 0 {
			sb.WriteByte('\n')
		}
		for x := range cols {
			top := h.over(scaled.RGBAAt(x, y*2))
			bottom := h.over(scaled.RGBAAt(x, y*2+1))
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm%s",
				top.R, top.G, top.B, bottom.R, bottom.G, bottom.B, halfBlock)
		}
		sb.WriteString("\x1b[0m")
	}
	return sb.String(), nil
}
