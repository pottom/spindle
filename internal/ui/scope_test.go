package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func scopeModel(w, h int) Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "playing", Artists: []string{"someone"}, Playing: true, DeviceName: "spindle"}
	m.width, m.height = w, h
	m.resize()
	return m
}

// The trace is off until asked for, and the key both shows it and starts the
// loop that moves it.
func TestScopeKeyTogglesTheTrace(t *testing.T) {
	m := scopeModel(100, 40)
	if strings.Contains(plain(m.render()), "⠀") {
		t.Fatal("the trace was drawn before it was asked for")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if cmd == nil {
		t.Fatal("v produced no command, so nothing would move the trace")
	}

	on := tm.(Model)
	if !on.scopeVisible() || !on.scope.running {
		t.Fatalf("scope visible=%v running=%v, want both", on.scopeVisible(), on.scope.running)
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if tm.(Model).scopeVisible() {
		t.Error("the second press did not put the trace away")
	}
}

// Turning the trace on must not move anything: it goes into rows that were
// already blank. A visualiser is not worth making the cover jump.
func TestScopeMovesNothing(t *testing.T) {
	for _, size := range [][2]int{{100, 44}, {100, 40}, {120, 50}} {
		off := scopeModel(size[0], size[1])
		if !off.scopeAvailable() {
			t.Fatalf("%dx%d: no room for the trace, so there is nothing to test", size[0], size[1])
		}
		before := plain(off.render())

		on := off
		on.scope.on = true
		after := plain(on.render())

		b := strings.Split(before, "\n")
		a := strings.Split(after, "\n")
		if len(a) != len(b) {
			t.Fatalf("%dx%d: %d rows with the trace, %d without", size[0], size[1], len(a), len(b))
		}
		// Exactly the trace's own rows may differ; everything else has to be
		// character for character what it was.
		first := tabBarHeight + off.scopeTop(off.layout()) + scopeChrome
		for i := range b {
			if i >= first && i < first+scopeRows {
				continue
			}
			if a[i] != b[i] {
				t.Errorf("%dx%d: row %d moved\n  off: %q\n  on:  %q", size[0], size[1], i, b[i], a[i])
			}
		}
		// And the artwork keeps its size, or the cover would be re-rendered.
		if on.layout().artHeight != off.layout().artHeight {
			t.Errorf("%dx%d: the artwork changed size", size[0], size[1])
		}
	}
}

// Where there are not enough blank rows the trace is not offered, and the key
// says so by doing nothing rather than by rearranging the screen.
func TestScopeIsNotOfferedWithoutRoom(t *testing.T) {
	m := scopeModel(80, minHeight+2)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if cmd != nil {
		t.Error("v started the trace on a terminal too short for it")
	}
	if tm.(Model).scopeVisible() {
		t.Error("the trace is visible on a terminal too short for it")
	}
}

// Thirty redraws a second is the whole cost of the feature, so it has to stop
// the moment the trace leaves the screen.
func TestScopeStopsWhenItLeavesTheScreen(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.on, m.scope.running = true, true

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '2', Text: "2"}) // to the queue tab
	if got := tm.(Model); got.scopeVisible() {
		t.Fatal("the trace survived the move to another tab")
	}

	// The tick already in flight has to be the last one.
	tm, cmd := tm.Update(msgScopeTick())
	if cmd != nil {
		t.Error("the trace scheduled another frame after leaving the screen")
	}
	if tm.(Model).scope.running {
		t.Error("the loop still reports itself as running")
	}
}

// The trace is a line, not a scatter: consecutive samples are joined, so a steep
// slope stays continuous instead of breaking into separate dots.
func TestScopeDrawsAContinuousLine(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.on = true
	m.resize()

	// A step from bottom to top inside two dots: every row between has to be lit.
	m.scope.frame = []float32{-1, -1, 1, 1}
	m.scope.follow(m.scope.frame)
	lines := m.scopeLines(4)
	if len(lines) != scopeRows {
		t.Fatalf("scopeLines = %d rows, want %d", len(lines), scopeRows)
	}
	for i, line := range lines {
		if strings.TrimSpace(plain(line)) == "" {
			t.Errorf("row %d is blank, so the line broke where it rose", i)
		}
	}

	// Silence rests on the centre line, and only there.
	m.scope.frame = []float32{0, 0, 0, 0}
	m.scope.follow(m.scope.frame)
	lit := 0
	for _, line := range m.scopeLines(4) {
		if strings.TrimSpace(plain(line)) != "" {
			lit++
		}
	}
	if lit != 1 {
		t.Errorf("silence lit %d rows, want 1 — the centre line", lit)
	}
}

