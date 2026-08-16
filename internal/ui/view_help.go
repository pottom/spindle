package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/build"
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
// The words are written out here rather than derived from the key map: the
// bindings know which keys they are, and this knows what they are for — which
// is a different question, is what a reader is asking, and is worth writing in
// sentences somebody chose.
//
// The keys themselves are not written out. They are the same names the bindings
// are made from, so a key that moves moves here too, and this page cannot go on
// teaching a letter nothing answers to.
func helpGroups() []helpGroup {
	return []helpGroup{{
		title: "Getting around",
		keys: [][2]string{
			{tabDigitRange(" – "), "go to a screen"},
			{keyNextTab, "the next screen"},
			{keyBack, "back, out of a list or a search"},
			{keyHelp, "these keys"},
			{keyQuit, "leave, and let the music play on"},
			{keyQuitAll, "leave and stop the music"},
		},
	}, {
		title: "Playing",
		keys: [][2]string{
			{keyPlayPause, "play or pause"},
			{pair(keyNext, keyPrev), "next or previous track"},
			{"← / →", "seek five seconds — shift+← / → where a list has the arrows"},
			{"↑ / ↓", "the music's volume, by five — shift+↑ / ↓ in a list"},
			{keyMute, "mute, and back to where it was"},
			{keyShuffle, "shuffle"},
			{keyRepeat, "repeat: off, all, one"},
			{keyDevices, "play somewhere else"},
			{keyStage, "the full screen picture, from wherever you are"},
		},
	}, {
		title: "On the player",
		keys: [][2]string{
			{keyScope, "waveform, spectrum, water, lamps, off — and no off on the queue"},
			{keyLyrics, "the words, as they are sung"},
			{keyPeek, "what is coming next"},
		},
	}, {
		title: "On the full screen",
		keys: [][2]string{
			{keyScope, "which picture it is"},
			{keyPlayPause + " ← → ↑ ↓", "the transport, without leaving"},
			{pair(keyNext, keyPrev), "next or previous track"},
			{keyMarks, "another company of dancers"},
			{keyTell, "what is playing, said there and then"},
			{pair(keyShuffle, keyRepeat), "shuffle and repeat, said by the one with the placard"},
			{pair(keyBack, keyQuit) + " / " + keyStage, "back — and nothing else is, so it can be leaned on"},
		},
	}, {
		title: "On this page",
		keys: [][2]string{
			{"pgup/pgdn", "the keys scroll under the head"},
			{"^u / ^d", "half a screenful"},
			{pair(keyFirstVim, keyLastVim), "the top, the end"},
		},
	}, {
		title: "In a list",
		keys: [][2]string{
			{"↑ ↓", "move — and ← → as well on the library, which is a wall"},
			{"pgup/pgdn", "a screenful"},
			{"^u / ^d", "half a screenful"},
			{pair(keyFirstVim, keyLastVim), "the top, the end"},
			{keyFind, "find in this list"},
			{pair(keyKindPrev, keyKindNext), "the kinds a tab holds: playlists, albums, artists, recent"},
			{pair(keyFindNext, keyFindPrev), "the next match, the one before"},
			{keyEnter, "play it, or open it"},
			{keyPlayOne, "play only this one"},
			{keyEnqueue, "add it to the queue"},
			{keyActions, "everything else it can do"},
		},
	}, {
		title: "In the queue",
		keys: [][2]string{
			{keyClose, "fold the top away: the picture, the player, both"},
			{pair(keyMoveDn, keyMoveUp), "move a track down or up"},
			{keyDrop, "take it out"},
			{keyEnter, "bring it forward and play it"},
		},
	}, {
		title: "Searching",
		keys: [][2]string{
			{keyFind, "type a query"},
			{"^t", "tracks, albums, artists, playlists"},
			{keyEnter, "play a track, open anything else"},
			{"^a", "add to the queue while typing"},
		},
	}, {
		title: "In the library",
		keys: [][2]string{
			{"^t", "playlists, albums, artists, recent"},
			{keyEnter, "open it"},
			{keyEnqueue, "add the whole of it to the queue"},
		},
	}, {
		title: "On the settings",
		keys: [][2]string{
			{"← / →", "change what the cursor is on"},
			{keyRestart, "restart the device, to hear it"},
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
	head := m.helpHead(w)

	// The keys are a page under a fixed head rather than part of one long page.
	//
	// Because the head has a picture in it, and a picture put into the terminal
	// by the kitty protocol is placed on the screen rather than in the text: it
	// stays where it was put while whatever is under it moves. Scrolling both
	// would slide the words out from under the logo. A letterhead does not
	// scroll anyway.
	body := max(rows-len(head), 0)
	keys := m.helpColumns(helpGroups(), w, helpKeyRows(w))
	keys = keys[min(m.helpAt, max(len(keys)-body, 0)):]

	lines := append(head, keys...)
	for len(lines) < rows {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines[:rows]
}

// helpHead is what the page is headed by: the program's own picture where there
// is room for one, its name, the build it was made from and its licence.
func (m Model) helpHead(w int) []string {
	head := m.splashRows()
	if len(head) > 0 {
		head = append(head, strings.Repeat(" ", w))
	}
	return append(head,
		fit(m.styles.Title.Render("spindle "+build.Version()), w),
		fit(m.styles.Album.Render("the best-looking Spotify player in a terminal — GPL-3.0"), w),
		strings.Repeat(" ", w),
	)
}

// helpKeyRows is how many rows to lay the keys out in before any of it is
// windowed: all of them, since the page scrolls.
//
// Measured, so the number is not a guess that quietly cuts a group off: the
// groups come to 36 rows at anything from 120 cells wide and 68 at 80, because
// a narrow terminal gets fewer columns. Twice the widest measured is room to
// grow into.
func helpKeyRows(w int) int { return 140 }

// helpScroll moves the keys under the head, and reports whether the key was one
// of the ones that do.
func (m *Model) helpScroll(k tea.KeyPressMsg, page int) bool {
	return m.scrolled(k, &m.helpAt, page, helpKeyRows(0))
}

// scrolled moves a row offset by the keys everything that scrolls is moved by,
// and says whether the key was one of them.
//
// One set of keys for the page of keys and for the box holding a record's
// story, because there is one answer to "how do I get further down this": the
// arrows, the pages, and the two that go to the ends. A second thing that
// scrolled differently would be a second thing to learn for no reason.
func (m Model) scrolled(k tea.KeyPressMsg, at *int, page, last int) bool {
	switch {
	case m.pressed(k, m.keys.Down):
		*at++
	case m.pressed(k, m.keys.Up):
		*at--
	case m.pressed(k, m.keys.PageDown):
		*at += page
	case m.pressed(k, m.keys.PageUp):
		*at -= page
	case m.pressed(k, m.keys.HalfDown):
		*at += max(page/2, 1)
	case m.pressed(k, m.keys.HalfUp):
		*at -= max(page/2, 1)
	case m.pressed(k, m.keys.First), m.pressed(k, m.keys.FirstVim):
		*at = 0
	case m.pressed(k, m.keys.Last), m.pressed(k, m.keys.LastVim):
		*at = last
	default:
		return false
	}
	*at = min(max(*at, 0), max(last, 0))
	return true
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
