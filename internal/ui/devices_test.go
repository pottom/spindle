package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func devices(active string, names ...string) []player.Device {
	out := make([]player.Device, 0, len(names))
	for _, n := range names {
		out = append(out, player.Device{ID: n, Name: n, Type: "computer", Active: n == active})
	}
	return out
}

// Devices come back from Spotify in whatever order it feels like. A cursor that
// stays on an index rather than on a device is how you transfer to the wrong
// speaker.
func TestDeviceCursorFollowsTheDevice(t *testing.T) {
	var p devicePane
	p.adopt(devices("", "laptop", "phone", "speaker"))
	p.cursor.move(2, 3) // on "speaker"

	p.adopt(devices("", "phone", "speaker", "laptop"))

	if sel := p.selected(); sel == nil || sel.ID != "speaker" {
		t.Errorf("cursor landed on %v, want speaker", sel)
	}
}

// With nothing to hold on to, start where playback already is.
func TestDeviceCursorStartsOnTheActiveOne(t *testing.T) {
	var p devicePane
	p.adopt(devices("phone", "laptop", "phone", "speaker"))

	if sel := p.selected(); sel == nil || sel.ID != "phone" {
		t.Errorf("cursor started on %v, want the active device", sel)
	}
}

func TestDeviceCursorSurvivesADeviceVanishing(t *testing.T) {
	var p devicePane
	p.adopt(devices("", "laptop", "phone", "speaker"))
	p.cursor.move(2, 3)

	p.adopt(devices("", "laptop"))

	if sel := p.selected(); sel == nil || sel.ID != "laptop" {
		t.Errorf("cursor landed on %v, want laptop", sel)
	}
}

func TestDeviceSelectionOnEmptyList(t *testing.T) {
	var p devicePane
	p.adopt(nil)
	if sel := p.selected(); sel != nil {
		t.Errorf("selected = %v, want nil", sel)
	}
}

// Opening and closing the picker must not disturb what is playing.
func TestPickerTogglesWithD(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Title: "something"}
	m.devices.adopt(devices("laptop", "laptop", "phone"))

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !tm.(Model).devices.open {
		t.Fatal("d did not open the picker")
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if tm.(Model).devices.open {
		t.Error("esc did not close the picker")
	}
}

// The no-device screen is the same list with nothing behind it, so esc has
// nothing to close and must not make it disappear.
func TestNoDeviceScreenCannotBeClosed(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	var tm tea.Model = m
	tm, _ = tm.Update(msg.NoActiveDevice{})
	tm, _ = tm.Update(msg.DevicesFetched{Devices: devices("", "laptop", "phone")})

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !tm.(Model).noDevice {
		t.Error("esc dismissed a screen with nothing behind it")
	}

	// The arrows still have to work: this screen is where you choose.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if sel := tm.(Model).devices.selected(); sel == nil || sel.ID != "phone" {
		t.Errorf("cursor on %v, want phone", sel)
	}
}
