package ui

import (
	"testing"

	"github.com/pottom/spindle/internal/ui/cover"
)

func TestListMoveClamps(t *testing.T) {
	var l listState

	l.move(-1, 5)
	if l.cursor != 0 {
		t.Errorf("cursor = %d, want 0 at the top", l.cursor)
	}

	l.move(99, 5)
	if l.cursor != 4 {
		t.Errorf("cursor = %d, want 4 at the bottom", l.cursor)
	}
}

func TestListMoveOnEmptyList(t *testing.T) {
	l := listState{cursor: 3, top: 2}
	l.move(1, 0)
	if l.cursor != 0 || l.top != 0 {
		t.Errorf("cursor, top = %d, %d; want 0, 0", l.cursor, l.top)
	}
}

// The window has to scroll only as far as it must: jumping the viewport around
// a cursor that is already visible is what makes a list feel unsteady.
func TestListWindowScrollsMinimally(t *testing.T) {
	const count, height = 20, 5
	var l listState

	if from, to := l.window(count, height); from != 0 || to != 5 {
		t.Fatalf("initial window = %d..%d, want 0..5", from, to)
	}

	// Moving inside the visible range must not scroll at all.
	l.move(4, count)
	if from, to := l.window(count, height); from != 0 || to != 5 {
		t.Errorf("window = %d..%d after moving to the last visible row, want 0..5", from, to)
	}

	// One step past it scrolls by exactly one.
	l.move(1, count)
	if from, to := l.window(count, height); from != 1 || to != 6 {
		t.Errorf("window = %d..%d, want 1..6", from, to)
	}

	// Jumping back to the top brings the window with it.
	l.move(-99, count)
	if from, to := l.window(count, height); from != 0 || to != 5 {
		t.Errorf("window = %d..%d after returning to the top, want 0..5", from, to)
	}
}

func TestListWindowClampsToContents(t *testing.T) {
	l := listState{cursor: 40, top: 30}
	from, to := l.window(3, 10)
	if from != 0 || to != 3 {
		t.Errorf("window = %d..%d, want 0..3 when the list shrank underneath", from, to)
	}
	if l.cursor != 2 {
		t.Errorf("cursor = %d, want 2", l.cursor)
	}
}

func TestListWindowHandlesNothingToShow(t *testing.T) {
	var l listState
	if from, to := l.window(0, 5); from != 0 || to != 0 {
		t.Errorf("window = %d..%d, want 0..0 for an empty list", from, to)
	}
	if from, to := l.window(10, 0); from != 0 || to != 0 {
		t.Errorf("window = %d..%d, want 0..0 with no room", from, to)
	}
}

// defaultTestCell is a measured cell of the shape a modern terminal reports, so
// layout-dependent tests do not silently exercise the fallback.
var defaultTestCell = cover.CellSize{Width: 17, Height: 41, Measured: true}
