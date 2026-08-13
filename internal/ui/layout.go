package ui

import "github.com/pottom/spindle/internal/ui/cover"

// Dimensions from SCREENS.md section 3. The table is authoritative; the ASCII
// drawings are not.
const (
	minWidth  = 64
	minHeight = 20

	// A table earns its width; prose does not. The lists take whatever the
	// terminal gives — a title cut short at column 60 on a 200-column screen is
	// the one thing nobody would choose.
	maxTableWidth = 200

	// The player used to be capped narrower than this, which left a third of a
	// large terminal blank. The reason written down for it — that a wide line of
	// text is harder to read — is an argument about the text rather than about
	// the frame it sits in, so it moved to where it belongs, one line down.
	//
	// maxInfoCols is that place: the column of words beside
	// the picture. A title, an artist and a caption set much wider than this stop
	// being a caption and start being a paragraph, and the eye has to travel back
	// across the picture to find the start of the next line.
	maxInfoCols = 64

	// compactBelow is where the artwork stops being worth its columns. Two
	// terminals side by side on a laptop is about eighty columns each, and the
	// picture has to survive that — it is most of why the screen looks the way
	// it does. Only well below that is there truly no room for it.
	compactBelow = 68

	// wideAbove is where there is room to spare rather than room to fit, and
	// narrowBelow is where it has to be shared out carefully.
	wideAbove   = 132
	narrowBelow = 96

	// maxArtPx is how large the cover may get on screen, in device pixels down
	// its side. In cells it would not be a cap at all: shrinking the terminal
	// font makes a cell smaller and the same number of cells physically larger,
	// which is exactly the way it kept growing.
	//
	// Set to what it measures at on this machine at a comfortable size, because
	// that is the size it was tuned to look right at — past it the sleeve starts
	// competing with the room rather than sitting in it.
	maxArtPx = 1100

	// maxBrowseArt caps the picture on the screens where it is a preview rather
	// than the subject. Left to grow with the terminal it would take half of a
	// very wide screen from the list, which is the thing being read there.
	maxBrowseArt = 40

	leftMargin  = 3
	rightMargin = 2
	columnGap   = 3
	minInfoCols = 32

	// artMargin is the blank line kept above and below the artwork.
	artMargin = 1

	// tabBarHeight is the labels, the rule beneath the active one, and a blank
	// line separating them from the body.
	tabBarHeight = 3
)

// widthTier is how much room the terminal has given. It decides the one thing
// width alone can decide: whether there is space for a picture.
type widthTier int

const (
	// tierCompact has no artwork. Everything else keeps working.
	tierCompact widthTier = iota
	tierNormal
	tierWide
)

func tierFor(w int) widthTier {
	switch {
	case w < compactBelow:
		return tierCompact
	case w >= wideAbove:
		return tierWide
	default:
		return tierNormal
	}
}

// layoutMode is how a screen divides its body. The two differ in what the
// artwork is competing with: nothing on the player, and a list beneath it on
// every screen that is a list.
type layoutMode int

const (
	modePlayer layoutMode = iota
	modeList
)

// layout is the geometry of one screen, derived purely from the terminal size
// and the pixel size of a cell.
type layout struct {
	interior   int // content width, capped so an ultrawide terminal stays readable
	artWidth   int // artwork area in cells, or zero when there is no room for one
	artHeight  int
	artRows    int // rows the picture will really fill, which can be fewer
	infoWidth  int // the column beside the artwork, or the whole width without one
	bodyHeight int // rows above the help bar
}

// hasArt reports whether this layout has room for a picture.
func (l layout) hasArt() bool { return l.artWidth > 0 && l.artHeight > 0 }

