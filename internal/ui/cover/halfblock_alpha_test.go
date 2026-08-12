package cover

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// A see-through picture is laid on the terminal's own background, not on black.
//
// A cell can only be one colour, so a picture with transparency has to be
// flattened before it can be drawn at all — there is no way to say "whatever is
// behind" in an SGR sequence. It used to be flattened against nothing, which is
// black: measured on a logo 57% of which is clear, 29% of the cells came out
// pure black. On a dark terminal that is a rectangle slightly darker than the
// screen; on a light one it is a black box round the picture.
func TestASeeThroughPictureIsLaidOnTheTerminalsBackground(t *testing.T) {
	// Half the picture clear, half solid red.
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			if y < 4 {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
			}
		}
	}

	h := NewHalfblock(CellSize{Width: 10, Height: 20, Measured: true})

	// Nothing said yet: black, which is what it always did.
	dark, err := h.Render(img, 8, 8, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dark, ";48;2;0;0;0m") {
		t.Error("with no background given, the clear half is not black")
	}

	// And once the terminal has said what its background is.
	h.SetBehind(color.RGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 255})
	on, err := h.Render(img, 8, 8, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(on, ";48;2;0;0;0m") {
		t.Error("the clear half is still black after the background was given")
	}
	if !strings.Contains(on, "48;2;30;30;46m") {
		t.Errorf("the clear half is not the background that was given:\n%q", on)
	}
	// The solid half is untouched by any of it.
	if !strings.Contains(on, "38;2;255;0;0") {
		t.Error("the solid half lost its colour")
	}
}

// A pixel that is only half there is mixed, rather than snapping one way.
func TestAHalfClearPixelIsMixedWithWhatIsBehindIt(t *testing.T) {
	h := NewHalfblock(CellSize{Width: 10, Height: 20, Measured: true})
	h.SetBehind(color.RGBA{R: 200, G: 200, B: 200, A: 255})

	// Premultiplied: white at half alpha.
	got := h.over(color.RGBA{R: 128, G: 128, B: 128, A: 128})
	if got.R < 220 || got.R > 240 {
		t.Errorf("half of white over light grey came out %d, want about 227", got.R)
	}
	if got.A != 255 {
		t.Errorf("the result is still see-through: alpha %d", got.A)
	}
	// And something solid is returned as it was, whatever is behind.
	if solid := h.over(color.RGBA{R: 10, G: 20, B: 30, A: 255}); solid != (color.RGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Errorf("a solid pixel was changed: %v", solid)
	}
}
