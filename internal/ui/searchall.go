package ui

import (
	"strconv"
	"strings"

	"github.com/pottom/spindle/internal/player"
)

// The view a query lands on: the strongest answer, and then the songs.
//
// Spotify calls it All, and what it is for is the first second after pressing
// enter. Searching for an artist and being given a list of their songs is
// nearly right and quietly wrong: what was asked for was the artist, and the
// answer is one row that has to be found among twenty that look like it.
//
// So the top result is the first row, and the cursor arrives on it. The panel
// beside the artwork describes whatever the cursor is on, as it does on every
// other screen, which means the picture on arrival is the picture of the thing
// that was searched for.
//
// It is the one view that mixes kinds. Every other list here holds one sort of
// thing, and that is worth keeping: a screen of headings and sections is a
// screen where nothing can be walked through with one key. One row of a
// different kind, named as what it is, is the smallest departure that answers
// the question.

// searchAll is the kind of the view that holds all of them. It is spindle's
// own: the Web API has no such type, and asking for it is asking for every
// type at once, which is what a query with no kind already does. See askKind.
const searchAll player.SearchKind = "all"

// searchViews are the views the query's answers are shown under, in the order
// the kinds bar walks them.
var searchViews = append([]player.SearchKind{searchAll}, player.SearchKinds...)

// askKind is what a view asks Spotify for. Everything, for the view that shows
// everything.
func askKind(view player.SearchKind) player.SearchKind {
	if view == searchAll {
		return ""
	}
	return view
}

// topResult names the strongest answer to a query: which kind it is, and where
// it sits in that kind's own list.
//
// A place rather than a copy, so that what is drawn and what is acted on are
// the same row as the kind's own view would give — and so that a page arriving
// later cannot leave two answers disagreeing about one thing.
type topResult struct {
	kind player.SearchKind
	at   int
}

// found reports whether there is one.
func (t topResult) found() bool { return t.kind != "" }

// topOf works out the strongest answer.
//
// An artist whose name is the query is what was asked for: somebody typing
// "queen" wants Queen, not the twenty songs with queen in the title. Failing
// that a record or a list by that exact name, and failing that the first song —
// which is what a query that is a phrase rather than a name usually wants.
//
// Exact after folding, rather than anything cleverer. A near-match rule that
// decides "queen" means "Queens of the Stone Age" is worse than no rule: the
// cursor lands on it, the picture is theirs, and the screen has answered a
// question nobody asked.
func (s *searchPane) topOf(query string) topResult {
	name := foldName(query)
	if name == "" {
		return topResult{}
	}

	if artists := s.of(player.SearchArtists).artists; len(artists) > 0 {
		for i, a := range artists {
			if foldName(a.Name) == name {
				return topResult{kind: player.SearchArtists, at: i}
			}
		}
	}
	for i, a := range s.of(player.SearchAlbums).albums {
		if foldName(a.Name) == name {
			return topResult{kind: player.SearchAlbums, at: i}
		}
	}
	for i, p := range s.of(player.SearchPlaylists).playlists {
		if foldName(p.Name) == name {
			return topResult{kind: player.SearchPlaylists, at: i}
		}
	}

	// Nothing is named after the query, so the songs answer it. The first of
	// them is the top result, and it is already the first row: naming it as well
	// would be the same track on two rows, which is a screen contradicting
	// itself.
	if len(s.of(player.SearchTracks).tracks) > 0 {
		return topResult{}
	}

	// No songs either. Whatever there is of the rest is better than an empty
	// screen with the answer one key away.
	for _, kind := range []player.SearchKind{player.SearchArtists, player.SearchAlbums, player.SearchPlaylists} {
		if s.of(kind).count() > 0 {
			return topResult{kind: kind}
		}
	}
	return topResult{}
}

// allRows is how many rows the all view holds: the top result where it is
// something other than a song, and then the songs.
func (m Model) allRows() int {
	rows := len(m.search.of(player.SearchTracks).tracks)
	if m.search.top.found() {
		rows++
	}
	return rows
}

