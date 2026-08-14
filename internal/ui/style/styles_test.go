package style

import (
	"charm.land/lipgloss/v2"
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

// The marked block's ground is a small step off the screen's own, in the
// screen's own colour.
//
// The terminal says what colour it is when it is asked, and it used to be asked
// only whether it was dark. That is enough for text and wrong for a ground: a
// light theme is not white — Solarized's is cream — and a raised block painted
// in a cool grey on it is a card from another program.
func TestTheRaisedGroundFollowsTheTerminal(t *testing.T) {
	for _, ground := range []string{"#000000", "#0D1117", "#1E1E2E", "#FFFFFF", "#FDF6E3", "#F5F5F5"} {
		on := lipgloss.Color(ground)
		h, sat, l := toHSL(on)

		s := New(l < 0.5, nil).On(on)
		bg, ok := s.Raised.GetBackground().(color.Color)
		if !ok {
			t.Fatalf("%s: the raised style has no ground", ground)
		}
		gotH, gotSat, gotL := toHSL(bg)

		// A step away, and the right way.
		if l < 0.5 && gotL <= l {
			t.Errorf("%s: the ground came out at %.2f, no lighter than the screen's %.2f", ground, gotL, l)
		}
		if l >= 0.5 && gotL >= l {
			t.Errorf("%s: the ground came out at %.2f, no darker than the screen's %.2f", ground, gotL, l)
		}
		if d := gotL - l; d < -groundStep-0.01 || d > groundStep+0.01 {
			t.Errorf("%s: the ground is %.3f away, want a step of %.2f", ground, d, groundStep)
		}

		// And in the screen's own colour: a cream terminal gets a cream card.
		if sat > 0.05 {
			if dh := gotH - h; dh < -5 || dh > 5 {
				t.Errorf("%s: the ground is hue %.0f, want the screen's %.0f", ground, gotH, h)
			}
			if gotSat < sat*0.8 {
				t.Errorf("%s: the ground lost its colour: %.2f against the screen's %.2f", ground, gotSat, sat)
			}
		}
	}

	// With no answer from the terminal it keeps the theme's own, rather than
	// coming up with nothing.
	if s := New(true, nil); s.Raised.GetBackground() == nil {
		t.Error("with no ground reported the block has no ground at all")
	}
}
