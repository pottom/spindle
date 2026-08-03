package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// A poll dispatched before a key press can land after it, still carrying the old
// playback flags. Inside the optimistic window that must not undo the local
// change, but new metadata should still come through. See DESIGN.md 4.2.
func TestAdoptInsideOptimisticWindow(t *testing.T) {
	m := Model{
		ps:              &player.State{Title: "old", Playing: true, Progress: time.Minute},
		localProgress:   time.Minute,
		optimisticUntil: time.Now().Add(optimisticWindow),
	}
	m.ps.Playing = false // the user just hit pause

	m.adopt(&player.State{Title: "new", Playing: true, Progress: 2 * time.Minute})

	if m.ps.Playing {
		t.Error("playback flag was overwritten inside the optimistic window")
	}
	if m.ps.Title != "new" {
		t.Errorf("metadata not adopted: title = %q, want %q", m.ps.Title, "new")
	}
	if m.localProgress != time.Minute {
		t.Errorf("localProgress = %v, want %v", m.localProgress, time.Minute)
	}
}

func TestAdoptAfterOptimisticWindow(t *testing.T) {
	m := Model{
		ps:              &player.State{Title: "old", Playing: false},
		localProgress:   time.Minute,
		optimisticUntil: time.Now().Add(-time.Second),
	}

	m.adopt(&player.State{Title: "new", Playing: true, Progress: 2 * time.Minute})

	if !m.ps.Playing {
		t.Error("server state was not adopted once the window had closed")
	}
	if m.localProgress != 2*time.Minute {
		t.Errorf("localProgress = %v, want %v", m.localProgress, 2*time.Minute)
	}
}

func TestNextRepeatCycles(t *testing.T) {
	want := []string{player.RepeatContext, player.RepeatTrack, player.RepeatOff}
	mode := player.RepeatOff
	for i, w := range want {
		mode = nextRepeat(mode)
		if mode != w {
			t.Errorf("step %d: mode = %q, want %q", i, mode, w)
		}
	}
}
