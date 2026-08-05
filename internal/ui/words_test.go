package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// A lyric is written as one line however long it is, and one long line across a
// screen is a row of specks. It is broken until its letters have dots enough to
// be letters.
func TestWordsBreakLongLinesUntilTheyRead(t *testing.T) {
	const w, h = 200 * dotsPerCellX, 40 * dotsPerCellY

	for _, line := range []string{
		"Alone",
		"Do you think you're better off alone?",
		"Never gonna give you up, never gonna let you down, never gonna run around and desert you",
	} {
		lines := wordsWrap(line, w, h)
		longest := 0
		for _, l := range lines {
			longest = max(longest, len([]rune(l)))
		}
		perLetter := int(float64(w)*(1-wordsMargin)) / max(longest, 1)
		t.Logf("%d chars → %d lines, %d dots a letter", len([]rune(line)), len(lines), perLetter)

		if len(lines) > wordsMostLines {
			t.Errorf("broken into %d lines, want at most %d", len(lines), wordsMostLines)
		}
		if perLetter < wordsReadable && len(lines) < wordsMostLines {
			t.Errorf("%d dots a letter over %d lines, want it broken further", perLetter, len(lines))
		}
	}
}

// The face carries Latin and Cyrillic and not CJK, so the picture has to know
// when it cannot draw a song: an empty box for every letter is worse than
// offering something else.
func TestWordsRefuseWhatTheyCannotDraw(t *testing.T) {
	const w, h = 100 * dotsPerCellX, 20 * dotsPerCellY

	for _, line := range []string{"Árvíztűrő tükörfúrógép", "Привет мир", "Über alles"} {
		if _, ok := wordsImage([]string{line}, w, h); !ok {
			t.Errorf("refused to draw %q, which the face has every letter of", line)
		}
	}
	for _, line := range []string{"こんにちは", "안녕하세요"} {
		if _, ok := wordsImage([]string{line}, w, h); ok {
			t.Errorf("drew %q, which the face has no letters of", line)
		}
	}
}

// What the picture shows: the line being sung where there is one, and the record
// itself where there is not — a song with no words in the database must not
// leave the screen empty for three minutes.
func TestWordsShowTheLineOrTheRecord(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 100, 40
	m.ps.TrackID = "now"

	if got := m.wordsNow(); len(got) == 0 || !strings.Contains(got[0], "playing") {
		t.Errorf("with no lyrics the picture shows %q, want the record", got)
	}

	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "first line"},
		{At: 10_000, Words: "the line being sung"},
		{At: 20_000, Words: "later line"},
	}
	m.ps.Progress = 12 * time.Second
	m.setProgress(12 * time.Second)

	got := m.wordsNow()
	if len(got) == 0 || !strings.Contains(strings.Join(got, " "), "being sung") {
		t.Errorf("with lyrics the picture shows %q, want the line being sung", got)
	}
}

// The dots gather: a line arrives scattered and comes together over the moment
// after it, which is what makes a line change something you see.
func TestWordsGatherAfterALineChanges(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	lines := wordsWrap("better off alone", w*dotsPerCellX, rows*dotsPerCellY)
	img, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.text = w, rows, "x"

	filled := func(at time.Duration) (lit int, cells int) {
		m.words.since = time.Now().Add(-at)
		for _, line := range m.wordsLines(w, rows) {
			plain := strings.TrimSpace(ansiOff(line))
			if plain != "" {
				lit++
			}
			cells += len(plain) - strings.Count(plain, " ")
		}
		return lit, cells
	}

	scatteredRows, _ := filled(0)
	settledRows, _ := filled(wordsGather)
	t.Logf("scattered over %d rows, settled into %d", scatteredRows, settledRows)

	if scatteredRows <= settledRows {
		t.Errorf("the line arrived on %d rows and settled onto %d, want it scattered first", scatteredRows, settledRows)
	}

	// And once it has settled it stays settled, rather than drifting on.
	a, _ := filled(wordsGather)
	b, _ := filled(wordsGather * 3)
	if a != b {
		t.Errorf("the settled picture moved from %d rows to %d", a, b)
	}
}

