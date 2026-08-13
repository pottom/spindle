package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/daemon"
)

// deviceRevived says a fresh daemon was started in place of one that had gone.
type deviceRevived struct{}

// Bringing the device back, which is the other half of the daemon's watchdog.
//
// The daemon ends itself when its playback loop stops answering — a wifi that
// drops can wedge it, and a wedged one never comes back on its own; see
// internal/daemon/watchdog.go. Ending it is only half an answer, though. The
// interface goes on working, because everything falls back to the Web API, and
// the device simply is not there any more: no local queue, no waveform, no
// words, and nothing on screen saying why.
//
// So the interface watches for it and starts one. It is the only part of the
// program that is still running when the daemon is not, and starting one costs
// nothing when there is already one there — see daemon.Spawn, which takes the
// lock or leaves.

// reviveEvery is how often a missing device is worth another attempt. Half a
// minute: long enough that a laptop with no network is not spawning daemons in
// a loop, short enough that somebody who plugged the cable back in is playing
// again before they wonder.
const reviveEvery = 30 * time.Second

// revive starts a daemon when the local one has gone, and no more often than
// reviveEvery.
func (m *Model) revive() tea.Cmd {
	live, ok := m.player.(interface{ Live() bool })
	if !ok || live.Live() {
		m.deviceLostAt = time.Time{}
		return nil
	}

	now := time.Now()
	if !m.deviceLostAt.IsZero() && now.Sub(m.deviceLostAt) < reviveEvery {
		return nil
	}
	m.deviceLostAt = now

	return func() tea.Msg {
		if _, started, err := daemon.Spawn(); err != nil || !started {
			// Already there, or not startable: either way there is nothing to
			// say. The next tick will look again.
			return nil
		}
		return deviceRevived{}
	}
}
