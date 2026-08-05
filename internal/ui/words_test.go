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

	early := m.words.sung
	first := m.words.paint[early]
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
	t.Logf("word %d kept hue %d, and the word being sung is now %d", early, first.hue, m.words.sung)

	if m.words.sung <= early {
		t.Fatalf("the singer never got past word %d", early)
	}
	if got := m.words.paint[m.words.sung]; !got.set || got.hue == first.hue {
		t.Errorf("a word sung over a cymbal took hue %d, the same as one sung over a bass note", got.hue)
	}
	// The earlier word kept what it was sung in rather than following the music.
	if m.words.paint[early].hue != first.hue {
		t.Errorf("the earlier word repainted itself from %d to %d", first.hue, m.words.paint[early].hue)
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

	// A song with no lyrics at all shows its name as it starts, and then gives
	// the screen over as well: a title is worth reading once.
	m.lyrics.synced = false
	m.setProgress(2 * time.Second)
	if m.wordsSilent() {
		t.Error("a song with no lyrics gave up its title as it started")
	}
	m.setProgress(wordsTitle + time.Second)
	if !m.wordsSilent() {
		t.Error("a song with no lyrics held its title up long after it started")
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

// A lyric sheet says when a line starts and nothing else, so how far into it the
// singer has got is a guess. Spreading the words evenly over the gap to the next
// line is a bad guess — measured over three hundred lines, a line fills about
// half of its gap — so the length is guessed from the line itself instead, and
// the guess is made to fail early rather than late.
func TestWordsGuessTheLineLengthFromTheLine(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 90, 20
	m.ps.TrackID = "now"

	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	// A short line with a long gap after it: the singer is done well before the
	// next line arrives, and the old arithmetic would have been halfway through
	// the line when they had finished it.
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "one two three four"},
		{At: 30_000, Words: "long after"},
	}
	m.words.where.Count = 4

	// Two seconds in, a line of eighteen characters is over at any singing rate.
	m.setProgress(2 * time.Second)
	if got := m.wordsSung(); got < 3 {
		t.Errorf("two seconds into a four word line the picture is on word %d, want it at the end", got)
	}

	// And it holds there rather than running past it.
	m.setProgress(20 * time.Second)
	if got := m.wordsSung(); got != 3 {
		t.Errorf("twenty seconds in the picture is on word %d, want it held on the last", got)
	}

	// A line that genuinely fills its gap is still followed through it.
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "one two three four"},
		{At: 1_500, Words: "hard after"},
	}
	m.setProgress(200 * time.Millisecond)
	early := m.wordsSung()
	m.setProgress(900 * time.Millisecond)
	if late := m.wordsSung(); late <= early {
		t.Errorf("through a tight line the picture went from word %d to %d", early, late)
	}
}

// The note a lyric sheet puts where a line would be is not a line: nobody sings
// it, so nothing paints it once and keeps it. It beats with the music instead.
func TestWordsBeatWhenThereIsNothingToSing(t *testing.T) {
	for _, line := range []string{"♪", "♪ ♪ ♪", "…"} {
		if !wordsBeats(line) {
			t.Errorf("%q was taken for words", line)
		}
	}
	for _, line := range []string{"better off alone", "1979", "♪ and then"} {
		if wordsBeats(line) {
			t.Errorf("%q was taken for a mark", line)
		}
	}

	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	lines := []string{"x"}
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not be drawn")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
	m.words.since = time.Now().Add(-time.Second)
	m.words.beats = true

	paint := func() string { return strings.Join(m.wordsLines(w, rows), "") }

	bass := make([]float32, 28)
	for i := range bass {
		bass[i] = 0.1
	}
	bass[1] = 1
	m.scope.bands = bass
	for range 30 {
		m.wordsFlow(w, rows)
	}
	was := paint()

	high := make([]float32, 28)
	for i := range high {
		high[i] = 0.1
	}
	high[26] = 1
	m.scope.bands = high
	for range 30 {
		m.wordsFlow(w, rows)
	}

	if paint() == was {
		t.Error("the mark kept its colour while the music moved under it")
	}
}

// The water crosses the lyric, and has to be seen through it rather than
// against it: a drop is spent by the time it is up among the letters, or it
// reads as part of one.
func TestWordsWaterFadesAsItClimbs(t *testing.T) {
	const w, rows = 90, 30

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	// Silent, so the only ink in the picture is the water itself.
	m.scope.bands = make([]float32, 28)

	lines := wordsWrap("better off alone", w*dotsPerCellX, rows*dotsPerCellY)
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
	m.words.since = time.Now().Add(-time.Second)

	_, tall := m.wordsRoom(rows)
	band := float32(tall * dotsPerCellY)

	// One drop in the band, one well up in the lyric, both thrown as brightly.
	m.stage.drops = []stageDrop{
		{col: 10, at: band / 2, bright: 1},
		{col: 30, at: band + (float32(rows*dotsPerCellY)-band)/2, bright: 1},
	}

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}
	m.wordsUnder(grid, paint, hue, w, rows, tall)

	level := func(col int) int8 {
		best := int8(-1)
		for r := range rows {
			if grid[r*w+col] != 0 {
				best = max(best, paint[r*w+col])
			}
		}
		return best
	}

	low, high := level(10/dotsPerCellX), level(30/dotsPerCellX)
	t.Logf("a drop in the band draws at %d, one up among the words at %d", low, high)

	if low < 0 || high < 0 {
		t.Fatalf("a drop went missing: band %d, sky %d", low, high)
	}
	if high >= low {
		t.Errorf("a drop up in the lyric draws at %d against %d in the band, want it spent", high, low)
	}
}

