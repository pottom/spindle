package cover

import (
	"fmt"
	"image"
	"strings"

	"golang.org/x/image/draw"
)

// halfBlock is the upper half block: its foreground paints the top pixel of the
// cell and its background the bottom one, so one cell carries two pixel rows.
const halfBlock = "▀"

// Halfblock renders with 24-bit colour and the ▀ character. It needs nothing
// from the terminal beyond truecolour, so it is the universal fallback.
type Halfblock struct {
	Cell CellSize
}

func NewHalfblock(cell CellSize) *Halfblock {
	return &Halfblock{Cell: cell}
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
			top := scaled.RGBAAt(x, y*2)
			bottom := scaled.RGBAAt(x, y*2+1)
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm%s",
				top.R, top.G, top.B, bottom.R, bottom.G, bottom.B, halfBlock)
		}
		sb.WriteString("\x1b[0m")
	}
	return sb.String(), nil
}
