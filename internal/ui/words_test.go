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
// comes together in one of eight ways, and which one is its own business rather
// than a coin toss, so a song plays the same way twice.
func TestWordsGatherEveryWay(t *testing.T) {
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
	t.Logf("all %d moves look different halfway through, and land in the same place", wordsMoves)
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

// The whole line is lit, and every word answers its own share of the sound: the
// first beats with the bass and the last with the cymbals. Nothing claims to
// know where the singer is inside the line, because a lyric sheet cannot say —
// and a guess that is right most of the time is worse than none, since the times
// it is wrong are the times you are looking straight at it.
func TestWordsBeatWithTheirOwnPartOfTheSound(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	lines := wordsWrap("better off alone tonight", w*dotsPerCellX, rows*dotsPerCellY)
	img, layout, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the face could not draw the test line")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
	m.words.since = time.Now().Add(-time.Second)

	freqs, levels := len(m.styles.Words), len(m.styles.Words[0])
	count := layout.Count
	if count < 3 {
		t.Fatalf("the line came out in %d words", count)
	}

	// A hit in the bass alone: the first word answers it and the last does not.
	bass := make([]float32, 28)
	for i := range bass {
		bass[i] = 0.05
	}
	for i := range 4 {
		bass[i] = 1
	}
	m.scope.bands = bass

	first := m.wordsBeatPaint(0, count, freqs, levels)
	last := m.wordsBeatPaint(count-1, count, freqs, levels)
	t.Logf("on a bass hit the first word burns at %d and the last at %d", first.level, last.level)
	if first.level <= last.level {
		t.Errorf("the bass lit the first word at %d and the last at %d, want the first brighter", first.level, last.level)
	}

	// And a cymbal the other way about.
	high := make([]float32, 28)
	for i := range high {
		high[i] = 0.05
	}
	for i := 24; i < 28; i++ {
		high[i] = 1
	}
	m.scope.bands = high

	first, last = m.wordsBeatPaint(0, count, freqs, levels), m.wordsBeatPaint(count-1, count, freqs, levels)
	t.Logf("on a cymbal the first word burns at %d and the last at %d", first.level, last.level)
	if last.level <= first.level {
		t.Errorf("the cymbal lit the last word at %d and the first at %d, want the last brighter", last.level, first.level)
	}

	// Every word is lit, whatever is playing: the line is read as a line.
	for i := range count {
		if p := m.wordsBeatPaint(i, count, freqs, levels); p.level <= 0 {
			t.Errorf("word %d went out at %d", i, p.level)
		}
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
	m.setProgress(10*time.Second - wordsGather/2)
	lines, starts := m.wordsComing()
	if len(lines) == 0 {
		t.Fatal("the line was not asked for as its gathering came due")
	}
	if starts != 10_000 {
		t.Errorf("the line says it starts at %dms, want 10000", starts)
	}

	// Which is what the arrival uses: half the gathering is already spent, so
	// the picture is half made when it appears.
	wait := time.Duration(starts-m.wordsClock()) * time.Millisecond
	spent := wordsGather - wait
	t.Logf("the line arrives %v before it is sung, %v of the gathering already spent", wait, spent)
	if spent < 0 || spent > wordsGather {
		t.Errorf("%v of the gathering is spent on arrival, want between 0 and %v", spent, wordsGather)
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

	lines := []string{wordsNotes}
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

	// The meter hangs from the very first row: the two that were held back for
	// the track's name went with it.
	if strings.TrimSpace(ansiOff(drawn[0])) == "" {
		t.Error("the top row of the screen is empty, want the meter starting there")
	}
	if strings.TrimSpace(ansiOff(drawn[rows-1])) == "" {
		t.Error("the bottom row of the screen is empty, want the meter standing on it")
	}
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

// A lyric sheet marks the bars it has no words for with a note. The note is set
// like any other line — with the meter above and below it and the water crossing
// it — but it keeps no colour: nobody is singing it, so it follows the music.
func TestWordsSetTheNoteThroughASolo(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 60, 16
	m.ps.TrackID = "now"
	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "♪"},
		{At: 30_000, Words: "and then the words"},
	}

	m.setProgress(2 * time.Second)
	lines, _ := m.wordsComing()
	if len(lines) != 1 || lines[0] != wordsNotes {
		t.Errorf("an instrumental bar set %q, want %q", lines, wordsNotes)
	}
	if m.wordsSilent() {
		t.Error("an instrumental bar was taken for silence")
	}

	// It is drawn as a mark rather than as a word: no colour is kept for it.
	m.wordsGrind()
	if !m.words.beats {
		t.Error("the note was taken for words")
	}

	// And an ordinary line is not.
	m.setProgress(31 * time.Second)
	m.wordsGrind()
	if m.words.beats {
		t.Error("a line of words was taken for a mark")
	}
}

// A mark standing in for a line is set smaller than a line would be, so the
// meter has its band above and below it: one character given the whole height
// is a note two hundred dots tall with nothing else on the screen.
func TestTheNoteLeavesRoomForTheMeter(t *testing.T) {
	const w, rows = 60, 30

	tall := func(line string) (top, bottom int) {
		_, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
		if !ok {
			t.Fatalf("%q could not be drawn", line)
		}
		return layout.Tops[0], layout.Bottoms[0]
	}

	noteTop, noteBottom := tall("♪")
	wordTop, wordBottom := tall("A")
	t.Logf("the note covers rows %d..%d, a letter %d..%d", noteTop, noteBottom, wordTop, wordBottom)

	if noteBottom-noteTop >= wordBottom-wordTop {
		t.Errorf("the note is %d dots tall and a letter %d, want the note held back",
			noteBottom-noteTop, wordBottom-wordTop)
	}

	// And what it leaves is enough for the meter to be drawn in.
	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	_, layout, _ := wordsImage([]string{"♪"}, w*dotsPerCellX, rows*dotsPerCellY)
	m.words.where = layout

	if _, room := m.wordsRoom(rows); room < wordsBand {
		t.Errorf("the note leaves %d rows under it, want at least %d", room, wordsBand)
	}
	if head := m.wordsHeadroom(rows); head < wordsBand*dotsPerCellY {
		t.Errorf("the note leaves %d dots over it, want at least %d", head, wordsBand*dotsPerCellY)
	}
}

// A line that is simply replaced looks like a slide changing. The one before
// goes out the way it came in — the same arithmetic run backwards, fading as it
// gets there — so the two lines are one movement rather than two pictures.
func TestTheLineBeforeLeaves(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	set := func(line string) cover.Grain {
		img, _, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
		if !ok {
			t.Fatalf("%q could not be drawn", line)
		}
		return cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	}

	m.words.have = set("second")
	m.words.cellsX, m.words.cellsY = w, rows
	m.words.since = time.Now().Add(-time.Second)
	_, layout, _ := wordsImage([]string{"second"}, w*dotsPerCellX, rows*dotsPerCellY)
	m.words.where = layout

	alone := strings.Join(m.wordsLines(w, rows), "")

	// With one on its way out there is more on the screen than the new line
	// alone, and once it has gone there is not.
	m.words.was, m.words.went, m.words.leave = set("first"), time.Now(), wordsRising
	both := strings.Join(m.wordsLines(w, rows), "")

	m.words.went = time.Now().Add(-wordsLeaving * 2)
	after := strings.Join(m.wordsLines(w, rows), "")

	ink := func(s string) int { return len(ansiOff(s)) - strings.Count(ansiOff(s), " ") }
	t.Logf("the new line alone is %d cells, with the old one leaving %d, once it is gone %d",
		ink(alone), ink(both), ink(after))

	if ink(both) <= ink(alone) {
		t.Error("the line before it left nothing on the screen on its way out")
	}
	if ink(after) != ink(alone) {
		t.Error("the line before it never finished leaving")
	}
}

// The notes ride the part of the sound they answer for — the first jumps on the
// kick and the last on the cymbals — and so may a line of words, but less far
// and only when its own line drew it. Never a dot at a time either way: the
// letters of a word move together or the word comes apart.
func TestTheNotesRideTheBeat(t *testing.T) {
	const w, rows = 70, 16

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true

	img, layout, ok := wordsImage([]string{wordsNotes}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the notes could not be drawn")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.cellsX, m.words.cellsY, m.words.where = w, rows, layout
	m.words.since = time.Now().Add(-time.Second)
	m.words.beats = true

	rowsOf := func() map[int]bool {
		out := map[int]bool{}
		for r, line := range m.wordsLines(w, rows) {
			if strings.TrimSpace(ansiOff(line)) != "" {
				out[r] = true
			}
		}
		return out
	}

	quiet := make([]float32, 28)
	m.scope.bands = quiet
	still := rowsOf()

	loud := make([]float32, 28)
	for i := range loud {
		loud[i] = 1
	}
	m.scope.bands = loud
	jumped := rowsOf()

	t.Logf("still on rows %v, on a beat %v", keys(still), keys(jumped))
	if len(still) == 0 || len(jumped) == 0 {
		t.Fatal("the notes drew nothing")
	}
	if same(still, jumped) {
		t.Error("the notes did not move when the music did")
	}

	// A line of words keeps time too, but only when its own line drew it, and
	// never as far: some stand perfectly still.
	m.words.beats = false
	kinds := map[wordsRide]int{}
	for _, line := range []string{
		"better off alone", "do you think", "never gonna give you up", "so close no matter how far",
		"hello darkness my old friend", "is this the real life", "sultans of swing", "arra gondolok",
		"we are the champions", "another one bites the dust", "under pressure", "life on mars",
	} {
		kinds[wordsRideFor(line)]++
	}
	t.Logf("of twelve lines: %d stand still, %d nod together, %d ride word by word",
		kinds[wordsStill], kinds[wordsRideLine], kinds[wordsRideWords])

	for kind := wordsStill; kind < wordsRides; kind++ {
		if kinds[kind] == 0 {
			t.Errorf("no line of twelve got ride %d", kind)
		}
	}

	// And a still one really is still.
	for _, line := range []string{"better off alone", "do you think", "never gonna give you up"} {
		if wordsRideFor(line) != wordsStill {
			continue
		}
		m.words.text = line
		m.scope.bands = quiet
		a := rowsOf()
		m.scope.bands = loud
		if b := rowsOf(); !same(a, b) {
			t.Errorf("%q was drawn still and moved anyway", line)
		}
		break
	}
}

func keys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func same(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// Some lines lean. Not many, and not far: a couple of degrees, so the line looks
// hand-set rather than typeset, and nobody has to tip their head.
func TestSomeLinesLean(t *testing.T) {
	lines := []string{
		"better off alone", "do you think", "never gonna give you up", "so close no matter how far",
		"hello darkness my old friend", "is this the real life", "sultans of swing", "arra gondolok",
		"we are the champions", "another one bites the dust", "under pressure", "life on mars",
	}

	var leaning int
	for _, line := range lines {
		if wordsLeans(line) {
			leaning++
		}
	}
	t.Logf("%d of %d lines lean", leaning, len(lines))
	if leaning == 0 || leaning == len(lines) {
		t.Errorf("%d of %d lines lean, want some of them", leaning, len(lines))
	}

	// A line that leans leans by a word, some one way and some the other, and
	// each about its own middle rather than about the screen's.
	const w, rows = 74, 20
	m := scopeModel(100, 44)
	m.width, m.height = w, rows

	var leaned bool
	for _, line := range lines {
		if !wordsLeans(line) {
			continue
		}

		_, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
		if !ok {
			continue
		}
		m.words.where, m.words.text = layout, line

		tilt, middle := m.wordsTilting(layout.Count)
		if len(tilt) != layout.Count {
			t.Fatalf("%q leans but gave %d tilts for %d words", line, len(tilt), layout.Count)
		}

		var last int
		for i := range tilt {
			if abs32(tilt[i]) > wordsTilt {
				t.Errorf("a word leans by %.3f, want no more than %.3f", tilt[i], wordsTilt)
			}
			if middle[i] < last {
				t.Error("the words lean about middles that are not in reading order")
			}
			last = middle[i]
		}
		leaned = true
		t.Logf("%q leans its words by %v", line, tilt)
		break
	}
	if !leaned {
		t.Error("no line of the twelve could be drawn leaning")
	}

	// And one that does not lean is given nothing to lean by.
	for _, line := range lines {
		if wordsLeans(line) {
			continue
		}
		m.words.text = line
		if tilt, _ := m.wordsTilting(4); tilt != nil {
			t.Errorf("%q does not lean but was given %v", line, tilt)
		}
		break
	}
}

// The notes are always the same three marks, so asked of their text they would
// lean in every solo of every record or in none of them. Theirs comes from when
// the bar is, the way their moves do.
func TestTheNotesLeanBySolo(t *testing.T) {
	var leaning int
	for at := range int64(30) {
		m := Model{}
		m.words.text, m.words.beats, m.words.starts = wordsNotes, true, at*7_000
		if m.wordsLeanSeed()%3 == 0 {
			leaning++
		}
	}
	t.Logf("%d of thirty solos lean", leaning)

	if leaning == 0 || leaning == 30 {
		t.Errorf("%d of thirty solos lean, want some of them", leaning)
	}

	// And the same bar answers the same way twice.
	one := Model{}
	one.words.text, one.words.beats, one.words.starts = wordsNotes, true, 12_345
	two := one
	if one.wordsLeanSeed() != two.wordsLeanSeed() {
		t.Error("one bar answered differently twice")
	}
}
