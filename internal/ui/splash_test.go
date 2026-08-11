package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

func splashModel(t *testing.T, w, rows int) Model {
	t.Helper()
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.width, m.height = w, rows
	m.tab = tabPlayer
	m.noDevice = true
	return m
}

// The picture is up while the device is being waited for, and gone when it is
// not.
//
// The wait is real and cannot be argued away — the interface draws in a quarter
// of a second and the device answers a second later, or twenty after the machine
// has been asleep — so the screen that says "no device yet" is given the logo
// rather than left blank above a list.
func TestTheLogoFillsTheWait(t *testing.T) {
	m := splashModel(t, 120, 40)
	m.splashFlow()
	if len(m.splashRows()) == 0 {
		t.Fatal("nothing was drawn while the device was being waited for")
	}
	w, rows := m.splashRoom()
	if got := len(m.splashRows()); got > rows+1 {
		t.Errorf("the logo came out %d rows in a box of %d", got, rows)
	}
	for _, line := range m.splashRows() {
		if n := lipgloss.Width(line); n > w {
			t.Errorf("a row of the logo is %d cells wide in a box of %d", n, w)
		}
	}

	// The device arrives: it goes, and it goes properly rather than being left
	// in the terminal for whatever comes next to draw over.
	m.noDevice = false
	m.splashFlow()
	if len(m.splashRows()) != 0 {
		t.Error("the logo stayed after the device arrived")
	}

	// And somebody choosing a device is past looking at a logo.
	m.noDevice, m.devices.open = true, true
	m.splashFlow()
	if len(m.splashRows()) != 0 {
		t.Error("the logo was up over the device picker")
	}
}

// A terminal with no room for it gets the words instead.
func TestTheLogoKnowsWhenThereIsNoRoom(t *testing.T) {
	m := splashModel(t, 30, 12)
	m.splashFlow()
	if len(m.splashRows()) != 0 {
		t.Error("the logo was drawn into a box too small to read it in")
	}
	// The panel still says what it has to say.
	panel := strings.Join(m.noDevicePanel(m.layout(), 10), "\n")
	if !strings.Contains(panel, "No active playback device") {
		t.Error("the panel lost its words")
	}
}
