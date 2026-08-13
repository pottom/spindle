package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func trackAt(id, title string) player.Track {
	return player.Track{ID: id, Title: title, Artists: []string{"someone"}, Duration: 3 * time.Minute}
}

// Pressing n should put the next track on screen at once. Waiting for Spotify to
// confirm it costs about half a second, which is the difference between a key
// that responds and one that lags.
func TestSkipShowsTheQueuedTrackAtOnce(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "first", Playing: true}
	m.queue = []player.Track{trackAt("b", "second"), trackAt("c", "third")}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})

	got := tm.(Model)
	if got.ps.Title != "second" {
		t.Errorf("title = %q, want the queued track straight away", got.ps.Title)
	}
	if got.elapsed() > 100*time.Millisecond {
		t.Errorf("elapsed = %v, want it reset", got.elapsed())
	}
	if len(got.queue) != 1 || got.queue[0].ID != "c" {
		t.Errorf("queue = %v, want the track just used to be gone", got.queue)
	}
	if got.awaitingTrack != "a" {
		t.Errorf("awaitingTrack = %q, want the track being left", got.awaitingTrack)
	}
}

// Spotify keeps reporting the old track for a moment after a skip. Adopting that
// would flash the previous title back over the one just shown.
func TestStaleSnapshotDoesNotUndoASkip(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "first", Playing: true}
	m.queue = []player.Track{trackAt("b", "second")}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})

	// The confirming fetch comes back still showing the track we left.
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "a", Title: "first", Playing: true}})
	if got := tm.(Model); got.ps.Title != "second" {
		t.Fatalf("title = %q, want the skip to survive a stale snapshot", got.ps.Title)
	}

	// And once Spotify catches up, the chase ends.
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "b", Title: "second", Playing: true}})
	if got := tm.(Model); got.awaitingTrack != "" {
		t.Errorf("awaitingTrack = %q, want it cleared once confirmed", got.awaitingTrack)
	}
}

// With an empty queue there is nothing to guess, so the skip falls back to
// asking — it must not blank the screen in the meantime.
func TestSkipWithoutAQueueKeepsTheCurrentTrack(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "first", Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})

	got := tm.(Model)
	if got.ps.Title != "first" {
		t.Errorf("title = %q, want the current track held until confirmation", got.ps.Title)
	}
	if got.awaitingTrack != "a" {
		t.Errorf("awaitingTrack = %q, want the chase started anyway", got.awaitingTrack)
	}
}

// The queue belongs to whatever the server says is playing. Re-asking for one we
// already hold would spend a request per poll.
func TestQueueIsFetchedOncePerConfirmedTrack(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)

	var tm tea.Model = m
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "a"}})
	if got := tm.(Model); got.queueFor != "a" {
		t.Fatalf("queueFor = %q, want a", got.queueFor)
	}

	// The same track again must not start another fetch.
	before := tm.(Model).queueFor
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "a"}})
	if got := tm.(Model); got.queueFor != before {
		t.Errorf("queueFor changed to %q on an unchanged track", got.queueFor)
	}
}
