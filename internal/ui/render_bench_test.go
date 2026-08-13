package ui

import (
	"math"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// What one frame of the big screen costs, at the size a terminal actually is.
func benchModel(w, rows int, mode scopeMode) Model {
	m := scopeModel(w, rows)
	m.stage.on = true
	m.stage.mode = mode
	m.ps.Duration = 4 * time.Minute
	m.lyrics.forTrack, m.lyrics.missing = m.ps.TrackID, true
	m.setProgress(30 * time.Second)

	m.scope.bands = make([]float32, 28)
	for i := range m.scope.bands {
		m.scope.bands[i] = 0.3 + 0.6*float32(i%7)/6
	}
	m.scope.frame = make([]float32, 2*512)
	for i := range m.scope.frame {
		m.scope.frame[i] = float32(math.Sin(float64(i) / 9))
	}
	m.scope.beat = player.Beat{Period: 500 * time.Millisecond, Loud: -20}
	m.scope.beatAt = time.Now()

	// A bar of marks, set the way the screen sets one.
	line := wordsMarks(w*dotsPerCellX, rows*dotsPerCellY)
	if img, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY); ok {
		m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
		m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
		m.words.text, m.words.beats = line, true
		m.words.since = time.Now().Add(-time.Second)
	}
	return m
}

func benchFrame(b *testing.B, w, rows int, mode scopeMode) {
	m := benchModel(w, rows, mode)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		m.swayFlow()
		m.swellFlow()
		m.wordsFlow(w, rows)
		m.stageFlow(w, rows)
		sink = m.render()
	}
}

var sink string

func BenchmarkFrameWords120x40(b *testing.B)  { benchFrame(b, 120, 40, scopeWords) }
func BenchmarkFrameWords200x50(b *testing.B)  { benchFrame(b, 200, 50, scopeWords) }
func BenchmarkFrameWords300x80(b *testing.B)  { benchFrame(b, 300, 80, scopeWords) }
func BenchmarkFrameBars200x50(b *testing.B)   { benchFrame(b, 200, 50, scopeBars) }
func BenchmarkFrameLadder200x50(b *testing.B) { benchFrame(b, 200, 50, scopeLadder) }
func BenchmarkFrameWave200x50(b *testing.B)   { benchFrame(b, 200, 50, scopeWave) }
