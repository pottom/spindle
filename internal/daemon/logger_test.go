package daemon

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// held is a clock that does not move unless a test moves it.
type held struct {
	mu sync.Mutex
	at time.Time
}

func (h *held) now() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.at
}

func (h *held) set(at time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.at = at
}

func stamped(out *strings.Builder, clock *held) *logger {
	return &logger{sink: &sink{out: out, now: clock.now}}
}

// Every line says when. A log that answers "what happened" and refuses "when"
// costs an afternoon the first time somebody has to know whether a refusal in it
// was from an hour ago or from a week.
func TestEveryLineSaysWhen(t *testing.T) {
	var out strings.Builder
	clock := &held{at: time.Date(2026, 8, 16, 22, 4, 5, 0, time.Local)}
	log := stamped(&out, clock)

	log.Infof("loaded track %q", "Füttyös")
	log.WithError(errStub{}).Errorf("failed put state after update")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("wrote %d lines, want the day and two entries:\n%s", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "──── 2026-08-16") {
		t.Errorf("the first line is %q, want the day it began", lines[0])
	}
	for _, line := range lines[1:] {
		if !strings.HasPrefix(line, "22:04:05 ") {
			t.Errorf("line %q does not begin with the time", line)
		}
	}
	if !strings.Contains(lines[2], `error="no"`) {
		t.Errorf("the fields did not survive the stamp: %q", lines[2])
	}
}

// A daemon left running overnight is the normal way this one lives, so the day
// changing is said once rather than stamped on every line after it.
func TestANewDayIsAnnouncedOnce(t *testing.T) {
	var out strings.Builder
	clock := &held{at: time.Date(2026, 8, 16, 23, 59, 59, 0, time.Local)}
	log := stamped(&out, clock)

	log.Info("before midnight")
	clock.set(time.Date(2026, 8, 17, 0, 0, 1, 0, time.Local))
	log.Info("after")
	log.Info("and after that")

	got := out.String()
	if strings.Count(got, "──── ") != 2 {
		t.Errorf("the days are announced %d times, want two:\n%s", strings.Count(got, "──── "), got)
	}
	if !strings.Contains(got, "──── 2026-08-17") {
		t.Errorf("the new day was never announced:\n%s", got)
	}
}

// Everything derived from the first logger writes to the same file and shares
// what is known about it, so two goroutines cannot announce the same day twice.
func TestDerivedLoggersShareTheDay(t *testing.T) {
	var out strings.Builder
	clock := &held{at: time.Date(2026, 8, 16, 12, 0, 0, 0, time.Local)}
	log := stamped(&out, clock)

	log.Info("first")
	log.WithField("uri", "spotify:track:1").Info("second")
	log.WithError(errStub{}).WithField("n", 2).Warn("third")

	if n := strings.Count(out.String(), "──── "); n != 1 {
		t.Errorf("the day was announced %d times, want once:\n%s", n, out.String())
	}
}

type errStub struct{}

func (errStub) Error() string { return "no" }
