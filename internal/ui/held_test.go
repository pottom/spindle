package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// A stopped record says so, in the type this screen already writes in.
//
// A figure holding a pause up was here before, and a badge is what it read as:
// one small drawing in the middle of a big picture, saying its one thing the
// same way every time. The word is the screen's own voice at the screen's own
// size.
func TestAStoppedRecordSaysSo(t *testing.T) {
	m := stageWords("a")
	m.setProgress(40 * time.Second)
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	if m.words.text == wordsHeld {
		t.Fatal("a playing record said it was paused")
	}

	m.ps.Playing = false
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	if m.words.text != wordsHeld {
		t.Errorf("a stopped record put up %q, want %q", m.words.text, wordsHeld)
	}
	if m.words.have.DotsX == 0 {
		t.Error("the word was chosen and never drawn")
	}
	if !m.wordsStill() {
		t.Error("the picture went on answering a record that is not playing")
	}

	// It is a line like any other, which is what gives it an arrival and a
	// departure. A bar of marks is neither.
	if m.words.beats {
		t.Error("the word was taken for a bar of marks")
	}
	if m.words.cast != "" {
		t.Errorf("the word was dealt the %q drawings", m.words.cast)
	}

	// Stopped beats silenced: a record nobody is playing is not playing to
	// anybody, silenced or not.
	m.toggleMute()
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	if m.words.text != wordsHeld {
		t.Errorf("stopped and silenced at once put up %q", m.words.text)
	}

	// And a device that has said nothing yet is not a stopped record.
	m.ps = &player.State{}
	if m.held() {
		t.Error("a status with no track in it read as stopped")
	}
}

// It arrives and leaves the way every other line does, and it is dealt its move
// like every other line: a picture that only ever does one thing is furniture.
func TestTheWordIsDealtAMoveLikeAnyOtherLine(t *testing.T) {
	seen := map[wordsMove]bool{}
	for _, at := range []time.Duration{9, 23, 41, 67, 88, 113, 140, 171} {
		m := stageWords("a")
		m.setProgress(at * time.Second)
		if cmd := m.wordsGrind(); cmd != nil {
			m.wordsTake(cmd)
		}
		m.ps.Playing = false
		if cmd := m.wordsGrind(); cmd != nil {
			m.wordsTake(cmd)
		}
		if m.words.text != wordsHeld {
			t.Fatalf("stopped at %s and put up %q", at, m.words.text)
		}
		seen[m.words.move] = true
	}
	if len(seen) < 2 {
		t.Errorf("stopped eight times and came in %d way(s); it is a badge again", len(seen))
	}
}

// The frames stop when the music does — silence should cost nothing — and
// stopping is itself a change of picture. Cut on the frame the sound sinks and
// the word is never drawn: what stays up is whatever the last frame of the
// music left, and the first resize wipes even that. Measured on screen.
func TestTheLoopWaitsForTheWordBeforeItStops(t *testing.T) {
	m := stageWords("a")
	m.setProgress(40 * time.Second)
	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	if !m.wordsSettled() {
		t.Fatal("a picture that is up and finished was called unsettled")
	}

	// The moment it stops, what is on screen is not what should be.
	m.ps.Playing = false
	if m.wordsSettled() {
		t.Fatal("the loop was free to stop before the word had been asked for")
	}

	if cmd := m.wordsGrind(); cmd != nil {
		m.wordsTake(cmd)
	}
	if m.wordsSettled() {
		t.Error("the loop was free to stop while the word was still gathering")
	}

	m.words.since = time.Now().Add(-time.Second)
	if !m.wordsSettled() {
		t.Error("the loop was held open after the word had arrived and stopped moving")
	}
}

// In the middle of the screen, at every size.
//
// A line is placed by the face's metrics, which reserve the whole ascent above
// the baseline whatever the letters actually reach. That is right for a lyric,
// where every line has to sit at the same height as the one before it, and
// wrong for a word that is alone: measured against the metrics this one sat two
// rows low at eighty by twenty-four and nearly four at a hundred and sixty by
// fifty, further the bigger the screen.
func TestTheWordSitsInTheMiddle(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 30}, {140, 40}, {160, 50}} {
		m := stageWords("a")
		m.width, m.height = size[0], size[1]
		m.resize()
		m.setProgress(40 * time.Second)
		m.ps.Playing = false
		if cmd := m.wordsGrind(); cmd != nil {
			m.wordsTake(cmd)
		}
		if m.words.text != wordsHeld {
			t.Fatalf("%dx%d put up %q", size[0], size[1], m.words.text)
		}

		g := m.words.have
		top, bottom := -1, -1
		for y := range g.DotsY {
			for x := range g.DotsX {
				if g.Lum[y*g.DotsX+x] > 0 {
					if top < 0 {
						top = y
					}
					bottom = y
					break
				}
			}
		}
		if top < 0 {
			t.Fatalf("%dx%d drew nothing", size[0], size[1])
		}

		// Half a dot for the rounding, and not a dot more: the whole point is
		// that it does not drift as the screen grows.
		off := float64(top+bottom)/2 - float64(g.DotsY-1)/2
		if off < -0.5 || off > 0.5 {
			t.Errorf("%dx%d: the word's ink runs %d..%d of %d dots, %+.1f off the middle",
				size[0], size[1], top, bottom, g.DotsY, off)
		}
	}
}

// And the word is one word, not a sentence: it is read at a glance across a
// whole terminal.
func TestTheWordIsOneWord(t *testing.T) {
	if strings.Fields(wordsHeld) == nil || len(strings.Fields(wordsHeld)) != 1 {
		t.Errorf("what a stopped record says is %q", wordsHeld)
	}
}
