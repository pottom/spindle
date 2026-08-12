package ui

import (
	"strings"
	"testing"
	"time"
)

// The screen says when what it is showing is old news.
//
// A stuck daemon goes on answering, so nothing on the screen looks wrong: the
// track, the artist, the cover and the volume are all still there, and all of
// them were true a while ago. It outranks the other notices because it is about
// them — saying anything else first would be saying it about the past.
func TestTheScreenSaysWhenThePlayerHasStoppedResponding(t *testing.T) {
	m := scopeModel(120, 44)
	m.ps.TrackID, m.ps.Duration, m.ps.Playing = "one", 3*time.Minute, true

	if _, _, ok := m.notice(); ok {
		t.Fatal("something was already being said")
	}

	m.ps.Stalled = true
	line, _, ok := m.notice()
	if !ok {
		t.Fatal("nothing was said about a daemon that had stopped responding")
	}
	if !strings.Contains(line, "last said") {
		t.Errorf("the line does not say the news is old: %q", line)
	}

	// It outranks the ones that are about the moment, because it is about
	// whether there is a moment to speak of.
	m.ps.Playing, m.queue = false, nil
	m.setProgress(0)
	if !m.ranOut() {
		t.Fatal("the list did not read as run out, so the ranking is untested")
	}
	if line, _, _ := m.notice(); !strings.Contains(line, "last said") {
		t.Errorf("a notice about the past was outranked by one about the present: %q", line)
	}

	// And it goes when the daemon comes back.
	m.ps.Stalled = false
	if line, _, _ := m.notice(); strings.Contains(line, "last said") {
		t.Error("it went on saying the news was old after the daemon came back")
	}
}
