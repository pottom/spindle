package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/pottom/spindle/internal/ui/cover"
)

// grainModel is the big screen showing the record, with a sleeve ground for it.
func grainModel(w, h int) Model {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeGrain
	m.stage.on = true
	m.width, m.height = w, h

	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := range 300 {
		for x := range 300 {
			d := math.Hypot(float64(x-150), float64(y-150))
			v := uint8(max(0, 230-d*1.6))
			img.Set(x, y, color.RGBA{R: v, G: v / 2, B: uint8(float64(v) * 0.8), A: 255})
		}
	}
	m.grain.have = cover.Grind(img, w, h, dotsPerCellX, dotsPerCellY)
	m.grain.url, m.grain.cellsX, m.grain.cellsY = "cover", w, h
	m.cover.url = "cover"

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.5
	}
	m.scope.bands = bands
	return m
}

// The record is still the record: bright where the sleeve is bright, empty
// where it is dark.
func TestGrainDrawsTheRecord(t *testing.T) {
	m := grainModel(80, 20)
	lines := m.grainLines(80, 20)

	if len(lines) != 20 {
		t.Fatalf("drew %d rows, want 20", len(lines))
	}

	middle := strings.Count(ansiOff(lines[10]), " ")
	edge := strings.Count(ansiOff(lines[0]), " ")
	t.Logf("the middle row has %d blank cells, the top row %d", middle, edge)

	if middle >= edge {
		t.Errorf("the middle of the picture is no fuller than its edge (%d against %d blanks)", middle, edge)
	}
}

// Until a sleeve has been ground for this exact screen, nothing is drawn: half a
// picture stretched over the wrong shape is worse than waiting a frame for the
// right one.
func TestGrainWaitsForItsOwnSize(t *testing.T) {
	m := grainModel(80, 20)
	if got := m.grainLines(60, 20); got != nil {
		t.Errorf("drew %d rows at a size it was not ground for", len(got))
	}
}

// The music moves it: the bend follows the bands, and a rise in loudness sends
// a ring out from the middle.
func TestGrainMovesWithTheMusic(t *testing.T) {
	m := grainModel(80, 20)

	for range 20 {
		m.grainFlow(80, 20)
	}
	var bent bool
	for _, v := range m.grain.bend {
		if v > 0.5 {
			bent = true
		}
	}
	if !bent {
		t.Error("the spectrum did not bend the picture at all")
	}

	// A hit: the envelope jumps, which is what starts a ring.
	m.scope.envelope = 0.2
	m.grainFlow(80, 20)
	m.scope.envelope = 0.9
	m.grainFlow(80, 20)

	if m.grain.ringPush <= 0 {
		t.Fatal("a hit sent no ring out")
	}
	t.Logf("the hit pushed by %.1f dot rows", m.grain.ringPush)

	// And it dies away rather than ringing for ever.
	for range 200 {
		m.grainFlow(80, 20)
	}
	if m.grain.ringPush > 0.05 {
		t.Errorf("the ring is still pushing by %.2f long after the hit", m.grain.ringPush)
	}
}

// It is a picture for the whole screen: the strip's key steps over it, because
// four rows of a photograph is a smudge and the artwork is already on that
// screen a few cells away.
func TestGrainIsNotOfferedInTheStrip(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeLadder

	seen := map[scopeMode]bool{}
	mode := scopeMode(scopeOff)
	for range int(scopeModes) * 2 {
		mode = mode.next()
		for mode.big() {
			mode = mode.next()
		}
		seen[mode] = true
	}
	if seen[scopeGrain] {
		t.Error("the strip's cycle stopped on a picture meant for the whole screen")
	}
	if len(seen) != int(scopeModes)-1 {
		t.Errorf("the strip cycles %d pictures, want %d", len(seen), int(scopeModes)-1)
	}
}
