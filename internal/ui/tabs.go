package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// headerGap is the least space kept between the device name and the tabs.
const headerGap = 2

// tabID identifies a screen. Each one works differently: the player shows what
// is sounding, the queue what follows, the library browses what has been saved,
// the search hunts for a track.
type tabID int

const (
	tabPlayer tabID = iota
	tabQueue
	tabLibrary
	tabSearch
	tabSettings
	tabHelp

	// tabCount sizes the per-tab state: each screen remembers its own
	// visualiser, and an array indexed by the tab keeps that beside the tab
	// rather than in a parallel structure that can fall out of step.
	tabCount = iota
)

var tabNames = [...]string{
	tabPlayer:   "now playing",
	tabQueue:    "queue",
	tabLibrary:  "library",
	tabSearch:   "search",
	tabSettings: "settings",
	tabHelp:     "help",
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

// tabDigits is the keys that go straight to a screen: one per screen there is,
// counted from the tabs themselves so that adding or dropping one cannot leave
// a digit bound to nothing or a screen with no digit. Nine screens is where
// counting on one hand runs out, and there are six.
func tabDigits() []string {
	out := make([]string, len(tabNames))
	for i := range tabNames {
		out[i] = string(rune('1' + i))
	}
	return out
}

// tabDigitRange names that run of digits, with the dash set as the caller sets
// its ranges: the bar is one line and has no room for air around it, the page
// of keys has both.
func tabDigitRange(dash string) string {
	return "1" + dash + string(rune('0'+len(tabNames)))
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

// tabGap is the air between two labels. Named rather than written twice,
// because the pointer has to step by the same distance the drawing sets by.
const tabGap = "   "

// tabSpans is where each label ends up on the row, for a click to be answered.
//
// The bar is set flush right, so it begins wherever it must to end at the
// margin — and the row it is on is the padded one, which starts a margin in.
// Nothing is written down as the screen is drawn; this reads the same names and
// the same air the drawing does, and TestAClickLandsOnTheTabItIsOver puts the
// drawn row through it column by column.
func tabSpans(l layout) []span {
	inner := l.interior - leftMargin - rightMargin
	width := labelsWidth(tabNames[:], len(tabGap))
	if width > inner {
		// Cut off at the edge. Where a label ended can no longer be said, and
		// guessing would land a click on the wrong screen.
		return nil
	}
	return labelSpans(tabNames[:], len(tabGap), leftMargin+inner-width)
}

// tabs renders the tab labels and the rule that marks the active one. No boxes —
// the underline is the only chrome on the screen.
func (m Model) tabs() (labels, rule string) {
	var names, marks strings.Builder
	for i, name := range tabNames {
		if i > 0 {
			names.WriteString(tabGap)
			marks.WriteString(tabGap)
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
