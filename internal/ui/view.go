package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// barRune fills both halves of the progress bar; only the colour differs between
// the played and the remaining part.
const barRune = '━'

// Transport glyphs. SCREENS.md 4.1.
const (
	iconPrev  = "⏮"
	iconPlay  = "⏵"
	iconPause = "⏸"
	iconNext  = "⏭"
	deviceDot = "●"
)

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
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
	l := computeLayout(m.width, m.height, m.helpHeight(), m.err != nil)
	f := frame{styles: m.styles, width: l.frameWidth}

	lines := []string{f.top(m.deviceLabel(), m.deviceStyle())}
	for _, row := range m.body(l) {
		lines = append(lines, f.row(row))
	}
	if m.err != nil {
		lines = append(lines, f.separator())
		lines = append(lines, f.row(" "+m.styles.Error.Render("✕ "+m.err.Error())))
	}
	lines = append(lines, f.separator())
	for _, row := range strings.Split(m.help.View(m.keys), "\n") {
		lines = append(lines, f.row(" "+row))
	}
	lines = append(lines, f.bottom())

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, strings.Join(lines, "\n"))
}

// body returns exactly bodyHeight interior lines: the artwork box beside the
// track information, with the leftover height below it.
func (m Model) body(l layout) []string {
	var block string
	if m.ps == nil {
		block = m.spinner.View() + " " + m.styles.Detail.Render("Connecting…")
	} else {
		cover := lipgloss.NewStyle().
			Border(m.styles.Border).
			BorderForeground(m.styles.Theme.Border).
			Width(coverBoxWidth).
			Height(coverBoxHeight).
			MarginLeft(leftMargin).
			Render("")
		info := lipgloss.NewStyle().
			MarginLeft(columnGap).
			Render(strings.Join(m.infoColumn(l.infoWidth), "\n"))
		block = lipgloss.JoinHorizontal(lipgloss.Top, cover, info)
	}

	// A blank line above the block, but only while there is height to spare:
	// when the expanded help squeezes the player, the artwork matters more.
	lines := strings.Split(block, "\n")
	if len(lines) < l.bodyHeight {
		lines = append([]string{""}, lines...)
	}
	for len(lines) < l.bodyHeight {
		lines = append(lines, "")
	}
	return lines[:l.bodyHeight]
}

// infoColumn is the right-hand column, always coverBoxHeight lines tall so the
// layout cannot jump when a field is empty.
func (m Model) infoColumn(w int) []string {
	s := m.styles
	ps := m.ps

	rows := make([]string, coverBoxHeight)
	rows[0] = s.Title.Render(ps.Title)
	rows[1] = s.Artist.Render(strings.Join(ps.Artists, ", "))
	rows[2] = s.Album.Render(ps.Album)
	rows[5] = m.progressBar(w)
	rows[6] = m.timeRow(w)
	rows[9] = m.transportRow()
	rows[10] = padLeft(s.Volume.Render(fmt.Sprintf("vol %d", ps.Volume)), w)

	for i, row := range rows {
		rows[i] = fit(row, w)
	}
	return rows
}

func (m Model) progressBar(w int) string {
	bar := m.progress
	bar.SetWidth(w)
	bar.EmptyColor = m.styles.Theme.Border
	bar.FullColor = m.styles.Theme.Accent
	if !m.ps.Playing {
		bar.FullColor = m.styles.Theme.Muted
	}

	var percent float64
	if m.ps.Duration > 0 {
		percent = min(float64(m.localProgress)/float64(m.ps.Duration), 1)
	}
	return bar.ViewAs(percent)
}

func (m Model) timeRow(w int) string {
	left := m.styles.Time.Render(formatDuration(m.localProgress))
	right := m.styles.Time.Render(formatDuration(m.ps.Duration))
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) transportRow() string {
	s := m.styles
	playPause := iconPause
	if !m.ps.Playing {
		playPause = iconPlay
	}

	transport := s.Controls.Render(iconPrev + "  " + playPause + "  " + iconNext)
	return transport +
		s.Rule.Render(" "+s.Border.Left+" ") +
		m.toggle("shuf", m.ps.Shuffle) + "  " +
		m.toggle("rep", m.ps.Repeat != player.RepeatOff)
}

// toggle renders a shuffle or repeat label: Accent when on, Faint when off.
func (m Model) toggle(label string, on bool) string {
	text := label + " off"
	if on {
		text = label + " on"
	}
	if label == "rep" && m.ps.Repeat != player.RepeatOff {
		text = label + " " + m.ps.Repeat
	}
	if on {
		return m.styles.ToggleOn.Render(text)
	}
	return m.styles.ToggleOff.Render(text)
}

func (m Model) deviceLabel() string {
	if m.ps == nil || m.ps.DeviceName == "" {
		return ""
	}
	return deviceDot + " " + m.ps.DeviceName
}

func (m Model) deviceStyle() lipgloss.Style {
	if m.ps != nil && m.ps.Playing {
		return m.styles.DeviceOn
	}
	return m.styles.DeviceOff
}

func (m Model) renderTooSmall() string {
	if m.width < tooSmallMinWidth || m.height < tooSmallBoxHeight {
		return m.styles.Heading.Render(fmt.Sprintf("need %dx%d", minWidth, minHeight))
	}

	lines := []string{
		"",
		"  " + m.styles.Heading.Render("Window too small"),
		"",
		"  " + m.styles.Detail.Render(fmt.Sprintf("current:  %d × %d", m.width, m.height)),
		"  " + m.styles.Detail.Render(fmt.Sprintf("needed:   %d × %d", minWidth, minHeight)),
		"",
	}
	box := lipgloss.NewStyle().
		Border(m.styles.Border).
		BorderForeground(m.styles.Theme.Border).
		Width(min(tooSmallBoxWidth, m.width)).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
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
