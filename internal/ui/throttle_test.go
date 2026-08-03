package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// The resting cadence is one call per five seconds: DESIGN.md 4.1 picked it as
// the point where drift is invisible and the quota is comfortable.
func TestRestingCadence(t *testing.T) {
	var m Model
	m.notePolled()

	if m.shouldResync() {
		t.Error("polled again immediately")
	}
	m.nextPollAt = time.Now().Add(-time.Millisecond)
	if !m.shouldResync() {
		t.Error("did not poll once due")
	}

	m.notePolled()
	if got := time.Until(m.nextPollAt); got > idlePoll || got < idlePoll-time.Second {
		t.Errorf("next poll due in %v, want about %v", got, idlePoll)
	}
}

// A change nobody here asked for means someone is using another device. Making
// them wait out the resting cadence for every following change is what makes a
// remote skip feel slow.
func TestFollowsAnotherDeviceClosely(t *testing.T) {
	m := Model{ps: &player.State{TrackID: "a", Playing: true}}

	if !m.drivenFromElsewhere(&player.State{TrackID: "b", Playing: true}) {
		t.Error("a track change we did not cause went unnoticed")
	}
	if !m.drivenFromElsewhere(&player.State{TrackID: "a", Playing: false}) {
		t.Error("a pause we did not cause went unnoticed")
	}

	// Our own optimistic change must not be mistaken for someone else's.
	m.optimisticUntil = time.Now().Add(optimisticWindow)
	if m.drivenFromElsewhere(&player.State{TrackID: "a", Playing: false}) {
		t.Error("our own change was read as a remote one")
	}

	m.optimisticUntil = time.Time{}
	m.followUntil = time.Now().Add(followWindow)
	m.notePolled()
	if got := time.Until(m.nextPollAt); got > activePoll {
		t.Errorf("next poll due in %v, want the faster %v while following", got, activePoll)
	}
}

// A track finishing by itself is a skip nobody pressed. The clock knows when it
// happened and the queue knows what follows, so it should behave like one.
func TestTrackRunningOutAdvancesLikeASkip(t *testing.T) {
	m := Model{
		ps:    &player.State{TrackID: "a", Title: "first", Playing: true, Duration: 2 * time.Second},
		queue: []player.Track{{ID: "b", Title: "second"}},
		// The clock, not the tick count, is what says the track is over.
		progressAt: time.Now().Add(-3 * time.Second),
	}

	var tm tea.Model = m
	for range 3 {
		tm, _ = tm.Update(msg.Tick{})
	}

	got := tm.(Model)
	if got.ps.Title != "second" {
		t.Errorf("title = %q, want the queued track once the clock ran out", got.ps.Title)
	}
	if got.awaitingTrack != "a" {
		t.Errorf("awaitingTrack = %q, want the finished track", got.awaitingTrack)
	}
	if len(got.queue) != 0 {
		t.Errorf("queue = %v, want the used track gone", got.queue)
	}
}

// Under repeat-one the same track starts again, so guessing from the queue would
// put the wrong title up.
func TestTrackRunningOutUnderRepeatOne(t *testing.T) {
	m := Model{
		ps: &player.State{
			TrackID: "a", Title: "first", Playing: true,
			Duration: 2 * time.Second, Repeat: player.RepeatTrack,
		},
		queue:      []player.Track{{ID: "b", Title: "second"}},
		progressAt: time.Now().Add(-3 * time.Second),
	}

	var tm tea.Model = m
	for range 3 {
		tm, _ = tm.Update(msg.Tick{})
	}

	got := tm.(Model)
	if got.ps.Title != "first" {
		t.Errorf("title = %q, want the same track under repeat-one", got.ps.Title)
	}
	if len(got.queue) != 1 {
		t.Errorf("queue = %v, want it left alone", got.queue)
	}
}

// Carrying on polling through a 429 is how a short throttle becomes a long one.
func TestThrottleSuspendsPolling(t *testing.T) {
	m := Model{rateLimitedUntil: time.Now().Add(time.Minute)}

	for range 20 {
		if m.shouldResync() {
			t.Fatal("polled while rate limited")
		}
	}

	m.rateLimitedUntil = time.Now().Add(-time.Second)
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

// Waiting out a debounce before anything is audible makes the key feel broken,
// which is how it felt. The first press of a run goes out at once.
func TestVolumeFirstPressIsNotDelayed(t *testing.T) {
	m := Model{ps: &player.State{Volume: 50}}

	if cmd := m.setVolume(55); cmd == nil {
		t.Fatal("the first press sent nothing")
	}
	if m.volumeSent != 55 {
		t.Errorf("volumeSent = %d, want 55 to have gone out immediately", m.volumeSent)
	}
}

// The presses that follow are collapsed, so holding the key costs two requests
// rather than one per repeat — but the value that lands is the final one.
func TestVolumeRunIsCollapsed(t *testing.T) {
	m := Model{ps: &player.State{Volume: 50}}

	m.setVolume(55) // leading edge, sent at once
	sentAfterFirst := m.volumeSent

	for v := 60; v <= 75; v += volumeStep {
		if cmd := m.setVolume(v); cmd == nil {
			t.Fatal("a press produced no command at all")
		}
	}

	if m.ps.Volume != 75 {
		t.Errorf("volume = %d, want 75 — the reading has to move at once", m.ps.Volume)
	}
	if m.volumeSent != sentAfterFirst {
		t.Errorf("volumeSent = %d, want the run to be collapsed rather than sent per press", m.volumeSent)
	}

	var tm tea.Model = m
	if _, cmd := tm.Update(msg.VolumeSettled{Seq: 2}); cmd != nil {
		t.Error("a settle message from an earlier keystroke sent a stale volume")
	}

	tm, cmd := tm.Update(msg.VolumeSettled{Seq: m.volumeSeq})
	if cmd == nil {
		t.Fatal("the newest settle message sent nothing")
	}
	if got := tm.(Model).volumeSent; got != 75 {
		t.Errorf("volumeSent = %d, want the final value 75", got)
	}
}

// Nothing more to say once the value has already gone out.
func TestVolumeSettleIsSkippedWhenNothingChanged(t *testing.T) {
	m := Model{ps: &player.State{Volume: 50}}
	m.setVolume(55)

	var tm tea.Model = m
	if _, cmd := tm.Update(msg.VolumeSettled{Seq: m.volumeSeq}); cmd != nil {
		t.Error("sent the same volume twice")
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
