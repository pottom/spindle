package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/ui/cover"
)

// What the library's wall is made of, and where its pictures come from.
//
// The wall itself is grid.go, which knows nothing about music: it divides a
// screen into tiles and draws what it is handed. This is the half that knows
// what a playlist is.

// gridSlotFrom is the first renderer slot the wall may use, and gridSlots how
// many it has. Below it are the pictures a screen holds one of — the cursor's,
// what is playing, the next record's colour, the program's own.
const (
	gridSlotFrom = 4
	gridSlots    = cover.Slots - gridSlotFrom
)

// gridKeep is how far past the screen the wall holds on to pictures it has
// already fetched and drawn.
//
// A wheel scrolls a row and back in a second, and without this every cover that
// passed the edge would be fetched again, rendered again and sent to the
// terminal again — which is the difference between a wall that scrolls and a
// wall that stutters. The keys never made it worth having: walking a row at a
// time, the row that left was rarely wanted back.
//
// A page either side, held to what the renderer can tell apart. A slot is one
// picture to the terminal and the wall's slots run i mod gridSlots, so two
// pictures kept at once whose places are a multiple of that apart would be one
// picture drawn twice. Three pages fit inside the sixty for any wall a terminal
// can hold; where they would not, nothing is kept and the wall behaves as it did
// before. See slotFor.
func gridKeep(g gridShape) int {
	return max(min(g.page(), (gridSlots-g.page())/2), 0)
}

// slotFor is the renderer slot a thing on the wall draws in.
//
// Taken from where the thing is in the library rather than from where it is on
// screen. A slot is one picture to the terminal, and a picture re-sent under the
// same slot is uploaded again: keyed by screen position, scrolling one row would
// re-send every cover on the wall. Keyed by the thing itself, a cover that only
// moved stays where it is and nothing goes over the wire.
func slotFor(at int) int { return gridSlotFrom + at%gridSlots }

// libraryTile is one thing on the wall, before its picture has been found.
type libraryTile struct {
	id   string
	url  string
	name string
	sub  string
}

// libraryTiles is what the kind on screen puts on the wall.
//
// The second line says what a thing is rather than repeating what its name
// already says: who a playlist belongs to, when a record came out, what an
// artist plays. The tile has one line for it, so it is one fact.
func (m Model) libraryTiles() []libraryTile {
	var out []libraryTile
	switch m.library.kind {
	case libraryAlbums:
		for _, a := range m.library.albums {
			sub := strings.Join(a.Artists, ", ")
			if year := releaseYear(a.Released); year != "" {
				sub = year + " · " + sub
			}
			out = append(out, libraryTile{id: a.ID, url: a.CoverURL, name: a.Name, sub: sub})
		}
	case libraryArtists:
		for _, a := range m.library.artists {
			sub := strings.Join(a.Genres, ", ")
			if sub == "" && a.Followers > 0 {
				sub = formatCount(a.Followers) + " following"
			}
			out = append(out, libraryTile{id: a.ID, url: a.ImageURL, name: a.Name, sub: sub})
		}
	case libraryRecent:
		for _, t := range m.library.recent {
			out = append(out, libraryTile{
				id: t.ID, url: t.CoverURL, name: t.Title, sub: strings.Join(t.Artists, ", "),
			})
		}
	default:
		for _, p := range m.library.playlists {
			sub := p.Owner
			if p.Tracks > 0 {
				sub = fmt.Sprintf("%d tracks · %s", p.Tracks, p.Owner)
			}
			out = append(out, libraryTile{id: p.ID, url: p.CoverURL, name: p.Name, sub: sub})
		}
	}

	// A thing with no artwork gets the drawn stand-in rather than a hole. A gap
	// in a wall of covers reads as a picture that failed to load; a drawing says
	// there was never one to load. See cover.NoneURL.
	for i := range out {
		if out[i].url == "" {
			out[i].url = cover.NoneURL
		}
	}
	return out
}

// libraryShape is the wall as this terminal divides it.
func (m Model) libraryShape(l layout, rows int) gridShape {
	return gridFor(l.interior-leftMargin-rightMargin-gridGutter-gridEdge,
		rows-gridChromeRows-frameRows-m.finderTakes(), m.cell)
}

// gridChromeRows is what the wall spends above itself: the heading with the
// kinds set against it, and the blank under it.
const gridChromeRows = 2