// onTop reports that the cursor of the all view is resting on the top result.
func (m Model) onTop() bool {
	return m.search.kind == searchAll && m.search.top.found() &&
		m.search.of(searchAll).cursor.cursor == 0
}

// allTrack is the song a row of the all view holds, or nil where the row is the
// top result.
func (m Model) allTrack(row int) *player.Track {
	if m.search.top.found() {
		row--
	}
	return at(m.search.of(player.SearchTracks).tracks, row)
}

// topAt reports that a row of the all view is the top result.
func (m Model) topAt(row int) bool { return m.search.top.found() && row == 0 }

// topRow draws the top result: what it is called, and what it is.
//
// The kind is said in words beside the name because the row is the only one of
// its sort on the screen, and a row that looks like a song and is not is worse
// than one that says so.
func (m Model) topRow(w int, selected bool) string {
	top := m.search.top
	found := m.search.of(top.kind)

	var name, what, trailing string
	switch top.kind {
	case player.SearchArtists:
		if a := atArtist(found.artists, top.at); a != nil {
			name, what = a.Name, "artist"
			trailing = artistLine(*a)
		}
	case player.SearchAlbums:
		if a := atAlbum(found.albums, top.at); a != nil {
			name, what = a.Name, "album"
			trailing = albumLine(*a)
		}
	case player.SearchPlaylists:
		if p := atPlaylist(found.playlists, top.at); p != nil {
			name, what = p.Name, "playlist"
			trailing = playlistOwner(*p)
		}
	}
	if name == "" {
		return strings.Repeat(" ", max(w, 0))
	}

	primary := m.styles.RowPrimary
	if selected {
		primary = m.styles.RowSelected
	}
	return m.drawRow(w, selected, rowCells{
		primary:   m.leadIn(m.styles.Cursor.Render(topMark)) + m.lit(primary, name),
		secondary: m.styles.RowSecondary.Render(what),
		third:     m.lit(m.styles.RowTrailing, trailing),
	})
}

// topMark stands where a track's number would be, on the one row that is not a
// track. Up, because it is the top of the answer.
const topMark = "★"

// viewName is what the counts call a view. The player's own name for a kind
// does not know about this one — the Web API has no such type — so the one view
// spindle invented is named here.
func viewName(view player.SearchKind) string {
	if view == searchAll {
		return "all"
	}
	return view.String()
}

// viewLabels is what the box's right-hand side says: every view that matched
// anything, named and counted.
func (m Model) viewLabels() []string {
	var out []string
	for _, view := range searchViews {
		n := m.viewCount(view)
		if n == 0 {
			continue
		}

		count := strconv.Itoa(n)
		if view != searchAll && m.search.of(view).pages.more {
			// What has been read, not what exists: Spotify's totals are not
			// worth carrying through three layers for a heading.
			count += "+"
		}
		out = append(out, viewName(view)+" "+count)
	}
	return out
}

// viewAt is which view a label belongs to, for a press on one.
func (m Model) viewAt(label int) player.SearchKind {
	at := 0
	for _, view := range searchViews {
		if m.viewCount(view) == 0 {
			continue
		}
		if at == label {
			return view
		}
		at++
	}
	return ""
}

// pagesOf is the paging of whichever list a view actually reads from.
//
// The all view holds the strongest answer and then the songs, so what runs out
// as the cursor goes down it is the songs' own list — and so is what is waited
// for. Marking the composed view as loading instead left a spinner turning for
// ever over a screen that had already answered: nothing ever cleared it, because
// nothing ever answered for that bucket. Measured on a real query.
func (m *Model) pagesOf(view player.SearchKind) *paging {
	if view == searchAll {
		return &m.search.of(player.SearchTracks).pages
	}
	return &m.search.of(view).pages
}

// loadingOf is the same question asked of a screen that is only being drawn.
func (m Model) loadingOf(view player.SearchKind) bool {
	if view == searchAll {
		return m.search.of(player.SearchTracks).pages.loading
	}
	return m.search.of(view).pages.loading
}
