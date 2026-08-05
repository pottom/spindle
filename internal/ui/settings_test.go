package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/daemon"
)

// The screen says what is set, what each setting is for, and which of them the
// device has to be restarted to hear.
func TestTheSettingsScreenSaysWhatIsSet(t *testing.T) {
	m := New(nil, nil, defaultTestCell)
	m.tab = tabSettings
	m.width, m.height = 140, 40
	m.resize()
	m.settings.quality, m.settings.crossfade, m.settings.notify = daemon.QualityHigh, 6*time.Second, true

	page := plain(strings.Join(m.settingsPanel(m.layout(), m.layout().bodyHeight), "\n"))
	for _, want := range []string{"Sound quality", "high", "Crossfade", "6 seconds", "Track notifications", "on"} {
		if !strings.Contains(page, want) {
			t.Errorf("the settings screen does not say %q:\n%s", want, page)
		}
	}
}

// Turning a setting writes it down at once, and says the device has not heard
// it yet — which is the difference between a screen that can be trusted and a
// screen of switches that may or may not be connected to anything.
func TestTurningASettingSaysItNeedsARestart(t *testing.T) {
	m := New(nil, nil, defaultTestCell)
	m.tab = tabSettings
	m.width, m.height = 140, 40
	m.resize()
	m.settings.cursor.cursor = settingNotify

	if cmd := m.turnSetting(1); cmd == nil {
		t.Fatal("turning the notifications wrote nothing")
	}
	if !m.settings.notify || !m.settings.changed {
		t.Errorf("notify = %v, changed = %v; want both", m.settings.notify, m.settings.changed)
	}

	page := plain(strings.Join(m.settingsPanel(m.layout(), m.layout().bodyHeight), "\n"))
	if !strings.Contains(page, "Restart the device") {
		t.Error("nothing on the screen says the device has not heard the change")
	}
}

// The values cycle, and stop where the thing itself stops.
func TestTheValuesCycle(t *testing.T) {
	if got := turnQuality(daemon.QualityHigh, 1); got != daemon.QualityLow {
		t.Errorf("past the best quality came %q, want it to come round", got)
	}
	if got := turnQuality(daemon.QualityLow, -1); got != daemon.QualityHigh {
		t.Errorf("before the worst quality came %q, want it to come round", got)
	}

	if got := turnCrossfade(daemon.MaxCrossfade, 1); got != 0 {
		t.Errorf("past the longest overlap came %v, want none", got)
	}
	if got := turnCrossfade(0, -1); got != daemon.MaxCrossfade {
		t.Errorf("before no overlap came %v, want the longest", got)
	}
	if got := turnCrossfade(3*time.Second, 1); got != 4*time.Second {
		t.Errorf("a second past three seconds came out %v", got)
	}
}

// The artwork row is a fact about the terminal, not a choice.
func TestTheArtworkRowIsNotAChoice(t *testing.T) {
	m := New(nil, nil, defaultTestCell)
	m.settings.cursor.cursor = settingArtwork

	if cmd := m.turnSetting(1); cmd != nil {
		t.Error("the artwork row was turned into a setting")
	}
	if m.settings.changed {
		t.Error("reading a fact was recorded as a change")
	}
}

// The tab answers its own keys, and the digits still reach it.
func TestTheSettingsTabIsReachable(t *testing.T) {
	m := New(nil, nil, defaultTestCell)
	m.width, m.height = 140, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '5', Text: "5"})
	if got := tm.(Model).tab; got != tabSettings {
		t.Fatalf("5 landed on %v, want the settings", got)
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := tm.(Model).settings.cursor.cursor; got != 1 {
		t.Errorf("the cursor is on %d after one press, want the second setting", got)
	}
}
