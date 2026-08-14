package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/ui/style"
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
// So it is drawn: a bracket down the left of the band, and a line from it to the
// row it belongs to. Both are what somebody would draw on a printout with a pen,
// which is how the request for it arrived.
//
// A bracket rather than a frame. The frame ran all the way round — two rules the
// width of the panel and a second upright at its right edge — which is four lines
// to say what the one line down the left already says, on a screen whose subject
// is a photograph. What the right-hand upright said as well was where the panel
// ends; that is worth less than the quiet, and the panel's own edges say it.
//
// It costs no rows. The arms stand in the blank row above the band and the one
// below it, and the upright in the left margin every row already has — the same
// three cells the list's own cursor mark sits inside. Anything that took a row
// would move the whole screen every time the cursor left the first line, which is
// a worse answer to "what is this" than not answering.

const (
	// pointerAt is the column the frame's left edge and the line down from it
	// stand in: inside the left margin, clear of the list's own cursor mark.
	pointerAt = 1

	// pointerArm is how far the marks at the top and the foot of the band reach
	// in when there is no picture to measure against. With one they run to its
	// edge, which is the thing at that end for them to arrive at.
	pointerArm = 10

	// The pen. Light box drawing with rounded corners, because it is an
	// annotation over a picture rather than a part of it: a square corner reads
	// as a panel's border, and the band is not a panel — it is a thing somebody
	// has drawn a bracket beside. The two right-hand corners belong to the field
	// a list is searched in, which is a box and closes — see finder.go.
	pointerTL    = "╭"
	pointerTee   = "├"
	pointerTR    = "╮"
	pointerBR    = "╯"
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
	at := top + band + m.listChrome() + (cursor.cursor - from)

	// The arms stand in the blank rows either side of the band.
	head, foot := top-1, top+band
	if head < 0 || at >= len(lines) || at <= foot {
		return lines
	}

	// As far as the picture's own edge. The arms stand in the blank rows above
	// and below it, so they may run its whole width — and the width of the thing
	// they are holding is the one length either of them can be that means
	// something. With the picture folded away there is nothing to measure, and
	// they fall back to a hand's width.
	arm := max(leftMargin+l.artWidth-pointerAt-1, pointerArm)
	if arm >= l.interior-pointerAt-1 {
		arm = l.interior - pointerAt - 2
	}
	if arm < 1 {
		return lines
	}

	// In the accent of the record it is about, which is the cover standing inside
	// it rather than the one sounding — the same colour the name, the artists and
	// the stars beside it are set in. Everything else on the screen wears the
	// sounding record's; this bracket and what it holds are the one part of it
	// that is explicitly about another, and they say so together.
	//
	// It was the border grey, which reads as furniture — a panel's edge rather
	// than something drawn on the screen to answer a question.
	pen := m.coverStyles.Cursor

	// And the arms darken along their length, out into the screen. At one weight
	// the whole way they were two rules the width of the sleeve, which is most of
	// what the closed frame was taken down for; fading them leaves the corner —
	// where the bracket takes hold — and lets the rest go.
	ground := m.ground
	if ground == nil {
		ground = m.styles.Theme.Raised
	}
	var rule string
	for _, step := range style.Fade(m.coverStyles.Accent, ground, arm) {
		rule += step.Render(pointerH)
	}

	out := append([]string(nil), lines...)
	put := func(row int, at int, s string) {
		if row < 0 || row >= len(out) {
			return
		}
		out[row] = overwrite(out[row], at, s, l.interior)
	}

	// The bracket, and nothing else. It was a closed frame with a ground inside
	// it for a while — a step off the screen's, in the hue of the cover — and
	// beside a photograph a shaded panel reads as another window laid over this
	// one, while a rule round all four sides is four lines saying what the one
	// down the left already says.

	put(head, pointerAt, pen.Render(pointerTL)+rule)
	put(foot, pointerAt, pen.Render(pointerTee)+rule)
	for row := head + 1; row < foot; row++ {
		put(row, pointerAt, pen.Render(pointerV))
	}

	// And down the margin to the row it belongs to, where it turns and points.
	for row := foot + 1; row < at; row++ {
		put(row, pointerAt, pen.Render(pointerV))
	}
	// The row's own cursor mark is the head of it: drawing a second arrowhead
	// beside it put two of them on the one row, both saying the same thing.
	put(at, pointerAt, pen.Render(pointerElbow+pointerH))
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
