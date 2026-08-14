package ui

import (
	"strings"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The library as a wall of covers.
//
// A library is a shelf. What is on a shelf is found by its spine and its sleeve,
// not by reading a table of names — and the one thing a music player has that a
// file manager does not is the artwork. Drawn as a list, the library spent a
// third of the screen on one enormous cover belonging to whichever row the cursor
// happened to be on, and showed the other thirty as words.
//
// So every row is a tile: the cover, the name under it, and one line saying what
// it is. The tiles are as large as the terminal can hold at a readable size, and
// how many go across is whatever fits — the wall fills the room it is given.

const (
	// tileGap is the air between two tiles across, and tileRowGap the blank row
	// between two rows of them. Wider across than down because the tiles are
	// taller than they are wide: equal air in cells reads as more air across.
	//
	// Both are held to what the frame round the tile under the cursor needs: an
	// upright either side of a tile and a column of air between two of them, and
	// a row over the picture and under the words. The frame is drawn into that
	// air rather than between the tiles, so nothing on the wall moves when the
	// cursor arrives — and a wall whose gaps were tighter than this would have
	// the frame standing on a picture.
	tileGap    = 2*frameCols + 1
	tileRowGap = frameRows

	// frameCols and frameRows are what that ring takes on each side.
	frameCols = 1
	frameRows = 1

	// tileTextRows is what a tile spends on words: the name, and a line saying
	// what the thing is. Two, always — a tile that took a third row for a long
	// name would stand out of line with the ones beside it.
	tileTextRows = 2

	// tileWant is the width a tile is drawn at where the room allows, and
	// tileLeast the narrowest worth drawing at all. Under that a cover is a
	// smudge and a name is two words of three.
	tileWant  = 22
	tileLeast = 12

	// gridGutter is the air kept to the left of the wall and gridEdge to the
	// right of it, for the frame that marks the tile under the cursor.
	//
	// Every tile has a column to itself on either side — the first and last from
	// these, the rest from the gap between tiles — so the frame is drawn in air
	// that is already there and nothing on the wall moves when the cursor
	// arrives. A mark that shifted the tiles would repaint every picture on the
	// screen to say which one of them is being pointed at.
	gridGutter = frameCols + 1
	gridEdge   = frameCols
)

// gridShape is how a wall of tiles divides a screen.
type gridShape struct {
	cols int // tiles across
	rows int // rows of tiles down

	tileW     int // cells one tile takes across
	coverRows int // rows its picture fills
	tileH     int // rows the whole tile takes, picture and words
}

// page is how many tiles are on screen at once.
func (g gridShape) page() int { return g.cols * g.rows }

// ok reports whether there is room for a wall at all.
func (g gridShape) ok() bool { return g.cols > 0 && g.rows > 0 && g.tileW >= tileLeast }

// gridFor divides a screen of the given width and height into tiles.
//
// The count of columns comes first, from the width a tile wants: what is left
// over is spread between them rather than left at the edge, so the wall reaches
// both margins whatever the terminal is.
func gridFor(width, height int, cell cover.CellSize) gridShape {
	if width <= 0 || height <= 0 {
		return gridShape{}
	}

	cols := max((width+tileGap)/(tileWant+tileGap), 1)
	tileW := (width - tileGap*(cols-1)) / cols
	for cols > 1 && tileW < tileLeast {
		cols--
		tileW = (width - tileGap*(cols-1)) / cols
	}

	// The picture is square, and a cell is not. The same arithmetic the artwork
	// area uses, so a tile's cover is as square as any other cover on screen.
	coverRows := max(tileW*cell.Width/cell.Height, 1)
	tileH := coverRows + tileTextRows

	rows := max((height+tileRowGap)/(tileH+tileRowGap), 0)
	return gridShape{cols: cols, rows: rows, tileW: tileW, coverRows: coverRows, tileH: tileH}
}

// gridWindow is the run of tiles on screen, scrolling a whole row at a time.
//
// A row at a time because a wall that scrolled by one tile would put the same
// row of covers across two rows of the screen, and the eye reads a wall by its
// rows.
func (l *listState) gridWindow(count int, g gridShape) (from, to int) {
	if count == 0 || !g.ok() {
		return 0, 0
	}
	l.cursor = min(max(l.cursor, 0), count-1)

	last := (count - 1) / g.cols
	topRow := min(max(l.top/g.cols, 0), max(last-g.rows+1, 0))
	if row := l.cursor / g.cols; row < topRow {
		topRow = row
	} else if row >= topRow+g.rows {
		topRow = row - g.rows + 1
	}

	l.top = topRow * g.cols
	return l.top, min(l.top+g.page(), count)
}

// gridTile is one thing on the wall: its picture, and the two lines under it.
type gridTile struct {
	art      string // the cover as cells, or empty while it is on its way
	name     string
	sub      string
	selected bool
}

// drawGrid lays the tiles out, given the shape they were measured for.
func (m Model) drawGrid(tiles []gridTile, g gridShape, width, height int) []string {
	// A row of air over the wall, so the frame round a tile in the first row has
	// somewhere to stand.
	out := []string{strings.Repeat(" ", width)}
	blank := strings.Repeat(" ", width)

	for at := 0; at < len(tiles); at += g.cols {
		row := tiles[at:min(at+g.cols, len(tiles))]
		if at > 0 {
			for range tileRowGap {
				out = append(out, blank)
			}
		}
		out = append(out, m.drawTileRow(row, g, width)...)
	}
	for len(out) < height {
		out = append(out, blank)
	}
	out = out[:min(len(out), height)]

	for i, tile := range tiles {
		if tile.selected {
			m.frameTile(out, i, 1, g, width)
			break
		}
	}
	return out
}

// drawTileRow draws one row of the wall: the pictures side by side, then the
// names, then the line under them.
func (m Model) drawTileRow(row []gridTile, g gridShape, width int) []string {
	out := make([]string, 0, g.tileH)
	gap := strings.Repeat(" ", tileGap)
	lead := strings.Repeat(" ", gridGutter)

	for line := range g.coverRows {
		var cells []string
		for _, tile := range row {
			cells = append(cells, fit(artLine(tile.art, line), g.tileW))
		}
		out = append(out, fit(lead+strings.Join(cells, gap), width))
	}

	// The words start where the picture starts, both lines of them.
	for _, words := range []func(gridTile) string{
		func(t gridTile) string { return m.tileName(t) },
		func(t gridTile) string { return m.styles.Empty.Render(t.sub) },
	} {
		var cells []string
		for _, tile := range row {
			cells = append(cells, fit(words(tile), g.tileW))
		}
		out = append(out, fit(lead+strings.Join(cells, gap), width))
	}

	return out
}

// frameTile draws the ring round the tile under the cursor, over rows already
// laid out: the row above its picture, the row under its words, and a column
// either side.
//
// Rounded corners and the accent, the same pen the band on a list is bracketed
// with. What it says is where the cursor is, which on a wall of pictures is a
// question about one of them rather than about a row of words — so it goes round
// the picture rather than beside the name.
func (m Model) frameTile(rows []string, at, top int, g gridShape, width int) {
	col := at % g.cols
	left := gridGutter + col*(g.tileW+tileGap) - 1
	right := left + g.tileW + 1
	head := top + (at/g.cols)*(g.tileH+tileRowGap) - 1
	foot := head + g.tileH + 1
	if head < 0 || foot >= len(rows) || left < 0 || right >= width {
		return
	}

	pen := m.styles.Cursor
	rule := strings.Repeat(pointerH, g.tileW)
	rows[head] = overwrite(rows[head], left, pen.Render(pointerTL+rule+pointerTR), width)
	rows[foot] = overwrite(rows[foot], left, pen.Render(pointerElbow+rule+pointerBR), width)
	for row := head + 1; row < foot; row++ {
		rows[row] = overwrite(rows[row], left, pen.Render(pointerV), width)
		rows[row] = overwrite(rows[row], right, pen.Render(pointerV), width)
	}
}

// tileName is the name under a picture, lit where the cursor is on it.
//
// The name rather than a frame round the picture. A frame would have to stand
// either inside the cover, over the artwork, or outside it, which moves every
// tile beside it by a column — and the name is where the eye goes to read what a
// tile is anyway.
func (m Model) tileName(t gridTile) string {
	if t.selected {
		return m.styles.RowSelected.Render(t.name)
	}
	return m.styles.RowPrimary.Render(t.name)
}

// artLine is one row of a rendered cover, or blank where the picture has not
// arrived or has run out of rows.
func artLine(art string, line int) string {
	if art == "" {
		return ""
	}
	lines := strings.Split(art, "\n")
	if line >= len(lines) {
		return ""
	}
	return lines[line]
}
