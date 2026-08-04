package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

func peekModel() Model {
	m := lyricsModel(120, 44)
	for i, n := range []string{"first", "second", "third", "fourth", "fifth", "sixth"} {
		m.queue = append(m.queue, player.Track{ID: fmt.Sprint(i), Title: n,
			Artists: []string{"someone"}, Duration: time.Duration(180+i*9) * time.Second})
	}
	return m
}

// The glance goes into the band above the artwork, which was blank. Nothing
// below it moves — the same promise the words and the trace make.
func TestPeekMovesNothing(t *testing.T) {
	m := peekModel()
	m.peek.on = false
	off := strings.Split(plain(m.render()), "\n")

	m.peek.on = true
	on := strings.Split(plain(m.render()), "\n")
	if !m.peekVisible() {
		t.Fatal("the glance is not showing")
	}

	// Only the band itself may differ.
	first := tabBarHeight
	for i := range off {
		if i >= first && i < first+peekRows+peekChrome {
			continue
		}
		if off[i] != on[i] {
			t.Errorf("row %d moved\n  off: %q\n  on:  %q", i, off[i], on[i])
		}
	}
}

// It shows what is coming, in order, and no more than a glance's worth.
func TestPeekShowsWhatIsNext(t *testing.T) {
	m := peekModel()
	m.peek.on = true

	out := plain(m.render())
	for _, want := range []string{"Up next", "first", "second", "third", "fourth"} {
		if !strings.Contains(out, want) {
			t.Errorf("the glance does not show %q", want)
		}
	}
	if strings.Contains(out, "fifth") {
		t.Errorf("the glance shows more than %d tracks", peekRows)
	}
}

// It is off to begin with, and the key is what brings it out.
func TestPeekKey(t *testing.T) {
	m := peekModel()
	if m.peek.on {
		t.Error("the glance is on before being asked for")
	}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if !tm.(Model).peekVisible() {
		t.Fatal("u did not bring out the glance")
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if tm.(Model).peekVisible() {
		t.Error("u did not put it away")
	}
}

// Where the band above the artwork is too shallow the glance is not offered,
// and the key says so by doing nothing rather than by rearranging the screen.
func TestPeekNeedsTheBand(t *testing.T) {
	m := lyricsModel(100, minHeight+4)
	if m.peekAvailable() {
		t.Skip("this terminal has room after all")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if cmd != nil || tm.(Model).peekVisible() {
		t.Error("the glance appeared with no band to put it in")
	}
}

// The glance has no cursor and never reaches double figures, so it gives up
// both the column the cursor would stand in and the second digit of the
// ordinal: without that the whole list sits indented from its own heading.
func TestPeekRowsAreFlushWithTheHeading(t *testing.T) {
	m := peekModel()
	m.peek.on = true

	var heading, first int = -1, -1
	for _, row := range strings.Split(plain(m.render()), "\n") {
		switch {
		case strings.Contains(row, "Up next"):
			heading = strings.Index(row, "Up next")
		case first < 0 && strings.Contains(row, "first"):
			first = strings.Index(row, "1")
		}
	}
	if heading < 0 || first < 0 {
		t.Fatal("could not find the heading and the first row")
	}
	if heading != first {
		t.Errorf("the heading starts at column %d and the first row at %d, want them flush", heading, first)
	}
}