// The meter hangs from the ceiling as well as standing on the floor, so the
// lyric sits between two of it and the screen reads as one picture rather than
// as a strip with a heading over it.
func TestWordsHangTheMeterFromTheCeilingToo(t *testing.T) {
	const w, rows = 70, 34

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.7
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
	if tall < wordsBand {
		t.Fatalf("only %d rows were left for the meter", tall)
	}

	drawn := m.wordsLines(w, rows)
	ink := func(lines []string) int {
		var n int
		for _, line := range lines {
			for _, r := range ansiOff(line) {
				if r != ' ' {
					n++
				}
			}
		}
		return n
	}

	head := ink(drawn[wordsCeiling : wordsCeiling+tall])
	foot := ink(drawn[from:])
	t.Logf("the meter draws %d cells hanging and %d standing", head, foot)

	if head == 0 {
		t.Error("nothing hangs from the ceiling")
	}
	if foot == 0 {
		t.Error("nothing stands on the floor")
	}
	// The same reading drawn twice: the two have to be within a hair of each
	// other, allowing for the room over the words being the shorter of the two.
	if head > foot {
		t.Errorf("%d cells hang and %d stand, want the hanging one no larger", head, foot)
	}
	if head*3 < foot {
		t.Errorf("%d cells hang against %d standing, want them a pair", head, foot)
	}
}

// A lyric sheet marks the bars it has no words for with a note, and a note two
// hundred dots tall is a dull thing to look at for the length of a solo. Those
// bars are drawn instead — a face or the chase — so nothing is set in type for
// them at all.
func TestWordsDrawRatherThanSetTheSolos(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 60, 16
	m.ps.TrackID = "now"
	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "♪"},
		{At: 60_000, Words: "and then the words"},
	}
	m.setProgress(2 * time.Second)

	if lines, _ := m.wordsComing(); len(lines) != 0 {
		t.Errorf("an instrumental bar asked for %q to be set, want it drawn", lines)
	}

	// And the screen is not blank for it: the bar is one of the drawn ones.
	m.chaseNow()
	if !m.chase.on && !m.words.drawn {
		t.Error("an instrumental bar drew neither a face nor the chase")
	}
	if m.wordsSilent() {
		t.Error("a bar with a face on it was taken for silence")
	}

	// Forty of them are not all the same face.
	seen := map[faceKind]bool{}
	for at := range int64(40) {
		seen[faceFor(at*7_000)] = true
	}
	t.Logf("forty instrumental bars pulled %d of the %d faces", len(seen), faceKinds)
	if len(seen) < int(faceKinds)-1 {
		t.Errorf("forty bars only pulled %d faces of %d", len(seen), faceKinds)
	}

	// And the same bar always pulls the same one.
	if a, b := faceFor(12_345), faceFor(12_345); a != b {
		t.Errorf("one bar pulled face %d and then %d", a, b)
	}
}

// The cheering face goes up through a solo with its arms in the air, and it
// would be a waste to let it simply vanish when the singer comes back: the hands
// throw on the beat, and let go of everything a moment before the words return.
func TestCheerThrowsFromItsHands(t *testing.T) {
	const w, rows = 80, 20

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true
	m.ps.TrackID = "now"

	m.words.drawn = true
	m.face = faceState{kind: faceCheer}

	// A beat: a spark or two from each hand.
	m.scope.envelope, m.words.wasLoud = 0.9, 0.2
	m.wordsCheerFlow(w, rows)
	if len(m.stage.drops) == 0 {
		t.Error("the hands threw nothing on the beat")
	}
	onBeat := len(m.stage.drops)

	// And everything they had, just before the words come back.
	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{{At: 0, Words: "♪"}, {At: 10_000, Words: "words"}}
	m.words.ends = 10_000
	m.setProgress(10*time.Second - wordsCheerBurst/2 - lyricsAhead)

	m.wordsCheerFlow(w, rows)
	t.Logf("%d sparks on a beat, %d once it let go", onBeat, len(m.stage.drops))

	if len(m.stage.drops) < onBeat+wordsCheerMost/2 {
		t.Errorf("the burst threw %d sparks, want at least %d", len(m.stage.drops)-onBeat, wordsCheerMost/2)
	}

	// Only once, though.
	before := len(m.stage.drops)
	m.wordsCheerFlow(w, rows)
	if len(m.stage.drops) > before+4 {
		t.Error("it let go twice")
	}
}
