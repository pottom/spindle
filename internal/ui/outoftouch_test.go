package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// A device that has lost its connection plays out what it has already fetched
// and answers everything asked of it, so the screen goes on looking right while
// nothing new can be started. It is a state that lasts now — the daemon never
// stops trying — which is exactly why it has to be said.
func TestAnOutageSaysSoOnScreen(t *testing.T) {
	m := playerModel()
	m.width, m.height = 120, 40
	m.resize()

	if strings.Contains(ansi.Strip(m.render()), "Out of touch") {
		t.Fatal("a connected device was reported as out of touch")
	}

	m.ps.OutOfTouch = map[string]time.Duration{"accesspoint": 5 * time.Minute}
	screen := ansi.Strip(m.render())
	if !strings.Contains(screen, "Out of touch with Spotify") {
		t.Errorf("an outage says nothing:\n%s", screen)
	}
	if !strings.Contains(screen, "5m") {
		t.Error("the line does not say how long it has been going on")
	}
}

// Dozens of reconnections a day land in under two seconds. None of them is news,
// and a warning that flashes up for every one of them teaches people to ignore
// the line.
func TestAShortHiccupIsNotWorthSaying(t *testing.T) {
	m := playerModel()
	m.ps.OutOfTouch = map[string]time.Duration{"accesspoint": 3 * time.Second}

	if got, _, ok := m.notice(); ok {
		t.Errorf("a three-second gap said %q", got)
	}
}

// The two connections are worth different words: one carries the audio, the
// other only the remote control.
func TestOnlyTheAudioConnectionPutsTheSoundAtStake(t *testing.T) {
	m := playerModel()
	m.ps.OutOfTouch = map[string]time.Duration{"dealer": 12 * time.Minute}

	got, _, ok := m.notice()
	if !ok {
		t.Fatal("said nothing about a dealer that has been gone twelve minutes")
	}
	if !strings.Contains(got, "Connect's reach") {
		t.Errorf("the notice says %q, want it to be about Connect", got)
	}
	if m.soundAtStake() {
		t.Error("called the sound at stake when only the remote control was gone")
	}

	m.ps.OutOfTouch["accesspoint"] = 30 * time.Second
	if got, _, _ := m.notice(); !strings.Contains(got, "playing what is here") {
		t.Errorf("the notice says %q, want it to be about the audio", got)
	}
	if !m.soundAtStake() {
		t.Error("the audio connection was gone and the sound was called safe")
	}
}

// It outranks what it causes. With the network gone the Web API fails too, and
// whatever the transport said about a socket says less than this does.
func TestAnOutageOutranksTheErrorsItCauses(t *testing.T) {
	m := playerModel()
	m.err = errors.New("dial tcp: no such host")
	m.ps.OutOfTouch = map[string]time.Duration{"accesspoint": time.Minute}

	if got, _, _ := m.notice(); !strings.Contains(got, "Out of touch") {
		t.Errorf("the notice says %q, want the outage behind it", got)
	}
}

// The longest gone is the one worth naming: two connections lost are one outage.
func TestTheLongestOutageIsTheOneNamed(t *testing.T) {
	m := playerModel()
	m.ps.OutOfTouch = map[string]time.Duration{"dealer": time.Hour, "accesspoint": time.Minute}

	if got := m.outOfTouch(); got != time.Hour {
		t.Errorf("reported %s out of touch, want the hour", got)
	}
}

func TestHowLongIsSpelledTheWayItIsSaid(t *testing.T) {
	for _, c := range []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m"},
		{59 * time.Minute, "59m"},
		{time.Hour, "1h"},
		{2*time.Hour + 4*time.Minute, "2h 4m"},
		{25 * time.Hour, "25h"},
	} {
		if got := howLong(c.in); got != c.want {
			t.Errorf("howLong(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}
