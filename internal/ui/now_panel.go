package ui

import (
	"strings"
)

// The library's top band is in two halves: what the cursor is resting on, and
// what is playing.
//
// The queue spends that right-hand half on the trace, and the player tab is
// nothing but what is playing, so neither wants this. The library did not have
// it and did not want it either while it was a list of playlists on a narrow
// screen — but at any width worth calling wide, half the band was a blank
// rectangle, and the one thing a music player should never make you leave the
// screen for is what is playing.

// showsNowPanel reports whether that half is on screen. It asks the layout as
// well as the tab: a panel that has to break the title across four lines says
// less than the rows it costs.
func (m Model) showsNowPanel() bool {
	if m.tab != tabLibrary || m.devices.open || m.noDevice || m.ps == nil {
		return false
	}
	w, _ := nowCoverBox(m.layout())
	return w >= minCoverCols
}

// minCoverCols is the narrowest a cover may be drawn and still read as a cover
// rather than as a smudge.
const minCoverCols = 8

// nowPanel draws it: the cover on the left of the half, the words and the
// playhead beside it. The same order as the screen's other half, so the band
// reads across rather than as two unrelated boxes.
func (m Model) nowPanel(l layout, w, rows, foot int) []string {
	// The caption is set in the same rows as the panel on the other half, so
	// the two read as one band rather than as two boxes that happen to be side
	// by side. Those rows end at the foot of the pictures: a line hanging below
	// them reads as a mistake.
	coverW, _ := nowCoverBox(l)
	captionWidth := w - coverW - columnGap
	caption := stack(m.nowCaption(captionWidth), captionWidth, foot)

	// And the picture is centred in the same rows rather than hung from the top
	// of the band. It is half the height of the one on the other half now, and
	// left at the top it floated above its own words with the band's air under
	// it — two things that belong together, arranged as though they did not.
	art := center(strings.Split(m.nowCover.art, "\n"), coverW, min(foot, rows))
	for len(caption) < len(art) {
		caption = append(caption, strings.Repeat(" ", captionWidth))
	}

	out := make([]string, 0, rows)
	gap := strings.Repeat(" ", columnGap)
	for i := range art {
		out = append(out, art[i]+gap+caption[i])
	}
	for len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}
	return out[:rows]
}

// nowCaption is what the panel says about the track: what it is, and how far
// through it the playhead is.
//
// Deliberately less than the panel on the other half says. That one describes
// what is being chosen and wants every fact; this one answers "what is this?"
// while you look at something else, and a column of facts here would compete
// with the list for attention it should not have.
func (m Model) nowCaption(w int) []string {
	if m.ps == nil {
		return nil
	}
	s := m.styles

	// Marked with the note the queue marks the playing row with, rather than
	// labelled "Now playing": the label was two lines, and two lines is what
	// pushed this panel out of line with the one it stands beside. The mark and
	// the moving playhead under it say the same thing in the space of a glyph.
	lines := []string{
		m.styles.Cursor.Render(nowMark) + " " + s.Title.Render(m.ps.Title),
		s.Artist.Render(strings.Join(m.ps.Artists, ", ")),
	}
	if m.ps.Album != "" && m.ps.Album != m.ps.Title {
		lines = append(lines, s.Album.Render(m.ps.Album))
	}

	return append(lines,
		"",
		m.progressLine(w),
		spread(
			s.Time.Render(formatDuration(m.elapsed())),
			s.Time.Render(formatDuration(m.ps.Duration)),
			w,
		),
	)
}
