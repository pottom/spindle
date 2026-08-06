package style

import (
	"image/color"
	"testing"
)

// The pictures are drawn in the record's own hue, but not in its own greyness.
//
// An accent is an average of a photograph and comes back a fifth to a half
// saturated. At a fifth, a screenful of single dots reads as grey with a tint,
// whatever it is multiplied by — so there is a floor under the colour as well
// as a lift over it, and what is taken from the cover is only the washing out.
func TestThePicturesKeepTheirColour(t *testing.T) {
	for _, accent := range []struct {
		name string
		c    color.Color
	}{
		{"a strong cover", color.RGBA{0xc4, 0x41, 0x6e, 0xff}},
		{"a washed one", color.RGBA{0x6a, 0x7f, 0x9a, 0xff}},
		{"a muddy one", color.RGBA{0x8c, 0x7a, 0x5e, 0xff}},
		{"a grey one", color.RGBA{0x88, 0x88, 0x8c, 0xff}},
	} {
		hue, was, _ := toHSL(accent.c)
		p := huePalette(Theme{Text: color.RGBA{0xe8, 0xe6, 0xf0, 0xff}}, accent.c, 170)

		// The middle of the spectrum, half way up a bar: the colour most of the
		// screen is drawn in.
		fg := p[len(p)/2][len(p[0])/2].GetForeground()
		r, g, b, _ := fg.RGBA()
		got, sat, _ := toHSL(color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff})

		t.Logf("%-15s accent hue %3.0f sat %.2f → drawn at hue %3.0f sat %.2f", accent.name, hue, was, got, sat)

		if sat < 0.30 {
			t.Errorf("%s is drawn at %.2f saturation, which on single dots is grey", accent.name, sat)
		}
		if sat <= was && was < 0.6 {
			t.Errorf("%s came back no stronger than it went in (%.2f → %.2f)", accent.name, was, sat)
		}
	}
}