// With no backend to ask, the trace rests flat rather than failing: there is
// nothing to explain to anyone.
func TestScopeWithoutASourceRestsFlat(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.on = true
	m.resize()
	m.scope.frame = nil

	if got := m.scopeLines(20); len(got) != scopeRows {
		t.Fatalf("scopeLines = %d rows, want %d", len(got), scopeRows)
	}
}

func msgScopeTick() tea.Msg { return msg.WaveformReady{} }

// Measured against a live stream, peaks within one track ran from 0.06 to 0.87.
// At a fixed scale that is a flat line for half the track and a clipped one for
// the rest, so the trace follows the recent loudness instead.
func TestScopeFollowsTheLoudness(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.on = true
	m.resize()

	quiet := []float32{0.06, -0.06, 0.05, -0.05}
	loud := []float32{0.87, -0.87, 0.8, -0.8}

	// A quiet passage still reaches most of the way up.
	for range 40 {
		m.scope.follow(quiet)
	}
	m.scope.frame = quiet
	if got := m.scopeSample(0, 4); got < 0.7 {
		t.Errorf("quiet passage scaled to %.2f, want most of the deflection", got)
	}

	// A sudden hit louder than anything before is drawn at the edge, not past it.
	m.scope.follow(loud)
	m.scope.frame = loud
	if got := m.scopeSample(0, 4); got > 1 || got < 0.9 {
		t.Errorf("loud hit scaled to %.2f, want it pinned at the edge", got)
	}

	// And the gain does not run away in silence, where only noise is left.
	for range 500 {
		m.scope.follow([]float32{0, 0, 0, 0})
	}
	if m.scope.envelope < scopeFloor {
		t.Errorf("envelope fell to %.3f in silence, want it held at %.2f", m.scope.envelope, scopeFloor)
	}
}

// helpHeight has to ask for the help without knowing whether the waveform key
// is offered, because the layout decides that and the layout needs the height.
// That is only safe while the bar is the same height either way.
func TestHelpHeightDoesNotDependOnTheScope(t *testing.T) {
	for _, full := range []bool{false, true} {
		m := scopeModel(100, 44)
		m.help.ShowAll = full

		with := lipgloss.Height(m.help.View(m.keys.forPlayer(true)))
		without := lipgloss.Height(m.help.View(m.keys.forPlayer(false)))
		if with != without {
			t.Errorf("ShowAll=%v: help is %d rows with the waveform key and %d without", full, with, without)
		}
	}
}

// The trace stops whenever it leaves the screen, and nothing else would start
// it again: leaving for a playlist and coming back used to kill it until it was
// switched off and on by hand.
func TestScopeResumesOnItsOwn(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.on = true

	var tm tea.Model = m
	tm, _ = tm.Update(msg.WaveformReady{Samples: []float32{0.5, -0.5}})

	// Away to another tab: the frame in flight is the last one.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	tm, cmd := tm.Update(msg.WaveformReady{})
	if cmd != nil || tm.(Model).scope.running {
		t.Fatal("the trace kept running after leaving the player")
	}

	// Back again, and it picks itself up without being asked.
	tm, cmd = tm.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if cmd == nil {
		t.Fatal("returning to the player did not restart the trace")
	}
	if !tm.(Model).scope.running {
		t.Error("the trace is not running after returning to the player")
	}
}

// A device arriving, or the first state landing, also leaves the trace stopped.
// The one-second tick is the safety net for everything the tab switch misses.
func TestScopeResumesOnTheTick(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.on = true
	m.noDevice = true

	var tm tea.Model = m
	tm, _ = tm.Update(msg.WaveformReady{})
	if tm.(Model).scope.running {
		t.Fatal("the trace ran with no device")
	}

	got := tm.(Model)
	got.noDevice = false
	tm, cmd := tea.Model(got).Update(msg.Tick{})
	if cmd == nil {
		t.Fatal("the tick produced no command")
	}
	if !tm.(Model).scope.running {
		t.Error("the tick did not pick the trace back up")
	}
}
