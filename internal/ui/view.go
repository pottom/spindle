package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/build"
	"github.com/pottom/spindle/internal/player"
)

// Glyphs. The screen carries no box drawing: hierarchy comes from colour and
// space, so the only line work is the progress and volume meters.
const (
	iconPrev  = "⏮"
	iconPlay  = "⏵"
	iconPause = "⏸"
	iconNext  = "⏭"
	iconShuf  = "⇄"
	iconRep   = "↻"

	meterFull  = "━"
	meterEmpty = "─"
	knob       = "●"
	deviceDot  = "●"

	// noCoverGlyph stands in for artwork that could not be loaded.
	noCoverGlyph = "♪"

	// volumeCells is the width of the bar beside the volume reading. Long
	// enough that a step of five is a step you can see.
	volumeCells = 16
)

func (m Model) View() tea.View {
	// The other half of a frame's timing: what the update decided, and what it
	// cost to draw. See slow.go.
	began := time.Now()
	v := tea.NewView(m.debugOver(m.render()))
	slowRenderDone(m, time.Since(began))

	// And, while the bar is up, the same numbers to a file. See debug.go.
	m.debugNote()

	v.AltScreen = true
	v.WindowTitle = m.windowTitle()

	// Where the pointer is and what it presses, in cells. Not every movement of
	// it: nothing on this screen follows the pointer about, and a message a
	// frame for something nobody reads is a cost with no answer to it.
	//
	// This is what takes the terminal's own drag-to-select away, which is worth
	// saying out loud because it is the price of the whole thing. Shift held
	// down gives it back, on every terminal this is drawn on. See mouse.go.
	v.MouseMode = tea.MouseModeCellMotion

	// Ask the terminal to say which key a press came from as well as which
	// letter it produced, so a binding written as a letter is the key that
	// letter sits on wherever somebody is typing. See keypress.go.
	//
	// Three things rather than one, because the first alone changed nothing. A
	// key that produces text is sent as that text, and the key it came from
	// travels only with the escape-coded form — so every key has to be asked for
	// that way, and the text asked for alongside it or a query would be typed in
	// whatever a US keyboard has there. Measured on a Hungarian layout: the
	// terminal agreed to report alternate keys and then reported none, because
	// nothing was ever sent in the form that carries them.
	//
	// Terminals that cannot do this ignore the request, and everything is as it
	// was: the letter that arrived, matched as itself.
	v.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportAlternateKeys:        true,
		ReportAllKeysAsEscapeCodes: true,
		ReportAssociatedText:       true,
	}
	return v
}

func (m Model) windowTitle() string {
	if m.ps == nil || m.ps.Title == "" {
		return "spindle"
	}
	return m.ps.Title + " · " + strings.Join(m.ps.Artists, ", ")
}

func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if !fitsMinimum(m.width, m.height) {
		return m.renderTooSmall()
	}
	// The big screen answers before anything else is laid out: it is not a
	// panel inside the player, it is instead of it.
	if m.stage.on {
		if art := m.stageView(); art != "" {
			return art
		}
	}
	return m.renderPlayer()
}

