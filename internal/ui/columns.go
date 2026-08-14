package ui

import (
	"image/color"

	"github.com/pottom/spindle/internal/ui/style"
)

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

// columnRule is the line under those names.
//
// The accent in the middle of it, walked out to the screen's own ground at both
// ends — the same pen as the bracket that marks the band above the list, and the
// same reason: at one weight it is a rule the width of the terminal, which is the
// loudest thing on a screen made of words.
//
// Out to both ends rather than out to one. A line that fades left to right has a
// bright end and a dead one, which says the left of the row matters more than the
// right, and nothing about a table is truer on one side than the other. The
// bracket beside the picture does fade one way, because it starts at a corner and
// takes hold there; this starts nowhere and only divides.
//
// The sounding record's accent rather than the cover's, like the names over it.
// The one thing on these screens that wears the cover's is the band the cursor is
// resting on, which is explicitly about another record.
func (m Model) columnRule(w int) string {
	var out string
	for _, step := range style.Crest(m.styles.Accent, m.screenGround(), w) {
		out += step.Render(pointerH)
	}
	return out
}

// screenGround is the terminal's own background colour, or the nearest thing to
// it worth fading into before the terminal has said what it is.
func (m Model) screenGround() color.Color {
	if m.ground != nil {
		return m.ground
	}
	return m.styles.Theme.Raised
}

// columnHead is a list's names, dressed and laid out like one of its rows.
//
// In the accent and in lower case. Lower case because a row of capitals across
// the top of a list is the loudest thing on a screen whose whole point is the
// words under it; the accent because the names are not one more row of the list
// and read as one when they are set in the same greys — the colour is what the
// line under them is doing, said again in a way that survives a narrow terminal
// dropping the line's columns out from under it.
func (m Model) columnHead(w int, lead string, names rowCells) string {
	s := m.styles.Columns
	dress := func(text string) string {
		if text == "" {
			return ""
		}
		return s.Render(text)
	}
	return m.drawRow(w, false, rowCells{
		primary:   lead + dress(names.primary),
		secondary: dress(names.secondary),
		album:     dress(names.album),
		stars:     dress(names.stars),
		liked:     dress(names.liked),
		tempo:     dress(names.tempo),
		trailing:  dress(names.trailing),
	})
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
	return m.columnHead(w, lead, rowCells{
		primary:   "title",
		secondary: "artist",
		album:     "album",
		stars:     "stars",
		liked:     "liked",
		tempo:     "tempo",
		trailing:  "time",
	})
}

// albumColumns names a list of records, and artistColumns a list of people.
func (m Model) albumColumns(w int, lead string) string {
	return m.columnHead(w, lead, rowCells{primary: "album", secondary: "artist", trailing: "released"})
}

func (m Model) artistColumns(w int, lead string) string {
	return m.columnHead(w, lead, rowCells{primary: "artist", secondary: "genres", trailing: "followers"})
}

func (m Model) playlistColumns(w int) string {
	return m.columnHead(w, blankMark, rowCells{primary: "playlist", secondary: "owner", trailing: "tracks"})
}
