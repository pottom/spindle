package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// typing types one letter at a time and says how many searches went out.
func typing(t *testing.T, m Model, word string) (Model, int) {
	t.Helper()

	asked := 0
	var tm tea.Model = m
	for _, letter := range word {
		var cmd tea.Cmd
		tm, cmd = tm.Update(tea.KeyPressMsg{Code: letter, Text: string(letter)})
		if cmd != nil && ranSearch(cmd) {
			asked++
		}
	}
	return tm.(Model), asked
}

// ranSearch reports whether running this command asks Spotify anything. The
// mock answers, so running it is safe and is the only way to tell a batch that
// searches from a batch that only spins.
func ranSearch(cmd tea.Cmd) bool {
	return foundSearch(cmd())
}

func foundSearch(message tea.Msg) bool {
	switch message := message.(type) {
	case tea.BatchMsg:
		for _, one := range message {
			if one != nil && foundSearch(one()) {
				return true
			}
		}
	case msg.SearchResults:
		return true
	}
	return false
}

// A query is asked once it has been typed, not once per letter.
//
// Typing "jolene" spent six requests and read one answer: the five before it
// were about words nobody had finished. Measured in a real terminal against the
// real API — six letters, six lines in the request log.
func TestAQueryIsAskedOnceTypingStops(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 100, 40
	m.resize()
	m.tab = tabSearch
	m.search.typing = true

	after, asked := typing(t, m, "jolene")
	if asked != 0 {
		t.Errorf("typing six letters asked %d times before anybody had stopped typing", asked)
	}
	if after.search.input.Value() != "jolene" {
		t.Fatalf("the box holds %q, so the keys never reached it", after.search.input.Value())
	}

	// And when the typing settles, the query goes out — once.
	var tm tea.Model = after
	_, cmd := tm.Update(searchSettled{seq: after.search.seq})
	if cmd == nil {
		t.Error("the typing settled and nothing was asked")
	}

	// A tick for a word somebody was halfway through is spent on nothing.
	_, stale := tm.Update(searchSettled{seq: after.search.seq - 3})
	if stale != nil {
		t.Error("a tick from three letters ago asked its own question")
	}
}
