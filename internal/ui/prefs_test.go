package ui

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// Each screen keeps its own visualiser, and the file has to carry that back.
func TestPrefsRememberEachTabSeparately(t *testing.T) {
	m := scopeModel(200, 45)
	m.scope.modes[tabPlayer] = scopeBars
	m.scope.modes[tabQueue] = scopeOff
	m.lyrics.on = true

	var saved prefs
	data, err := json.Marshal(prefs{
		Scope:  append([]scopeMode(nil), m.scope.modes[:]...),
		Lyrics: m.lyrics.on,
		Peek:   m.peek.on,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	back := scopeModel(200, 45)
	back.applyPrefs(saved)
	if back.scope.modes[tabPlayer] != scopeBars || back.scope.modes[tabQueue] != scopeOff {
		t.Errorf("the tabs came back as %v, want the bars on the player and nothing on the queue", back.scope.modes)
	}
	if !back.lyrics.on {
		t.Error("the words did not come back on")
	}
}

// A file written by another version must not put the model into a state its own
// code cannot produce.
func TestPrefsIgnoreWhatItCannotUse(t *testing.T) {
	m := scopeModel(200, 45)
	m.applyPrefs(prefs{Scope: []scopeMode{scopeModes + 3, scopeBars, scopeOff, scopeOff, scopeOff, scopeOff}})
	if m.scope.modes[tabPlayer] != scopeWave {
		t.Errorf("the player took mode %d from the file, want its default", m.scope.modes[tabPlayer])
	}
	if m.scope.modes[tabQueue] != scopeBars {
		t.Error("a mode that was in range was dropped along with the one that was not")
	}
}

// The trace goes into the column the detail panel has never filled, and nothing
// below it moves when it appears.
func TestQueueDrawsTheTraceWithoutMovingTheList(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "playing", Playing: true, DeviceName: "spindle"}
	m.tab = tabQueue
	m.width, m.height = 200, 45
	m.resize()

	if !m.scopeVisible() {
		t.Fatal("the queue does not draw the trace on a terminal with room for it")
	}
	m.scope.frame = []float32{-1, -1, 1, 1}
	m.scope.follow(m.scope.frame)

	on := ansiOff(m.render())
	if !hasBraille(on) {
		t.Error("no trace was drawn on the queue")
	}

	// The same key cycles it here as on the player, and only this tab's setting
	// moves.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	got := tm.(Model)
	if got.scope.modes[tabQueue] != scopeBars {
		t.Errorf("v put the queue in mode %d, want the bars", got.scope.modes[tabQueue])
	}
	if got.scope.modes[tabPlayer] != scopeWave {
		t.Error("cycling the queue's visualiser changed the player's")
	}

	m.scope.modes[tabQueue] = scopeOff
	off := ansiOff(m.render())
	if rowOf(on, "Queue") != rowOf(off, "Queue") {
		t.Errorf("the heading sits on row %d with the trace and %d without it",
			rowOf(on, "Queue"), rowOf(off, "Queue"))
	}
}

// columnOf is where want sits inside its own line, which is what "the same
// column" means on a screen whose lines are not all the same length.
func columnOf(screen, want string) int {
	for _, line := range strings.Split(screen, "\n") {
		if at := strings.Index(line, want); at >= 0 {
			return at
		}
	}
	return -1
}

func rowOf(screen, want string) int {
	for i, line := range strings.Split(screen, "\n") {
		if strings.Contains(line, want) {
			return i
		}
	}
	return -1
}

// hasBraille reports whether anything was plotted: an empty cell is drawn as a
// space, so a braille rune on screen means the trace is there.
func hasBraille(s string) bool {
	for _, r := range s {
		if r > brailleBase && r < brailleBase+0x100 {
			return true
		}
	}
	return false
}

// The trace hangs from the same column the artists below start at. Two blocks
// that nearly line up read as a mistake; the same line reads as a column.
func TestQueueTraceStartsWhereTheArtistsDo(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "playing", Playing: true, DeviceName: "spindle"}
	m.tab = tabQueue
	m.queue = []player.Track{{ID: "a", Title: "a", Artists: []string{"someone"}}}
	m.width, m.height = 200, 45
	m.resize()

	l := m.layout()
	if !m.scopeVisible() {
		t.Fatal("the trace is not drawn on a terminal with room for it")
	}
	trace := l.artWidth + columnGap + queueDetailWidth(l) + columnGap
	if want := queueSplit(l); trace != want {
		t.Errorf("the trace starts at column %d and the artists at %d", trace, want)
	}
}

// The right-hand columns must not step sideways when the list grows past the
// screen and the scrollbar appears.
func TestQueueColumnsDoNotMoveWhenTheBarAppears(t *testing.T) {
	at := func(tracks int) int {
		m := queueModel(0)
		for i := range tracks {
			m.queue = append(m.queue, player.Track{ID: string(rune('a' + i)), Title: "title", Artists: []string{"THE ARTIST"}})
		}
		m.width, m.height = 180, 40
		m.resize()
		return columnOf(ansiOff(m.render()), "THE ARTIST")
	}
	if short, long := at(3), at(60); short != long {
		t.Errorf("the artists sit at %d with a short list and %d with a scrollbar", short, long)
	}
}
