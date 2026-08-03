package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/ui/style"
)

const appName = "spindle"

// frame draws the outer chrome of SCREENS.md 4.1 one line at a time, so the
// content rows stay plain strings the rest of the view can compose freely.
type frame struct {
	styles style.Styles
	width  int // including both border columns
}

// top renders the header, optionally with a right-aligned label such as the
// active device.
func (f frame) top(right string, rightStyle lipgloss.Style) string {
	b := f.styles.Border
	head := b.Top + " " + appName + " "
	tail := ""
	if right != "" {
		tail = " " + right + " " + b.Top
	}
	fill := max(f.width-2-lipgloss.Width(head)-lipgloss.Width(tail), 0)

	var sb strings.Builder
	sb.WriteString(f.styles.Rule.Render(b.TopLeft + b.Top + " "))
	sb.WriteString(f.styles.AppName.Render(appName))
	sb.WriteString(f.styles.Rule.Render(" " + strings.Repeat(b.Top, fill)))
	if right != "" {
		sb.WriteString(f.styles.Rule.Render(" "))
		sb.WriteString(rightStyle.Render(right))
		sb.WriteString(f.styles.Rule.Render(" " + b.Top))
	}
	sb.WriteString(f.styles.Rule.Render(b.TopRight))
	return sb.String()
}

// row wraps one line of content, fitting it to the interior width.
func (f frame) row(content string) string {
	b := f.styles.Border
	return f.styles.Rule.Render(b.Left) + fit(content, f.width-2) + f.styles.Rule.Render(b.Right)
}

func (f frame) separator() string {
	b := f.styles.Border
	return f.styles.Rule.Render(b.MiddleLeft + strings.Repeat(b.Top, f.width-2) + b.MiddleRight)
}

func (f frame) bottom() string {
	b := f.styles.Border
	return f.styles.Rule.Render(b.BottomLeft + strings.Repeat(b.Top, f.width-2) + b.BottomRight)
}

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

// center places lines in the middle of a w × h block, padding the rest with
// spaces so every returned line is exactly w cells wide.
func center(lines []string, w, h int) []string {
	out := make([]string, h)
	blank := strings.Repeat(" ", w)
	for i := range out {
		out[i] = blank
	}

	top := max((h-len(lines))/2, 0)
	for i, line := range lines {
		row := top + i
		if row >= h {
			break
		}
		lw := lipgloss.Width(line)
		left := max((w-lw)/2, 0)
		out[row] = strings.Repeat(" ", left) + line + strings.Repeat(" ", max(w-left-lw, 0))
	}
	return out
}

// padLeft right-aligns s inside w cells.
func padLeft(s string, w int) string {
	return strings.Repeat(" ", max(w-lipgloss.Width(s), 0)) + s
}
