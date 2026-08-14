package ui

import (
	"fmt"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

func press(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

// Skipping a track is on the plain keys, and typing still types.
//
// n and N used to walk the matches of a search inside a list, and the transport
// had ctrl+n and ctrl+p. That was the wrong way round in use: skipping is wanted
// on every screen there is and is the most pressed key after play, while walking
// matches is wanted in a list that has just been searched.
func TestSkippingIsOnThePlainKeys(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)

	if !key.Matches(press("n"), m.keys.Next) {
		t.Error("n does not skip forward")
	}
	if !key.Matches(press("p"), m.keys.Prev) {
		t.Error("p does not skip back")
	}

	// And nothing is left on the held ones. They sat beside the letters after
	// the transport took them, doing the same thing a finger later — and not
	// even that where it would have mattered: a query being typed swallows
	// them, so the case they were kept for was one they never answered.
	for _, k := range []struct {
		code rune
		b    key.Binding
	}{{'n', m.keys.Next}, {'p', m.keys.Prev}} {
		if key.Matches(tea.KeyPressMsg{Code: k.code, Mod: tea.ModCtrl}, k.b) {
			t.Errorf("ctrl+%c still works the transport", k.code)
		}
	}

	// The matches moved to what vim repeats a search with.
	if !key.Matches(press(";"), m.keys.FindNext) || !key.Matches(press(","), m.keys.FindPrev) {
		t.Error("the matches are not on ; and ,")
	}
	if key.Matches(press("n"), m.keys.FindNext) {
		t.Error("n still walks matches, so it cannot skip a track in a list")
	}

	// And a query being written still gets every letter of it.
	m.find.typing = true
	if _, handled := m.findKey(press("n")); !handled {
		t.Error("typing n into a search skipped a track instead")
	}
	if got := m.find.query; got != "n" {
		t.Errorf("the query came out %q", got)
	}
}

// The transport keeps a second pair of arrows, for the screens where the plain
// ones belong to what is on them.
//
// A list took up and down for its cursor and the volume has been unreachable
// there ever since; the wall takes left and right as well. Shift gives both back
// by one rule, on every screen.
func TestShiftArrowsSeekAndSetTheVolume(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	m.ps = &player.State{TrackID: "a", Playing: true, Volume: 50,
		Duration: 3 * time.Minute, Progress: time.Minute}
	for i := range 8 {
		m.library.playlists = append(m.library.playlists,
			player.Playlist{ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i)})
	}

	for _, c := range []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"shift+right", tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}},
		{"shift+left", tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}},
		{"shift+up", tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}},
		{"shift+down", tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}},
	} {
		was := m.library.cursors[m.library.kind].cursor
		next, cmd := m.handleKey(c.key)
		if cmd == nil {
			t.Errorf("%s did nothing", c.name)
		}
		if now := next.library.cursors[next.library.kind].cursor; now != was {
			t.Errorf("%s moved the cursor from %d to %d", c.name, was, now)
		}
	}

	// And the plain arrows still walk the wall.
	for _, c := range []tea.KeyPressMsg{{Code: tea.KeyRight}, {Code: tea.KeyDown}} {
		was := m.library.cursors[m.library.kind].cursor
		next, _ := m.handleKey(c)
		if now := next.library.cursors[next.library.kind].cursor; now == was {
			t.Errorf("a plain arrow did not move the cursor from %d", was)
		}
		m = next
	}
}
