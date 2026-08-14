package ui

// The row that names a list's columns.
//
// Every list on screen is four or five columns of words with nothing to say what
// any of them is. On the queue that reads as a title, a second title, a number
// and a time — and the second title is the artists, the number is a beat rate,
// and the one everybody guesses wrong is the album, because an album named after
// its single is the same word twice across the row.
//
// It costs no rows. The blank between the heading and the first track is already
// there, held so the heading stands clear of the list; the names go in it.
//
// They are set through the same function that lays out a row, given the same
// width, so a column that comes and goes with the terminal's takes its name with
// it. Nothing here decides which columns fit — that is the row's business, and a
// header that decided it separately would be a second opinion to keep in step.

// columnHead is a list's names, dressed and laid out like one of its rows.
//
// In the accent and in lower case. Lower case because a row of capitals across
// the top of a list is the loudest thing on a screen whose whole point is the
// words under it; the accent because the names are not one more row of the list
// and read as one when they are set in the same greys — the colour is what the
// line under them is doing, said again in a way that survives a narrow terminal
// dropping the line's columns out from under it.
func (m Model) columnHead(w int, lead, primary, secondary, album, tempo, trailing string) string {
	s := m.styles.Columns
	dress := func(text string) string {
		if text == "" {
			return ""
		}
		return s.Render(text)
	}
	return m.rowColsAlbum(w, false,
		lead+dress(primary), dress(secondary), dress(album), dress(tempo), dress(trailing))
}

// trackColumns names a list of tracks. The ordinal keeps its column: it is the
// number people call a track by when they ask for one.
func (m Model) trackColumns(w int, numbered bool) string {
	lead := ""
	if numbered {
		lead = m.leadIn(m.styles.Columns.Render("#"))
	}
	// Whatever a row of this list holds open in front of the title — the queue's
	// hearts, the marks for what is playing — held open here too, so the names
	// stand over the words they name rather than two columns left of them.
	lead += m.blankQueuedColumn()

	// tempo rather than bpm, because the cell under it says bpm already: a
	// column named after its own unit is the unit written twice.
	return m.columnHead(w, lead, "title", "artist", "album", "tempo", "time")
}

// albumColumns names a list of records, and artistColumns a list of people.
func (m Model) albumColumns(w int, lead string) string {
	return m.columnHead(w, lead, "album", "artist", "", "", "released")
}

func (m Model) artistColumns(w int, lead string) string {
	return m.columnHead(w, lead, "artist", "genres", "", "", "followers")
}

func (m Model) playlistColumns(w int) string {
	return m.columnHead(w, blankMark, "playlist", "owner", "", "", "tracks")
}