func (m Model) renderPlayer() string {
	l := m.layout()

	inner := l.interior - leftMargin - rightMargin

	// Outlined after the padding, not before, and so at the frame's width rather
	// than the content's. Every block on the screen is inset by the same margin,
	// so bordering one inside it and the next outside it drew two edges a few
	// columns apart that were saying the same thing.
	var header []string
	for _, row := range m.header(inner) {
		header = append(header, m.pad(row, l))
	}

	lines := m.outline(header, l.interior, "header")
	lines = append(lines, m.pad("", l))

	body := m.body(l)
	// The queue draws its trace inside its own top block, beside the detail.
	if m.tab == tabPlayer && m.scopeVisible() {
		body = m.drawScope(body, l)
	}
	lines = append(lines, m.outline(body, l.interior, "body")...)

	// After the body is in place, because the glance starts in the blank row
	// above it and so needs the frame rather than the body.
	if m.peekVisible() {
		lines = m.drawPeek(lines, len(lines)-len(body), l)
	}

	// And for the same reason: where the band at the top belongs to a different
	// record from the one the list begins with, the frame and the line that say
	// so stand in the blank rows either side of it. See pointer.go.
	lines = m.pointAtCursor(lines, l, len(lines)-len(body))

	// And the field a list is searched in, over the head of the list itself.
	// Last, because it is the thing being typed into: nothing on the screen may
	// be drawn over the letters going in. See finder.go.
	lines = m.drawFinder(lines, l, len(lines)-len(body))

	// Except the menu, which is what is being answered while it is up and so
	// stands over everything, the field included. See menu.go.
	lines = m.drawMenu(lines, l)

	// A blank row before the bottom block, so the help never reads as one more
	// entry in whatever list ends above it.
	lines = append(lines, m.pad("", l))
	if text, style, ok := m.notice(); ok {
		lines = append(lines, m.pad(style.Render(text), l))
	}
	var help []string
	for _, row := range strings.Split(m.help.View(m.helpKeys()), "\n") {
		help = append(help, m.pad(row, l))
	}
	lines = append(lines, m.outline(help, l.interior, "help")...)

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, strings.Join(lines, "\n"))
}

// pad indents one line to the left margin and squares it off to the content
// width, so every row of the screen is the same length.
func (m Model) pad(s string, l layout) string {
	return fit(strings.Repeat(" ", leftMargin)+s, l.interior)
}

// body is everything above the help bar: the artwork beside the track
// information, centred in what is left once the status line has its row.
func (m Model) body(l layout) []string {
	var pane []string
	switch {
	// The screen that stands in for the player when nothing is playing anywhere.
	// Not while the picker is up over it: that box is the same list, and two of
	// them is one too many.
	case m.tab == tabPlayer && m.noDevice && !m.devices.open:
		pane = m.noDevicePanel(l, max(l.bodyHeight-1, 1))

	case m.tab == tabPlayer && m.ps == nil:
		// The picture, while there is nothing else to look at. See splash.go.
		pane = append(m.splashRows(), "", m.styles.Detail.Render("Connecting…"))

	case m.tab == tabSettings:
		pane = m.settingsPanel(l, max(l.bodyHeight, 1))

	case m.tab == tabHelp:
		pane = m.helpPanel(l, max(l.bodyHeight, 1))

	case m.open() != nil:
		pane = m.openPageView(l, max(l.bodyHeight, 1))

	case m.tab == tabQueue:
		pane = m.queueBlock(l, max(l.bodyHeight, 1))

	case m.tab == tabLibrary:
		pane = m.libraryPaneView(l, max(l.bodyHeight, 1))

	case m.tab == tabSearch:
		pane = m.searchPaneView(l, max(l.bodyHeight, 1))

	default:
		rows := m.playerPaneRows(l)

		var right []string
		switch {
		case m.lyricsVisible():
			// The words need room, so the information goes to the top of the
			// body rather than sitting in the middle of it, and they take
			// everything under it.
			// Outlined inside, as "player" and "lyrics": one border round both
			// would say less than the two say separately.
			// Outlined inside, as "player" and "lyrics": one border round both
			// would say less than the two say separately.
			right = m.infoWithLyrics(l, rows)
		default:
			right = m.place(m.playerBlock(), l.infoWidth, rows)
		}

		// Without a picture the text and the list take the whole width, which
		// is the point of dropping it.
		if !l.hasArt() {
			pane = right
			break
		}

		// The picture stays where it is. It is laid out in its own area first
		// and only then padded into whatever the pane turns out to be, so
		// showing the words moves the column beside it and nothing else —
		// centring it in the taller pane instead lands a row out, because two
		// halvings round differently from one. The browsing tabs align it to
		// the top, where it heads a list rather than sitting beside a caption.
		// The player centres its cover twice: once in the box the layout gave it,
		// then again in the taller pane, so showing the words moves the column
		// beside it and nothing else.
		var art []string
		if m.tab == tabPlayer {
			art = m.place(block{"art", func(w, h int) []string {
				return center(center(strings.Split(m.artworkCells(), "\n"), w, l.artHeight), w, h)
			}}, l.artWidth, rows)
		} else {
			art = m.place(m.artBlock(false), l.artWidth, rows)
		}
		gap := strings.Repeat(" ", columnGap)

		pane = make([]string, rows)
		for i := range art {
			pane[i] = art[i] + gap + right[i]
		}
	}

	lines := make([]string, 0, l.bodyHeight)
	top := max((l.bodyHeight-len(pane))/2, 0)
	for range top {
		lines = append(lines, m.pad("", l))
	}
	for _, row := range pane {
		lines = append(lines, m.pad(row, l))
	}
	for len(lines) < l.bodyHeight {
		lines = append(lines, m.pad("", l))
	}
	return lines[:l.bodyHeight]
}

