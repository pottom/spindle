package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Saying whose track the band at the top belongs to.
//
// On the queue the band across the top — the cover, the name, the facts — is
// whatever the cursor is resting on, and the list under it begins with the track
// that is sounding. Most of the time those are the same thing and there is
// nothing to say. The moment the cursor moves off it they are two different
// records, and nothing on the screen said so: the picture changed and the only
// way to know why was to have watched it change.
//
// So it is drawn: a frame round the band and a line from it down to the row it
// belongs to. Both are what somebody would draw on a printout with a pen, which
// is how the request for it arrived.
//
// It costs no rows. The frame stands in the blank row above the band and the one
// below it, and in the left margin every row already has — the same three cells
// the list's own cursor mark sits inside. Anything that took a row would move
// the whole screen every time the cursor left the first line, which is a worse
// answer to "what is this" than not answering.

const (
	// pointerAt is the column the frame's left edge and the line down from it
	// stand in: inside the left margin, clear of the list's own cursor mark.
	pointerAt = 1

	// The pen. Light box drawing with rounded corners, because it is an
	// annotation over a picture rather than a part of it: a square corner reads
	// as a panel's border, and the band is not a panel — it is a thing somebody
	// has drawn a ring around.
	pointerTL    = "╭"
	pointerTR    = "╮"
	pointerBR    = "╯"
	pointerTee   = "├"
	pointerH     = "─"
	pointerV     = "│"
	pointerElbow = "╰"
)

// pointAtCursor draws the frame and the line, over rows already laid out.
//
// Over rather than into: the band is assembled by whatever screen is up, and a
// mark that had to be threaded through every block would be a mark every block
// had to know about. What it needs instead is where things ended up, which is
// this arithmetic — the same the list itself pages by.
func (m Model) pointAtCursor(lines []string, l layout, top int) []string {
	band := m.listBandRows(l)
	if m.tab != tabQueue || m.devices.open || m.open() != nil || band <= 0 {
		return lines
	}
	if !m.queuePane.room.showsNow() {
		// Nothing up there describes anything, so there is nothing to point at.
		return lines
	}

	// Only when the two are different records. On the row that is sounding the
	// band is about the track the list begins with, and a frame round it would
	// be pointing at what it is standing on.
	if _, playing := m.nowPlayingRow(); playing && m.queuePane.cursor.cursor == 0 {
		return lines
	}

	// Where the row under the cursor ended up: the band, the list's own chrome,
	// and how far down its window the cursor has got.
	cursor := m.queuePane.cursor
	from, _ := cursor.window(len(m.queueRows()), m.visibleListRows())
	at := top + band + listChromeRows + (cursor.cursor - from)

	// The frame stands in the blank rows either side of the band.
	head, foot := top-1, top+band
	if head < 0 || at >= len(lines) || at <= foot {
		return lines
	}

	// A column clear of the panel rather than hard against it: the clock and the
	// length are set to the panel's right edge, and drawn at that column the
	// frame took the air out from under them. The gap between the panel and the
	// picture is three columns wide, so this stands in it.
	right := leftMargin + l.artWidth + columnGap + queueDetailWidth(l) + 1
	if !m.scopeVisible() || !m.queuePane.room.showsTrace() {
		right = leftMargin + l.artWidth + columnGap + l.infoWidth
	}
	if right >= l.interior-1 {
		right = l.interior - 2
	}
	if right <= pointerAt+2 {
		return lines
	}

	// Quiet on purpose. It is an annotation over a picture: what it has to do is
	// answer "which track is this" when the question comes up, not compete with
	// the picture it is drawn around.
	style := m.styles.Rule
	rule := strings.Repeat(pointerH, right-pointerAt-1)

	out := append([]string(nil), lines...)
	put := func(row int, at int, s string) {
		if row < 0 || row >= len(out) {
			return
		}
		out[row] = overwrite(out[row], at, style.Render(s), l.interior)
	}

	// The block's own ground, a step off the screen's, so that what is inside
	// the frame reads as standing apart rather than as the same screen with a
	// line drawn round it. In the hue of the cover it stands beside, like
	// everything else inside the frame: the frame says another record, the
	// colour says which, and the ground says it without a word.
	//
	// From the picture's right edge only. The cover is put on the screen by the
	// terminal rather than written into the row, and cutting a row apart at a
	// column the placement runs through would take the picture with it.
	if from := leftMargin + l.artWidth; from < right {
		for row := head + 1; row < foot; row++ {
			if row < 0 || row >= len(out) {
				continue
			}
			out[row] = raise(out[row], from, right, m.coverStyles.Raised, l.interior)
		}
	}

	put(head, pointerAt, pointerTL+rule+pointerTR)
	put(foot, pointerAt, pointerTee+rule+pointerBR)
	for row := head + 1; row < foot; row++ {
		put(row, pointerAt, pointerV)
		put(row, right, pointerV)
	}

	// And down the margin to the row it belongs to, where it turns and points.
	for row := foot + 1; row < at; row++ {
		put(row, pointerAt, pointerV)
	}
	// The row's own cursor mark is the head of it: drawing a second arrowhead
	// beside it put two of them on the one row, both saying the same thing.
	put(at, pointerAt, pointerElbow+pointerH)
	return out
}

// raise lays a background under part of a row, keeping the colours already in
// it.
//
// A style wrapped round text that has styles of its own does not survive: every
// one of them ends in a reset, and a reset clears the background along with
// everything else — measured, and the tint stopped at the first word that had a
// colour. So the background is put back after every reset inside the span.
func raise(row string, from, to int, bg lipgloss.Style, w int) string {
	if to <= from {
		return row
	}
	row = fit(row, w)

	open := bg.Render("")
	if at := strings.Index(open, "m"); at >= 0 {
		open = open[:at+1] // the sequence that opens it, without its own reset
	}

	span := ansi.Truncate(ansi.TruncateLeft(row, from, ""), to-from, "")
	for _, reset := range []string{"\x1b[m", "\x1b[0m"} {
		span = strings.ReplaceAll(span, reset, reset+open)
	}

	left := ansi.Truncate(row, from, "")
	return fit(left+open+span+"\x1b[m"+ansi.TruncateLeft(row, to, ""), w)
}

// overwrite puts s into a row at a column, keeping what is either side of it.
//
// ANSI-aware in both directions: the rows are styled, and cutting one by byte
// would leave half an escape sequence running into whatever is written next.
func overwrite(row string, at int, s string, w int) string {
	if at < 0 {
		return row
	}
	row = fit(row, w)
	wide := lipgloss.Width(s)

	left := ansi.Truncate(row, at, "")
	if got := lipgloss.Width(left); got < at {
		left += strings.Repeat(" ", at-got)
	}
	return fit(left+s+ansi.TruncateLeft(row, at+wide, ""), w)
}