// A change that happens the same way every time stops being a change: each line
// comes apart and goes back together in one of four ways, and which one is its
// own business rather than a coin toss, so a song plays the same way twice.
func TestWordsGatherFourWays(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	lines := wordsWrap("better off alone", w*dotsPerCellX, rows*dotsPerCellY)
	img, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY = w, rows

	picture := func(at time.Duration) string {
		m.words.since = time.Now().Add(-at)
		var sb strings.Builder
		for _, line := range m.wordsLines(w, rows) {
			sb.WriteString(ansiOff(line))
			sb.WriteByte('\n')
		}
		return sb.String()
	}

	half := map[string]wordsMove{}
	var settled string
	for move := wordsDrifting; move < wordsMoves; move++ {
		m.words.move = move

		if got := picture(wordsGather / 2); half[got] != 0 || len(half) > 0 && got == "" {
			t.Errorf("move %d halfway looks like move %d", move, half[got])
		} else {
			half[got] = move
		}

		// However it arrives, it arrives at the same place.
		done := picture(wordsGather)
		if settled == "" {
			settled = done
		} else if done != settled {
			t.Errorf("move %d settled into a different picture", move)
		}
	}
	if len(half) != int(wordsMoves) {
		t.Errorf("%d of the %d moves look alike halfway through", int(wordsMoves)-len(half), wordsMoves)
	}
}

// Which move a line gets is worked out from the line, so it is the same every
// time that line comes round — and different lines get different ones.
func TestWordsMoveComesFromTheLine(t *testing.T) {
	if a, b := wordsMoveFor("better off alone"), wordsMoveFor("better off alone"); a != b {
		t.Errorf("the same line got moves %d and %d", a, b)
	}

	seen := map[wordsMove]bool{}
	for _, line := range []string{
		"Do you think you're better off alone",
		"Never gonna give you up",
		"Árvíztűrő tükörfúrógép",
		"So close, no matter how far",
		"Hello darkness my old friend",
		"We're no strangers to love",
		"Is this the real life",
		"Sultans of swing",
	} {
		seen[wordsMoveFor(line)] = true
	}
	t.Logf("eight lines used %d of the %d moves", len(seen), wordsMoves)
	if len(seen) < 3 {
		t.Errorf("eight lines only ever got %d moves, want them spread", len(seen))
	}
}

// A settled line stands still and its colour moves: the letters are here to be
// read, and the music is carried by how they burn rather than by where they are.
// The bass lights the left of the line and the cymbals the right, so a beat runs
// through the words the way it runs through the spectrum.
func TestWordsColourFollowsTheMusic(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	lines := wordsWrap("better off alone", w*dotsPerCellX, rows*dotsPerCellY)
	img, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY = w, rows
	m.words.since = time.Now().Add(-time.Second)

	// The shape of the line, with the colours taken off, and the whole of it
	// with them left on.
	shape := func() string {
		var sb strings.Builder
		for _, line := range m.wordsLines(w, rows) {
			sb.WriteString(ansiOff(line))
		}
		return sb.String()
	}
	painted := func() string {
		return strings.Join(m.wordsLines(w, rows), "")
	}

	quiet := make([]float32, 28)
	for i := range quiet {
		quiet[i] = 0.1
	}
	m.scope.bands = quiet
	for range 40 {
		m.wordsFlow(w, rows)
	}
	wasShape, wasPaint := shape(), painted()

	// A hit in the bass alone: the left of the line has to light and the right
	// stay where it was.
	loud := make([]float32, 28)
	copy(loud, quiet)
	for i := range 6 {
		loud[i] = 1
	}
	m.scope.bands = loud
	for range 10 {
		m.wordsFlow(w, rows)
	}

	if shape() != wasShape {
		t.Error("the letters moved when the music did, want them still")
	}
	if painted() == wasPaint {
		t.Error("the colour did not change with the music")
	}

	// Left louder than right: the glow under the first columns has to have
	// risen further than the glow under the last.
	left, right := m.words.glow[2], m.words.glow[len(m.words.glow)-3]
	t.Logf("after a bass hit the line burns %.2f on the left and %.2f on the right", left, right)
	if left <= right {
		t.Errorf("the bass lit the line at %.2f on the left and %.2f on the right, want the left hotter", left, right)
	}

	// And it never goes out: a line whose quiet end is dark is a line with holes
	// in it.
	m.scope.bands = quiet
	for range 200 {
		m.wordsFlow(w, rows)
	}
	if shape() != wasShape {
		t.Error("the line lost letters once the music went quiet")
	}
}
