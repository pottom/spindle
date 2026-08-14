package ui

import (
	"fmt"
	"math"
	"strings"
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

// benchLeaving hands the model a line on its way out, the way wordsAdopt does
// when the picture changes.
func benchLeaving(m *Model, w, rows int, leave wordsMove) {
	m.words.was, m.words.wasWhere = m.words.have, m.words.where
	m.words.went, m.words.leave = time.Now(), leave
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

// What a picture on its way out costs on top of the one arriving. Measured
// because the recording said it was everything: of 153 frames that went missing
// on a wall-sized screen, 146 had a line leaving, and every one of the 146 was
// going off by popping.
func benchFrameLeaving(b *testing.B, w, rows int, leave wordsMove) {
	m := benchModel(w, rows, scopeWords)
	benchLeaving(&m, w, rows, leave)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sink = m.render()
	}
}

func BenchmarkFrameWords120x40(b *testing.B)  { benchFrame(b, 120, 40, scopeWords) }
func BenchmarkFrameWords200x50(b *testing.B)  { benchFrame(b, 200, 50, scopeWords) }
func BenchmarkFrameWords300x80(b *testing.B)  { benchFrame(b, 300, 80, scopeWords) }
func BenchmarkFrameBars200x50(b *testing.B)   { benchFrame(b, 200, 50, scopeBars) }
func BenchmarkFrameLadder200x50(b *testing.B) { benchFrame(b, 200, 50, scopeLadder) }
func BenchmarkFrameWave200x50(b *testing.B)   { benchFrame(b, 200, 50, scopeWave) }

func BenchmarkFrameStill352x84(b *testing.B) { benchFrame(b, 352, 84, scopeWords) }
func BenchmarkFramePopping352x84(b *testing.B) {
	benchFrameLeaving(b, 352, 84, wordsPopping)
}
func BenchmarkFrameSpilling352x84(b *testing.B) {
	benchFrameLeaving(b, 352, 84, wordsSpilling)
}
func BenchmarkFrameWiping352x84(b *testing.B) {
	benchFrameLeaving(b, 352, 84, wordsWiping)
}

// What one frame of the library's wall costs, with every tile carrying a
// picture the size a terminal really draws one.
func BenchmarkTheWall(b *testing.B) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabLibrary
	m.width, m.height = 200, 50
	m.resize()

	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	m.tiles = map[string]coverState{}
	for i := range 60 {
		id := fmt.Sprintf("p%d", i)
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: id, Name: fmt.Sprintf("List %d", i), Owner: "somebody", Tracks: 30,
			CoverURL: "http://example.invalid/" + id,
		})
		// A picture the size of one the kitty renderer hands back: a cell is a
		// placeholder character, two diacritics and a colour.
		var art strings.Builder
		for range g.artRows {
			art.WriteString("\x1b[38;2;1;2;3m")
			for range g.tileW {
				art.WriteString("\U0010EEEE̅̅")
			}
			art.WriteString("\x1b[m\n")
		}
		tile := coverState{url: "http://example.invalid/" + id,
			width: g.boxW, height: g.boxH}
		tile.took(art.String())
		m.tiles[id] = tile
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = m.libraryPaneView(m.layout(), m.layout().bodyHeight)
	}
}