// playerPaneRows is the height the column beside the picture is laid out in.
//
// The player centres its text against the cover. The browsing tabs, and the
// player once the words are showing, give the right-hand column every row there
// is instead.
//
// Read by the body to draw it and by the pointer to find the transport in it —
// neither may work it out for itself, or a click lands a row off centre on
// exactly the terminals where the two halvings round differently.
func (m Model) playerPaneRows(l layout) int {
	if m.tab != tabPlayer || !l.hasArt() || m.lyricsVisible() {
		return max(l.bodyHeight, l.artHeight)
	}
	return l.artHeight
}

// artworkCells is the cover: the picture itself, a spinner while it downloads,
// or a single note glyph when there is none. The area around it is reserved
// whatever it holds, so nothing moves when a cover finishes loading.
func (m Model) artworkCells() string {
	switch {
	case m.cover.art != "":
		return m.cover.art
	case m.cover.loading():
		return m.spinner.View()
	default:
		return m.styles.NoCover.Render(noCoverGlyph)
	}
}

// The two rows of the block a pointer can act on, counted back from its foot.
//
// From the foot because the head of the block is as tall as the caption needs,
// and the caption is a different height for a track, an episode and a thing
// with no album. Counted from the end, the last four rows are the same rows
// whatever is above them.
//
// A number written here and a row written in infoBlock are two records of one
// thing, and the way that is kept true is a test rather than a promise:
// TestTheBlockPutsTheBarAndTheTransportWhereTheyAreNamed.
const (
	playerTransportUp = 0
	playerBarUp       = 4
)

// infoBlock is the text beside the artwork, as a compact run of lines. The
// caller centres it against the cover rather than stretching it, so the two
// columns read as one composition instead of two edges with a hole between them.
func (m Model) infoBlock(w int) []string {
	s, ps := m.styles, m.ps

	// The heart, where this one is in the collection. Beside the name, which is
	// where the row in a list carries it too, and where the eye already is: a key
	// that changes something invisible is a key nobody trusts. See collect.go.
	title := s.Title.Render(ps.Title)
	if m.library.saved(ps.TrackID) {
		title += "  " + m.styles.Queued.Render(likedMark)
	}

	lines := []string{
		title,
		s.Artist.Render(strings.Join(ps.Artists, ", ")),
	}
	if t, ok := m.nowPlayingRow(); ok {
		lines = append(lines, m.trackCaption(t, w)...)
	} else {
		lines = append(lines, s.Album.Render(ps.Album))
	}

	// A device with nothing on it has nothing to say about where it has got to.
	// Every other player leaves this blank rather than drawing a bar at nought
	// against a length of nought — and a clock that runs anyway, over a track
	// that is not there, says something that is not true. The transport stays:
	// pressing play is how something gets loaded.
	if !m.loaded() {
		return append(lines, "", "", "", "", "", "", m.transportLine(w))
	}

	return append(lines,
		"",
		"",
		m.progressLine(w),
		spread(
			// The same reading the bar is drawn from, or the clock and the bar
			// would say two different things about the same drag.
			s.Time.Render(formatDuration(m.playhead())),
			s.Time.Render(formatDuration(ps.Duration)),
			w,
		),
		"",
		"",
		m.transportLine(w),
	)
}

