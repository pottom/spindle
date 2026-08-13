package ui

import (
	"testing"

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
