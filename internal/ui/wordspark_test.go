package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// sparkModel is a line of type on the full screen with a beat to spark on.
func sparkModel(t *testing.T, w, rows int) Model {
	t.Helper()
	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on, m.stage.loose = scopeWords, true, true

	img, layout, ok := wordsImage([]string{"jaj de jo"}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.text = w, rows, "jaj de jo"
	m.words.where = layout
	m.words.since = time.Now().Add(-5 * time.Second)
	m.words.went = time.Now().Add(-5 * time.Second)

	m.ps = &player.State{TrackID: "one", Duration: 4 * time.Minute, Playing: true}
	m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
	m.run.count, m.run.forTrack = 8, "one"
	return m
}

// sparkRows is how many rows of the picture the sparks are responsible for, at a
// moment inside the beat.
//
// Measured as the difference the sparks make to the drawn picture rather than as
// dots they light, because they do not land on empty screen: the meter's band is
// under the type and that is where they fall. Counting lit cells found two of
// two hundred and thirty-nine and called a working effect broken.
func sparkRows(m Model, w, rows int, into time.Duration) int {
	m.scope.beatAt = time.Now().Add(-into)

	m.stage.loose = true
	with := m.wordsLines(w, rows)
	m.stage.loose = false
	without := m.wordsLines(w, rows)

	n := 0
	for i := range with {
		if i < len(without) && with[i] != without[i] {
			n++
		}
	}
	return n
}

// The type sparks on the beat, all through it, and stops before the next.
func TestTheTypeSparksThroughTheBeat(t *testing.T) {
	const w, rows = 90, 30
	m := sparkModel(t, w, rows)

	for _, into := range []time.Duration{0, 100, 200, 300} {
		if n := sparkRows(m, w, rows, into*time.Millisecond); n == 0 {
			t.Errorf("%v into the beat, the sparks changed nothing", into*time.Millisecond)
		}
	}
	// Out of light before the next dozen let go, or four glints a bar becomes a
	// drizzle, which is what the water is for.
	if n := sparkRows(m, w, rows, 440*time.Millisecond); n != 0 {
		t.Errorf("at the end of the beat the sparks are still lighting %d rows", n)
	}
}

// Loose, there are none. There is no beat to spark on, and a rhythm the record
// has not got is worse than no rhythm.
func TestTheTypeDoesNotSparkWhenItIsNotKeepingTime(t *testing.T) {
	const w, rows = 90, 30
	m := sparkModel(t, w, rows)
	m.scope.beatAt = time.Now().Add(-150 * time.Millisecond)

	// The same picture at two points of the beat, because with no beat kept
	// there is nothing for a point of the beat to mean.
	m.stage.loose = false
	early := strings.Join(m.wordsLines(w, rows), "\n")
	m.scope.beatAt = time.Now().Add(-350 * time.Millisecond)
	late := strings.Join(m.wordsLines(w, rows), "\n")
	if early != late {
		t.Error("the picture moved with the beat while the screen was not keeping it")
	}
}

// The same beat sparks the same way, so a record plays the same twice.
func TestTheSparksAreDealtFromTheBeat(t *testing.T) {
	for k := range wordsSparkEach {
		if wordsSparkRoll(11, k) == wordsSparkRoll(12, k) {
			t.Errorf("spark %d comes off the same column on two beats running", k)
		}
	}
	// And the dozen do not all come from one place.
	seen := map[uint64]bool{}
	for k := range wordsSparkEach {
		seen[wordsSparkRoll(7, k)%360] = true
	}
	if len(seen) < wordsSparkEach/2 {
		t.Errorf("the dozen sparks came off only %d columns", len(seen))
	}
}

// The sparks fall under the same pull as the line that lets go, and as the water
// and the volume's lamps. Three unrelated things obeying one gravity is what
// makes them one picture rather than three effects, so it is worth a test rather
// than a comment.
func TestTheSparksFallUnderTheSamePullAsTheLine(t *testing.T) {
	if wordsSparkPull != wordsSpillPull {
		t.Errorf("the sparks fall at %v and the line at %v", wordsSparkPull, wordsSpillPull)
	}
}
