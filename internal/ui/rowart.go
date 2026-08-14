package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// A record's tracks, each with its own sleeve.
//
// Inside an opened playlist the covers are all different — a playlist is a
// hundred records rather than one — and the sleeve is how anybody who has ever
// used a music player recognises a track. So the row carries it: the picture on
// the left, the title beside it, the artists under the title, and the columns
// this program's tables have always had out to the right.
//
// It costs half the list. A row two rows tall shows half as many tracks, which is
// the price of the pictures and is paid nowhere else: the queue is a working list
// and keeps its table.

const (
	// openRowRows is how many screen rows one track takes with its sleeve on it,
	// and rowArtRows how many of those the sleeve fills. The row that is left is
	// air: drawn edge to edge, one sleeve runs into the next and a column of
	// covers reads as one long picture.
	//
	// Two rows was the least a square picture can be drawn in and it looked it —
	// forty pixels across is a colour rather than a cover. Three is half as much
	// again in each direction and twice the picture, and the two lines a track has
	// to say stand beside it.
	openRowRows = 4
	rowArtRows  = 3

	// rowArtCols is how wide that sleeve is. Twice the rows, because a cell is
	// about twice as tall as it is wide and a sleeve is square.
	rowArtCols = 2 * rowArtRows

	// rowArtGap is the air between the picture and the words.
	rowArtGap = 2
)

// showsRowArt reports whether the tracks of an opened record carry their
// sleeves.
//
// Only where there is a picture worth drawing and a terminal to draw it: below a
// certain width the sleeve is taking room the title needs, and a list of titles
// beats a list of smudges.
func (m Model) showsRowArt() bool {
	page := m.open()
	if page == nil || page.holdsAlbums() {
		return false
	}
	return m.width >= rowArtFrom
}

// rowArtFrom is the width at which a track can afford to carry its sleeve.
const rowArtFrom = 80

// openTrackRows draws one track of an opened record: the sleeve, the title with
// the table's columns beside it, and the artists under it.
func (m Model) openTrackRows(t player.Track, w int, selected bool, number int) []string {
	art := strings.Split(m.tiles[t.ID].art, "\n")
	lead := make([]string, openRowRows)
	for i := range lead {
		cell := ""
		if i < min(len(art), rowArtRows) {
			cell = art[i]
		}
		lead[i] = fit(cell, rowArtCols) + strings.Repeat(" ", rowArtGap)
	}

	primary := m.styles.RowPrimary
	switch {
	case selected:
		primary = m.styles.RowSelected
	case m.ps != nil && m.ps.TrackID == t.ID:
		primary = m.styles.RowPlaying
	}

	// The title takes the row the table would give it, with everything that
	// stands to the right of a title still standing there. The artists have the
	// row under it to themselves, which is where a name that long belongs.
	body := w - rowArtCols - rowArtGap
	title := m.drawRow(body, selected, rowCells{
		primary:  m.leadIn(primary.Render(numberOf(number))) + m.queuedColumn(t) + m.withMark(t, m.lit(primary, t.Title), body),
		album:    m.lit(m.styles.RowTrailing, t.Album),
		stars:    m.starsCell(t),
		liked:    m.likedCell(t),
		tempo:    m.tempoCell(t),
		trailing: m.styles.RowTrailing.Render(formatDuration(t.Duration)),
	})
	// The row under holds those two columns open rather than drawing them again:
	// the ordinal and the mark for a track already queued belong to the track,
	// and a track is one thing however many rows it takes.
	under := m.drawRow(body, false, rowCells{
		primary: m.leadIn(" ") + strings.Repeat(" ", lipgloss.Width(m.queuedColumn(t))) +
			m.lit(m.styles.RowSecondary, strings.Join(t.Artists, ", ")),
	})

	// The words sit against the middle of the picture rather than at the top of
	// it: two lines against three rows of sleeve, and hung from the top they
	// would read as a caption that had come loose.
	out := make([]string, openRowRows)
	at := (rowArtRows - 2 + 1) / 2
	for i := range out {
		switch i {
		case at:
			out[i] = lead[i] + title
		case at + 1:
			out[i] = lead[i] + under
		default:
			out[i] = lead[i]
		}
	}
	return out
}

// numberOf is a track's place in the record, or nothing at all for the one that
// is sounding — which is marked instead, like every other list here.
func numberOf(number int) string {
	if number <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", number)
}

// syncRowCovers sends for the sleeves of the tracks on screen.
//
// The same rules as the wall's: one request per track and only once, keyed by the
// track rather than by where it sits, and what has scrolled away is forgotten.
// See syncGridCovers, which this is the twin of.
func (m *Model) syncRowCovers() tea.Cmd {
	if m.covers == nil || !m.showsRowArt() || !fitsMinimum(m.width, m.height) {
		return nil
	}

	page := m.open()
	from, to := page.cursor.window(page.count(), m.visibleListRows())
	if m.tiles == nil {
		m.tiles = map[string]coverState{}
	}

	seen := make(map[string]bool, to-from)
	var cmds []tea.Cmd
	for i := from; i < to; i++ {
		t := at(page.tracks, i)
		if t == nil {
			continue
		}
		seen[t.ID] = true
		if t.CoverURL == "" || m.tiles[t.ID].matches(t.CoverURL, rowArtCols, rowArtRows) {
			continue
		}
		m.tiles[t.ID] = coverState{url: t.CoverURL, width: rowArtCols, height: rowArtRows}
		cmds = append(cmds, coverCmd(m.covers, t.CoverURL, rowArtCols, rowArtRows, slotFor(i)))
	}
	for id := range m.tiles {
		if !seen[id] {
			delete(m.tiles, id)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}
