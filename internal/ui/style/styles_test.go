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

// The marked block's ground is the record's hue at the screen's own weight.
//
// A multiplier was the wrong instrument: covers come back at every lightness
// there is, and the same factor makes one album a dark slab and the next a lit
// one. What a ground has to be is the same distance off the screen's whatever
// record it belongs to.
func TestTheRaisedGroundKeepsItsWeight(t *testing.T) {
	light := func(c color.Color) float64 {
		_, _, l := toHSL(c)
		return l
	}

	for _, accent := range []string{"#1DB954", "#E8734A", "#4A9BE8", "#D8C24A", "#8B5CF6"} {
		for _, dark := range []bool{true, false} {
			s := New(dark, lipgloss.Color(accent))
			bg, ok := s.Raised.GetBackground().(color.Color)
			if !ok {
				t.Fatalf("%s: the raised style has no ground", accent)
			}
			if got, want := light(bg), raisedLight(dark); got < want-0.01 || got > want+0.01 {
				t.Errorf("%s dark=%v: the ground is at lightness %.2f, want %.2f", accent, dark, got, want)
			}

			// And it is the record's hue, not a grey.
			wantHue, _, _ := toHSL(lipgloss.Color(accent))
			gotHue, sat, _ := toHSL(bg)
			if sat < 0.2 {
				t.Errorf("%s: the ground came out at saturation %.2f, which is a grey", accent, sat)
			}
			// Within a few degrees: a ground this dark has only a handful of
			// steps of each channel to be built out of, so the hue lands where
			// eight bits will let it.
			if d := gotHue - wantHue; d < -5 || d > 5 {
				t.Errorf("%s: the ground is hue %.0f, want the record's %.0f", accent, gotHue, wantHue)
			}
		}
	}
}
