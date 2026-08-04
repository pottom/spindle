package style

import (
	"image/color"
	"math"
	"testing"
)

func hueOf(t *testing.T, c color.Color) float64 {
	t.Helper()
	h, _, _ := toHSL(c)
	return h
}

// A hue is a circle, so the distance between two of them wraps.
func hueGap(a, b float64) float64 {
	d := math.Abs(a - b)
	return min(d, 360-d)
}

var accents = map[string]color.RGBA{
	"warm sand": {R: 0xde, G: 0x89, B: 0x5f, A: 0xff},
	"pale blue": {R: 0x8d, G: 0xb9, B: 0xd5, A: 0xff},
	"gold":      {R: 0xcf, G: 0xa9, B: 0x4f, A: 0xff},
	"green":     {R: 0x1d, G: 0xb9, B: 0x54, A: 0xff},
}

// The extremes of the waveform are drawn in a different colour, not a dimmer
// one. A tip that is merely a faded accent disappears into the trace; the point
// of it is that it reads as another material.
func TestScopeEdgeIsNotJustADimmerAccent(t *testing.T) {
	for name, accent := range accents {
		s := New(true, accent)

		core := s.ScopeCore[len(s.ScopeCore)-2].GetForeground()
		edge := s.ScopeEdge[len(s.ScopeEdge)-2].GetForeground()

		_, coreSat, _ := toHSL(core)
		_, edgeSat, _ := toHSL(edge)
		if edgeSat >= coreSat {
			t.Errorf("%s: the edge is as saturated as the trace (%.2f vs %.2f), so it will read as the same colour",
				name, edgeSat, coreSat)
		}
	}
}

// The trace runs through a little of the hue on either side of the accent, so
// it is not one flat colour — but it never leaves the accent's family, or it
// would stop belonging to the artwork.
func TestScopeCoreTravelsWithoutLeavingTheAccent(t *testing.T) {
	for name, accent := range accents {
		s := New(true, accent)
		base := hueOf(t, accent)

		lowest := hueOf(t, s.ScopeCore[0].GetForeground())
		highest := hueOf(t, s.ScopeCore[len(s.ScopeCore)-1].GetForeground())

		if travel := hueGap(lowest, highest); travel < 15 {
			t.Errorf("%s: the palette travels %.0f° of hue, want enough to see", name, travel)
		}
		for _, h := range []float64{lowest, highest} {
			if gap := hueGap(h, base); gap > 45 {
				t.Errorf("%s: a step sits %.0f° from the accent, want it to stay in the family", name, gap)
			}
		}
	}
}

// Round-tripping a colour through HSL has to give it back, or every derived
// shade is quietly wrong.
func TestHSLRoundTrip(t *testing.T) {
	for name, c := range accents {
		h, s, l := toHSL(c)
		back := fromHSL(h, s, l)
		r1, g1, b1, _ := c.RGBA()
		r2, g2, b2, _ := back.RGBA()

		for i, pair := range [][2]uint32{{r1, r2}, {g1, g2}, {b1, b2}} {
			if d := int(pair[0]>>8) - int(pair[1]>>8); d > 1 || d < -1 {
				t.Errorf("%s: channel %d came back %d instead of %d", name, i, pair[1]>>8, pair[0]>>8)
			}
		}
	}
}

// A bar carries two facts, so the palette has two dimensions: where the energy
// sits, and how much of it there is.
func TestBarPaletteHasTwoDimensions(t *testing.T) {
	for name, accent := range accents {
		s := New(true, accent)
		if len(s.Bars) < 4 || len(s.Bars[0]) < 4 {
			t.Fatalf("%s: palette is %dx%d, too small to read", name, len(s.Bars), len(s.Bars[0]))
		}

		lum := func(c color.Color) float64 {
			r, g, b, _ := c.RGBA()
			return (0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(b>>8)) / 255
		}

		// Up a bar: the tip is far brighter than the foot.
		top := len(s.Bars[0]) - 1
		foot, tip := lum(s.Bars[0][0].GetForeground()), lum(s.Bars[0][top].GetForeground())
		if tip <= foot*2 {
			t.Errorf("%s: a bar runs %.2f to %.2f, want the tip to burn", name, foot, tip)
		}

		// Across the width: the hue travels, and stays in the accent's family.
		base := hueOf(t, accent)
		lo := hueOf(t, s.Bars[0][top].GetForeground())
		hi := hueOf(t, s.Bars[len(s.Bars)-1][top].GetForeground())
		if travel := hueGap(lo, hi); travel < 20 {
			t.Errorf("%s: the spectrum travels %.0f° of hue, want the sweep to show", name, travel)
		}
		for _, h := range []float64{lo, hi} {
			if gap := hueGap(h, base); gap > barHueArc {
				t.Errorf("%s: an end sits %.0f° from the accent, want it in the family", name, gap)
			}
		}
	}
}
