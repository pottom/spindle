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
		if _, _, ok := wordsImage([]string{line}, w, h); !ok {
			t.Errorf("refused to draw %q, which the face has every letter of", line)
		}
	}
	for _, line := range []string{"こんにちは", "안녕하세요"} {
		if _, _, ok := wordsImage([]string{line}, w, h); ok {
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
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.text = w, rows, "x"
	m.words.where = layout

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
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout

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

// The picture's whole idea: a word is coloured by the sound that was in the air
// as it went by, and keeps that colour for the rest of the line. What has been
// sung is a record of how it sounded, what is being sung burns, and what is to
// come waits — and the letters themselves never move.
func TestWordsKeepTheColourTheyWereSungIn(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true
	m.ps.TrackID = "now"

	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "better off alone tonight"},
		{At: 8_000, Words: "and after"},
	}

	lines := wordsWrap("better off alone tonight", w*dotsPerCellX, rows*dotsPerCellY)
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	if layout.Count != 4 {
		t.Fatalf("the layout found %d words in a line of four", layout.Count)
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
	m.words.since = time.Now().Add(-time.Second)

	shape := func() string {
		var sb strings.Builder
		for _, line := range m.wordsLines(w, rows) {
			sb.WriteString(ansiOff(line))
		}
		return sb.String()
	}
	painted := func() string { return strings.Join(m.wordsLines(w, rows), "") }

	// The first word, sung over a bass note.
	bass := make([]float32, 28)
	for i := range bass {
		bass[i] = 0.1
	}
	bass[1] = 1
	m.scope.bands = bass
	m.setProgress(500 * time.Millisecond)
	m.wordsFlow(w, rows)

	first := m.words.paint[0]
	if !first.set {
		t.Fatal("the word being sung was not painted at all")
	}

	// The third word, sung over a cymbal: a different colour entirely.
	high := make([]float32, 28)
	for i := range high {
		high[i] = 0.1
	}
	high[26] = 1
	m.scope.bands = high
	m.setProgress(5 * time.Second)

	was := shape()
	m.wordsFlow(w, rows)
	t.Logf("word 0 kept hue %d, and the word being sung is %d", first.hue, m.words.sung)

	if m.words.sung == 0 {
		t.Fatal("the singer never reached the second word")
	}
	if got := m.words.paint[m.words.sung]; !got.set || got.hue == first.hue {
		t.Errorf("a word sung over a cymbal took hue %d, the same as one sung over a bass note", got.hue)
	}
	// The earlier word kept what it was sung in rather than following the music.
	if m.words.paint[0].hue != first.hue {
		t.Errorf("the first word repainted itself from %d to %d", first.hue, m.words.paint[0].hue)
	}
	// And nothing moved.
	if shape() != was {
		t.Error("the letters moved when the colour did, want them still")
	}
	if painted() == "" {
		t.Error("nothing was drawn")
	}
}

// A lyric rarely fills a screen, and empty is the one thing this picture can
// least afford: what the words leave over goes to the music.
func TestWordsGiveWhatIsLeftToTheMusic(t *testing.T) {
	const w, rows = 90, 30

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.8
	}
	m.scope.bands = bands

	lines := wordsWrap("better off alone", w*dotsPerCellX, rows*dotsPerCellY)
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
	m.words.since = time.Now().Add(-time.Second)

	from, tall := m.wordsRoom(rows)
	t.Logf("the words leave rows %d to %d, %d of them", from, rows, tall)
	if tall < wordsBand {
		t.Fatalf("only %d rows were left over, so there is nothing to test", tall)
	}

	drawn := m.wordsLines(w, rows)
	var band int
	for _, line := range drawn[from:] {
		if strings.TrimSpace(ansiOff(line)) != "" {
			band++
		}
	}
	if band == 0 {
		t.Error("the rows under the words are empty, want the music in them")
	}

	// And the words are still above it.
	var above int
	for _, line := range drawn[:from] {
		if strings.TrimSpace(ansiOff(line)) != "" {
			above++
		}
	}
	if above == 0 {
		t.Error("the words went missing when the music was put under them")
	}
}

// A song between two lines is a song playing: the screen goes to the music
// rather than holding the last line up or going blank. A song with no lyrics at
// all is a different case, and keeps its title.
func TestWordsGiveTheScreenBackBetweenLines(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 90, 30
	m.ps.TrackID = "now"

	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 20_000, Words: "the first line, long after the start"},
		{At: 30_000, Words: ""},
		{At: 40_000, Words: "and after the solo"},
	}

	// The opening bars, before anything is sung.
	m.setProgress(time.Second)
	if !m.wordsSilent() {
		t.Error("the screen is showing words before the first line is sung")
	}

	// The line itself.
	m.setProgress(22 * time.Second)
	if m.wordsSilent() {
		t.Error("the screen went to the music while a line was being sung")
	}

	// The gap the sheet marks with an empty line.
	m.setProgress(34 * time.Second)
	if !m.wordsSilent() {
		t.Error("the screen held a line up through the solo")
	}

	// A song with no lyrics at all keeps its title instead.
	m.lyrics.synced = false
	if m.wordsSilent() {
		t.Error("a song with no lyrics gave up its title")
	}
}

// The words are complete as the line begins, rather than starting to gather
// then: a picture that arrives half a second late is a picture that is always
// behind the singer.
func TestWordsLandOnTheLine(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 90, 20
	m.ps.TrackID = "now"

	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 10_000, Words: "the line to come"},
		{At: 20_000, Words: "and the next"},
	}

	// Well before the line: nothing to set yet.
	m.setProgress(5 * time.Second)
	if lines, _ := m.wordsComing(); len(lines) != 0 {
		t.Errorf("the line was asked for %v early, want it left until its gathering starts", 5*time.Second)
	}

	// Inside the gathering's length of it: asked for, with its own timestamp so
	// that the arrival can be wound back to land on it.
	m.setProgress(10*time.Second - wordsGather/2 - lyricsAhead)
	lines, starts := m.wordsComing()
	if len(lines) == 0 {
		t.Fatal("the line was not asked for as its gathering came due")
	}
	if starts != 10_000 {
		t.Errorf("the line says it starts at %dms, want 10000", starts)
	}

	// Which is what the arrival uses: half the gathering is already spent, so
	// the picture is half made when it appears.
	wait := time.Duration(starts-m.lyricsClock()) * time.Millisecond
	spent := wordsGather - wait
	t.Logf("the line arrives %v before it is sung, %v of the gathering already spent", wait, spent)
	if spent < 0 || spent > wordsGather {
		t.Errorf("%v of the gathering is spent on arrival, want between 0 and %v", spent, wordsGather)
	}
}
