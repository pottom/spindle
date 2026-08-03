package cover

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// The placeholder grid is the whole reason the kitty backend survives Bubble
// Tea's line diff: it has to measure as ordinary text of exactly the right
// width, and every cell has to carry its own row and column.
func TestPlaceholderGrid(t *testing.T) {
	const cols, rows = 20, 10

	grid := placeholderGrid(cols, rows)
	lines := strings.Split(grid, "\n")
	if len(lines) != rows {
		t.Fatalf("got %d lines, want %d", len(lines), rows)
	}

	index := make(map[rune]int, len(rowColumnDiacritics))
	for i, r := range rowColumnDiacritics {
		index[r] = i
	}

	for row, line := range lines {
		if w := lipgloss.Width(line); w != cols {
			t.Errorf("row %d measures %d cells, want %d", row, w, cols)
		}

		cells := splitPlaceholders(line)
		if len(cells) != cols {
			t.Fatalf("row %d has %d placeholder cells, want %d", row, len(cells), cols)
		}
		for col, marks := range cells {
			if len(marks) != 2 {
				t.Fatalf("row %d col %d has %d combining marks, want 2", row, col, len(marks))
			}
			if got, ok := index[marks[0]]; !ok || got != row {
				t.Errorf("row %d col %d encodes row %d", row, col, got)
			}
			if got, ok := index[marks[1]]; !ok || got != col {
				t.Errorf("row %d col %d encodes column %d", row, col, got)
			}
		}
	}
}

func TestPlaceholderGridRefusesOversizedArea(t *testing.T) {
	if got := placeholderGrid(len(rowColumnDiacritics)+1, 1); got != "" {
		t.Errorf("got %q, want an empty grid when the area exceeds the diacritic table", got)
	}
}

// splitPlaceholders returns the combining marks attached to each placeholder in
// a rendered row, ignoring the surrounding colour escapes.
func splitPlaceholders(line string) [][]rune {
	var cells [][]rune
	for _, r := range line {
		switch {
		case r == placeholder:
			cells = append(cells, nil)
		case len(cells) > 0 && r >= 0x0300:
			last := len(cells) - 1
			cells[last] = append(cells[last], r)
		}
	}
	return cells
}
