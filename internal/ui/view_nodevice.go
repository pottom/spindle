package ui

import "github.com/pottom/spindle/internal/player"

// deviceListCols keeps the device names and their kinds within reading distance
// of each other on a wide terminal.
const deviceListCols = 52

// noDevicePanel is the player tab when nothing is playing anywhere. It is the
// most common way to arrive at spindle, so it explains rather than complains —
// including why the list below it may be empty, which otherwise looks exactly
// like a bug.
func (m Model) noDevicePanel(l layout, rows int) []string {
	s := m.styles

	lines := []string{
		s.Title.Render("No active playback device"),
		"",
		s.Detail.Render("Start Spotify on one of your devices, or pick one below."),
		"",
	}

	// The list reads as a list, not as two columns pinned to opposite edges of
	// the screen, so it gets a column of its own rather than the full width.
	w := l.interior - leftMargin - rightMargin
	listWidth := min(w, deviceListCols)

	if len(m.devices) == 0 {
		lines = append(lines,
			s.Empty.Render("No devices reported."),
			"",
			"",
			s.Empty.Render("Spotify only lists devices that were recently active. Open the"),
			s.Empty.Render("app on one of them and play something for a moment."),
		)
	} else {
		for _, d := range m.devices {
			lines = append(lines, m.deviceRow(d, listWidth))
		}
		lines = append(lines,
			"",
			"",
			s.Empty.Render("Missing one? Spotify only lists devices that were recently"),
			s.Empty.Render("active — open the app on it and play something for a moment."),
		)
	}

	return stack(lines, w, rows)
}

// deviceRow names a device and its kind, marking whichever one Spotify still
// considers active.
func (m Model) deviceRow(d player.Device, w int) string {
	mark := "  "
	name := m.styles.RowPrimary
	if d.Active {
		mark = m.styles.Cursor.Render(deviceDot) + " "
		name = m.styles.RowSelected
	}

	body := max(w-2, 0)
	return mark + spread(name.Render(d.Name), m.styles.RowSecondary.Render(d.Type), body)
}
