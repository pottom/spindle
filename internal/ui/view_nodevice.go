package ui

// deviceListCols keeps the device names and their kinds within reading distance
// of each other on a wide terminal.
const deviceListCols = 52

// noDevicePanel is the player tab when nothing is playing anywhere. It is the
// most common way to arrive at spindle, so it explains rather than complains —
// including why the list below it may be empty, which otherwise looks exactly
// like a bug.
//
// The list is the same one the picker uses, and it is just as live: this screen
// is where you choose where to start.
func (m Model) noDevicePanel(l layout, rows int) []string {
	s := m.styles
	w := l.interior - leftMargin - rightMargin

	// The program's own picture, as large as what is left allows, while the
	// device is being waited for. See splash.go.
	lines := m.splashRows()
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines,
		s.Title.Render("No active playback device"),
		"",
		s.Detail.Render("Start Spotify on one of your devices, or pick one below."),
		"",
	)
	lines = append(lines, m.deviceRows(min(w, deviceListCols), true)...)
	lines = append(lines, "", "")

	if len(m.devices.items) == 0 {
		lines = append(lines,
			s.Empty.Render("Spotify only lists devices that were recently active. Open the"),
			s.Empty.Render("app on one of them and play something for a moment."),
		)
	} else {
		lines = append(lines,
			s.Empty.Render("Missing one? Spotify only lists devices that were recently"),
			s.Empty.Render("active — open the app on it and play something for a moment."),
		)
	}

	return stack(lines, w, rows)
}

// devicePicker is the same list opened over the player, in place of the track
// information. No border and no dimming: the artwork stays put, the heading and
// the help bar say plainly where you are.
func (m Model) devicePicker(w, rows int) []string {
	lines := []string{
		m.styles.Title.Render("Devices"),
		m.styles.Album.Render("Move playback somewhere else"),
		"",
	}
	lines = append(lines, m.deviceRows(w, true)...)
	return stack(lines, w, rows)
}