// libraryPaneGrid draws the whole tab: the heading, and the wall under it.
func (m Model) libraryPaneGrid(l layout, rows int) []string {
	w := l.interior - leftMargin - rightMargin
	g := m.libraryShape(l, rows)

	// The kinds, drawn as a tab bar of their own at the left, and the spinner out
	// at the right: the tiles already on the wall are not always all of them, and
	// nothing else on the screen would say so.
	labels, rule := m.libraryKinds()
	spinner := ""
	if m.listLoading() {
		spinner = m.spinner.View()
	}
	out := []string{
		spread(labels, spinner, w),
		fit(rule, w),
	}

	// The room the field a wall is searched in stands in, under the heading and
	// over the pictures. The wall steps down for it the way a table does, rather
	// than having it drawn over the covers. See finder.go.
	for range m.finderTakes() {
		out = append(out, strings.Repeat(" ", w))
	}

	items := m.libraryTiles()
	switch {
	case !g.ok():
		out = append(out, fit(m.styles.Empty.Render("Not enough room for the covers."), w))
	case len(items) == 0:
		out = append(out, fit(m.libraryWaiting(), w))
	default:
		state := &m.library.cursors[m.library.kind]
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

// libraryWaiting is what the wall says with nothing on it yet.
func (m Model) libraryWaiting() string {
	waiting := map[libraryKind]string{
		libraryAlbums:  "Reading your albums…",
		libraryArtists: "Reading who you follow…",
		libraryRecent:  "Reading what you played…",
	}[m.library.kind]
	if waiting == "" {
		waiting = "Reading your library…"
	}
	if m.listLoading() {
		return m.spinner.View() + " " + m.styles.Empty.Render(waiting)
	}

	empty := map[libraryKind]string{
		libraryAlbums:  "No saved albums.",
		libraryArtists: "Nobody followed yet.",
		libraryRecent:  "Nothing played yet.",
	}[m.library.kind]
	if empty == "" {
		empty = "Nothing saved yet."
	}
	return m.styles.Empty.Render(empty)
}

// syncGridCovers sends for the pictures of the tiles on screen, and forgets the
// ones that have scrolled away.
//
// One request per tile, once: a picture already rendered at the size it is
// wanted at is left alone, so walking the wall asks for nothing and only
// scrolling or resizing does. The renderer's slot is the tile's place on screen,
// which is what lets a terminal drawing real pictures tell them apart — see
// gridSlotFrom.
func (m *Model) syncGridCovers() tea.Cmd {
	if m.covers == nil || m.tab != tabLibrary || m.open() != nil || !fitsMinimum(m.width, m.height) {
		return nil
	}

	l := m.layout()
	g := m.libraryShape(l, l.bodyHeight)
	if !g.ok() {
		return nil
	}

	items := m.libraryTiles()
	state := &m.library.cursors[m.library.kind]
	from, to := state.gridWindow(len(items), g)

	if m.tiles == nil {
		m.tiles = map[string]coverState{}
	}

	// What is on screen now and a little either side of it, so what is neither
	// can go: the wall of a long library is a picture for every playlist
	// somebody has ever saved otherwise.
	keep := max(from-gridKeep(g), 0)
	until := min(to+gridKeep(g), len(items))
	seen := make(map[string]bool, until-keep)
	for i := keep; i < until; i++ {
		seen[items[i].id] = true
	}

	// Asked for, though, only what is on screen. A picture nobody has looked at
	// is a request against somebody's quota; a picture already here costs
	// nothing to hold on to.
	var cmds []tea.Cmd
	for i := from; i < to; i++ {
		item := items[i]
		if item.url == "" {
			continue
		}
		if m.tiles[item.id].matches(item.url, g.boxW, g.boxH) {
			continue
		}
		m.tiles[item.id] = coverState{url: item.url, width: g.boxW, height: g.boxH, want: g.tileW}
		cmds = append(cmds, coverCmd(m.covers, item.url, g.boxW, g.boxH, slotFor(i)))
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

// gridTook files a picture that has arrived under the tile that asked for it.
func (m *Model) gridTook(url string, w, h int, art string) {
	for id, tile := range m.tiles {
		if tile.matches(url, w, h) {
			tile.took(art)
			m.tiles[id] = tile
		}
	}
}

// gridFailed marks a tile whose picture will not come.
func (m *Model) gridFailed(url string, w, h int) {
	for id, tile := range m.tiles {
		if tile.matches(url, w, h) {
			tile.failed = true
			m.tiles[id] = tile
		}
	}
}

// libraryGridKey moves the cursor about the wall. Across as well as down, which
// is the whole difference between a wall and a list.
func (m *Model) libraryGridKey(k tea.KeyPressMsg) bool {
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	if !g.ok() {
		return false
	}

	state := m.library.cursor()
	count := len(m.libraryTiles())
	switch {
	case m.pressed(k, m.keys.Down):
		state.move(g.cols, count)
	case m.pressed(k, m.keys.Up):
		state.move(-g.cols, count)
	case m.pressed(k, m.keys.NextTile):
		state.move(1, count)
	case m.pressed(k, m.keys.PrevTile):
		state.move(-1, count)
	case m.pressed(k, m.keys.PageDown):
		state.move(g.page(), count)
	case m.pressed(k, m.keys.PageUp):
		state.move(-g.page(), count)
	case m.pressed(k, m.keys.HalfDown):
		state.move(max(g.page()/2, g.cols), count)
	case m.pressed(k, m.keys.HalfUp):
		state.move(-max(g.page()/2, g.cols), count)
	case m.pressed(k, m.keys.First), m.pressed(k, m.keys.FirstVim):
		state.move(-count, count)
	case m.pressed(k, m.keys.Last), m.pressed(k, m.keys.LastVim):
		state.move(count, count)
	default:
		return false
	}
	return true
}

// cursorTile is what the cursor is resting on, for the keys that act on it.
func (m Model) cursorTile() *libraryTile {
	items := m.libraryTiles()
	at := m.library.cursors[m.library.kind].cursor
	if at < 0 || at >= len(items) {
		return nil
	}
	return &items[at]
}
