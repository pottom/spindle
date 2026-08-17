package ui

import (
	"testing"

	"github.com/pottom/spindle/internal/player"
)

// Pressing next showed the track twice for a moment.
//
// The skip is drawn the instant it is pressed — that is what makes it feel
// immediate — but the list of what comes next still belongs to the track before
// it until the device answers. So the track that has just started was both the
// row at the top and the head of what is still to come.
func TestASkipDoesNotShowTheTrackTwice(t *testing.T) {
	m := queueModel(0, "b", "c", "d")
	m.queue[0].Title = "Jump Into The Fire"

	// The queue on screen was read while "a" was playing, and "b" has just been
	// skipped to.
	was := player.Track{ID: "a", Title: "Before"}
	m.nowQueued = &was
	m.ps = &player.State{TrackID: "b", Title: "Jump Into The Fire", Playing: true}

	rows := m.queueRows()
	seen := 0
	for _, row := range rows {
		if row.ID == "b" {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("the track that has just started is on %d rows, want one:\n%v", seen, ids(rows))
	}
	if len(rows) != 3 || rows[1].ID != "c" {
		t.Errorf("the rest of the queue was disturbed: %v", ids(rows))
	}
}

// And a queue that has caught up says what it means: a track queued twice, or a
// record repeating itself, is two rows because it is two plays.
func TestATrackQueuedTwiceIsStillTwoRows(t *testing.T) {
	m := queueModel(0, "b", "b", "c")
	now := player.Track{ID: "b", Title: "Twice"}
	m.nowQueued = &now
	m.ps = &player.State{TrackID: "b", Title: "Twice", Playing: true}

	rows := m.queueRows()
	seen := 0
	for _, row := range rows {
		if row.ID == "b" {
			seen++
		}
	}
	if seen != 3 {
		t.Errorf("a track playing and queued twice is on %d rows, want three: %v", seen, ids(rows))
	}
}
