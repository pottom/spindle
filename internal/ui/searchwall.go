package ui

import (
	"strconv"
	"strings"

	"github.com/pottom/spindle/internal/player"
)

// The answers as a wall of covers.
//
// A record, an artist and a playlist are things somebody recognises by their
// sleeve — which is the whole argument the library's wall was built on, and it
// does not stop being true because the list came from a query. A table of names
// is the other half of the same argument: it holds twice as many answers on a
// screen, it says how many tracks and what year, and it is what somebody
// scanning for one word wants.
//
// So both, and a key to choose. Neither is the right answer for everybody, and
// which one somebody wants changes with what they are looking for.
//
// The songs are never a wall. A list of tracks off one record is the same sleeve
// twenty times over, and what tells two songs apart is their names.

// searchWallable reports that a view is of things a picture says something
// about.
func searchWallable(view player.SearchKind) bool {
	switch view {
	case player.SearchAlbums, player.SearchArtists, player.SearchPlaylists:
		return true
	default:
		return false
	}
}

// searchWall reports that the answers on screen are being shown as a wall.
func (m Model) searchWall() bool {
	return m.tab == tabSearch && m.open() == nil &&
		m.search.wall && searchWallable(m.search.kind) && m.counted() > 0
}

// turnSearchWall is the key that chooses. It is remembered across the views, so
// walking the kinds does not keep changing the answer.
func (m *Model) turnSearchWall() {
	m.search.wall = !m.search.wall
}

// searchWallChrome is what the wall spends above itself: the query, the views
// the answers are shown under, and the mark under whichever is on screen.
func (m Model) searchWallChrome() int { return m.searchHeadRows() + 1 + searchChromeRows }

// searchWallShape divides what is left into tiles.
//
// The same count of columns as the library's wall, which is what ctrl and the
// wheel change: it is one wall in two places, and a program with two sizes of
// cover is a program that has forgotten which one somebody set. See resizeWall.
func (m Model) searchWallShape(l layout, rows int) gridShape {
	return gridFor(l.interior-leftMargin-rightMargin-gridGutter-gridEdge,
		rows-m.searchWallChrome()-frameRows, m.library.cols, m.cell)
}

// searchTiles is what the view on screen puts on the wall.
//
// The second line says what a thing is rather than repeating its name, exactly
// as the library's tiles do — one fact, and the same fact the table would put in
// its second column. See libraryTiles.
func (m Model) searchTiles() []libraryTile {
	found := m.search.current()

	var out []libraryTile
	switch m.search.kind {
	case player.SearchAlbums:
		for _, a := range found.albums {
			sub := strings.Join(a.Artists, ", ")
			if year := releaseYear(a.Released); year != "" {
				sub = year + " · " + sub
			}
			out = append(out, libraryTile{id: a.ID, url: a.CoverURL, name: a.Name, sub: sub})
		}
	case player.SearchArtists:
		for _, a := range found.artists {
			sub := strings.Join(a.Genres, ", ")
			if sub == "" && a.Followers > 0 {
				sub = formatCount(a.Followers) + " followers"
			}
			out = append(out, libraryTile{id: a.ID, url: a.ImageURL, name: a.Name, sub: sub})
		}
	case player.SearchPlaylists:
		for _, p := range found.playlists {
			sub := p.Owner
			if p.Tracks > 0 {
				sub = strconv.Itoa(p.Tracks) + " tracks · " + sub
			}
			out = append(out, libraryTile{id: p.ID, url: p.CoverURL, name: p.Name, sub: sub})
		}
	}
	return out
}

// searchWallView draws the tab as a wall: the query, the views, the covers.
//
// No band over it. The band is a cover and a panel about the one thing the
// cursor is on, and a wall is every cover at once — the two together would be
// the same picture twice, and the rows it costs are the rows the wall wants.
func (m Model) searchWallView(l layout, rows int) []string {
	w := l.interior - leftMargin - rightMargin

	out := m.searchHead(w)

	// The views, with the spinner out at the right: what is on the wall is not
	// always all of it, and nothing else here would say so. The library's wall
	// says it in the same place.
	names, rule := m.searchViewsBar()
	spinner := ""
	if m.listLoading() {
		spinner = m.spinner.View()
	}
	out = append(out, spread(names, spinner, w), fit(rule, w))

	g := m.searchWallShape(l, rows)
	items := m.searchTiles()
	switch {
	case !g.ok():
		out = append(out, fit(m.styles.Empty.Render("Not enough room for the covers."), w))
	case len(items) == 0:
		out = append(out, fit(m.styles.Empty.Render("Nothing matched."), w))
	default:
		state := &m.search.current().cursor
		from, to := state.gridWindow(len(items), g)

		tiles := make([]gridTile, 0, to-from)
		for i := from; i < to; i++ {
			tiles = append(tiles, gridTile{
				art:      m.tiles[items[i].id].lines,
				name:     items[i].name,
				sub:      items[i].sub,
				selected: i == state.cursor,
			})
		}
		out = append(out, m.drawGrid(tiles, g, w, rows-len(out))...)
	}

	for len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}
	return out[:rows]
}

// searchWallSpot is what the wall has at a point: the views over it, or one of
// the covers.
//
// Read back from the same pieces it is drawn from, which is the only way a
// pointer and a picture stay in step. See wallSpot, which does this for the
// library's own.
func (m Model) searchWallSpot(l layout, x, row int) spot {
	head := m.searchHeadRows()
	if row == head {
		return spot{spotKinds, spanAt(m.searchViewSpans(), x)}
	}
	row -= m.searchWallChrome()
	if row < 0 {
		return spot{spotQuery, -1}
	}

	here := spot{spotTile, -1}
	g := m.searchWallShape(l, l.bodyHeight)
	if !g.ok() {
		return spot{spotNothing, -1}
	}

	r := row / (g.tileH + tileRowGap)
	col := x - leftMargin - gridGutter
	step := g.tileW + g.gap
	if r >= g.rows || col < 0 || col%step >= g.tileW {
		return here
	}
	i := col / step
	if i >= g.cols {
		return here
	}

	items := m.searchTiles()
	state := &m.search.current().cursor
	from, to := state.gridWindow(len(items), g)
	if at := from + r*g.cols + i; at < to {
		return spot{spotTile, at}
	}
	return here
}