// barCells is how much of a row of that width the bar itself is: the playhead
// takes a cell of its own, so the bar is one shorter.
//
// Both meters are drawn from it and a press on either is measured back through
// it, so where the pointer landed is the fraction the bar would have been drawn
// at. See atShare.
func barCells(w int) int { return max(w-1, 1) }

// progressLine is a thin rule with the playhead riding on it. Paused, the whole
// thing goes grey: the state has to be readable without hunting for an icon.
func (m Model) progressLine(w int) string {
	elapsed, remaining := m.styles.Elapsed, m.styles.Knob
	if !m.ps.Playing {
		elapsed, remaining = m.styles.Time, m.styles.Time
	}

	// Where the pointer has dragged it to, while it has hold of it, and where
	// the track really is otherwise. See drag.go.
	var fraction float64
	if m.ps.Duration > 0 {
		fraction = min(float64(m.playhead())/float64(m.ps.Duration), 1)
	}

	bar := barCells(w)
	filled := min(max(int(fraction*float64(bar)+0.5), 0), bar)
	return elapsed.Render(strings.Repeat(meterFull, filled)) +
		remaining.Render(knob) +
		m.styles.Remaining.Render(strings.Repeat(meterEmpty, bar-filled))
}

// volumeLine is the volume drawn like the progress bar: what is set in the
// artwork's accent, the rest of the way faint, and the playhead on the join.
// One control, one shape — a meter of its own beside it read as a different
// kind of thing entirely.
func (m Model) volumeLine(w int) string {
	// Paused, the whole thing goes grey, exactly as the progress bar does: the
	// state has to be readable without hunting for an icon, and one control
	// answering it differently from the other would undo that.
	filled, marker := m.styles.Elapsed, m.styles.Knob
	if !m.ps.Playing {
		filled, marker = m.styles.Time, m.styles.Time
	}

	bar := barCells(w)
	at := min(max(m.heldVolume()*bar/100, 0), bar)

	return filled.Render(strings.Repeat(meterFull, at)) +
		marker.Render(knob) +
		m.styles.Remaining.Render(strings.Repeat(meterEmpty, bar-at))
}

// control names one of the things on the transport row, in the order they are
// drawn. A click has to say which one it landed on, and so does the row itself.
type control int

const (
	ctlPrev control = iota
	ctlPlay
	ctlNext
	ctlShuffle
	ctlRepeat

	ctlCount = iota
)

// controlAir is the space kept after each of them. The last has none: what
// follows it is the whole width of the row.
var controlAir = [ctlCount]int{ctlPrev: 3, ctlPlay: 3, ctlNext: 4, ctlShuffle: 2}

// controlGlyphs is what each of them is drawn as, in that order.
//
// The two toggles keep a fixed two-cell slot, so turning one on cannot nudge the
// rest of the row sideways.
func (m Model) controlGlyphs() [ctlCount]string {
	s := m.styles

	playPause := iconPause
	if !m.ps.Playing {
		playPause = iconPlay
	}

	shuffle := s.ToggleOff.Render(iconShuf + " ")
	if m.ps.Shuffle {
		shuffle = s.ToggleOn.Render(iconShuf + " ")
	}
	repeat := s.ToggleOff.Render(iconRep + " ")
	switch m.ps.Repeat {
	case player.RepeatContext:
		repeat = s.ToggleOn.Render(iconRep + " ")
	case player.RepeatTrack:
		repeat = s.ToggleOn.Render(iconRep + "1")
	}

	return [ctlCount]string{
		ctlPrev:    s.Controls.Render(iconPrev),
		ctlPlay:    s.Controls.Render(playPause),
		ctlNext:    s.Controls.Render(iconNext),
		ctlShuffle: shuffle,
		ctlRepeat:  repeat,
	}
}

// controlSpans is where those glyphs end up on the row, for a click to be
// answered: the same glyphs and the same air, walked instead of written.
func (m Model) controlSpans() []span {
	glyphs := m.controlGlyphs()
	out := make([]span, ctlCount)
	at := 0
	for i, glyph := range glyphs {
		w := lipgloss.Width(glyph)
		out[i] = span{at: at, w: w}
		at += w + controlAir[i]
	}
	return out
}

