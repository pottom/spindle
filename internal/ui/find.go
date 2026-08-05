package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// find is a search inside the list on screen: not a request to Spotify, but a
// way through what is already in front of you.
//
// The two are worth telling apart. The search tab asks the catalogue a question
// and waits; this walks a list you are already reading — a playlist of three
// hundred, a queue you have been filling — and needs no network at all. They
// share the key because they are the same act at different scales: / looks for
// something where you are.
type find struct {
	// typing is whether the keyboard belongs to the query. The query is only
	// live while it is being written or while its matches are being walked.
	typing bool
	query  string

	// matches are the rows that matched, in order, and at is which of them the
	// cursor last landed on. They are kept after typing ends so n and N can
	// walk them.
	matches []int
	at      int
}

// findable reports whether the screen has a list to look through. The player
// tab has none, and there / means the other thing: go and ask the catalogue.
func (m Model) findable() bool {
	switch {
	case m.open() != nil, m.tab == tabQueue, m.tab == tabLibrary:
		return true
	default:
		return false
	}
}

// findKey drives the query while it is being written. Everything reaches it,
// because every printable key is part of what is being looked for.
func (m *Model) findKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.find.typing {
		return nil, false
	}

	switch {
	case key.Matches(k, m.keys.Back):
		// Nothing was chosen, so nothing is left behind: the cursor goes back
		// to where the search started from.
		m.find = find{}
		return nil, true

	case key.Matches(k, m.keys.Enter):
		m.find.typing = false
		return nil, true

	case k.Code == tea.KeyBackspace:
		if q := []rune(m.find.query); len(q) > 0 {
			m.find.query = string(q[:len(q)-1])
		}
		m.refind()
		return nil, true
	}

	if text := k.Text; text != "" {
		m.find.query += text
		m.refind()
		return nil, true
	}
	return nil, true
}

// startFind opens the query over whatever list is on screen.
func (m *Model) startFind() {
	m.find = find{typing: true}
}

// refind works out what the query matches now, and moves the cursor to the
// first match at or after where it already is — so a query typed a letter at a
// time walks forward rather than jumping back to the top of the list on every
// keystroke.
func (m *Model) refind() {
	m.find.matches, m.find.at = nil, 0
	if m.find.query == "" {
		return
	}

	needle := strings.ToLower(m.find.query)
	state, count := m.listCursor()
	if state == nil {
		return
	}

	for i := range count {
		if strings.Contains(strings.ToLower(m.rowText(i)), needle) {
			m.find.matches = append(m.find.matches, i)
		}
	}
	if len(m.find.matches) == 0 {
		return
	}

	for i, row := range m.find.matches {
		if row >= state.cursor {
			m.find.at = i
			break
		}
	}
	state.moveTo(m.find.matches[m.find.at], count)
}

// findStep moves to the next match, or the previous one, wrapping at the ends:
// a list read to the bottom is a list you want to start again, not one that
// stops answering.
func (m *Model) findStep(delta int) bool {
	if len(m.find.matches) == 0 {
		return false
	}
	state, count := m.listCursor()
	if state == nil {
		return false
	}

	// Where the cursor is now decides where "next" is, because it may have been
	// moved by hand since the last match.
	switch {
	case delta > 0:
		m.find.at = len(m.find.matches) - 1
		for i, row := range m.find.matches {
			if row > state.cursor {
				m.find.at = i
				break
			}
		}
		if m.find.matches[m.find.at] <= state.cursor {
			m.find.at = 0
		}
	default:
		m.find.at = 0
		for i := len(m.find.matches) - 1; i >= 0; i-- {
			if m.find.matches[i] < state.cursor {
				m.find.at = i
				break
			}
		}
		if m.find.matches[m.find.at] >= state.cursor {
			m.find.at = len(m.find.matches) - 1
		}
	}

	state.moveTo(m.find.matches[m.find.at], count)
	return true
}

// listCursor is the cursor of the list on screen, and how many rows it has.
// Nil where the screen has no list of its own.
func (m *Model) listCursor() (*listState, int) {
	switch {
	case m.open() != nil:
		page := m.openMut()
		return &page.cursor, page.count()
	case m.tab == tabQueue:
		return &m.queuePane.cursor, len(m.queueRows())
	case m.tab == tabLibrary:
		return m.library.cursor(), m.library.count()
	default:
		return nil, 0
	}
}

// rowText is what a row says, for the query to be matched against. What is
// drawn rather than what is stored: a search through a list looks for what is
// on the screen, and the columns beyond the first are part of that — a playlist
// is found by its owner as readily as by its name.
func (m Model) rowText(i int) string {
	if page := m.open(); page != nil {
		if page.holdsAlbums() {
			if a := atAlbum(page.albums, i); a != nil {
				return a.Name + " " + strings.Join(a.Artists, ", ")
			}
			return ""
		}
		if t := at(page.tracks, i); t != nil {
			return trackText(*t)
		}
		return ""
	}

	switch m.tab {
	case tabQueue:
		rows := m.queueRows()
		if i >= 0 && i < len(rows) {
			return trackText(rows[i])
		}
	case tabLibrary:
		switch m.library.kind {
		case libraryAlbums:
			if a := atAlbum(m.library.albums, i); a != nil {
				return a.Name + " " + strings.Join(a.Artists, ", ")
			}
		case libraryArtists:
			if a := atArtist(m.library.artists, i); a != nil {
				return a.Name + " " + strings.Join(a.Genres, ", ")
			}
		case libraryRecent:
			if t := at(m.library.recent, i); t != nil {
				return trackText(*t)
			}
		default:
			if p := atPlaylist(m.library.playlists, i); p != nil {
				return p.Name + " " + p.Owner
			}
		}
	}
	return ""
}

func trackText(t player.Track) string {
	return t.Title + " " + strings.Join(t.Artists, ", ") + " " + t.Album
}

// findLine is the query as the screen shows it, or empty when nothing is being
// looked for. It says how many rows matched, because a query that matches
// nothing and a query that matches everything look the same on a list you
// cannot see the end of.
func (m Model) findLine() string {
	if !m.find.typing && m.find.query == "" {
		return ""
	}

	line := "/" + m.find.query
	switch {
	case m.find.query == "":
	case len(m.find.matches) == 0:
		line += m.styles.Empty.Render("   no match")
	default:
		line += m.styles.Album.Render(fmt.Sprintf("   %d of %d", m.find.at+1, len(m.find.matches)))
	}
	return line
}
