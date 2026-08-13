package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Outlines: a border round every block of the screen, with its name and size, so
// that how the parts sit against each other can be seen rather than guessed at.
//
// Drawn *over* the block's own edge cells, never around them. A border that
// added a row or a column would move everything inside it, and the screen being
// inspected would stop being the screen that ships — which is the one thing this
// must not do. The price is that the outermost cells of each block are covered
// while it is on, and that is the right way round: a title with its last letter
// hidden still says where the block ends, and a block a column out of place is
// exactly what this is for.
//
// It lives on a debug level rather than behind a build tag or a flag, so turning
// it off costs nothing and there is nothing to take out afterwards.

const (
	outlineH  = "─"
	outlineV  = "│"
	outlineTL = "┌"
	outlineTR = "┐"
	outlineBL = "└"
	outlineBR = "┘"
)

// outline draws the border of one block, named, without changing its size.
//
// The name and the measurements go in the top edge, which is where they cost
// nothing: that row is a rule either way. Where the block is too narrow to
// carry them they are dropped a piece at a time — first the name, then the
// size — rather than being cut into something unreadable.
func (m Model) outline(rows []string, w int, name string) []string {
	if m.debug.level != debugOutlines || w < 4 || len(rows) < 2 {
		return rows
	}

	style := m.styles.Detail
	label := outlineLabel(name, w, len(rows))

	out := make([]string, len(rows))
	out[0] = style.Render(outlineTL + label + strings.Repeat(outlineH, w-2-lipgloss.Width(label)) + outlineTR)
	out[len(rows)-1] = style.Render(outlineBL + strings.Repeat(outlineH, w-2) + outlineBR)

	edge := style.Render(outlineV)
	for i := 1; i < len(rows)-1; i++ {
		out[i] = edge + fit(middleOf(rows[i], w), w-2) + edge
	}
	return out
}

// outlineLabel is what fits in the top edge: the name and the size, then the
// size alone, then nothing.
func outlineLabel(name string, w, h int) string {
	size := fmt.Sprintf("%d×%d", w, h)
	for _, try := range []string{
		" " + name + " " + size + " ",
		" " + size + " ",
	} {
		if lipgloss.Width(try) <= w-2 {
			return try
		}
	}
	return ""
}

// middleOf is the block's own row with the first and last cell taken off, so the
// border can stand in their place. ANSI-aware, because these rows are styled and
// cutting one by byte would leave half a sequence running into the next block.
func middleOf(row string, w int) string {
	row = fit(row, w)
	row = ansi.TruncateLeft(row, 1, "")
	return ansi.Truncate(row, w-2, "")
}
