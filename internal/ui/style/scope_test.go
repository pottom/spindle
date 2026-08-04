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
