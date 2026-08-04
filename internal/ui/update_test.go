package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A poll dispatched before a key press can land after it, still carrying the old
// playback flags. Inside the optimistic window that must not undo the local
// change, but new metadata should still come through. See DESIGN.md 4.2.
func TestAdoptInsideOptimisticWindow(t *testing.T) {
	m := Model{
		ps:              &player.State{Title: "old", Playing: true, Progress: time.Minute},
		progressAt:      time.Now(),
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
	if got := m.ps.Progress; got != time.Minute {
		t.Errorf("progress anchor = %v, want %v", got, time.Minute)
	}
}

func TestAdoptAfterOptimisticWindow(t *testing.T) {
	m := Model{
		ps:              &player.State{Title: "old", Playing: false},
		optimisticUntil: time.Now().Add(-time.Second),
	}

	m.adopt(&player.State{Title: "new", Playing: true, Progress: 2 * time.Minute})

	if !m.ps.Playing {
		t.Error("server state was not adopted once the window had closed")
	}
	if got := m.ps.Progress; got != 2*time.Minute {
		t.Errorf("progress anchor = %v, want %v", got, 2*time.Minute)
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

// The position on screen is the last reported one carried forward by the clock,
// not a counter of its own. A counter and a poll are two answers to the same
// question, and they disagree by whatever the tick has drifted — which showed
// up as the playhead jumping backwards whenever the poll won.
func TestElapsedCarriesForwardFromTheAnchor(t *testing.T) {
	m := Model{
		ps:         &player.State{Playing: true, Progress: time.Minute, Duration: 5 * time.Minute},
		progressAt: time.Now().Add(-2 * time.Second),
	}

	got := m.elapsed()
	if got < 61500*time.Millisecond || got > 62500*time.Millisecond {
		t.Errorf("elapsed = %v, want about 62s", got)
	}
}

// Ticks used to add a second each. They no longer add anything, so an adopted
// snapshot cannot disagree with them.
func TestTicksDoNotAdvanceThePosition(t *testing.T) {
	var tm tea.Model = Model{
		ps:         &player.State{Playing: true, Progress: time.Minute, Duration: 5 * time.Minute},
		progressAt: time.Now(),
	}

	for range 5 {
		tm, _ = tm.Update(msg.Tick{})
	}

	if got := tm.(Model).ps.Progress; got != time.Minute {
		t.Errorf("anchor = %v, want it untouched by ticks", got)
	}
}

// Pausing stops the clock being carried forward, so the anchor has to already
// hold everything that had accumulated — otherwise the playhead drops back to
// wherever the last poll left it.
func TestPauseFreezesWhereItWas(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Playing: true, Progress: time.Minute, Duration: 5 * time.Minute}
	m.progressAt = time.Now().Add(-3 * time.Second)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	got := tm.(Model)
	if got.ps.Playing {
		t.Fatal("space did not pause")
	}
	if elapsed := got.elapsed(); elapsed < 62500*time.Millisecond {
		t.Errorf("elapsed = %v, want the three seconds already played to be kept", elapsed)
	}
}

// Paused, it stays put however long it sits there.
func TestPausedPositionDoesNotDrift(t *testing.T) {
	m := Model{
		ps:         &player.State{Playing: false, Progress: time.Minute, Duration: 5 * time.Minute},
		progressAt: time.Now().Add(-time.Hour),
	}

	if got := m.elapsed(); got != time.Minute {
		t.Errorf("elapsed = %v, want it unchanged at 1m", got)
	}
}

// The names come off the audio stream and there is not always one, but the
// queue still knows what is playing. A blank screen over music that is plainly
// playing is the one answer that helps nobody.
func TestABlankStateIsFilledFromTheQueue(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.QueueFetched{
		Current: []player.Track{{
			ID: "now", Title: "Sandokan", Artists: []string{"Neoton Familia"},
			Album: "Sandokan", CoverURL: "http://cover", Duration: 3*time.Minute + 27*time.Second,
		}},
	})

	got := tm.(Model)
	if got.ps.Title != "Sandokan" || got.ps.TrackID != "now" {
		t.Errorf("state = %q/%q, want the queue's answer", got.ps.TrackID, got.ps.Title)
	}
	if got.ps.Duration == 0 || got.ps.CoverURL == "" {
		t.Errorf("duration = %v, cover = %q — want both borrowed too", got.ps.Duration, got.ps.CoverURL)
	}
}

// What the device does say is never overwritten by the queue: the device is the
// one that knows, and the queue lags it.
func TestTheQueueDoesNotOverwriteWhatTheDeviceSaid(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "the device's answer", Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.QueueFetched{
		Current: []player.Track{{ID: "now", Title: "the queue's answer"}},
	})

	if got := tm.(Model).ps.Title; got != "the device's answer" {
		t.Errorf("title = %q, want the device's", got)
	}
}
