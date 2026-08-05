package ui

import (
	"strings"
	"testing"
)

// chaseModel is a screen with the chase walking across it.
func chaseModel(w, h int) Model {
	m := scopeModel(100, 44)
	m.width, m.height = w, h
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true
	m.chase = chaseState{on: true}

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.5
	}
	m.scope.bands = bands
	return m
}

// It walks: what is on the left of the screen one moment is further right the
// next, and it eats its way along as it goes.
func TestChaseWalksAndEats(t *testing.T) {
	const w, rows = 80, 20
	m := chaseModel(w, rows)

	middle := func() string {
		lines := m.chaseLines(w, rows)
		if len(lines) != rows {
			t.Fatalf("drew %d rows, want %d", len(lines), rows)
		}
		return ansiOff(lines[rows/2])
	}

	first := middle()
	if strings.TrimSpace(first) == "" {
		t.Fatal("nothing was drawn")
	}

	for range 60 {
		m.chaseFlow(w, rows)
	}
	later := middle()

	if first == later {
		t.Error("it never moved")
	}
	if m.chase.eaten == 0 {
		t.Error("it walked past pellets without eating any")
	}

	// The pellets behind it are gone: the left of the row is empty once it has
	// walked past, where it started full.
	left := func(s string) int {
		var n int
		for _, r := range []rune(s)[:20] {
			if r != ' ' {
				n++
			}
		}
		return n
	}
	t.Logf("the first twenty cells held %d marks, and hold %d once it has passed", left(first), left(later))
	if left(later) > left(first) {
		t.Error("it left more behind it than it found")
	}
}

// One bar in three is a chase rather than a face, and which way it walks is the
// bar's own business — so a record plays the same way twice.
func TestChaseComesRoundSometimes(t *testing.T) {
	var chases, backs int
	for at := range int64(300) {
		chase, back := chaseFor(at * 3_000)
		if chase {
			chases++
			if back {
				backs++
			}
		}
	}
	t.Logf("%d of 300 bars were a chase, %d of those walking the other way", chases, backs)

	if chases < 60 || chases > 140 {
		t.Errorf("%d of 300 bars were a chase, want about a third", chases)
	}
	if backs == 0 || backs == chases {
		t.Errorf("%d of %d chases walked the other way, want both directions", backs, chases)
	}

	if a, _ := chaseFor(1234); func() bool { b, _ := chaseFor(1234); return a != b }() {
		t.Error("the same bar answered differently twice")
	}
}
