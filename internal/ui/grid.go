package ui

import (
	"strings"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/style"
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

	// frameCols and frameRows are how far the corners stand off the tile on each
	// side: two columns and one row.
	//
	// Not the same number, because a cell is not square. A terminal cell is about
	// twice as tall as it is wide, so a column of air is half the air a row is —
	// drawn one of each, the corners sat visibly closer to the sides of a picture
	// than to its top and foot. Two and one is the pair that measures the same in
	// pixels.
	frameCols = 2
	frameRows = 1

	// tileTextRows is what a tile spends under its picture: a row of air, the
	// name, and a line saying what the thing is.
	//
	// Always these three — a tile that took a fourth row for a long name would
	// stand out of line with the ones beside it. The air is the picture's own:
	// the frame round the tile under the cursor closes there, so what it marks is
	// the cover and not the caption.
	tileTextRows = 3

	// coverSquare is the shape a cover is, for asking the renderer how many cells
	// it will really fill. Album art is square, whatever the pixels of it are.
	coverSquare = 640

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

	// The box a cover is asked for, and the rectangle of it the renderer will
	// really draw in. They are not the same: a square picture in whole cells
	// hardly ever is, and the drawn one is what everything here lines up
	// against. The box has to be passed on unchanged, or the renderer fits the
	// picture inside it a second time and it shrinks again.
	boxW, boxH int

	tileW   int // cells one tile takes across, which is what is drawn
	artRows int // and rows, likewise
	tileH   int // rows the whole tile takes, picture and words
	gap     int // cells between two tiles across
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

	// And the tile is exactly the rectangle the renderer will draw the cover in,
	// asked of the renderer rather than worked out again here.
	//
	// Because the tile is the picture: the frame stands against its edges and the
	// words start at its left. A box a cell wider than what is drawn in it puts
	// every one of those marks out of true — and on the renderer that hands the
	// rectangle to the terminal, it leaves a band down one side of the cover that
	// nothing here can see. See cover.FitCells, which both sides now ask.
	boxW, boxH := tileW, coverRows
	tileW, artRows := cover.FitCells(coverSquare, coverSquare, boxW, boxH, cell)

	// What that gives back goes between the tiles, so the wall still reaches both
	// margins and the gap is never tighter than the frame needs.
	gap := tileGap
	if cols > 1 {
		gap = max((width-cols*tileW)/(cols-1), tileGap)
	}

	tileH := artRows + tileTextRows
	rows := max((height+tileRowGap)/(tileH+tileRowGap), 0)
	return gridShape{
		cols: cols, rows: rows, boxW: boxW, boxH: boxH,
		tileW: tileW, artRows: artRows, tileH: tileH, gap: gap,
	}
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
	// art is the cover, a row of cells to a line, already squared off to the
	// tile's width. Empty while it is on its way.
	art      []string
	name     string
	sub      string
	selected bool
}

// drawGrid lays the tiles out, given the shape they were measured for.
//
// A row of tiles has a row of air over it and under it, which is where the arms
// of the frame round the tile under the cursor stand. So a wall is a gap row, a
// row of tiles, a gap row, and so on to the foot of it.
func (m Model) drawGrid(tiles []gridTile, g gridShape, width, height int) []string {
	out := make([]string, 0, height)

	rows := (len(tiles) + g.cols - 1) / g.cols
	for r := range rows {
		row := tiles[r*g.cols : min((r+1)*g.cols, len(tiles))]
		at := selectedIn(tiles, g, r)
		out = append(out, m.armRow(g, width, at, pointerTL, pointerTR))
		out = append(out, m.drawTileRow(row, g, width, at)...)
	}

	blank := strings.Repeat(" ", width)
	for len(out) < height {
		out = append(out, blank)
	}
	return out[:min(len(out), height)]
}

// selectedIn is which tile of a row of the wall the cursor is on, counted from
// the left of that row, or -1 where it is on none of them.
func selectedIn(tiles []gridTile, g gridShape, row int) int {
	if row < 0 {
		return -1
	}
	for i := row * g.cols; i < min((row+1)*g.cols, len(tiles)); i++ {
		if tiles[i].selected {
			return i - row*g.cols
		}
	}
	return -1
}

