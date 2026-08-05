package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The keys live on a screen now. Unfolding them out of the bar pushed the list
// off the bottom of exactly the tabs whose keys somebody was looking up.
func TestQuestionMarkGoesToTheKeys(t *testing.T) {
	m := New(nil, nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.resize()

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if got := tm.(Model).tab; got != tabHelp {
		t.Errorf("? landed on %v, want the keys", got)
	}
}

// Every key the screens answer is named, and each one only once.
func TestTheKeysAreAllNamed(t *testing.T) {
	seen := map[string]string{}
	for _, group := range helpGroups() {
		for _, pair := range group.keys {
			if pair[1] == "" {
				t.Errorf("%q under %q says nothing about itself", pair[0], group.title)
			}
			if where, ok := seen[pair[0]+group.title]; ok {
				t.Errorf("%q is listed twice under %q (%s)", pair[0], group.title, where)
			}
			seen[pair[0]+group.title] = group.title
		}
	}

	// The ones that would be missed first.
	page := plain(strings.Join(helpScreen(t, 200, 46), "\n"))
	for _, want := range []string{"play or pause", "find in this list", "waveform", "add it to the queue"} {
		if !strings.Contains(page, want) {
			t.Errorf("the keys screen does not mention %q", want)
		}
	}
}

// Groups are kept whole: a heading in one column with its keys in the next is
// worse than an uneven bottom edge.
func TestTheColumnsKeepGroupsWhole(t *testing.T) {
	for _, w := range []int{80, 120, 160, 200} {
		m := New(nil, nil, defaultTestCell)
		m.width, m.height = w, 46
		m.resize()

		columns, width := helpColumnFit(m.layout().interior - leftMargin - rightMargin)
		if columns < 1 || width < helpColumnMin {
			t.Errorf("at %d columns the width came out %d", w, width)
		}

		// Every heading has at least one key under it on the same column.
		lines := plain(strings.Join(m.helpColumns(helpGroups(), m.layout().interior-leftMargin-rightMargin, 40), "\n"))
		for _, group := range helpGroups() {
			if !strings.Contains(lines, group.title) {
				t.Errorf("at width %d the group %q is missing", w, group.title)
			}
		}
	}
}

func helpScreen(t *testing.T, w, h int) []string {
	t.Helper()

	m := New(nil, nil, defaultTestCell)
	m.tab = tabHelp
	m.width, m.height = w, h
	m.resize()
	return m.helpPanel(m.layout(), m.layout().bodyHeight)
}