// computeLayout resolves the frame geometry for a terminal of w × h cells. It
// assumes the size already passed fitsMinimum.
func computeLayout(w, h, helpHeight int, hasBanner bool, mode layoutMode, cell cover.CellSize) layout {
	tier := tierFor(w)

	// The lists take the width and fill it with columns. The player works the
	// other way round: it is as wide as its two parts need, and centred in what
	// is left, because a caption stretched across a very wide terminal is a
	// worse use of the room than the blank either side of it.
	interior := min(w, maxTableWidth)

	// Above the body: the tab labels, their rule and a blank line. Below it: a
	// blank line and the help bar, plus one more for a banner.
	chrome := tabBarHeight + 1 + helpHeight
	if hasBanner {
		chrome++
	}
	bodyHeight := max(h-chrome, 0)

	var artWidth, artHeight, artRows int
	infoWidth := interior - leftMargin - rightMargin
	if tier != tierCompact {
		// The player is measured against the whole terminal rather than the
		// capped interior, so that a picture on a very wide screen can grow into
		// the rows it has. Everything the words then hand back goes to it too.
		room := interior
		if mode == modePlayer {
			room = w
		}

		artWidth, artHeight = artworkArea(room, bodyHeight, mode, cell)
		infoWidth = room - leftMargin - rightMargin - artWidth - columnGap

		if mode == modePlayer {
			// The picture takes what it can of the surplus first, since it is the
			// subject of this screen. It is usually held back by the rows rather
			// than the columns, so this runs out quickly.
			if infoWidth > maxInfoCols {
				artWidth, artHeight = squareOff(artWidth+infoWidth-maxInfoCols, bodyHeight, cell)
			}

			// And the rest goes to the column beside it, all of it.
			//
			// It was capped here, and the frame closed around the two of them and
			// centred, on the reasoning that a caption two hundred columns wide is
			// not more readable for it. That reasoning is sound and the result was
			// still wrong: it left a blank band down both sides of a wide terminal,
			// which is exactly what shrinking the font is meant to get rid of. The
			// line length is the text's business — the words are short and ragged
			// right, so a wider column costs nothing — and the frame's business is
			// to fill the screen.
			infoWidth = max(interior-leftMargin-rightMargin-artWidth-columnGap, minInfoCols)
		}
		artRows = artworkRows(artWidth, artHeight, cell)
	}

	return layout{
		interior:   interior,
		artWidth:   artWidth,
		artHeight:  artHeight,
		artRows:    artRows,
		infoWidth:  infoWidth,
		bodyHeight: bodyHeight,
	}
}

// artworkArea gives the artwork as much room as the frame can spare while keeping
// it square on screen and leaving the information column its minimum width. Cells
// are taller than they are wide, so a square area needs roughly twice as many
// columns as rows.
// The artwork never takes more than half the width: past that it stops being a
// player and starts being a picture viewer with captions. The browsing tabs give
// it less still, because a list of titles and artists needs the room more than a
// preview does.
//
// A list screen puts its list underneath rather than beside, so the artwork
// competes for rows instead of columns and is held to a third of the body.
// Anything more and the list it heads would be a footnote.
func artworkArea(interior, bodyHeight int, mode layoutMode, cell cover.CellSize) (width, height int) {
	share := interior / 2
	switch {
	case mode != modePlayer:
		share = min(interior*2/5, maxBrowseArt)
	case interior < narrowBelow:
		// Where there is little to go round, the words get more of it: a title
		// cut in half is a worse loss than a smaller picture.
		share = interior * 2 / 5
	}

	maxWidth := max(min(interior-leftMargin-columnGap-minInfoCols-rightMargin, share), 1)
	if mode == modePlayer && cell.Width > 0 {
		maxWidth = max(min(maxWidth, maxArtPx/cell.Width), 1)
	}
	maxHeight := max(bodyHeight-2*artMargin-1, 1)

	// The picture never takes more than two thirds of the height it is given.
	// Filling the body edge to edge reads as cramped however large the picture
	// is, and the rows it leaves are where the waveform goes — a cover that
	// grows until there is no room for it would make the trace disappear on
	// exactly the terminals with the most space.
	if mode == modePlayer {
		// The picture stops where the trace begins rather than at a fixed share
		// of the body. Two thirds was the rule, and on a tall terminal it left a
		// third of the screen holding nothing: every other part of this screen is
		// a fixed height, so whatever the picture does not take is not taken at
		// all.
		//
		// Two thirds stays as a floor, and it is the floor that matters on a
		// short terminal: holding rows back there would shrink the picture to
		// make room for a trace that is not offered at that size anyway, which is
		// the worst of both. The two meet at a body of thirty-six rows, and above
		// that the picture is the one that grows.
		grown := max(bodyHeight-playerBelowArt, bodyHeight*2/3)
		maxHeight = max(min(maxHeight, grown), 1)
	}
	if mode == modeList {
		maxHeight = max(min(maxHeight, bodyHeight/3), 1)
	}

	height = max(min(maxHeight, maxWidth*cell.Width/cell.Height), 1)
	width = max(min(height*cell.Height/cell.Width, maxWidth), 1)
	return width, height
}

// playerBelowArt is the rows the player screen keeps back from the picture so
// that the trace has somewhere to go.
//
// Twice what the trace needs, because the picture is centred in the body: half
// of whatever is spare ends up above it and only the other half below, which is
// where the trace lives. Held back as one number rather than by shrinking the
// picture when the key is pressed — a visualiser that moved the cover every time
// it was turned on would not be worth having.
const playerBelowArt = 2 * (scopeRows + scopeChrome + artMargin)

