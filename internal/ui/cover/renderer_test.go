package cover

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestFitCellsPreservesAspect(t *testing.T) {
	cell := CellSize{Width: 10, Height: 20}

	cases := []struct {
		name             string
		w, h             int
		wCells, hCells   int
		wantCols, wantRw int
	}{
		// A square image in a box that is square on screen fills it exactly.
		{"square fills square box", 640, 640, 20, 10, 20, 10},
		// A wide image keeps its width and loses height.
		{"wide letterboxes", 640, 320, 20, 10, 20, 5},
		// A tall image keeps its height and loses width.
		{"tall pillarboxes", 320, 640, 20, 10, 10, 10},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cols, rows, pxW, pxH := fitCells(solid(c.w, c.h, color.White), c.wCells, c.hCells, cell)
			if cols != c.wantCols || rows != c.wantRw {
				t.Errorf("got %dx%d cells, want %dx%d", cols, rows, c.wantCols, c.wantRw)
			}
			if pxW != cols*cell.Width || pxH != rows*cell.Height {
				t.Errorf("pixel size %dx%d does not match %d×%d cells", pxW, pxH, cols, rows)
			}
		})
	}
}

func TestHalfblockRenderDimensions(t *testing.T) {
	h := NewHalfblock(CellSize{Width: 10, Height: 20})

	art, err := h.Render(solid(640, 640, color.RGBA{R: 10, G: 200, B: 90, A: 255}), 20, 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(art, "\n")
	if len(lines) != 10 {
		t.Fatalf("got %d lines, want 10", len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != 20 {
			t.Errorf("line %d measures %d cells, want 20", i, w)
		}
		if !strings.Contains(line, "38;2;10;200;90") {
			t.Errorf("line %d lost the source colour", i)
		}
	}
}

func TestHalfblockRejectsEmptyArea(t *testing.T) {
	h := NewHalfblock(CellSize{Width: 10, Height: 20})
	if _, err := h.Render(solid(640, 640, color.White), 0, 0, 1); err == nil {
		t.Error("expected an error for a zero-sized area")
	}
}
