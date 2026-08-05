package ui

import (
	"strings"
)

// The keys, on a screen of their own.
//
// They used to unfold out of the bar at the bottom, which meant the one screen
// that lists everything was also the one screen with no room for it: the table
// pushed the list off the bottom on the tabs with the longest lists, and on a
// short terminal it pushed itself off. A screen has room, and the bar goes back
// to being what it is good at — the three or four keys worth naming in passing.
//
// Laid out in columns of grouped pairs rather than as a table: what somebody
// looks up here is "what does this screen do", and the answer is a short list
// under a heading, not a row in a grid of forty.

const (
	// helpKeyCols is the width the key itself is set in, so the descriptions
	// line up down a column and the eye can run along either one.
	helpKeyCols = 12

	// helpColumnMin is the narrowest a column may be before it starts cutting
	// the descriptions, and helpColumnMax the widest worth setting one in: past
	// that the key and its meaning drift apart across a field of space.
	helpColumnMin = 46
	helpColumnMax = 62

	// helpColumnGap is the air between two columns. Wide enough that the
	// columns read as columns without a rule drawn between them.
	helpColumnGap = 6

	// helpColumnsMost is as many as are worth having: three columns of keys is
	// already a page somebody scans rather than reads.
	helpColumnsMost = 3
)

// helpGroup is one heading and the keys under it.
type helpGroup struct {
	title string
	keys  [][2]string
}

// helpGroups is every key spindle answers, in the order somebody meets them.
//
// Written out here rather than derived from the key map: the bindings know
// which keys they are, and this knows what they are for — which is a different
// question, is what a reader is asking, and is worth writing in sentences
// somebody chose.
func helpGroups() []helpGroup {
	return []helpGroup{{
		title: "Getting around",
		keys: [][2]string{
			{"1 – 6", "go to a screen"},
			{"tab", "the next screen"},
			{"esc", "back, out of a list or a search"},
			{"?", "these keys"},
			{"q", "leave, and let the music play on"},
			{"Q", "leave and stop the music"},
		},
	}, {
		title: "Playing",
		keys: [][2]string{
			{"space", "play or pause"},
			{"^n / ^p", "next or previous track"},
			{"← / →", "seek five seconds"},
			{"↑ / ↓", "the music's volume, by five"},
			{"m", "mute, and back to where it was"},
			{"s", "shuffle"},
			{"r", "repeat: off, all, one"},
			{"d", "play somewhere else"},
		},
	}, {
		title: "On the player",
		keys: [][2]string{
			{"v", "waveform, spectrum, water, lamps, off"},
			{"f", "full screen, v switches it, F pulls a face"},
			{"l", "the words, as they are sung"},
			{"u", "what is coming next"},
		},
	}, {
		title: "In a list",
		keys: [][2]string{
			{"↑ ↓", "move"},
			{"pgup/pgdn", "a screenful"},
			{"^u / ^d", "half a screenful"},
			{"g / G", "the top, the end"},
			{"/", "find in this list"},
			{"n / N", "the next match, the one before"},
			{"enter", "play it, or open it"},
			{"o", "play only this one"},
			{"a", "add it to the queue"},
			{".", "everything else it can do"},
		},
	}, {
		title: "In the queue",
		keys: [][2]string{
			{"j / k", "move a track down or up"},
			{"x", "take it out"},
			{"enter", "bring it forward and play it"},
		},
	}, {
		title: "Searching",
		keys: [][2]string{
			{"/", "type a query"},
			{"^t", "tracks, albums, artists, playlists"},
			{"enter", "play a track, open anything else"},
			{"^a", "add to the queue while typing"},
		},
	}, {
		title: "In the library",
		keys: [][2]string{
			{"^t", "playlists, albums, artists, recent"},
			{"enter", "open it"},
			{"a", "add the whole of it to the queue"},
		},
	}, {
		title: "On the settings",
		keys: [][2]string{
			{"← / →", "change what the cursor is on"},
			{"R", "restart the device, to hear it"},
		},
	}, {
		title: "From the shell",
		keys: [][2]string{
			{"spindle", "--help lists every command"},
			{"status", "--line for a status bar"},
			{"daemon", "start, stop, restart, status"},
		},
	}}
}

// helpPanel draws the screen.
func (m Model) helpPanel(l layout, rows int) []string {
	w := l.interior - leftMargin - rightMargin

	head := []string{
		fit(m.styles.Title.Render("Keys"), w),
		fit(m.styles.Album.Render("and what they are for"), w),
		strings.Repeat(" ", w),
	}

	// Set from the top rather than centred: this is a page to read, and a page
	// that floats in the middle of the screen reads as a dialogue box.
	lines := append(head, m.helpColumns(helpGroups(), w, max(rows-len(head), 0))...)
	for len(lines) < rows {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines[:rows]
}

// helpColumns lays the groups out side by side.
//
// Balanced rather than filled: a column poured full before the next is started
// leaves the last one a stub, and a page whose columns end at different heights
// reads as a page that ran out rather than one that was set. The groups are
// kept whole — a heading in one column and its keys in the next is worse than
// an uneven bottom edge.
func (m Model) helpColumns(groups []helpGroup, w, rows int) []string {
	columns, width := helpColumnFit(w)

	blocks := make([][]string, 0, len(groups))
	total := 0
	for _, group := range groups {
		block := m.helpGroup(group)
		blocks = append(blocks, block)
		total += len(block)
	}

	// The height to aim for in each column, never taller than the screen: what
	// will not fit is what the screen cannot show either way.
	target := max((total+columns-1)/columns, 1)
	if rows > 0 {
		target = min(target, rows)
	}

	var built [][]string
	current := []string{}
	for _, block := range blocks {
		tall := len(current) > 0 && len(current)+len(block) > target
		if tall && len(built) < columns-1 {
			built = append(built, current)
			current = []string{}
		}
		current = append(current, block...)
	}
	if len(current) > 0 {
		built = append(built, current)
	}

	return joinColumns(built, width, helpColumnGap, w)
}

// helpColumnFit is how many columns the width can hold, and how wide each one
// is then set.
func helpColumnFit(w int) (columns, width int) {
	columns = max(min((w+helpColumnGap)/(helpColumnMin+helpColumnGap), helpColumnsMost), 1)
	width = min((w-(columns-1)*helpColumnGap)/columns, helpColumnMax)
	return columns, max(width, helpColumnMin)
}

// helpGroup is one heading and its pairs, with the blank line that separates it
// from the next.
func (m Model) helpGroup(g helpGroup) []string {
	out := []string{m.styles.FactLabel.Render(g.title)}
	for _, pair := range g.keys {
		key := m.styles.Cursor.Render(pair[0])
		out = append(out, key+strings.Repeat(" ", max(helpKeyCols-len([]rune(pair[0])), 1))+
			m.styles.Detail.Render(pair[1]))
	}
	return append(out, "")
}

// joinColumns sets blocks of lines beside each other, each squared off to its
// own width so the one to its right starts in the same place on every row.
func joinColumns(blocks [][]string, width, gap, w int) []string {
	height := 0
	for _, block := range blocks {
		height = max(height, len(block))
	}

	gapText := strings.Repeat(" ", gap)
	out := make([]string, height)
	for row := range height {
		var line strings.Builder
		for i, block := range blocks {
			if i > 0 {
				line.WriteString(gapText)
			}
			text := ""
			if row < len(block) {
				text = block[row]
			}
			line.WriteString(fit(text, width))
		}
		out[row] = fit(strings.TrimRight(line.String(), " "), w)
	}
	return out
}
