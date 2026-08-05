package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// recordModel is a screen with the record on it and a sleeve to wear.
func recordModel(w, h int) Model {
	m := scopeModel(100, 44)
	m.width, m.height = w, h
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true
	m.words.drawn = true

	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := range 200 {
		for x := range 200 {
			v := uint8(20)
			if (x/20+y/20)%2 == 0 {
				v = 240
			}
			img.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	m.grain.have = cover.Grind(img, w, h, dotsPerCellX, dotsPerCellY)

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.6
	}
	m.scope.bands = bands
	return m
}

// It is a disc: ink inside its radius and nothing outside it.
func TestRecordIsRound(t *testing.T) {
	const w, rows = 74, 26
	m := recordModel(w, rows)

	lines := m.recordLines(w, rows)
	if len(lines) != rows {
		t.Fatalf("drew %d rows, want %d", len(lines), rows)
	}

	// The corners are outside any disc that fits the screen.
	for _, r := range []int{0, rows - 1} {
		plain := ansiOff(lines[r])
		if strings.TrimSpace(plain[:8]) != "" || strings.TrimSpace(plain[len(plain)-8:]) != "" {
			t.Errorf("row %d has ink in its corners, so the disc is not round", r)
		}
	}
	if strings.TrimSpace(ansiOff(lines[rows/2])) == "" {
		t.Error("nothing was drawn across the middle of the disc")
	}
}

// It turns, and it turns at the speed a record turns at: a full revolution in
// about a second and four fifths.
func TestRecordTurns(t *testing.T) {
	const w, rows = 74, 26
	m := recordModel(w, rows)

	first := strings.Join(m.recordLines(w, rows), "")
	for range 5 {
		m.recordFlow()
	}
	if strings.Join(m.recordLines(w, rows), "") == first {
		t.Error("the record did not turn")
	}

	// A revolution, counted in frames.
	m.record.turned = 0
	var frames int
	for m.record.turned < 6.28 && frames < 1000 {
		m.recordFlow()
		frames++
	}
	seconds := float64(frames) * float64(scopeInterval) / float64(time.Second)
	t.Logf("a revolution took %d frames, %.2f seconds — %.1f rpm", frames, seconds, 60/seconds)

	if rpm := 60 / seconds; rpm < 32 || rpm > 35 {
		t.Errorf("the record turns at %.1f rpm, want thirty-three and a third", rpm)
	}
}

// A beat sends a wave out along the grooves, and it dies away.
func TestRecordAnswersTheBeat(t *testing.T) {
	m := recordModel(74, 26)

	m.scope.envelope, m.record.wasLoud = 0.2, 0.2
	m.recordFlow()
	if m.record.lit > 0.1 {
		t.Error("a wave went out with nothing to set it off")
	}

	m.scope.envelope = 0.9
	m.recordFlow()
	if m.record.lit <= 0 {
		t.Fatal("a beat sent no wave along the grooves")
	}
	t.Logf("the beat lit the grooves at %.2f", m.record.lit)

	for range 200 {
		m.recordFlow()
	}
	if m.record.lit > 0.02 {
		t.Errorf("the wave is still lit at %.2f long after the beat", m.record.lit)
	}
}

// The record goes on for a bar with no words in it, comes off when the singer
// returns, and can be put on by hand whenever.
func TestRecordGoesOnForTheSolos(t *testing.T) {
	m := recordModel(74, 26)
	m.ps.TrackID = "now"
	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "♪"},
		{At: 30_000, Words: "and the words"},
	}

	m.setProgress(2 * time.Second)
	if !m.recordNow() {
		t.Error("the record did not go on for a bar with no words")
	}

	m.setProgress(31 * time.Second)
	if m.recordNow() {
		t.Error("the record stayed on through the words")
	}

	// By hand, and it holds the screen for its few seconds.
	m.putOnTheRecord()
	if !m.recordNow() {
		t.Error("the key did not put the record on")
	}
	m.words.until = time.Now().Add(-time.Second)
	if m.recordNow() {
		t.Error("the record stayed on past its welcome")
	}
}
