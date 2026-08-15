package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// Three seconds for ever is twenty-eight thousand requests a day, and an
// application has a daily quota: spindle left open overnight came back to "rate
// limited for 23h4m", which is a lockout rather than a throttle. So the cadence
// rests while the answer keeps saying the same thing.
func TestThePollRestsWhenNothingIsHappening(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Playing: true, Volume: 40}

	same := &player.State{TrackID: "a", Playing: true, Volume: 40}

	var tm tea.Model = m
	for range 6 {
		tm, _ = tm.Update(msg.StateFetched{State: same})
	}
	got := tm.(Model)
	if got.restFor != restMost {
		t.Errorf("after six identical answers the poll rests %s, want %s", got.restFor, restMost)
	}

	// And the position moving is not news: it moves by itself.
	moved := &player.State{TrackID: "a", Playing: true, Volume: 40, Progress: time.Minute}
	tm, _ = tm.Update(msg.StateFetched{State: moved})
	if got := tm.(Model); got.restFor != restMost {
		t.Errorf("the playhead moving woke the poll up, leaving it at %s", got.restFor)
	}
}

// And it wakes the moment anything happens: something somebody did, or an answer
// that differs from the last.
func TestAnythingHappeningWakesThePoll(t *testing.T) {
	rested := func() tea.Model {
		m := New(player.NewMock(), nil, defaultTestCell)
		m.ps = &player.State{TrackID: "a", Playing: true}
		m.restFor = restMost

		var tm tea.Model = m
		return tm
	}

	// A key.
	tm, _ := rested().Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if got := tm.(Model); got.restFor != idlePoll {
		t.Errorf("a key left the poll resting at %s, want %s", got.restFor, idlePoll)
	}

	// A press of the pointer.
	tm, _ = rested().Update(tea.MouseClickMsg{X: 1, Y: 1, Button: tea.MouseLeft})
	if got := tm.(Model); got.restFor != idlePoll {
		t.Errorf("a press left the poll resting at %s, want %s", got.restFor, idlePoll)
	}

	// And somebody pressing play on another machine.
	tm, _ = rested().Update(msg.StateFetched{State: &player.State{TrackID: "b", Playing: true}})
	if got := tm.(Model); got.restFor != idlePoll {
		t.Errorf("a new track left the poll resting at %s, want %s", got.restFor, idlePoll)
	}
}

// What the rest actually saves, in the units that matter: requests against a
// quota, in a day of a window nobody is looking at.
//
// All three of them, because it was never one. A window left open with nothing
// playing asked after the state every three seconds and after the devices every
// three seconds — twice what anybody counted — and re-read whatever list was on
// screen twice a minute on top.
func TestTheRestIsWorthTheDay(t *testing.T) {
	day := 24 * time.Hour

	m := New(player.NewMock(), nil, defaultTestCell)
	quick := day/idlePoll + day/devicesEvery + day/staleAfter

	m.restFor = restMost
	rested := day/m.every(idlePoll) + day/m.every(devicesEvery) + day/m.every(staleAfter)

	if rested*10 > quick {
		t.Errorf("resting leaves %d requests a day against %d, want it under a tenth", rested, quick)
	}
	t.Logf("a quiet day: %d requests rather than %d", rested, quick)
}

// A list of devices is asked after while a list of devices is on screen, and
// not otherwise. Browsing the library with nothing playing asked Spotify what
// exists every three seconds, for a list nowhere on the screen.
func TestTheDevicesAreAskedForWhileTheyAreLookedAt(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.noDevice = true

	m.tab = tabPlayer
	if !m.devicesOnScreen() {
		t.Error("with nothing playing the player screen is the device list")
	}

	m.tab = tabLibrary
	if m.devicesOnScreen() {
		t.Error("the library was asking after devices it does not show")
	}

	// Opened over anything, it is on screen wherever you were.
	m.devices.open = true
	if !m.devicesOnScreen() {
		t.Error("the picker was open and nobody was keeping it current")
	}
}

// And the queue is not slowed, because the queue is free: it comes from
// spindle's own daemon over localhost while that is what is playing.
func TestTheDaemonsQueueIsNotSlowed(t *testing.T) {
	m := New(localQueue{Player: player.NewMock()}, nil, defaultTestCell)
	m.restFor = restMost

	if got := m.queueEvery(); got != queueStaleAfter {
		t.Errorf("the daemon's queue is asked for every %s at rest, want %s", got, queueStaleAfter)
	}
}

// localQueue is a backend whose queue comes from the daemon.
type localQueue struct{ player.Player }

func (localQueue) QueueIsLocal() bool { return true }
