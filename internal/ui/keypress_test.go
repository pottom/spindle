package ui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// A binding written as a letter is the key that letter sits on, whatever the
// keyboard in front of somebody puts there.
func TestAKeyIsMatchedByTheKeyItCameFrom(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)

	// A Hungarian keyboard: the key where z sits on a US board sends y, and the
	// terminal says so.
	hungarianZ := tea.KeyPressMsg{Code: 'y', Text: "y", BaseCode: 'z'}
	if !m.pressed(hungarianZ, key.NewBinding(key.WithKeys("z"))) {
		t.Error("the key where z sits does not answer to z")
	}

	// And the letter that actually arrived still answers as itself: somebody who
	// remapped their keyboard on purpose means the letters they typed.
	if !m.pressed(hungarianZ, key.NewBinding(key.WithKeys("y"))) {
		t.Error("the letter that arrived does not answer to itself")
	}

	// A key with nothing to do with either is still nothing to do with either.
	if m.pressed(hungarianZ, key.NewBinding(key.WithKeys("q"))) {
		t.Error("a key answered to a binding it has no business with")
	}

	// With no terminal saying anything about base keys, this is the ordinary
	// match and nothing else.
	plain := tea.KeyPressMsg{Code: 'y', Text: "y"}
	if m.pressed(plain, key.NewBinding(key.WithKeys("z"))) {
		t.Error("a press with no base key was matched as one")
	}
	if !m.pressed(plain, key.NewBinding(key.WithKeys("y"))) {
		t.Error("a press with no base key stopped matching itself")
	}
}

// The whole way through: the key where c sits folds the queue's band on a
// keyboard that sends something else from it.
func TestAFoldingKeyWorksOnAnotherLayout(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.resize()
	was := m.queuePane.room

	// A layout that sends "&" from the key where c sits.
	next, _ := m.handleKey(tea.KeyPressMsg{Code: '&', Text: "&", BaseCode: 'c'})
	if next.queuePane.room == was {
		t.Errorf("the key where c sits did not fold the band: still %v", was)
	}
}

// And the request for it is made of the terminal, or nothing ever reports one.
func TestTheTerminalIsAskedForTheBaseKey(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 100, 30
	m.resize()
	if !m.View().KeyboardEnhancements.ReportAlternateKeys {
		t.Error("the terminal is never asked which key a press came from")
	}
}
