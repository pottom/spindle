package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestClock(t *testing.T) {
	for _, c := range []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00"},
		{9 * time.Second, "0:09"},
		{94 * time.Second, "1:34"},
		{-time.Second, "0:00"},
		// A podcast episode: hours only appear when there are any.
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
	} {
		if got := clock(c.in); got != c.want {
			t.Errorf("clock(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The shape a status bar reads: one labelled field per line, no colour, and the
// position and duration in a form a person recognises.
func TestStatusShape(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)

	code, out, errOut := s.run("status")
	if code != exitOK {
		t.Fatalf("exit = %d (%s)", code, errOut)
	}

	want := strings.Join([]string{
		"state:    playing",
		"title:    Sultans of Swing",
		"artist:   Dire Straits",
		"album:    Dire Straits",
		"position: 1:34",
		"duration: 5:48",
		"volume:   50",
		"device:   spindle",
		"",
	}, "\n")
	if out != want {
		t.Errorf("status printed\n%s\nwant\n%s", out, want)
	}
	if strings.Contains(out, "\x1b") {
		t.Error("status printed an escape sequence; this is not the interface")
	}
}

func TestStatusSaysWhenPaused(t *testing.T) {
	s := newStub(t, pausedStatus, sampleQueue)

	_, out, _ := s.run("status")
	if !strings.HasPrefix(out, "state:    paused\n") {
		t.Errorf("status printed %q, want it to open with the paused state", out)
	}
}

// --json hands over what the daemon said, untouched, so jq can reach the fields
// the plain output leaves out.
func TestStatusJSONIsTheDaemonsOwn(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)

	code, out, errOut := s.run("status", jsonFlag)
	if code != exitOK {
		t.Fatalf("exit = %d (%s)", code, errOut)
	}
	if strings.TrimSpace(out) != playingStatus {
		t.Errorf("status --json printed %q, want the daemon's document", out)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("status --json is not json: %v", err)
	}
}

// Stopped still has an answer worth handing to jq, even though the plain output
// has nothing to say.
func TestStatusJSONWhileStopped(t *testing.T) {
	s := newStub(t, stoppedStatus, emptyQueue)

	code, out, _ := s.run("status", jsonFlag)
	if code != exitIdle {
		t.Errorf("exit = %d, want %d", code, exitIdle)
	}
	if strings.TrimSpace(out) != stoppedStatus {
		t.Errorf("status --json printed %q, want the daemon's document", out)
	}
}

func TestQueueShape(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)

	code, out, errOut := s.run("queue")
	if code != exitOK {
		t.Fatalf("exit = %d (%s)", code, errOut)
	}

	want := strings.Join([]string{
		"now  Sultans of Swing — Dire Straits  5:48",
		"1    Romeo and Juliet — Dire Straits  6:00",
		"",
	}, "\n")
	if out != want {
		t.Errorf("queue printed\n%s\nwant\n%s", out, want)
	}
}

func TestCommandJSON(t *testing.T) {
	s := newStub(t, playingStatus, sampleQueue)
	if _, out, _ := s.run("play", jsonFlag); strings.TrimSpace(out) != `{"command":"play"}` {
		t.Errorf("play --json printed %q", out)
	}

	s = newStub(t, playingStatus, sampleQueue)
	if _, out, _ := s.run("volume", "25", jsonFlag); strings.TrimSpace(out) != `{"volume":25}` {
		t.Errorf("volume --json printed %q", out)
	}

	s = newStub(t, playingStatus, sampleQueue)
	if _, out, _ := s.run("seek", "+30", jsonFlag); strings.TrimSpace(out) != `{"position":124000}` {
		t.Errorf("seek --json printed %q", out)
	}
}
