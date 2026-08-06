package ui

import (
	"strings"
	"testing"
)

// The figures come from drawings, converted by cmd/spindle-figures. What the
// generator has to hand over is a line drawing with a hollow head: the strokes
// are the figure, and the hole is where a face that can blink goes.
func TestAFigureArrivesAsALineDrawingWithAHollowHead(t *testing.T) {
	d, ok := figureFor("robot")
	if !ok {
		t.Fatal("the robot was not generated — run go run ./cmd/spindle-figures")
	}
	if d.licence == "" || d.from == "" {
		t.Error("a figure was generated without saying where it came from")
	}

	for _, tall := range []int{62, 100, 140} {
		p, ok := d.at(tall, "idle")
		if !ok {
			t.Fatalf("no idle pose at %d dots", tall)
		}
		if p.tall != tall {
			t.Errorf("asked for %d dots and got %d", tall, p.tall)
		}

		var lit int
		p.draw(func(_, _ int) { lit++ })
		share := float64(lit) / float64(p.wide*p.tall)
		t.Logf("%3d dots tall: %dx%d, %d lit (%.0f%%), face box %dx%d at %d,%d",
			tall, p.wide, p.tall, lit, share*100, p.headW, p.headH, p.headX, p.headY)

		if lit == 0 {
			t.Fatal("nothing was drawn")
		}
		// A line drawing, not a silhouette: a filled figure would be most of
		// its own box, and on this screen that is a hole rather than a picture.
		if share > 0.30 {
			t.Errorf("%.0f%% of the figure is lit, want a drawing rather than a mass", share*100)
		}

		// The head is hollow, and big enough to put something in.
		var inside int
		p.draw(func(x, y int) {
			if x >= p.headX && x < p.headX+p.headW && y >= p.headY && y < p.headY+p.headH {
				inside++
			}
		})
		if inside != 0 {
			t.Errorf("%d dots are still inside the head, want it cleared for the face", inside)
		}
		if p.headW < 8 || p.headH < 6 {
			t.Errorf("the face box is %dx%d, too small to put a face in", p.headW, p.headH)
		}
	}

	// And the whole walk is there.
	for i := range 8 {
		if _, ok := d.at(100, "walk"+string(rune('0'+i))); !ok {
			t.Errorf("walk frame %d is missing", i)
		}
	}
	for _, pose := range []string{"cheer", "talk", "think", "show", "duck"} {
		if _, ok := d.at(100, pose); !ok {
			t.Errorf("the %s pose is missing", pose)
		}
	}

	// The nearest size is the one that comes back, whatever is asked for.
	p, _ := d.at(1000, "idle")
	if p.tall != 140 {
		t.Errorf("asked for a figure 1000 dots tall and got %d, want the tallest there is", p.tall)
	}
}

// And it draws where it is put.
func TestAFigureDrawsWhereItIsPut(t *testing.T) {
	d, _ := figureFor("robot")
	p, ok := d.at(100, "walk2")
	if !ok {
		t.Fatal("no pose")
	}

	grid := make([][]byte, p.tall)
	for y := range grid {
		grid[y] = []byte(strings.Repeat(".", p.wide))
	}
	p.draw(func(x, y int) {
		if x < 0 || y < 0 || x >= p.wide || y >= p.tall {
			t.Fatalf("a dot at %d,%d is outside the %dx%d figure", x, y, p.wide, p.tall)
		}
		grid[y][x] = '#'
	})

	var rows int
	for _, row := range grid {
		if strings.ContainsRune(string(row), '#') {
			rows++
		}
	}
	t.Logf("the walking robot covers %d of its %d rows", rows, p.tall)
	if rows < p.tall/2 {
		t.Errorf("the figure covers %d of %d rows, want it filling its own box", rows, p.tall)
	}
}
