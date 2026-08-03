package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// fit truncates s with an ellipsis and pads it out to exactly w cells. Styled
// input is handled: widths and cuts are ANSI-aware. A string that already fits
// is passed through untouched, which keeps artwork escape sequences intact.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	width := lipgloss.Width(s)
	if width > w {
		s = ansi.Truncate(s, w, "…")
		width = lipgloss.Width(s)
	}
	return s + strings.Repeat(" ", max(w-width, 0))
}

// spread puts left and right on one line of w cells with the gap between them.
func spread(left, right string, w int) string {
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return fit(left+strings.Repeat(" ", gap)+right, w)
}

// stack centres lines vertically in an h-row block of w cells, leaving them left
// aligned. Used for text, where centring the lines themselves would look sloppy.
func stack(lines []string, w, h int) []string {
	out := make([]string, h)
	top := max((h-len(lines))/2, 0)
	for i := range out {
		row := i - top
		if row >= 0 && row < len(lines) {
			out[i] = fit(lines[row], w)
			continue
		}
		out[i] = strings.Repeat(" ", w)
	}
	return out
}

// padLeft right-aligns s inside w cells.
func padLeft(s string, w int) string {
	return strings.Repeat(" ", max(w-lipgloss.Width(s), 0)) + s
}

// center places lines in the middle of a w × h block, padding the rest with
// spaces so every returned line is exactly w cells wide.
func center(lines []string, w, h int) []string {
	return place(lines, w, h, max((h-len(lines))/2, 0))
}

// alignTop is center's sibling for the browsing tabs, where the artwork has to
// start on the same row as the list heading beside it.
func alignTop(lines []string, w, h int) []string {
	return place(lines, w, h, 0)
}

func place(lines []string, w, h, top int) []string {
	out := make([]string, h)
	blank := strings.Repeat(" ", w)
	for i := range out {
		out[i] = blank
	}

	for i, line := range lines {
		row := top + i
		if row >= h {
			break
		}
		lw := min(lipgloss.Width(line), w)
		left := max((w-lw)/2, 0)
		out[row] = fit(strings.Repeat(" ", left)+line, w)
	}
	return out
}

// meter draws a filled bar of w cells, using the two styles for the played and
// the remaining part.
func meter(fraction float64, w int, on, off lipgloss.Style) string {
	if w <= 0 {
		return ""
	}
	filled := min(max(int(fraction*float64(w)+0.5), 0), w)
	return on.Render(strings.Repeat(meterFull, filled)) +
		off.Render(strings.Repeat(meterEmpty, w-filled))
}