// volumeReading is the number beside the meter. It holds three columns whatever
// it says: the row is laid out from the right, so a number that narrows would
// drag the bar along with it.
func (m Model) volumeReading() string {
	// The held value while the meter is held, for the same reason the clock
	// beside the playhead shows the dragged position: a number that disagreed
	// with the bar it is written against is worse than no number.
	return m.styles.Volume.Render(fmt.Sprintf(" %3d", m.heldVolume()))
}

// volumeSpan is where the meter itself sits on a transport row of that width:
// hard against the right edge, with only the reading after it.
func (m Model) volumeSpan(w int) span {
	return span{at: w - volumeCells - lipgloss.Width(m.volumeReading()), w: volumeCells}
}

// transportLine holds the transport icons, the shuffle and repeat state, and the
// volume, all on one row.
func (m Model) transportLine(w int) string {
	var b strings.Builder
	for i, glyph := range m.controlGlyphs() {
		b.WriteString(glyph)
		b.WriteString(strings.Repeat(" ", controlAir[i]))
	}
	return spread(b.String(), m.volumeLine(volumeCells)+m.volumeReading(), w)
}

// statusLine names the device on the left and leaves the right to whatever the
// screen still owes the user.
func (m Model) statusLine() string {
	if m.ps == nil || m.ps.DeviceName == "" {
		return ""
	}

	device := m.styles.DeviceOff
	if m.ps.Playing {
		device = m.styles.DeviceOn
	}
	// Stopped is a state, not a moment, so the mark stops turning and settles
	// back into a plain dot rather than freezing mid-rotation.
	mark := deviceDot
	if m.ps.Playing {
		mark = m.device.View()
	}
	line := device.Render(mark + " " + m.ps.DeviceName)

	// Which build this is, beside the name. The version exists to settle "am I
	// looking at the fix or at the binary from before it", and the header is
	// where the eye already goes.
	line += m.styles.Quality.Render(" " + build.Version())

	// The bitrate is the stream actually arriving, so it only appears once one
	// is: a rate printed over a paused device would be describing nothing.
	if m.ps.Playing && m.ps.Bitrate > 0 {
		line += m.styles.Quality.Render(fmt.Sprintf("  %d kbps", m.ps.Bitrate))
	}
	// The tempo is measured from the audio, so it arrives a few seconds into a
	// track and is absent from anything without a steady beat.
	if m.ps.Playing && m.ps.Tempo > 0 {
		line += m.styles.Quality.Render(fmt.Sprintf("  %.0f bpm", m.ps.Tempo))
	}
	return line
}

func (m Model) renderTooSmall() string {
	lines := []string{
		m.styles.Heading.Render("Window too small"),
		"",
		m.styles.Detail.Render(fmt.Sprintf("current   %d × %d", m.width, m.height)),
		m.styles.Detail.Render(fmt.Sprintf("needed    %d × %d", minWidth, minHeight)),
	}
	if m.width < 26 {
		lines = []string{m.styles.Heading.Render(fmt.Sprintf("need %dx%d", minWidth, minHeight))}
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, strings.Join(lines, "\n"))
}

// helpHeight is how many lines the help bar currently occupies.
// helpHeight is how many rows the help bar takes. It asks without the waveform
// key: the layout decides whether that key is offered, and the layout needs
// this number, so it cannot depend on the answer. The bar is the same height
// either way, which TestHelpHeightDoesNotDependOnTheScope keeps true.
func (m Model) helpHeight() int {
	return lipgloss.Height(m.help.View(m.helpKeysWith(false, false, false)))
}

// formatDuration renders a position as m:ss, or h:mm:ss past an hour.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second).Seconds())
	h, mn, sec := total/3600, total/60%60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, mn, sec)
	}
	return fmt.Sprintf("%d:%02d", mn, sec)
}

// deviceSpinner is the mark beside the device name: a quartered circle turning
// once a second. It carries no style of its own, so the status line can colour
// it along with the name.
var deviceSpinner = spinner.Spinner{
	Frames: []string{"◐", "◓", "◑", "◒"},
	FPS:    time.Second / 8,
}
