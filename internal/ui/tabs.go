package ui

import "strings"

// tabID identifies a screen. Each one works differently: the player shows what
// is sounding, the queue what follows, the playlists browse a library, the
// search hunts for a track.
type tabID int

const (
	tabPlayer tabID = iota
	tabQueue
	tabPlaylists
	tabSearch
)

var tabNames = [...]string{
	tabPlayer:    "now playing",
	tabQueue:     "queue",
	tabPlaylists: "playlists",
	tabSearch:    "search",
}

func (t tabID) String() string { return tabNames[t] }

// next cycles through the tabs, wrapping in either direction.
func (t tabID) next(delta int) tabID {
	n := len(tabNames)
	return tabID(((int(t)+delta)%n + n) % n)
}

// tabBar renders the header: the active tab in full strength with a rule under
// it, the rest faint. No boxes — the underline is the only chrome on the screen.
func (m Model) tabBar(w int) []string {
	const gap = "   "

	var labels, rule strings.Builder
	for i, name := range tabNames {
		if i > 0 {
			labels.WriteString(gap)
			rule.WriteString(gap)
		}
		if tabID(i) == m.tab {
			labels.WriteString(m.styles.TabActive.Render(name))
			rule.WriteString(m.styles.TabRule.Render(strings.Repeat(meterFull, len(name))))
			continue
		}
		labels.WriteString(m.styles.TabIdle.Render(name))
		rule.WriteString(strings.Repeat(" ", len(name)))
	}

	return []string{fit(labels.String(), w), fit(rule.String(), w)}
}
