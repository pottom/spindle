package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// What just happened, in the middle of the screen.
//
// The transport keys work from every tab — space plays, the arrows seek, up and
// down set the volume — and on every tab but one there is nothing to see them
// on. Pressing volume-up while reading a playlist changed a number on a screen
// that was not being drawn, and the only way to know it had worked was to hear
// it, which is exactly what somebody adjusting the volume cannot do yet.
//
// So it is said in the middle of the screen and then it goes, the way a laptop
// says the same things: one thing at a time, over whatever was there, in the
// colour the rest of the program is wearing — which is the record's own. See
// tone.go.

const (
	// osdFor is how long it stands after the last press. A run of volume presses
	// keeps it up: each one starts the clock again, so it disappears a moment
	// after the hand stops rather than in the middle of a turn of the dial.
	osdFor = 1400 * time.Millisecond

	// osdWidth is the box's inside, and osdBar the meter drawn in it. Wide
	// enough that a step of five per cent moves the meter by a cell, and narrow
	// enough to read as a card laid on the screen rather than a screen of its
	// own.
	osdWidth = 23
	osdBar   = osdWidth - 4

	// osdFoot is how many rows are kept clear under it: the help bar, the notice
	// line that stands over it, and a row of air so the card does not read as
	// part of them.
	osdFoot = 3
)

// osdKind is what the card is about.
type osdKind int

const (
	osdNothing osdKind = iota
	osdVolume
	osdPlaying
	osdSeeking
)

// osdState is the card, and when it went up.
type osdState struct {
	kind osdKind
	at   time.Time
}

// osdMsg is the card's own tick: it takes itself down. Without one the card
// would sit there until something else redrew the screen, which on a quiet
// library tab is the next time the poll comes round.
type osdMsg struct{}

// showOSD puts the card up, or keeps it up. The clock starts again on every
// press, so a run of them is one card rather than a flicker of them.
func (m *Model) showOSD(kind osdKind) tea.Cmd {
	m.osd = osdState{kind: kind, at: time.Now()}
	return tea.Tick(osdFor, func(time.Time) tea.Msg { return osdMsg{} })
}

// osdUp reports whether the card is still standing.
func (m Model) osdUp() bool {
	return m.osd.kind != osdNothing && time.Since(m.osd.at) < osdFor
}

// osdOver lays the card over the finished screen.
//
// Over rather than in: it belongs to no tab, it must not move anything, and the
// screen underneath is what the reader was looking at a moment ago and will be
// looking at again in a second. The debug bar does the same thing at the top of
// the screen — see debugOver.
func (m Model) osdOver(screen string) string {
	if !m.osdUp() || m.width == 0 || m.height == 0 {
		return screen
	}

	card := m.osdCard()
	if len(card) == 0 {
		return screen
	}

	lines := strings.Split(screen, "\n")

	// As low as it can go. What the card is about is the sound rather than the
	// screen, and the screen's own middle is where the record is: the artwork on
	// the player, the picture on the big screen, the rows somebody is reading
	// everywhere else. Low is where a laptop puts it, and it is the part of the
	// screen the eye is not using.
	//
	// Never over the last rows, which are the notice and the keys: those are the
	// other things on the screen that answer a keypress, and one of them is
	// often answering the same one.
	top := max(len(lines)-len(card)-osdFoot, 0)
	left := max((m.width-lipgloss.Width(card[0]))/2, 0)

	for i, row := range card {
		at := top + i
		if at >= len(lines) {
			break
		}
		lines[at] = overlay(lines[at], row, left, m.width)
	}
	return strings.Join(lines, "\n")
}

// overlay puts one row of the card into a row of the screen, at a column,
// leaving what is either side of it alone.
//
// A reset before and after: the row underneath was drawn in whatever it was
// wearing, and half a style left open runs on into the card — measured, and the
// card came out in the colour of the row it happened to land on.
func overlay(under, over string, at, w int) string {
	left := ansi.Truncate(under, at, "")
	right := ansi.TruncateLeft(under, at+lipgloss.Width(over), "")

	return left + "\x1b[0m" + over + "\x1b[0m" + right
}

// osdCard is the card itself: what was changed on the left, what it now reads
// on the right, and the meter under them where the thing has one.
//
// Words rather than pictograms. The first draft wore a speaker and a stopwatch,
// which are rare enough code points that a terminal with an ordinary font drew
// nothing at all — reported from a real screen, where the card came up with a
// hole in it. Everything else in this program is set in words and rules, and
// they cannot fail to arrive.
func (m Model) osdCard() []string {
	s := m.styles

	var name, meter, reading string
	switch m.osd.kind {
	case osdVolume:
		volume := m.heldVolume()
		name, meter = "volume", m.osdMeter(volume, 100)
		reading = fmt.Sprintf("%d%%", volume)
		if volume == 0 {
			reading = "muted"
		}

	case osdPlaying:
		name = "playing"
		if m.ps != nil && !m.ps.Playing {
			name = "paused"
		}

	case osdSeeking:
		if m.ps == nil || m.ps.Duration <= 0 {
			return nil
		}
		at := m.playhead()
		name = formatDuration(at)
		reading = formatDuration(m.ps.Duration)
		meter = m.osdMeter(int(at/time.Second), int(m.ps.Duration/time.Second))

	default:
		return nil
	}

	// What is being changed in the record's own colour — which is what the rest
	// of the screen is wearing, see tone.go — and what it now reads beside it,
	// quiet. The same shape the player draws its own bars in: a line with its
	// two ends named above it.
	head := s.Knob.Render(name)
	if reading != "" {
		head = spread(s.Knob.Render(name), s.Detail.Render(reading), osdWidth)
	} else {
		head = middle(head, osdWidth)
	}

	rows := []string{head}
	if meter != "" {
		rows = append(rows, middle(meter, osdWidth))
	}
	return m.osdFrame(rows)
}

// osdMeter is the bar, drawn the way every other bar in the program is: what is
// set in the record's own colour, the rest of the way faint. One shape for one
// kind of reading — see progressLine, which this is deliberately a copy of.
func (m Model) osdMeter(at, of int) string {
	if of <= 0 {
		return ""
	}

	// The knob rides on the join, as it does on the player's own bars: without
	// one, a reading near either end is a bar with nothing in it, which says
	// "none" rather than "a little".
	bar := barCells(osdBar)
	filled := min(max(at*bar/of, 0), bar)
	return m.styles.Elapsed.Render(strings.Repeat(meterFull, filled)) +
		m.styles.Knob.Render(knob) +
		m.styles.Remaining.Render(strings.Repeat(meterEmpty, bar-filled))
}

// osdFrame draws the box round it, in the same pen as every other box on the
// screen. See menu.go.
func (m Model) osdFrame(rows []string) []string {
	edge := m.styles.Rule

	out := make([]string, 0, len(rows)+2)
	out = append(out, edge.Render("╭"+strings.Repeat("─", osdWidth+2)+"╮"))
	for _, row := range rows {
		out = append(out, edge.Render("│")+" "+row+" "+edge.Render("│"))
	}
	out = append(out, edge.Render("╰"+strings.Repeat("─", osdWidth+2)+"╯"))
	return out
}

// middle puts one line in the middle of a width it is narrower than. Its
// sibling center does the same for a block of them — see frame.go.
func middle(line string, w int) string {
	pad := w - lipgloss.Width(line)
	if pad <= 0 {
		return line
	}
	left := pad / 2
	return strings.Repeat(" ", left) + line + strings.Repeat(" ", pad-left)
}
