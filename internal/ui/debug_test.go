package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

// The bar takes rows from the picture and never adds any.
//
// It is written over the top of what was drawn rather than above it, so that a
// picture looked at with the numbers on is the picture drawn with them off. A
// bar that pushed the screen down a row would change every measurement made
// while it was up, which is the one thing a measuring instrument may not do.
func TestTheBarChangesNothingItIsMeasuring(t *testing.T) {
	m := stageWords("a")
	m.setProgress(30 * time.Second)
	screen := m.render()
	rows, wide := strings.Count(screen, "\n"), lipgloss.Width(strings.SplitN(screen, "\n", 2)[0])

	for level := range debugDepths {
		m.debug.level = level
		out := m.debugOver(screen)
		if got := strings.Count(out, "\n"); got != rows {
			t.Errorf("at depth %d the screen went from %d rows to %d", level, rows, got)
		}
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > wide {
				t.Errorf("at depth %d row %d ran to %d cells, past the %d there are", level, i, got, wide)
			}
		}
	}
}

// Off is off, and the key walks back round to it.
func TestTheKeyWalksThroughAndBackToNothing(t *testing.T) {
	m := stageWords("b")
	if m.debug.level != debugOff {
		t.Fatal("the numbers were up before anybody asked for them")
	}
	if rows := m.debugRows(); rows != nil {
		t.Errorf("something was drawn while the bar was off: %q", rows)
	}

	for want := 1; want <= debugDepths; want++ {
		if !m.debugKey("ctrl+shift+d") {
			t.Fatal("the key was not taken")
		}
		if got, expect := m.debug.level, want%debugDepths; got != expect {
			t.Fatalf("press %d left the bar at depth %d, not %d", want, got, expect)
		}
	}
	if m.debugKey("d") || m.debugKey("ctrl+d") {
		t.Error("a key that is not the toggle was swallowed by it")
	}
}

// Every reading on the block has something in it, on a model that is playing.
//
// Not what the numbers say — that is what the rest of the suite is for — but
// that each row was actually filled. A row that quietly comes back empty is a
// row nobody notices is missing until they need it at two in the morning.
func TestEveryRowSaysSomething(t *testing.T) {
	m := stageWords("c")
	m.setProgress(45 * time.Second)
	m.debug.level = debugFull

	rows := m.debugRows()
	if len(rows) < 8 {
		t.Fatalf("the block came back with %d rows", len(rows))
	}
	for i, row := range rows {
		if len(strings.Fields(row)) < 2 {
			t.Errorf("row %d says nothing but its own name: %q", i, row)
		}
	}
}

// What the bar says goes to a file as well, and goes with the session.
//
// The screen is in front of whoever is listening; everybody else is told what
// the numbers said rather than reading them. So it is written down — and
// deleted as spindle closes, because it is a page of working rather than a log.
func TestTheBarIsReadableOffTheDiskAndOnlyForNow(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	debugPen.began, debugPen.was, debugPen.at, debugPen.off = false, "", time.Time{}, false

	path, err := debugPath()
	if err != nil {
		t.Fatalf("no state directory: %v", err)
	}
	lines := func() int {
		raw, err := os.ReadFile(path)
		if err != nil {
			return 0
		}
		return len(strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(raw)), "\n", " \n ")))
	}

	m := stageWords("f")
	m.lyrics.forTrack, m.lyrics.missing = "f", true
	m.setProgress(30 * time.Second)
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	m.debugNote()
	if _, err := os.Stat(path); err == nil {
		t.Error("the bar was off and it wrote anyway")
	}

	m.debug.level = debugFull
	m.debugNote()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the bar was up and nothing was written: %v", err)
	}
	if !strings.Contains(string(raw), "\"words\"") || !strings.Contains(string(raw), "move") {
		t.Errorf("the deal is missing from what was written: %s", raw)
	}
	was := lines()

	// Nothing has happened, and it is not a second later.
	m.debugNote()
	if got := lines(); got != was {
		t.Errorf("a frame where nothing changed was written down anyway (%d fields, was %d)", got, was)
	}

	// Something has.
	m.words.move = (m.words.move + 1) % wordsMoves
	m.debugNote()
	if got := lines(); got <= was {
		t.Error("the deal changed and nothing was written down")
	}

	ForgetDebug()
	if _, err := os.Stat(path); err == nil {
		t.Error("the session ended and its working was left behind")
	}
}
