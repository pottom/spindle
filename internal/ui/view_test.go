package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{-time.Second, "0:00"},
		{0, "0:00"},
		{900 * time.Millisecond, "0:01"},
		{time.Second, "0:01"},
		{9 * time.Second, "0:09"},
		{time.Minute, "1:00"},
		{5*time.Minute + 55*time.Second, "5:55"},
		{12*time.Minute + 3*time.Second, "12:03"},
		{time.Hour, "1:00:00"},
		{time.Hour + 2*time.Minute + 4*time.Second, "1:02:04"},
	}

	for _, c := range cases {
		if got := formatDuration(c.in); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The bitrate answers "what am I actually hearing", so it belongs next to the
// device — but only while something is arriving.
func TestStatusLineShowsTheBitrateWhilePlaying(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: true, Bitrate: 320}

	if got := m.statusLine(); !strings.Contains(got, "320 kbps") {
		t.Errorf("statusLine() = %q, want the bitrate in it", got)
	}

	m.ps.Playing = false
	if got := m.statusLine(); strings.Contains(got, "kbps") {
		t.Errorf("statusLine() = %q, want no bitrate while paused", got)
	}

	m.ps.Playing, m.ps.Bitrate = true, 0
	if got := m.statusLine(); strings.Contains(got, "kbps") {
		t.Errorf("statusLine() = %q, want no bitrate when unknown", got)
	}
}

// The mark beside the device is the only thing on screen that says sound is
// coming out right now, so it turns while playing and settles when it stops.
func TestDeviceMarkTurnsOnlyWhilePlaying(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: true}

	first := m.statusLine()
	m.device, _ = m.device.Update(m.device.Tick().(spinner.TickMsg))
	if second := m.statusLine(); second == first {
		t.Errorf("statusLine() = %q on two frames, want the mark to have moved", first)
	}

	m.ps.Playing = false
	if got := m.statusLine(); !strings.Contains(got, deviceDot) {
		t.Errorf("statusLine() = %q, want the plain dot once stopped", got)
	}
}

// Nothing may drive the mark while the music is stopped: it would redraw the
// screen eight times a second to say nothing.
func TestDeviceMarkDoesNotTickWhileStopped(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: false}

	if cmd := m.spinDevice(); cmd != nil {
		t.Error("spinDevice() started a tick loop while stopped")
	}

	m.ps.Playing = true
	if cmd := m.spinDevice(); cmd == nil {
		t.Fatal("spinDevice() = nil while playing, want a tick")
	}
	// And a second caller must not start a competing loop.
	if cmd := m.spinDevice(); cmd != nil {
		t.Error("spinDevice() started a second tick loop")
	}
}

// The top row carries both bearings: which machine is making the sound, and
// which screen you are on. The rule has to land under the active tab, or it
// points at the wrong one.
func TestHeaderPutsTheDeviceLeftAndTheTabsRight(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: "spindle", Playing: true, Bitrate: 320}
	m.tab = tabQueue

	rows := m.header(96)
	if len(rows) != 2 {
		t.Fatalf("header() = %d rows, want 2", len(rows))
	}

	top := ansiOff(rows[0])
	if !strings.HasPrefix(strings.TrimLeft(top, " "), "◐") && !strings.Contains(top, "spindle") {
		t.Errorf("header row = %q, want the device on the left", top)
	}
	if !strings.HasSuffix(strings.TrimRight(top, " "), "search") {
		t.Errorf("header row = %q, want the tabs flush right", top)
	}

	// The rule sits under the active tab, which is "queue".
	rule := ansiOff(rows[1])
	if got, want := column(rule, "━"), column(top, "queue"); got != want {
		t.Errorf("rule starts at column %d, want %d — under the active tab", got, want)
	}
}

// column is where a substring begins on screen. Counting bytes would be wrong
// the moment a line carries anything outside ASCII, which every one of them does.
func column(line, want string) int {
	at := strings.Index(line, want)
	if at < 0 {
		return -1
	}
	return lipgloss.Width(line[:at])
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func ansiOff(s string) string { return ansiPattern.ReplaceAllString(s, "") }

// A long device name must not push the tabs off the edge: the name is a detail,
// the tabs are how you move around.
func TestHeaderKeepsTheTabsWhole(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{DeviceName: strings.Repeat("very long name ", 6), Playing: true, Bitrate: 320}

	top := ansiOff(m.header(minWidth - leftMargin - rightMargin)[0])
	if !strings.HasSuffix(strings.TrimRight(top, " "), "search") {
		t.Errorf("header row = %q, want the tabs intact", top)
	}
	if !strings.Contains(top, "…") {
		t.Errorf("header row = %q, want the device name cut instead", top)
	}
}
