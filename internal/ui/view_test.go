package ui

import (
	"strings"
	"testing"
	"time"

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
