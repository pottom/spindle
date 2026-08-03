package ui

import (
	"fmt"
	"strings"
	"time"

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

	// volumeCells is the width of the little bar beside the volume reading.
	volumeCells = 8
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

	lines := m.body(l)
	if m.err != nil {
		lines = append(lines, m.pad(m.styles.Error.Render("✕ "+m.err.Error()), l))
	}
	for _, row := range strings.Split(m.help.View(m.keys), "\n") {
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
	if m.ps == nil {
		block = []string{m.styles.Detail.Render("Connecting…")}
	} else {
		art := m.artwork(l)
		info := stack(m.infoBlock(l.infoWidth), l.infoWidth, l.artHeight)
		gap := strings.Repeat(" ", columnGap)

		block = make([]string, len(art))
		for i := range art {
			block[i] = art[i] + gap + info[i]
		}
	}

	lines := make([]string, 0, l.bodyHeight)
	top := max((l.bodyHeight-1-len(block))/2, 0)
	for range top {
		lines = append(lines, m.pad("", l))
	}
	for _, row := range block {
		lines = append(lines, m.pad(row, l))
	}
	for len(lines) < l.bodyHeight-1 {
		lines = append(lines, m.pad("", l))
	}
	if len(lines) < l.bodyHeight {
		lines = append(lines, m.pad(m.statusLine(), l))
	}
	return lines[:l.bodyHeight]
}

// artwork is the cover area: the picture itself, a spinner while it downloads,
// or a single note glyph when there is none. The area is reserved whatever it
// holds, so nothing moves when a cover finishes loading.
func (m Model) artwork(l layout) []string {
	switch {
	case m.cover.art != "":
		return center(strings.Split(m.cover.art, "\n"), l.artWidth, l.artHeight)
	case m.cover.loading():
		return center([]string{m.spinner.View()}, l.artWidth, l.artHeight)
	default:
		return center([]string{m.styles.NoCover.Render(noCoverGlyph)}, l.artWidth, l.artHeight)
	}
}

// infoBlock is the text beside the artwork, as a compact run of lines. The
// caller centres it against the cover rather than stretching it, so the two
// columns read as one composition instead of two edges with a hole between them.
func (m Model) infoBlock(w int) []string {
	s, ps := m.styles, m.ps

	return []string{
		s.Title.Render(ps.Title),
		s.Artist.Render(strings.Join(ps.Artists, ", ")),
		s.Album.Render(ps.Album),
		"",
		"",
		m.progressLine(w),
		spread(
			s.Time.Render(formatDuration(m.localProgress)),
			s.Time.Render(formatDuration(ps.Duration)),
			w,
		),
		"",
		"",
		m.transportLine(w),
	}
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
		fraction = min(float64(m.localProgress)/float64(m.ps.Duration), 1)
	}

	// The playhead takes a cell of its own, so the bar is one shorter.
	bar := max(w-1, 1)
	filled := min(max(int(fraction*float64(bar)+0.5), 0), bar)
	return elapsed.Render(strings.Repeat(meterFull, filled)) +
		remaining.Render(knob) +
		m.styles.Remaining.Render(strings.Repeat(meterEmpty, bar-filled))
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

	volume := meter(float64(m.ps.Volume)/100, volumeCells, s.MeterOn, s.MeterOff) +
		s.Volume.Render(fmt.Sprintf(" %d", m.ps.Volume))

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
	return device.Render(deviceDot + " " + m.ps.DeviceName)
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
func (m Model) helpHeight() int {
	return lipgloss.Height(m.help.View(m.keys))
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
