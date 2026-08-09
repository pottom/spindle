package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// sheet is a set of lines a second apart, as a record would send them.
func sheet(lines ...string) []player.Lyric {
	out := make([]player.Lyric, len(lines))
	for i, line := range lines {
		out[i] = player.Lyric{At: int64(i+1) * 1000, Words: line}
	}
	return out
}

// The chorus is found in the sheet, and found again through the punctuation a
// sheet puts in one time and leaves out the next.
func TestTheRefrainIsFoundInTheSheet(t *testing.T) {
	m := scopeModel(100, 40)
	m.adoptLyrics(msg.LyricsFetched{
		TrackID: "now",
		Synced:  true,
		Lines: sheet(
			"the first verse",
			"take me home",
			"another verse",
			"Take me home,",
			"one more verse",
			"take   me home",
		),
	})

	want := []bool{false, false, false, true, false, true}
	for i, line := range m.lyrics.lines {
		if got := m.refrain.again[line.At]; got != want[i] {
			t.Errorf("%q is a return: %v, want %v", line.Words, got, want[i])
		}
	}

	// And a record whose sheet says nothing twice has no refrain at all, rather
	// than one line standing in for one.
	m.adoptLyrics(msg.LyricsFetched{
		TrackID: "now",
		Synced:  true,
		Lines:   sheet("one", "two", "three"),
	})
	if len(m.refrain.again) != 0 {
		t.Errorf("a sheet with no repeat found %d returns", len(m.refrain.again))
	}
}

// The rhyme itself, which is what the shadow is there to show: a line that
// comes round again arrives exactly as it arrived the first time — the same way
// in, the same way of keeping time, the same words leaning the same way.
//
// It has been true since the moves were written, because all three are dealt
// from the line's own text. It is locked down here because it is the whole
// point of the picture the sheet's repetition buys, and nothing else in the
// tests would notice it being dealt from anything else.
func TestALineThatComesRoundAgainArrivesTheSameWay(t *testing.T) {
	const line = "do you think you're better off alone"

	first := wordsMoveFor(line)
	if again := wordsMoveFor(line); again != first {
		t.Errorf("the line came in as %d and back as %d, want the same way", first, again)
	}
	if a, b := wordsRideFor(line), wordsRideFor(line); a != b {
		t.Errorf("the line kept time as %d and came back as %d, want the same", a, b)
	}
	if a, b := wordsLeans(line), wordsLeans(line); a != b {
		t.Errorf("the line leaned %v and came back %v, want the same", a, b)
	}

	// And two different lines are not all dealt the one arrival, or there would
	// be nothing for a return to be the same as.
	var moves int
	seen := map[wordsMove]bool{}
	for _, other := range []string{
		"talk to me", "so why do you", "i can hear the wind", "hold me now",
		"and the night came down", "nothing left to say", "we were young",
	} {
		if !seen[wordsMoveFor(other)] {
			seen[wordsMoveFor(other)] = true
			moves++
		}
	}
	if moves < 3 {
		t.Errorf("seven lines were dealt %d different arrivals, want a spread", moves)
	}
}

// The shadow: while a returning line is on its way in, it is already faintly
// where it is going. A line the record has not sung yet arrives out of nothing.
func TestALineSungBeforeCastsItsShadow(t *testing.T) {
	const w, rows = 90, 14
	const line = "hey baby"

	// The picture at a given point in its arrival, with or without the record
	// having sung the line before.
	draw := func(again bool, at float32) string {
		m := scopeModel(100, 44)
		m.width, m.height = w, rows
		m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

		img, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
		if !ok {
			t.Fatalf("%q could not be drawn", line)
		}
		m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
		m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
		m.words.text, m.words.starts = line, 4000
		m.words.move = wordsMoveFor(line)

		m.words.since = time.Now().Add(-time.Duration(at * float32(wordsGather)))

		if again {
			m.adoptLyrics(msg.LyricsFetched{
				TrackID: "now", Synced: true,
				Lines: sheet("hey baby", "a verse", "of some kind", "hey baby"),
			})
			if !m.wordsAgain() {
				t.Fatal("the line was sung twice in the sheet and did not count as a return")
			}
		}
		return strings.Join(m.wordsLines(w, rows), "\n")
	}

	// Measured against the line's own place: the cells the picture lights once
	// it has arrived, counted while it is still on its way there. Dots crossing
	// those rows on their way in light some of them by chance, so the question
	// is how much of the finished picture is already standing.
	top, bottom := wordsRowsOf(t, line, w, rows)
	cells := func(picture string) map[[2]int]bool {
		out := map[[2]int]bool{}
		for r, row := range strings.Split(picture, "\n") {
			if r < top || r > bottom {
				continue
			}
			for c, ch := range []rune(ansiOff(row)) {
				if ch != ' ' {
					out[[2]int{r, c}] = true
				}
			}
		}
		return out
	}

	home := cells(draw(false, 4))
	standing := func(picture string) int {
		var n int
		for cell := range cells(picture) {
			if home[cell] {
				n++
			}
		}
		return n
	}

	for _, at := range []float32{0.1, 0.4} {
		fresh, back := standing(draw(false, at)), standing(draw(true, at))
		t.Logf("%.0f%% of the way in, %d of the line's %d cells are standing for a new line and %d for one sung before",
			at*100, fresh, len(home), back)

		if back < len(home) {
			t.Errorf("%d of the returning line's %d cells were standing at %.0f%%, want the whole shadow",
				back, len(home), at*100)
		}
		if fresh >= len(home)/2 {
			t.Errorf("a line the record has not sung had %d of its %d cells standing at %.0f%%, so there is nothing to tell apart",
				fresh, len(home), at*100)
		}
	}
}

// wordsRowsOf is the band of rows a line is set in, so a test can measure what
// is at its destination without counting the meter underneath it.
func wordsRowsOf(t *testing.T, line string, w, rows int) (top, bottom int) {
	t.Helper()
	layout := mustLayout(t, line, w, rows)

	top, bottom = rows, 0
	for _, row := range layout.Tops {
		top = min(top, row/dotsPerCellY)
	}
	for _, row := range layout.Bottoms {
		bottom = max(bottom, row/dotsPerCellY)
	}
	return top, bottom
}

func mustLayout(t *testing.T, line string, w, rows int) msg.WordLayout {
	t.Helper()
	_, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatalf("%q could not be drawn", line)
	}
	return layout
}
