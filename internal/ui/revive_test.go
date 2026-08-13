package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// deadDevice is a backend whose local daemon is not answering.
type deadDevice struct {
	player.Player
	live bool
}

func (d deadDevice) Live() bool { return d.live }

// A device that has gone is started again — and not on every tick, which would
// be a daemon spawned thirty times a second while the network is out.
func TestAMissingDeviceIsStartedAgain(t *testing.T) {
	m := New(deadDevice{Player: player.NewMock()}, nil, defaultTestCell)

	if m.revive() == nil {
		t.Fatal("the device is gone and nothing went to start one")
	}
	if m.revive() != nil {
		t.Error("a second attempt followed immediately, with no wait between")
	}

	// Long enough afterwards, it is worth another go: the first attempt may
	// have failed because the machine had no network at all.
	m.deviceLostAt = time.Now().Add(-reviveEvery - time.Second)
	if m.revive() == nil {
		t.Error("nothing tried again after the interval had passed")
	}
}

// A device that is answering is left alone, and the clock that paces the
// attempts is cleared — so a device that goes twice is not made to wait out the
// first outage's interval before the second is noticed.
func TestALiveDeviceIsLeftAlone(t *testing.T) {
	m := New(deadDevice{Player: player.NewMock(), live: true}, nil, defaultTestCell)
	m.deviceLostAt = time.Now()

	if m.revive() != nil {
		t.Error("a daemon was started for a device that is answering")
	}
	if !m.deviceLostAt.IsZero() {
		t.Error("the device came back and the attempt clock was left running")
	}
}

// A backend that has no local device at all — the mock, or the Web API on its
// own — is not something to start daemons for.
func TestABackendWithNoDeviceIsNotRevived(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	if m.revive() != nil {
		t.Error("the mock backend was taken for a daemon that had died")
	}
}
