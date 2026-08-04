package ui

import "github.com/pottom/spindle/internal/ui/cover"

// Dimensions from SCREENS.md section 3. The table is authoritative; the ASCII
// drawings are not.
const (
	minWidth  = 64
	minHeight = 20

	// A table earns its width; prose does not. The lists take whatever the
	// terminal gives — a title cut short at column 60 on a 200-column screen is
	// the one thing nobody would choose — while the player screen stays narrow,
	// because a line of text that wide is harder to read, not easier.
	maxTableWidth = 200
	maxProseWidth = 100

	// On a wide terminal the player screen is let out too, but only enough for
	// the artwork to grow: past this the text beside it starts to drift away
	// from the picture it belongs to.
	maxProseWidthWide = 140

	// compactBelow is where the artwork stops being worth its columns. Two
	// terminals side by side on a laptop is about eighty columns each, and the
	// picture has to survive that — it is most of why the screen looks the way
	// it does. Only well below that is there truly no room for it.
	compactBelow = 68

	// wideAbove is where there is room to spare rather than room to fit, and
	// narrowBelow is where it has to be shared out carefully.
	wideAbove   = 132
	narrowBelow = 96

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

// layoutMode is how a screen divides its body. The three differ in what the
// artwork is competing with: nothing on the player, a list beside it while
// browsing, a list beneath it on the queue.
type layoutMode int

const (
	modePlayer layoutMode = iota
	modeBrowse
	modeQueue
)

// layout is the geometry of one screen, derived purely from the terminal size
// and the pixel size of a cell.
type layout struct {
	interior   int // content width, capped so an ultrawide terminal stays readable
	artWidth   int // artwork area in cells, or zero when there is no room for one
	artHeight  int
	infoWidth  int // the column beside the artwork, or the whole width without one
	bodyHeight int // rows above the help bar
}

// hasArt reports whether this layout has room for a picture.
func (l layout) hasArt() bool { return l.artWidth > 0 && l.artHeight > 0 }

// computeLayout resolves the frame geometry for a terminal of w × h cells. It
// assumes the size already passed fitsMinimum.
func computeLayout(w, h, helpHeight int, hasBanner bool, mode layoutMode, cell cover.CellSize) layout {
	tier := tierFor(w)

	// The lists are tables and take the room; the player is read and does not.
	limit := maxProseWidth
	switch {
	case mode != modePlayer:
		limit = maxTableWidth
	case tier == tierWide:
		limit = maxProseWidthWide
	}
	interior := min(w, limit)

	// Above the body: the tab labels, their rule and a blank line. Below it: a
	// blank line and the help bar, plus one more for a banner.
	chrome := tabBarHeight + 1 + helpHeight
	if hasBanner {
		chrome++
	}
	bodyHeight := max(h-chrome, 0)

	var artWidth, artHeight int
	infoWidth := interior - leftMargin - rightMargin
	if tier != tierCompact {
		artWidth, artHeight = artworkArea(interior, bodyHeight, mode, cell)
		infoWidth -= artWidth + columnGap
	}

	return layout{
		interior:   interior,
		artWidth:   artWidth,
		artHeight:  artHeight,
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
// The queue screen is the odd one out: its list is underneath rather than
// beside, so the artwork competes for rows instead of columns and is held to a
// third of the body. Anything more and the list it heads would be a footnote.
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
	maxHeight := max(bodyHeight-2*artMargin-1, 1)

	// The picture never takes more than two thirds of the height it is given.
	// Filling the body edge to edge reads as cramped however large the picture
	// is, and the rows it leaves are where the waveform goes — a cover that
	// grows until there is no room for it would make the trace disappear on
	// exactly the terminals with the most space.
	if mode == modePlayer {
		maxHeight = max(min(maxHeight, bodyHeight*2/3), 1)
	}
	if mode == modeQueue {
		maxHeight = max(min(maxHeight, bodyHeight/3), 1)
	}

	height = max(min(maxHeight, maxWidth*cell.Width/cell.Height), 1)
	width = max(min(height*cell.Height/cell.Width, maxWidth), 1)
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

// queueDetailWidth is what the detail panel keeps once the trace has its share.
func queueDetailWidth(l layout) int {
	return queueSplit(l) - l.artWidth - 2*columnGap
}

// fitsMinimum reports whether the player screen can be drawn at all.
func fitsMinimum(w, h int) bool {
	return w >= minWidth && h >= minHeight
}
