package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// Polling once a second would hit the rate limit; once every fifth tick is the
// cadence DESIGN.md 4.1 settled on.
func TestResyncEveryFifthTick(t *testing.T) {
	var m Model
	var fetches int

	for range 20 {
		m.tickCount++
		if m.shouldResync() {
			fetches++
		}
	}
	if fetches != 4 {
		t.Errorf("20 ticks produced %d fetches, want 4", fetches)
	}
}

// Carrying on polling through a 429 is how a short throttle becomes a long one.
func TestThrottleSuspendsPolling(t *testing.T) {
	m := Model{rateLimitedUntil: time.Now().Add(time.Minute)}

	for range 20 {
		m.tickCount++
		if m.shouldResync() {
			t.Fatal("polled while rate limited")
		}
	}

	m.rateLimitedUntil = time.Now().Add(-time.Second)
	m.tickCount = resyncEvery
	if !m.shouldResync() {
		t.Error("polling did not resume once the throttle expired")
	}
}

func TestRateLimitedMessageSetsTheWindow(t *testing.T) {
	var tm tea.Model = Model{}
	tm, _ = tm.Update(msg.RateLimited{RetryAfter: 30 * time.Second})

	m := tm.(Model)
	if !m.throttled() {
		t.Fatal("a rate limit message did not suspend polling")
	}

	text, _, ok := m.notice()
	if !ok || !contains(text, "Rate limited") {
		t.Errorf("notice = %q, want a rate limit warning", text)
	}
}

// A free account can read playback but never change it. That is a standing
// explanation, not a passing error, and it clears when a control call works.
func TestPremiumNoticeClearsOnSuccess(t *testing.T) {
	var tm tea.Model = Model{}
	tm, _ = tm.Update(msg.Error{Err: player.ErrPremiumRequired})

	m := tm.(Model)
	if !m.noPremium || m.err != nil {
		t.Fatalf("noPremium = %v, err = %v; want the standing notice and no error", m.noPremium, m.err)
	}
	if text, _, ok := m.notice(); !ok || !contains(text, "Premium") {
		t.Errorf("notice = %q, want a Premium explanation", text)
	}

	tm, _ = tm.Update(msg.ControlDone{})
	if m := tm.(Model); m.noPremium {
		t.Error("the notice survived a control call that worked")
	}
}

// A held volume key must collapse into one request, and a settle message from an
// earlier keystroke must not fire a stale value at the API.
func TestVolumeDebounce(t *testing.T) {
	m := Model{ps: &player.State{Volume: 50}}

	for range 5 {
		m.setVolume(m.ps.Volume + volumeStep)
	}
	if m.ps.Volume != 75 {
		t.Errorf("volume = %d, want 75 — the reading has to move at once", m.ps.Volume)
	}
	if m.volumeSeq != 5 {
		t.Errorf("volumeSeq = %d, want 5", m.volumeSeq)
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(msg.VolumeSettled{Seq: 2})
	if cmd != nil {
		t.Error("a settle message from an earlier keystroke sent a stale volume")
	}

	if _, cmd = tm.Update(msg.VolumeSettled{Seq: 5}); cmd == nil {
		t.Error("the newest settle message sent nothing")
	}
}

func TestNoticePrecedence(t *testing.T) {
	m := Model{
		rateLimitedUntil: time.Now().Add(time.Minute),
		noPremium:        true,
		err:              player.ErrNoActiveDevice,
	}

	// A throttle explains why nothing is moving right now, so it comes first.
	if text, _, _ := m.notice(); !contains(text, "Rate limited") {
		t.Errorf("notice = %q, want the throttle to outrank the rest", text)
	}

	m.rateLimitedUntil = time.Time{}
	if text, _, _ := m.notice(); !contains(text, "Premium") {
		t.Errorf("notice = %q, want Premium to outrank a plain error", text)
	}

	m.noPremium = false
	if _, _, ok := m.notice(); !ok {
		t.Error("the plain error was not shown")
	}

	m.err = nil
	if _, _, ok := m.notice(); ok {
		t.Error("a notice survived with nothing left to report")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