// drawTileRow draws one row of the wall: the pictures side by side, then the
// names, then the line under them — with the uprights of the frame written in
// beside the tile the cursor is on.
func (m Model) drawTileRow(row []gridTile, g gridShape, width, at int) []string {
	out := make([]string, 0, g.tileH)

	// Everything here is already the width it says it is: the pictures were
	// squared off when they arrived and the words are cut to the tile. So the row
	// is written out and padded at the end, and nothing walks a picture through
	// an escape-sequence parser to ask how wide it is. See coverState.took.
	tall := min(frameTall, (g.artRows-1)/2)
	fade := style.Fade(m.styles.Accent, m.screenGround(), tall+1)

	join := func(cells []string, line int) string {
		// How far down the picture this line is, from whichever end is nearer,
		// and nothing at all where it is past the corners' reach or below the
		// picture altogether.
		edge := ""
		if at >= 0 && line < g.artRows {
			if step := min(line+1, g.artRows-line); step <= tall {
				edge = fade[step].Render(pointerV)
			}
		}

		var b strings.Builder
		b.Grow(width)
		for i, cell := range cells {
			gap := g.gap
			if i == 0 {
				gap = gridGutter
			}
			b.WriteString(m.frameGap(gap, edge, at, i))
			b.WriteString(cell)
		}
		if at == len(cells)-1 && edge != "" {
			b.WriteString(strings.Repeat(" ", frameCols-1) + edge)
		}

		used := gridGutter + len(cells)*g.tileW + (len(cells)-1)*g.gap
		if at == len(cells)-1 && edge != "" {
			used += frameCols
		}
		if pad := width - used; pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}
		return b.String()
	}

	cells := make([]string, len(row))
	for line := range g.artRows {
		for i, tile := range row {
			cells[i] = artLine(tile.art, line, g.tileW)
		}
		out = append(out, join(cells, line))
	}

	// The row of air under the picture, where the frame closes.
	out = append(out, m.armRow(g, width, at, pointerElbow, pointerBR))

	for _, words := range []func(gridTile) string{
		func(t gridTile) string { return m.tileName(t) },
		func(t gridTile) string { return m.styles.Empty.Render(t.sub) },
	} {
		for j, tile := range row {
			cells[j] = fit(words(tile), g.tileW)
		}
		out = append(out, join(cells, g.artRows))
	}
	return out
}

// frameGap is the air before a tile, with an upright of the frame in it where the
// tile beside it is the one under the cursor.
func (m Model) frameGap(gap int, edge string, at, tile int) string {
	if edge == "" || (tile != at && tile != at+1) {
		return strings.Repeat(" ", gap)
	}
	if tile == at {
		// Before the framed tile: the upright stands frameCols out from it.
		return strings.Repeat(" ", gap-frameCols) + edge + strings.Repeat(" ", frameCols-1)
	}
	// After it, in the same air.
	return strings.Repeat(" ", frameCols-1) + edge + strings.Repeat(" ", gap-frameCols)
}

// armRow is a row of air carrying the arms of the frame round one tile: the two
// top corners over a picture, or the two bottom ones under it.
//
// The frame is written into these rows rather than over them afterwards. Drawn
// over, every corner and every cell of every arm was a splice into a finished
// line — and a line of this wall is a few thousand bytes of placeholder cells,
// which has to be walked through an escape-sequence parser to find the column
// asked for. Measured on a wall of sixty covers: two hundred milliseconds a
// frame, four fifths of it in that splicing.
func (m Model) armRow(g gridShape, width, at int, near, far string) string {
	if at < 0 {
		return strings.Repeat(" ", width)
	}

	arm := min(frameArm, (g.tileW-1)/2)
	fade := style.Fade(m.styles.Accent, m.screenGround(), arm+1)

	var b strings.Builder
	b.Grow(width)
	left := gridGutter + at*(g.tileW+g.gap) - frameCols
	b.WriteString(strings.Repeat(" ", left))
	b.WriteString(fade[0].Render(near))
	for i := 1; i <= arm; i++ {
		b.WriteString(fade[i].Render(pointerH))
	}

	// The middle of a side is open: two corners rather than a ring.
	//
	// The row spans the tile and the air either side of it — the same columns
	// the uprights stand in, which is the whole point of it. It was a column
	// short, so every right-hand corner sat one column inside its own upright:
	// the arithmetic was written when the frame stood one column out and did not
	// follow when it went to two.
	inner := g.tileW + 2*frameCols - 2*(arm+1)
	if inner > 0 {
		b.WriteString(strings.Repeat(" ", inner))
	}
	for i := arm; i >= 1; i-- {
		b.WriteString(fade[i].Render(pointerH))
	}
	b.WriteString(fade[0].Render(far))

	if pad := width - left - 2*(arm+1) - max(inner, 0); pad > 0 {
		b.WriteString(strings.Repeat(" ", pad))
	}
	return b.String()
}

const (
	// frameArm is how far a corner reaches along the top or foot of a tile, and
	// frameTall how far down its side.
	//
	// Long enough for the fade to be a fade. At four cells and two rows the arms
	// were the right length for a corner and the wrong length for what is drawn
	// in them: four steps from the accent to the ground is a corner that stops
	// rather than one that goes out. Twice that gives the walk somewhere to
	// happen, and the halves are still nowhere near meeting.
	frameArm  = 8
	frameTall = 4
)

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

// artLine is one row of a rendered cover, or a blank of the tile's width where
// the picture has not arrived or has run out of rows.
func artLine(art []string, line, w int) string {
	if line >= len(art) {
		return strings.Repeat(" ", w)
	}
	return art[line]
}
