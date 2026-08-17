package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A device coming up says so a dozen times in three seconds, and every one of
// those used to go straight out as a fetch. Measured against the real Web API:
// nineteen requests in the first three seconds of an open window, paid for out
// of a daily quota that a device reconnecting all night would spend by morning.
//
// So a storm is one request, and the ones inside the gap are not dropped: the
// next poll moves forward to the end of it, and the last word is read once.
func TestAStormOfChangesIsOneRequest(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.nextPollAt = time.Now().Add(time.Hour)
	// Not a window that has just opened: Init's own fetch starts the clock, so a
	// storm in the first second is a storm that has already been answered.
	m.polledAt = time.Now().Add(-time.Minute)

	asked := 0
	var tm tea.Model = m
	for range 12 {
		var cmd tea.Cmd
		tm, cmd = tm.Update(msg.StateChanged{})
		if cmd != nil {
			asked++
		}
	}
	if asked != 1 {
		t.Errorf("twelve changes in a moment asked %d times, want one", asked)
	}

	// Nothing was dropped: what the last of them had to say is due within the
	// gap rather than in an hour.
	got := tm.(Model)
	if wait := time.Until(got.nextPollAt); wait > eventGap {
		t.Errorf("the last change of the storm is read in %s, want within %s", wait, eventGap)
	}
}

// And a change that stands on its own is read at once. Somebody pressing play on
// their phone may not wait a second for the screen to say so.
func TestAChangeOnItsOwnIsReadAtOnce(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.polledAt = time.Now().Add(-time.Minute)

	if _, cmd := m.answer(msg.StateChanged{}); cmd == nil {
		t.Error("a change nobody had just asked about went unread")
	}
}

// The answer a window gets all day while nobody is listening is "nothing is
// playing anywhere", and it has to rest the cadence like any other answer that
// says what the last one said. It did not, which left the one case the resting
// cadence exists for running at its quickest.
func TestNothingPlayingRestsTheCadenceToo(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)

	var tm tea.Model = m
	for range 6 {
		tm, _ = tm.Update(msg.NoActiveDevice{})
	}
	if got := tm.(Model); got.restFor != restMost {
		t.Errorf("six answers of nothing left the poll at %s, want %s", got.restFor, restMost)
	}

	// But the first one is news — it is how the screen learns to offer the
	// device list — and news wakes the cadence rather than easing it.
	fresh := New(player.NewMock(), nil, defaultTestCell)
	fresh.restFor = restMost
	fresh.ps = &player.State{TrackID: "a", Playing: true}
	var back tea.Model = fresh
	back, _ = back.Update(msg.NoActiveDevice{})
	if got := back.(Model); got.restFor != restMost {
		t.Errorf("music stopping somewhere else left the poll at %s, want it unchanged", got.restFor)
	}
}

// A device that never turns up is given up on. spotify-player gives its own five
// tries a second apart; this is the same half minute, and what it protects
// against is a daemon Spotify cannot see and a window left open in front of it —
// which would otherwise ask for the list every three seconds until morning.
func TestTheWaitForOurOwnDeviceEnds(t *testing.T) {
	m := New(&ownDaemon{Player: player.NewMock(), id: "never-turns-up"}, nil, defaultTestCell)
	m.noDevice = true
	m.adoptingSince = time.Now().Add(-2 * adoptWindow)
	m.devicesAt = time.Now().Add(-time.Hour)

	if !m.awaitingOwnDevice() {
		t.Fatal("a daemon whose device is not on the list is not being waited for")
	}
	if cmd := m.refreshDevices(); cmd != nil {
		t.Error("a device that has not appeared in half an hour was still being looked for")
	}

	// And anybody touching the window starts the half minute again: starting the
	// daemon by hand and coming back is exactly how that happens.
	m.stir()
	if cmd := m.refreshDevices(); cmd == nil {
		t.Error("somebody coming back to the window did not start it looking again")
	}
}
