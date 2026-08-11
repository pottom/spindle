package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"

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

	for _, k := range []string{"n", "ctrl+n"} {
		if !key.Matches(tea.KeyPressMsg{Code: 'n', Text: "n", Mod: modOf(k)}, m.keys.Next) {
			t.Errorf("%q does not skip forward", k)
		}
	}
	if !key.Matches(press("p"), m.keys.Prev) {
		t.Error("p does not skip back")
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

func modOf(k string) tea.KeyMod {
	if k == "ctrl+n" {
		return tea.ModCtrl
	}
	return 0
}
