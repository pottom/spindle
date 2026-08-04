package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// headerGap is the least space kept between the device name and the tabs.
const headerGap = 2

// tabID identifies a screen. Each one works differently: the player shows what
// is sounding, the queue what follows, the playlists browse a library, the
// search hunts for a track.
type tabID int

const (
	tabPlayer tabID = iota
	tabQueue
	tabPlaylists
	tabSearch

	// tabCount sizes the per-tab state: each screen remembers its own
	// visualiser, and an array indexed by the tab keeps that beside the tab
	// rather than in a parallel structure that can fall out of step.
	tabCount = iota
)

var tabNames = [...]string{
	tabPlayer:    "now playing",
	tabQueue:     "queue",
	tabPlaylists: "playlists",
	tabSearch:    "search",
}

func (t tabID) String() string { return tabNames[t] }

// tabAt maps a digit onto the tab drawn in that place, counting from one.
func tabAt(digit string) (tabID, bool) {
	n := int(digit[0] - '1')
	if len(digit) != 1 || n < 0 || n >= len(tabNames) {
		return 0, false
	}
	return tabID(n), true
}

// next cycles through the tabs, wrapping in either direction.
func (t tabID) next(delta int) tabID {
	n := len(tabNames)
	return tabID(((int(t)+delta)%n + n) % n)
}

// header is the top two rows: what is playing on the left, where you are on the
// right. The two belong together — one says which machine is making the sound,
// the other which screen you are looking at — and neither is worth a row of its
// own.
func (m Model) header(w int) []string {
	labels, rule := m.tabs()

	// The tabs are kept whole and the device name gives way: a long name is a
	// detail, but tabs cut off at the edge are navigation you cannot read.
	status := ""
	if room := w - lipgloss.Width(labels) - headerGap; room > 0 {
		status = fit(m.statusLine(), room)
	}

	return []string{
		spread(status, labels, w),
		// The rule has to sit under the tabs, which are flush right.
		padLeft(rule, w),
	}
}

// tabs renders the tab labels and the rule that marks the active one. No boxes —
// the underline is the only chrome on the screen.
func (m Model) tabs() (labels, rule string) {
	const gap = "   "

	var names, marks strings.Builder
	for i, name := range tabNames {
		if i > 0 {
			names.WriteString(gap)
			marks.WriteString(gap)
		}
		if tabID(i) == m.tab {
			names.WriteString(m.styles.TabActive.Render(name))
			marks.WriteString(m.styles.TabRule.Render(strings.Repeat(meterFull, len(name))))
			continue
		}
		names.WriteString(m.styles.TabIdle.Render(name))
		marks.WriteString(strings.Repeat(" ", len(name)))
	}

	return names.String(), marks.String()
}
