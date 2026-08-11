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

	// A record the database has answered about and has nothing for takes turns:
	// a card of its own for a few seconds, and the marks the rest of the time.
	// Either way there is something. Until the answer lands nothing is put up
	// on spec — see wordsWordless.
	m.lyrics.forTrack, m.lyrics.missing = "now", true
	m.setProgress(wordsSpell)
	if got := m.wordsNow(); len(got) == 0 || got[0] == "" {
		t.Errorf("with no lyrics the picture shows %q, want something", got)
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

	// The arrivals only. Popping is a way of leaving — a line that arrived by
	// un-bursting would be a film run backwards — and it is drawn by wordsPop
	// rather than by the gathering. See TestTheMarksPopInTurn.
	half := map[string]wordsMove{}
	var settled string
	for move := wordsDrifting; move < wordsPopping; move++ {
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
	if len(half) != int(wordsPopping) {
		t.Errorf("%d of the %d arrivals look alike halfway through", int(wordsPopping)-len(half), wordsPopping)
	}
	t.Logf("all %d arrivals look different halfway through, and land in the same place", wordsPopping)
}

// Which move a line gets is worked out from the line, from when it is sung, and
// from the one before it — so a record plays the same way twice, and no two
// arrivals in a row are the same.
//
// It used to come from the line alone, so a chorus came back exactly as it went.
// That is worth wanting — measured over thirty real sheets, half of what this
// screen shows is a line that has been sung before, 44% at the median and 85% at
// the top of the range — and it is not what happens: three arrivals dealt freely
// out of eight ways collide better than a third of the time, and a line coming
// in exactly as the last one did reads as a mistake rather than as a refrain.
// Measured on a chorus that comes round three times, two of the three drew the
// same way. So the one promise this deal makes is the one it can keep: never
// twice in a row.
func TestWordsMoveComesFromTheLine(t *testing.T) {
	if a, b := wordsMoveFor("better off alone", 1000, 0), wordsMoveFor("better off alone", 1000, 0); a != b {
		t.Errorf("the same line got moves %d and %d", a, b)
	}

	// And the rest of what a line is dealt comes back with it: which of its
	// words lean.
	if a, b := wordsLeans("better off alone"), wordsLeans("better off alone"); a != b {
		t.Errorf("the same line leaned %v and came back %v", a, b)
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
		seen[wordsMoveFor(line, 0, 0)] = true
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

// A song between two lines is a song playing. A short gap goes to the music; a
// long one goes to the marks, which keep time with it. Either way the last line
// sung is not left standing over the whole of a solo.
func TestWordsGiveTheScreenBackBetweenLines(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 90, 30
	m.ps.TrackID = "now"
	m.ps.Duration = 3 * time.Minute

	m.lyrics.synced, m.lyrics.forTrack = true, "now"
	m.lyrics.lines = []player.Lyric{
		{At: 20_000, Words: "the first line, long after the start"},
		{At: 26_000, Words: "and the second"},
		{At: 45_000, Words: "and after the solo"},
	}

	// The opening bars: twenty seconds of nothing sung, which is a bar of its
	// own and gets the marks.
	m.setProgress(time.Second)
	lines, _ := m.wordsComing()
	if len(lines) != 1 || !wordsBeats(lines[0]) {
		t.Errorf("before the first line the screen has %q, want the marks", lines)
	}

	// The line itself.
	m.setProgress(22 * time.Second)
	if m.wordsSilent() {
		t.Error("the screen went to the music while a line was being sung")
	}

	// Nineteen seconds with nothing in them: the line goes once it has had its
	// time, and the marks have the rest. Not the middle of it, which is where
	// the record says its own name — see solo.go.
	m.setProgress(34 * time.Second)
	lines, _ = m.wordsComing()
	if len(lines) != 1 || !wordsBeats(lines[0]) {
		t.Errorf("through the solo the screen has %q, want the marks", lines)
	}

	// A gap too short to be worth a change of picture keeps the line before it.
	//
	// It used to go blank instead, and that was the rule this screen was given
	// at the start: between two lines the music has the middle. Watched, it is
	// not what a gap looks like — a sheet that writes its rests down blanked the
	// screen for a second between two lines, and a sheet that leaves them out
	// held the line up through a gap ten times as long, on the same record. The
	// rest is the same bar written down, so it reads the same way now.
	m.lyrics.lines = []player.Lyric{
		{At: 20_000, Words: "one"},
		{At: 30_000, Words: ""},
		{At: 31_000, Words: "two"},
	}
	m.setProgress(30 * time.Second)
	lines, starts := m.wordsComing()
	if len(lines) == 0 {
		t.Error("a one second rest blanked the screen between two lines")
	} else if lines[0] != "one" || starts != 20_000 {
		t.Errorf("over the rest the screen has %q from %dms, want the line before it", lines, starts)
	}

	// A song with no lyrics at all is one long solo: the marks throughout, and
	// nothing written across them at any point. See idle.go.
	m.lyrics.synced = false
	m.ps.Artists, m.ps.Album = []string{"The Band"}, "An Album"
	for spell := range 12 {
		m.setProgress(time.Duration(spell)*wordsSpell + time.Second)
		if lines, _ := m.wordsComing(); len(lines) != 1 || !wordsBeats(lines[0]) {
			t.Errorf("spell %d of a wordless record has %q, want the marks", spell, lines)
		}
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

	// Well before the line: the intro, which the marks keep. What matters here
	// is that the line itself is not asked for yet — a picture built five
	// seconds early is five seconds of a line standing still before it is sung.
	m.setProgress(5 * time.Second)
	if lines, _ := m.wordsComing(); len(lines) != 1 || !wordsBeats(lines[0]) {
		t.Errorf("%v before the line the screen has %q, want the marks", 5*time.Second, lines)
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
	m.wordsUnder(grid, paint, hue, w, rows, tall, m.wordsHeadroom(rows))

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
	if len(lines) != 1 || !wordsBeats(lines[0]) {
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

	// A line of words moves too, and never as far: every word of it on its own
	// part of the sound. It used to be dealt — some lines nodding as one block
	// on the loudest thing playing — and the block is gone: watched against each
	// other, words that each live on their own part of the sound beat a line
	// moving as a slab.
	m.words.beats = false

	// Every one of them moves when the music does.
	for _, line := range []string{"better off alone", "do you think", "never gonna give you up"} {
		m.words.text = line
		m.scope.bands = quiet
		a := rowsOf()
		m.scope.bands = loud
		if b := rowsOf(); same(a, b) {
			t.Errorf("%q did not move when the music did", line)
		}
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
			if abs32(tilt[i]) > wordsTiltMost {
				t.Errorf("a word leans by %.3f, want no more than %.3f", tilt[i], wordsTiltMost)
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
// lean in every solo of every record or in none of them. Theirs comes from the
// bar they arrived under — not the bar playing now, because they stay up across
// a change of bar and a row that snaps to new angles without going anywhere is
// a row dealt again in place.
func TestTheNotesLeanBySolo(t *testing.T) {
	const w, rows = 74, 20
	_, layout, ok := wordsImage([]string{wordsNotes}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the notes would not draw")
	}

	m := scopeModel(100, 44)
	m.words.where, m.words.text, m.words.beats = layout, wordsNotes, true

	var leaning int
	for at := range int64(30) {
		m.words.starts, m.words.leanAt = at*7_000, at*7_000
		if tilt, _ := m.wordsTilting(layout.Count); tilt != nil {
			leaning++
		}
	}
	t.Logf("%d of thirty solos lean", leaning)

	if leaning == 0 || leaning == 30 {
		t.Errorf("%d of thirty solos lean, want some of them", leaning)
	}

	// A mark is slanted sideways, not tipped: what it turns about is a row
	// across the middle of the type rather than a column down the middle of
	// itself, which is what a note with a long straight stem needs to lean at
	// all. Anything else moves the stem without ever tipping it.
	for at := range int64(30) {
		m.words.starts, m.words.leanAt = at*7_000, at*7_000
		tilt, middle := m.wordsTilting(layout.Count)
		if tilt == nil {
			continue
		}

		if got := abs32(tilt[0]); got != 0 && got != wordsTiltMark {
			t.Errorf("a note slants by %.3f, want the marks' own slant", tilt[0])
		}
		if middle[0] < layout.Tops[0] || middle[0] > layout.Bottoms[0] {
			t.Errorf("a note leans about row %d, which is not inside the type at %d..%d",
				middle[0], layout.Tops[0], layout.Bottoms[0])
		}
		break
	}

	// And the same bar answers the same way twice.
	one := Model{}
	one.words.text, one.words.beats, one.words.starts = wordsNotes, true, 12_345
	two := one
	if one.wordsLeanSeed() != two.wordsLeanSeed() {
		t.Error("one bar answered differently twice")
	}
}

// A comma is not part of the word in front of it. It is cut off and counted as
// something of its own, so it answers its own part of the sound and rides it —
// while what is between the ends of a word, an apostrophe or a dash, stays put.
func TestPunctuationIsItsOwnWord(t *testing.T) {
	for _, c := range []struct {
		line string
		want []string
	}{
		{"Hello, is it me you're looking for?",
			[]string{"Hello", ",", "is", "it", "me", "you're", "looking", "for", "?"}},
		{"(this is a whisper) ... and then it stops.",
			[]string{"(", "this", "is", "a", "whisper", ")", "...", "and", "then", "it", "stops", "."}},
		{"don't stop me now — I'm having a good time;",
			[]string{"don't", "stop", "me", "now", "—", "I'm", "having", "a", "good", "time", ";"}},
		{`"stop", she said — 'cause we were singin'`,
			[]string{`"`, "stop", `",`, "she", "said", "—", "'", "cause", "we", "were", "singin", "'"}},
		{"no marks here at all", []string{"no", "marks", "here", "at", "all"}},
		{wordsNotes, []string{"♪", "♫", "♪"}},
	} {
		var got []string
		for _, p := range wordsPieces(c.line) {
			got = append(got, c.line[p.from:p.to])
		}
		if len(got) != len(c.want) {
			t.Errorf("%q cut into %q, want %q", c.line, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q cut into %q, want %q", c.line, got, c.want)
				break
			}
		}
	}
}

// And the mark's own dots are the ones it is given: a piece that claimed the
// wrong columns would ride the letters beside it about instead of itself.
func TestAMarkOwnsItsOwnDots(t *testing.T) {
	const w, rows = 90, 12
	const line = "Hello, world!"

	img, layout, ok := wordsImage([]string{line}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the line would not draw")
	}
	if layout.Count != 4 {
		t.Fatalf("%q came out as %d pieces, want the two words and the two marks", line, layout.Count)
	}

	// Every lit dot belongs to something, and the pieces are in reading order:
	// the comma to the right of "Hello" and the mark to the right of "world".
	first := make([]int, layout.Count)
	last := make([]int, layout.Count)
	for i := range first {
		first[i], last[i] = 1<<30, -1
	}
	for y := range rows * dotsPerCellY {
		for x := range w * dotsPerCellX {
			if img.GrayAt(x, y).Y < wordsLit {
				continue
			}
			at := layout.WordAt(x, y)
			if at < 0 {
				t.Fatalf("a lit dot at %d,%d belongs to no piece", x, y)
			}
			first[at], last[at] = min(first[at], x), max(last[at], x)
		}
	}

	for i := 1; i < layout.Count; i++ {
		if first[i] <= last[i-1] {
			t.Errorf("piece %d starts at %d, before piece %d ends at %d", i, first[i], i-1, last[i-1])
		}
	}
	t.Logf("the comma has %d columns of its own, the exclamation mark %d",
		last[1]-first[1]+1, last[3]-first[3]+1)
}

// A mark keeps its own time, but it is not carried over the word it hangs off.
// The end of a line answers the top of the range, where a hi-hat is going all
// the time, so without this the full stop at the end of a lyric floated halfway
// up the screen and stayed there.
func TestAMarkIsNeverCarriedAboveItsWord(t *testing.T) {
	m := scopeModel(100, 44)
	m.words.text = "and then it stops."

	// Loud at the top of the range, quiet at the bottom: the arrangement that
	// lifted the mark and left the words where they were.
	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = float32(i) / float32(len(bands)-1)
	}
	m.scope.bands = bands

	marks := m.wordsMarksNow()
	want := []bool{false, false, false, false, true}
	if len(marks) != len(want) {
		t.Fatalf("%q reads as %v, want five pieces", m.words.text, marks)
	}
	for i := range want {
		if marks[i] != want[i] {
			t.Fatalf("%q reads as %v, want %v", m.words.text, marks, want)
		}
	}

	ride := make([]int, len(marks))
	for i := range ride {
		ride[i] = -int(m.wordsBeatRide(i, len(marks)) * wordsWordRide)
	}
	t.Logf("left to itself the line rides %v", ride)

	settled := m.wordsSettleMarks(append([]int(nil), ride...))
	t.Logf("settled it rides %v", settled)

	if settled[4] < settled[3] {
		t.Errorf("the full stop rides %d against the word's %d, want it no higher",
			settled[4], settled[3])
	}
	for i := range 4 {
		if settled[i] != ride[i] {
			t.Errorf("word %d was moved from %d to %d, want the words left alone",
				i, ride[i], settled[i])
		}
	}
}

// A mark that opens something takes its place from the word after it, there
// being nothing in front of it to hang off.
func TestAnOpeningMarkFollowsTheWordAfterIt(t *testing.T) {
	marks := []bool{true, false, false, true}
	if by, ok := wordsBesideMark(marks, 0); !ok || by != 1 {
		t.Errorf("the opening mark follows piece %d (%v), want the word after it", by, ok)
	}
	if by, ok := wordsBesideMark(marks, 3); !ok || by != 2 {
		t.Errorf("the closing mark follows piece %d (%v), want the word before it", by, ok)
	}
	if _, ok := wordsBesideMark([]bool{true, true}, 0); ok {
		t.Error("a bar of nothing but marks found a word to follow")
	}
}

// A bar of music is a row of marks across the screen, not three of them sitting
// in the middle of it — as many as go beside each other at the size one is set
// at, each with its own colour off the spectrum behind it.
func TestABarOfMusicFillsTheRow(t *testing.T) {
	for _, size := range [][2]int{{60, 20}, {100, 30}, {160, 46}, {240, 60}} {
		w, rows := size[0], size[1]
		dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

		line := wordsMarks(dotsX, dotsY)
		marks := strings.Count(line, "♪") + strings.Count(line, "♫")

		_, layout, ok := wordsImage([]string{line}, dotsX, dotsY)
		if !ok {
			t.Fatalf("%dx%d: the row would not draw", w, rows)
		}
		t.Logf("%3dx%-3d %d marks, %d pieces: %s", w, rows, marks, layout.Count, line)

		if marks < 5 {
			t.Errorf("%dx%d: %d marks, want the row filled", w, rows, marks)
		}
		if marks > wordsMarksMost {
			t.Errorf("%dx%d: %d marks, past the %d that is a row rather than wallpaper", w, rows, marks, wordsMarksMost)
		}
		if layout.Count != marks {
			t.Errorf("%d marks came out as %d pieces, want each its own", marks, layout.Count)
		}

		// They are set no smaller for being many: the band decides the height,
		// and the row stops where the width would start shrinking them.
		one := wordsSize([]string{"♪"}, dotsX, int(wordsMark*float64(dotsY)))
		many := wordsSize([]string{line}, dotsX, int(wordsMark*float64(dotsY)))
		if many < one {
			t.Errorf("%dx%d: one mark is set at %d and the row at %d, want them the same", w, rows, one, many)
		}
	}

	// And each of them answers its own part of the sound, so the row shimmers
	// across rather than lighting all at once.
	m := scopeModel(160, 46)
	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = float32(i%7) / 6
	}
	m.scope.bands = bands

	levels, freqs := len(m.styles.Words[0]), len(m.styles.Words)
	seen := map[int8]bool{}
	var hues int8 = -1
	for i := range 7 {
		p := m.wordsBeatPaint(i, 7, freqs, levels)
		seen[p.level] = true
		if p.hue != hues {
			hues, _ = p.hue, 0
		}
	}
	t.Logf("across seven marks the sound gives %d different brightnesses", len(seen))
	if len(seen) < 3 {
		t.Errorf("the row lights at %d different levels, want it shimmering across", len(seen))
	}
}

// A row of marks does not slide or wipe away. It goes the way a row of anything
// goes when somebody walks down it: the first one bursts, then the next, then
// the next.
func TestTheMarksPopInTurn(t *testing.T) {
	const w, rows = 100, 30
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	m := scopeModel(160, 46)
	m.width, m.height = w, rows

	line := wordsMarks(dotsX, dotsY)
	img, layout, ok := wordsImage([]string{line}, dotsX, dotsY)
	if !ok {
		t.Fatal("the row would not draw")
	}
	m.words.was = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.wasWhere, m.words.leave = layout, wordsPopping

	// Which marks are still standing where they were drawn, as it goes off.
	standing := func(gone float32) []bool {
		out := make([]bool, layout.Count)
		for y := range dotsY {
			for x := range dotsX {
				if m.words.was.Lum[y*dotsX+x] < wordsLit {
					continue
				}
				piece := layout.WordAt(x, y)
				if piece < 0 {
					continue
				}
				if at, to, _, ok := m.wordsPop(x, y, gone, 6); ok && at == x && to == y {
					out[piece] = true
				}
			}
		}
		return out
	}

	var was int
	for _, gone := range []float32{0, 0.25, 0.5, 0.75, 1} {
		up := standing(gone)
		var n int
		for _, on := range up {
			if on {
				n++
			}
		}
		t.Logf("%.0f%% through, %d of %d marks are still standing: %v", gone*100, n, layout.Count, up)

		if gone == 0 && n != layout.Count {
			t.Errorf("%d of %d marks were already going as it began", layout.Count-n, layout.Count)
		}
		if gone > 0 && n > was {
			t.Errorf("%d marks are standing where %d were before, want them going one way", n, was)
		}
		// They go in order, left to right: once the row has started standing
		// again it does not stop, so there is no mark left behind in a gap.
		var back int
		for i := 1; i < len(up); i++ {
			if !up[i] && up[i-1] {
				back++
			}
		}
		if back > 1 {
			t.Errorf("at %.0f%% the row goes in and out %d times, want it going one way along", gone*100, back)
		}
		was = n
	}
	if was != 0 {
		t.Errorf("%d marks were still standing at the end", was)
	}
}

// The meter does not jump when the picture it is making room for changes.
//
// Every picture on this screen leaves the meter a different amount of room: a
// line of words, a row of marks, a card, a figure standing in the middle. Each
// of them used to be measured at the moment it was drawn, so the columns along
// the top and the foot of the screen changed height between one frame and the
// next — a change of picture read as a cut rather than as one picture giving
// way to another.
func TestTheMeterDoesNotJumpBetweenPictures(t *testing.T) {
	m := sung(10, 40, 44, 48)
	w, rows := m.width, m.height

	was, wasHead := -1, -1
	var most, mostHead int
	var when time.Duration

	for at := 5 * time.Second; at < 50*time.Second; at += 33 * time.Millisecond {
		m.setProgress(at)
		if cmd := m.wordsGrind(); cmd != nil {
			if got := cmd(); got != nil {
				tm, _ := m.Update(got)
				m = tm.(Model)
			}
		}
		m.faceFlow()
		m.wordsEase(w, rows)

		// Measured from the first picture on: before there is one there is no
		// room to ease from, and the meter arriving with it is the meter
		// arriving, not the meter jumping.
		tall, head := m.wordsBandNow(w, rows)
		if was > 0 {
			if abs(tall-was) > most {
				most, when = abs(tall-was), at
			}
			mostHead = max(mostHead, abs(head-wasHead))
		}
		was, wasHead = tall, head
	}

	t.Logf("over the record the meter's room moved at most %d rows in a frame (at %s), and %d dots over the head",
		most, when, mostHead)
	if most > 1 {
		t.Errorf("the meter's room moved %d rows in one frame, want it eased", most)
	}
	if mostHead > dotsPerCellY {
		t.Errorf("the room over the head moved %d dots in one frame, want it eased", mostHead)
	}
}

// A line of a song is lit evenly, and what each word's own part of the sound is
// doing shows in how far it rides rather than in how brightly it burns.
//
// Both were on the brightness before, and it cost the reading for nothing:
// measured over ninety seconds of a record at thirty frames a second, the
// brightest and dimmest word of a six word line stood two of the palette's six
// steps apart at the median and three in the top tenth, changing all the while.
// Nobody can read "the third word's band is loud" — what they can read is the
// line, and only if it is lit like one.
func TestALineIsLitEvenly(t *testing.T) {
	m := scopeModel(120, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.words.text = "do you think you're better off alone"

	// A spectrum that leans hard on one end, which is what used to pull a line
	// of type apart.
	m.scope.bands = make([]float32, 28)
	for i := range m.scope.bands {
		m.scope.bands[i] = 0.1 + 0.9*float32(i)/float32(len(m.scope.bands)-1)
	}

	const words, freqs, levels = 7, 10, 6
	var low, high int8 = levels, 0
	var hues []int8
	for i := range words {
		p := m.wordsPaint(i, words, freqs, levels)
		low, high = min(low, p.level), max(high, p.level)
		hues = append(hues, p.hue)
	}
	t.Logf("across a line of %d words the brightness runs %d to %d of %d, and the hues %v", words, low, high, levels, hues)

	if low != high {
		t.Errorf("the line is lit from %d to %d, want every word of it at one level", low, high)
	}

	// The line still breathes with the record: quiet music, a dimmer line.
	for i := range m.scope.bands {
		m.scope.bands[i] = 0.05
	}
	quiet := m.wordsPaint(0, words, freqs, levels)
	t.Logf("and over quiet music the same line is lit at %d against %d", quiet.level, high)
	if quiet.level >= high {
		t.Errorf("quiet music lit the line at %d and loud music at %d, want it breathing", quiet.level, high)
	}

	// The marks are not words: each of them is a part of the spectrum, and
	// burning by what that part is doing is the whole of what they say.
	m.words.beats = true
	var marksLow, marksHigh int8 = levels, 0
	for i := range m.scope.bands {
		m.scope.bands[i] = 0.1 + 0.9*float32(i)/float32(len(m.scope.bands)-1)
	}
	for i := range words {
		p := m.wordsPaint(i, words, freqs, levels)
		marksLow, marksHigh = min(marksLow, p.level), max(marksHigh, p.level)
	}
	t.Logf("a row of marks over the same spectrum runs %d to %d", marksLow, marksHigh)
	if marksLow == marksHigh {
		t.Error("the marks were lit evenly, want each of them burning by its own part of the sound")
	}
}

// Nothing of one record is left on the screen of the next.
//
// A picture is held until something replaces it, and what replaces it hands the
// old one to the leaving animation — so the last line of the record before flew
// out across the first line of this one, on every skip. The picture belongs to
// the record it was made for, and goes with it.
func TestAPictureGoesWithItsRecord(t *testing.T) {
	const w, rows = 90, 14

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.scope.modes[tabPlayer], m.stage.on = scopeWords, true
	m.ps.Duration = 3 * time.Minute
	m.lyrics.forTrack, m.lyrics.synced = m.ps.TrackID, true
	m.lyrics.lines = []player.Lyric{{At: 0, Words: "the line of the record before"}}
	m.setProgress(time.Second)

	// The record before, with a line of it on the screen.
	m.wordsGrind()
	img, layout, ok := wordsImage([]string{"the line of the record before"}, w*dotsPerCellX, rows*dotsPerCellY)
	if !ok {
		t.Fatal("the line could not be drawn")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.where, m.words.text = layout, "the line of the record before"
	m.words.cellsX, m.words.cellsY = w, rows
	if m.words.have.DotsX == 0 {
		t.Fatal("the record before has no picture, so there is nothing to leave behind")
	}

	// And now somebody skips.
	m.ps.TrackID = "the next record"
	m.lyrics.forTrack, m.lyrics.lines = "", nil
	m.wordsGrind()

	t.Logf("after the skip the screen holds %q, %d dots wide", m.words.text, m.words.have.DotsX)
	if strings.Contains(m.words.text, "record before") {
		t.Errorf("the line of the record before survived the skip: %q", m.words.text)
	}

	// What matters is not that the screen is empty — the marks go up at the top
	// of a record and that is the point — but that nothing of the record before
	// is queued up to fly out across it.
	if m.words.was.DotsX != 0 {
		t.Error("the record before is still waiting to leave across the next one")
	}
}

// A line of type has a measure, and it is not the width of the screen.
//
// The only rule was a floor — at least nine dots a letter, or break — which on a
// wide terminal is never reached, so every lyric that fitted was set on one line
// however long it was. Watched on the same words at two font sizes: forty-two
// letters across is a thin strip a third of the height it could be, and the same
// words broken in two at twenty-one fill the screen.
//
// What the measure buys is the thing the wrapping was worth having for: a
// sheet's lines are not all the same length, so some come up as one line and
// some as two, and the picture changes shape as the song goes.
func TestALineOfTypeHasAMeasure(t *testing.T) {
	const long = "Ami jól jött az a kis lé, és nem is féltem"

	// The same words, on a wide screen and a wider one: the same picture.
	wide, wider := wordsWrap(long, 300, 200), wordsWrap(long, 440, 200)
	if len(wide) != len(wider) {
		t.Errorf("the same line came out %d lines at 150 cells and %d at 220", len(wide), len(wider))
	}
	if len(wider) < 2 {
		t.Errorf("forty-two letters were set on %d line(s) across a wide screen", len(wider))
	}
	for _, l := range wider {
		if n := len([]rune(l)); n > wordsMeasure {
			t.Errorf("a line ran to %d letters, past the measure of %d: %q", n, wordsMeasure, l)
		}
	}

	// And a short line is left alone: the measure breaks what is too long, it
	// does not chop everything into halves.
	if got := wordsWrap("She's a lady", 440, 200); len(got) != 1 {
		t.Errorf("twelve letters were broken into %d lines: %q", len(got), got)
	}
}

// A long word full of hyphens is broken at them.
//
// "You got me like whoa-whoa-whoa-whoa-whoa-whoa-whoa-whoa-whoa" is four words
// to a reader and five to strings.Fields, and the fifth is forty-four letters
// long. With spaces as the only place to break, the line could not be broken at
// all: it was set on one line at whatever size that took, which on a screen two
// thousand pixels across came to nine dots a letter — a ribbon of specks rather
// than a lyric.
//
// Breaking after a hyphen is what type has always done, and it takes as many of
// them as it needs: one break still leaves twenty-two letters here.
func TestALineOfHyphensIsBrokenAtThem(t *testing.T) {
	const line = "You got me like whoa-whoa-whoa-whoa-whoa-whoa-whoa-whoa-whoa"

	lines := wordsWrap(line, 360, 120)
	if len(lines) < 3 {
		t.Errorf("the line came out on %d lines: %q", len(lines), lines)
	}
	longest := 0
	for _, l := range lines {
		longest = max(longest, len([]rune(l)))
	}
	if longest > wordsMeasure {
		t.Errorf("the longest line is %d letters, past the measure of %d: %q",
			longest, wordsMeasure, lines)
	}

	// The hyphen stays on the line it was on, which is where a hyphen goes.
	for i, l := range lines[:len(lines)-1] {
		if !strings.HasSuffix(l, "-") {
			t.Errorf("line %d does not end on its hyphen: %q", i, l)
		}
	}

	// Nothing invented and nothing dropped: what was set is what came in.
	if got := strings.Join(lines, ""); got != line {
		t.Errorf("the lines put together are %q, not the line that came in", got)
	}
}

// And an ordinary line is not broken at every hyphen it happens to contain.
func TestAShortLineKeepsItsHyphensToItself(t *testing.T) {
	for _, line := range []string{"well-known and much-loved", "Sultans of Swing"} {
		if lines := wordsWrap(line, 360, 120); len(lines) != 1 {
			t.Errorf("%q was broken into %q with room to spare", line, lines)
		}
	}
}
