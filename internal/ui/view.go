package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = m.windowTitle()
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
	return m.renderPlayer()
}

func (m Model) renderPlayer() string {
	l := m.layout()

	var lines []string
	for _, row := range m.header(l.interior - leftMargin - rightMargin) {
		lines = append(lines, m.pad(row, l))
	}
	lines = append(lines, m.pad("", l))

	body := m.body(l)
	if m.scopeVisible() {
		body = m.drawScope(body, l)
	}
	if m.peekVisible() {
		body = m.drawPeek(body, l)
	}
	lines = append(lines, body...)

	// A blank row before the bottom block, so the help never reads as one more
	// entry in whatever list ends above it.
	lines = append(lines, m.pad("", l))
	if text, style, ok := m.notice(); ok {
		lines = append(lines, m.pad(style.Render(text), l))
	}
	for _, row := range strings.Split(m.help.View(m.helpKeys()), "\n") {
		lines = append(lines, m.pad(row, l))
	}

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
	var block []string
	switch {
	case m.tab == tabPlayer && m.noDevice:
		block = m.noDevicePanel(l, max(l.bodyHeight-1, 1))

	case m.tab == tabPlayer && m.ps == nil:
		block = []string{m.styles.Detail.Render("Connecting…")}

	case m.tab == tabQueue:
		block = m.queueBlock(l, max(l.bodyHeight, 1))

	default:
		// The player centres its text against the cover. The browsing tabs, and
		// the player once the words are showing, give the right-hand column
		// every row there is instead.
		rows := l.artHeight
		full := m.tab != tabPlayer || !l.hasArt() || m.lyricsVisible()
		if full {
			rows = max(l.bodyHeight, l.artHeight)
		}

		right := m.browsePane(l, rows)
		switch {
		case m.devices.open:
			right = m.devicePicker(l.infoWidth, rows)
		case right == nil && m.lyricsVisible():
			// The words need room, so the information goes to the top of the
			// body rather than sitting in the middle of it, and they take
			// everything under it.
			right = m.infoWithLyrics(l, rows)
		case right == nil:
			right = stack(m.infoBlock(l.infoWidth), l.infoWidth, rows)
		}

		// Without a picture the text and the list take the whole width, which
		// is the point of dropping it.
		if !l.hasArt() {
			block = right
			break
		}

		// The picture stays where it is. It is laid out in its own area first
		// and only then padded into whatever the block turns out to be, so
		// showing the words moves the column beside it and nothing else —
		// centring it in the taller block instead lands a row out, because two
		// halvings round differently from one. The browsing tabs align it to
		// the top, where it heads a list rather than sitting beside a caption.
		cells := strings.Split(m.artworkCells(), "\n")
		var art []string
		if m.tab == tabPlayer {
			art = center(center(cells, l.artWidth, l.artHeight), l.artWidth, rows)
		} else {
			art = alignTop(cells, l.artWidth, rows)
		}
		gap := strings.Repeat(" ", columnGap)

		block = make([]string, rows)
		for i := range art {
			block[i] = art[i] + gap + right[i]
		}
	}

	lines := make([]string, 0, l.bodyHeight)
	top := max((l.bodyHeight-len(block))/2, 0)
	for range top {
		lines = append(lines, m.pad("", l))
	}
	for _, row := range block {
		lines = append(lines, m.pad(row, l))
	}
	for len(lines) < l.bodyHeight {
		lines = append(lines, m.pad("", l))
	}
	return lines[:l.bodyHeight]
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

// infoBlock is the text beside the artwork, as a compact run of lines. The
// caller centres it against the cover rather than stretching it, so the two
// columns read as one composition instead of two edges with a hole between them.
func (m Model) infoBlock(w int) []string {
	s, ps := m.styles, m.ps

	lines := []string{
		s.Title.Render(ps.Title),
		s.Artist.Render(strings.Join(ps.Artists, ", ")),
	}
	if t, ok := m.nowPlayingRow(); ok {
		lines = append(lines, m.trackCaption(t, w)...)
	} else {
		lines = append(lines, s.Album.Render(ps.Album))
	}

	return append(lines,
		"",
		"",
		m.progressLine(w),
		spread(
			s.Time.Render(formatDuration(m.elapsed())),
			s.Time.Render(formatDuration(ps.Duration)),
			w,
		),
		"",
		"",
		m.transportLine(w),
	)
}

// progressLine is a thin rule with the playhead riding on it. Paused, the whole
// thing goes grey: the state has to be readable without hunting for an icon.
func (m Model) progressLine(w int) string {
	elapsed, remaining := m.styles.Elapsed, m.styles.Knob
	if !m.ps.Playing {
		elapsed, remaining = m.styles.Time, m.styles.Time
	}

	var fraction float64
	if m.ps.Duration > 0 {
		fraction = min(float64(m.elapsed())/float64(m.ps.Duration), 1)
	}

	// The playhead takes a cell of its own, so the bar is one shorter.
	bar := max(w-1, 1)
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

	bar := max(w-1, 1)
	at := min(max(m.ps.Volume*bar/100, 0), bar)

	return filled.Render(strings.Repeat(meterFull, at)) +
		marker.Render(knob) +
		m.styles.Remaining.Render(strings.Repeat(meterEmpty, bar-at))
}

// transportLine holds the transport icons, the shuffle and repeat state, and the
// volume, all on one row.
func (m Model) transportLine(w int) string {
	s := m.styles

	playPause := iconPause
	if !m.ps.Playing {
		playPause = iconPlay
	}
	transport := s.Controls.Render(iconPrev + "   " + playPause + "   " + iconNext)

	// Both toggles keep a fixed two-cell slot, so turning one on cannot nudge
	// the rest of the row sideways.
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

	// The reading holds three columns whatever it says: the row is laid out
	// from the right, so a number that narrows drags the bar along with it.
	volume := m.volumeLine(volumeCells) + s.Volume.Render(fmt.Sprintf(" %3d", m.ps.Volume))

	return spread(transport+"    "+shuffle+"  "+repeat, volume, w)
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