// squareOff takes a width the picture has been offered and gives back the
// largest square that fits in it without passing the rows available. Used when
// the words hand back what they cannot use: the offer is a width, and a width
// alone would make a picture wider than it is tall.
func squareOff(offered, bodyHeight int, cell cover.CellSize) (width, height int) {
	// The ceiling holds here too. This is the path the words hand their surplus
	// back on, and it would otherwise walk straight past a cap the other path
	// obeys.
	if cell.Width > 0 {
		offered = max(min(offered, maxArtPx/cell.Width), 1)
	}
	rows := max(bodyHeight-playerBelowArt, 1)
	height = max(min(rows, offered*cell.Width/cell.Height), 1)
	width = max(min(height*cell.Height/cell.Width, offered), 1)
	return width, height
}

// queueScopeMin is the narrowest trace worth drawing beside the queue's detail
// panel. Below it the bands are wider than they are tall and read as a bar
// chart of nothing in particular.
const queueScopeMin = 28

// queueBlockWidth is the queue screen's content, artwork and all. queueRowWidth
// is what its rows are laid out in: the scrollbar's column is held whether there
// is a bar or not, so the columns do not step sideways the moment the list grows
// past the screen.
func queueBlockWidth(l layout) int { return l.artWidth + columnGap + l.infoWidth }
func queueRowWidth(l layout) int   { return queueBlockWidth(l) - scrollCols }

// queueSplit is where the right-hand half of the queue screen begins. The trace
// above and the artists below start on the same column, which is what makes the
// screen read as two columns rather than four.
func queueSplit(l layout) int { return rowSecondaryAt(queueRowWidth(l)) }

// queueScopeWidth is the trace's share of the row, and zero when the detail
// panel beside the artwork would be left too narrow to read.
func queueScopeWidth(l layout) int {
	if !l.hasArt() || queueDetailWidth(l) < minInfoCols {
		return 0
	}
	w := queueBlockWidth(l) - queueSplit(l)
	if w < queueScopeMin {
		return 0
	}
	return w
}

// nowPanelMin is the narrowest right-hand half worth giving to what is playing:
// a small cover and a caption beside it. Below this the caption is a column of
// broken words, and the rows are better left to the list.
const nowPanelMin = 34

// nowCaptionMin is what the words beside that cover need.
const nowCaptionMin = 18

// nowPanelWidth is the right-hand half of a browsing screen's top band, where
// what is playing goes. It starts at the same column the artists below do, so
// the band and the list read as the same two halves — the trace on the queue
// hangs from that column for the same reason.
func nowPanelWidth(l layout) int {
	if !l.hasArt() || queueDetailWidth(l) < minInfoCols {
		return 0
	}
	w := queueBlockWidth(l) - queueSplit(l)
	if w < nowPanelMin {
		return 0
	}
	return w
}

// nowCoverBox is the picture inside that panel: as tall as the picture beside
// it, and no wider than the panel can spare with the caption still readable.
func nowCoverBox(l layout) (w, h int) {
	panel := nowPanelWidth(l)
	if panel == 0 {
		return 0, 0
	}
	return min(l.artWidth, panel-nowCaptionMin-columnGap), l.artHeight
}

// queueDetailWidth is what the detail panel keeps once the trace has its share.
func queueDetailWidth(l layout) int {
	return queueSplit(l) - l.artWidth - 2*columnGap
}

// artworkRows is how many rows the picture will actually fill inside the box it
// has been given. A cover is square and cells are not, and no whole number of
// cells is exactly square, so one of the two dimensions always has slack: the
// box is as wide as it was asked to be and comes out a row or two short.
//
// Anything laid out beside the artwork has to know this. The box is where the
// picture may go; this is where it ends, and a line below that reads as a
// mistake. It is derived rather than measured so that it holds before a cover
// has arrived as well as after — a budget that changed when the picture loaded
// would show a row and then take it away again.
func artworkRows(artWidth, artHeight int, cell cover.CellSize) int {
	if artWidth <= 0 || artHeight <= 0 || cell.Width <= 0 || cell.Height <= 0 {
		return artHeight
	}
	side := min(artWidth*cell.Width, artHeight*cell.Height)
	return min(max(side/cell.Height, 1), artHeight)
}

// fitsMinimum reports whether the player screen can be drawn at all.
func fitsMinimum(w, h int) bool {
	return w >= minWidth && h >= minHeight
}
