package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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

// The rows come out of the body, and the screen stays exactly as tall as the
// terminal — a visualiser that pushes the help bar off the bottom is worse than
// no visualiser.
func TestScopeTakesItsRowsFromTheBody(t *testing.T) {
	for _, size := range [][2]int{{100, 40}, {80, 30}, {64, 28}} {
		off := scopeModel(size[0], size[1])
		before := off.layout().bodyHeight

		on := scopeModel(size[0], size[1])
		on.scope.on = true
		on.resize()

		if got := on.layout().bodyHeight; got != before-scopeRows-scopeChrome {
			t.Errorf("%dx%d: body = %d with the trace, want %d", size[0], size[1], got, before-scopeRows-scopeChrome)
		}
		rows := strings.Split(plain(on.render()), "\n")
		if len(rows) != size[1] {
			t.Errorf("%dx%d: render() = %d rows, want %d", size[0], size[1], len(rows), size[1])
		}
		if strings.TrimSpace(rows[len(rows)-1]) == "" {
			t.Errorf("%dx%d: the help bar was pushed off the bottom", size[0], size[1])
		}
	}
}

// A short terminal has better uses for five rows, and the key says so by doing
// nothing rather than by rearranging the screen.
func TestScopeIsNotOfferedOnAShortTerminal(t *testing.T) {
	m := scopeModel(80, scopeMinHeight-1)

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

func msgScopeTick() tea.Msg { return msg.ScopeTick{} }
