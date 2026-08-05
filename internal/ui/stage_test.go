package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// stageModel is a player screen with the big picture up and something to draw.
func stageModel(w, h int) Model {
	m := scopeModel(w, h)
	m.stage.on = true

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.6
	}
	m.scope.bands = bands
	return m
}

// The key gives the whole terminal to the visualiser, and the next key takes it
// back: it is a screen you watch rather than work on.
func TestStageTakesTheScreenAndGivesItBack(t *testing.T) {
	m := scopeModel(100, 40)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	got := tm.(Model)
	if !got.stage.on {
		t.Fatal("f did not open the big screen")
	}
	if cmd == nil && !got.scope.running {
		t.Error("nothing was left to fetch the frames it draws")
	}
	if !got.scopeVisible() {
		t.Error("the frames stop while the big screen is up")
	}

	// Anything at all comes back, including a key that means something else
	// everywhere: this is the way out and it has to be the first thing tried.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	back := tm.(Model)
	if back.stage.on {
		t.Error("a key press left the big screen up")
	}
	if back.tab != tabPlayer {
		t.Errorf("the key that closed the screen also changed tab, to %d", back.tab)
	}
}

// The music keeps its keys: stopping it or turning it down is not a reason to
// lose the picture.
func TestStageKeepsTheTransport(t *testing.T) {
	m := stageModel(100, 40)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !tm.(Model).stage.on {
		t.Error("the space bar closed the big screen instead of pausing")
	}
	if tm.(Model).ps.Playing {
		t.Error("the space bar did not reach the transport")
	}
}

// It fills the terminal exactly: every row, every column, no more and no less.
func TestStageFillsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 40}, {200, 60}} {
		m := stageModel(size[0], size[1])

		art := m.stageArt(size[0], size[1])
		if len(art) != size[1] {
			t.Fatalf("%dx%d: drew %d rows", size[0], size[1], len(art))
		}
		for i, line := range art {
			if got := len([]rune(ansiOff(line))); got != size[0] {
				t.Errorf("%dx%d: row %d is %d cells wide", size[0], size[1], i, got)
			}
		}
	}
}

// The picture is a reflection: what stands above the middle stands below it,
// which is what fills the frame rather than leaving the top half empty.
func TestStageMirrorsAboutTheMiddle(t *testing.T) {
	m := stageModel(100, 40)
	art := m.stageArt(100, 40)

	var lit []int
	for r, line := range art {
		if strings.TrimSpace(ansiOff(line)) != "" {
			lit = append(lit, r)
		}
	}
	if len(lit) < 2 {
		t.Fatalf("the picture lit %d rows, want it filling the frame", len(lit))
	}

	// The first and last lit rows have to be the same distance from the middle.
	first, last := lit[0], lit[len(lit)-1]
	above, below := 40/2-first, last-(40/2-1)
	if above-below > 1 || below-above > 1 {
		t.Errorf("the picture reaches %d rows up and %d down, want a reflection", above, below)
	}
}

// A band that jumps throws water into the air, and what goes up comes down: the
// air empties again once the music stops jumping.
func TestStageThrowsWaterAndTakesItBack(t *testing.T) {
	m := stageModel(100, 40)

	quiet := make([]float32, 28)
	for i := range quiet {
		quiet[i] = 0.2
	}
	m.scope.bands = quiet
	m.stageFlow()

	// A hit across the bottom of the range.
	hit := make([]float32, 28)
	copy(hit, quiet)
	for i := range 6 {
		hit[i] = 1
	}
	m.scope.bands = hit
	m.stageFlow()

	if len(m.stage.drops) == 0 {
		t.Fatal("a hit threw nothing into the air")
	}
	t.Logf("the hit threw %d drops", len(m.stage.drops))

	// Left alone, every one of them comes back.
	m.scope.bands = quiet
	for range 400 {
		m.stageFlow()
	}
	if len(m.stage.drops) != 0 {
		t.Errorf("%d drops are still in the air after the music stopped jumping", len(m.stage.drops))
	}
}
