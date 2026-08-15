package ui

import "github.com/pottom/spindle/internal/build"

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
	lines, _ := m.noDeviceLines(l)
	return stack(lines, l.interior-leftMargin-rightMargin, rows)
}

// noDeviceLines is what that screen holds, and where in it the devices start.
//
// Two answers from one function because there are two callers: the drawing, and
// the pointer working out which device it is over. A screen this tall — a
// picture of unknown height, four lines of explanation, the list, and a page of
// small print under it — cannot have its rows counted twice and stay in
// agreement. See deviceSpot.
func (m Model) noDeviceLines(l layout) (lines []string, at int) {
	s := m.styles
	w := l.interior - leftMargin - rightMargin

	// The program's own picture, as large as what is left allows, while the
	// device is being waited for. See splash.go.
	lines = m.splashRows()
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines,
		s.Title.Render("No active playback device"),
		"",
		s.Detail.Render("Start Spotify on one of your devices, or pick one below."),
		"",
	)
	at = len(lines)
	lines = append(lines, m.deviceRows(min(w, deviceListCols), true)...)
	lines = append(lines, "", "")

	// Which build this is. It belongs here more than anywhere: this is the
	// screen you are looking at while you wait, and half a day went on a picture
	// that had been fixed and went on looking broken because the binary running
	// was older than the fix.
	lines = append(lines, s.Empty.Render("spindle "+build.Version()), "")

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

	return lines, at
}

// devicesPopup is the picker as a box standing over whatever you were looking
// at, rather than in place of it.
//
// It opens under the device's name in the header, because that is the thing it
// changes: the name says where the music is, and the list of where else it could
// be belongs under it — the way a menu belongs under the word that opens it.
//
// In place of the player it was a panel that took half the screen to say three
// lines, and it took away the one thing somebody moving the music wants to keep
// looking at: what is playing.
func (m Model) devicesPopup() popup {
	rows := m.deviceRows(deviceListCols, true)
	if len(m.devices.items) == 0 {
		// deviceRows says so itself, in one line. Kept, because an empty box
		// with a heading is worse than a box that explains.
		rows = []string{m.styles.Empty.Render("No devices reported.")}
	}
	return popup{
		x: leftMargin + devicesUnderName, y: tabBarHeight - 1,
		title:    "Devices",
		subtitle: "Move playback somewhere else",
		rows:     rows,
	}
}

// devicesUnderName is how far in the box hangs from the left margin: under the
// name in the header rather than under the mark before it.
const devicesUnderName = 2
