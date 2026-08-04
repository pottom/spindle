package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func msgLyrics(id string) msg.LyricsFetched {
	return msg.LyricsFetched{TrackID: id, Synced: true,
		Lines: []player.Lyric{{At: 0, Words: "aaa"}, {At: 4000, Words: "bbb"}}}
}

func lyricsModel(w, h int) Model {
	m := scopeModel(w, h)
	m.ps.Duration = 3 * time.Minute
	m.lyrics.on = true
	m.lyrics.forTrack, m.lyrics.synced = m.ps.TrackID, true
	for i := range 12 {
		m.lyrics.lines = append(m.lyrics.lines, player.Lyric{
			At: int64(i) * 4000, Words: strings.Repeat("x", 6) + " " + string(rune('a'+i)),
		})
	}
	return m
}

// Turning the words off has to give back exactly the screen that was there
// before, the same promise the waveform makes.
func TestLyricsOffRestoresThePlayer(t *testing.T) {
	m := lyricsModel(120, 40)
	m.lyrics.on = false
	plainScreen := plain(m.render())

	m.lyrics.on = true
	withWords := plain(m.render())
	if withWords == plainScreen {
		t.Fatal("the words changed nothing")
	}

	m.lyrics.on = false
	if got := plain(m.render()); got != plainScreen {
		t.Error("the screen did not come back as it was")
	}
}

// A line too long for the column is wrapped, never cut: a line of a song that
// stops before its last word is not worth showing.
func TestLongLyricLinesWrap(t *testing.T) {
	m := lyricsModel(120, 40)
	l := m.layout()

	// Six words that cannot share one row.
	word := strings.Repeat("y", l.infoWidth/3)
	m.lyrics.lines[3].Words = strings.Join([]string{word, word, word, word, word, word}, " ")
	m.setProgress(12 * time.Second)

	rows := 0
	for _, row := range strings.Split(plain(m.render()), "\n") {
		if strings.Contains(row, word) {
			rows++
		}
		if len([]rune(row)) > m.width {
			t.Fatalf("a row ran past the frame: %q", row)
		}
	}
	if rows < 2 {
		t.Errorf("the long line took %d rows, want it wrapped over several", rows)
	}
	if strings.Contains(plain(m.render()), "…") {
		t.Error("the line was cut rather than wrapped")
	}
}

// Words are broken on spaces, and only mid-word when one word is longer than
// the whole row.
func TestWrapWords(t *testing.T) {
	got := wrapWords("one two three four", 9)
	if len(got) < 2 {
		t.Fatalf("wrapWords = %q, want it broken up", got)
	}
	for _, row := range got {
		if len(row) > 9 {
			t.Errorf("row %q is wider than 9", row)
		}
	}
	if strings.Join(got, " ") != "one two three four" {
		t.Errorf("wrapWords = %q, want every word kept", got)
	}
}

// Only the line being sung is at full strength; everything else recedes with
// distance, above and below alike.
func TestLyricsFadeBothWays(t *testing.T) {
	m := lyricsModel(120, 40)
	if len(m.styles.LyricFade) < 3 {
		t.Fatal("no fade to test")
	}

	if a, b := m.lyricStyle(-2), m.lyricStyle(2); a.GetForeground() != b.GetForeground() {
		t.Error("a line two ahead is not drawn like one two behind")
	}
	if now, near := m.lyricStyle(0), m.lyricStyle(1); now.GetForeground() == near.GetForeground() {
		t.Error("the line being sung is drawn like its neighbour")
	}
	// And the fade keeps going rather than flattening after one step.
	if near, far := m.lyricStyle(1), m.lyricStyle(4); near.GetForeground() == far.GetForeground() {
		t.Error("the fade stops after the first step")
	}
}

// The words follow the clock: the lit line is the one whose time has come.
func TestLyricsFollowTheClock(t *testing.T) {
	m := lyricsModel(120, 40)

	m.setProgress(0)
	if got := m.lyricsAt(); got != 0 {
		t.Errorf("at 0s the line is %d, want 0", got)
	}
	m.setProgress(9 * time.Second)
	if got := m.lyricsAt(); got != 2 {
		t.Errorf("at 9s the line is %d, want 2", got)
	}
	m.setProgress(2 * time.Minute)
	if got := m.lyricsAt(); got != len(m.lyrics.lines)-1 {
		t.Errorf("past the end the line is %d, want the last", got)
	}
}

// An answer that lands after a skip belongs to the wrong song, and captioning a
// track with another one's words is the only thing worse than none.
func TestLateLyricsAreDropped(t *testing.T) {
	m := lyricsModel(120, 40)
	m.lyrics.lines = nil
	m.lyrics.forTrack = ""

	m.adoptLyrics(msgLyrics("someone-else"))
	if m.lyrics.forTrack != "" {
		t.Error("the words of another track were adopted")
	}

	m.adoptLyrics(msgLyrics(m.ps.TrackID))
	if m.lyrics.forTrack != m.ps.TrackID {
		t.Error("the words of this track were not adopted")
	}
}

// The key is offered only where the words fit, and toggling it is what shows
// and hides them.
func TestLyricsKey(t *testing.T) {
	m := lyricsModel(120, 40)
	m.lyrics.on = false

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if !tm.(Model).lyricsVisible() {
		t.Fatal("l did not show the words")
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if tm.(Model).lyricsVisible() {
		t.Error("l did not put them away again")
	}
}

// A handful of lines around the one being sung is something to follow; the
// whole lyric at once is a page of text to search through.
func TestLyricsShowAWindow(t *testing.T) {
	m := lyricsModel(120, 44)
	l := m.layout()

	drawn := 0
	for _, row := range m.infoWithLyrics(l, l.bodyHeight) {
		if strings.Contains(plain(row), "xxxxxx") {
			drawn++
		}
	}
	if drawn > lyricsMaxRows {
		t.Errorf("%d lines on screen, want no more than %d", drawn, lyricsMaxRows)
	}
	if drawn < 3 {
		t.Errorf("%d lines on screen, want enough for context", drawn)
	}
}

// The line being sung keeps its place on screen from the first line to the
// last, so the eye following it never has to move — the opening of a track is
// padded above rather than starting the block at the top.
func TestLitLineHoldsItsPlace(t *testing.T) {
	m := lyricsModel(120, 44)

	litRow := func() int {
		for i, row := range m.lyricsBlock(44, lyricsMaxRows) {
			if strings.Contains(row, "\x1b[1;") {
				return i
			}
		}
		return -1
	}

	for _, at := range []time.Duration{0, 20 * time.Second, 32 * time.Second, 44 * time.Second} {
		m.setProgress(at)
		if got := litRow(); got != lyricsLead {
			t.Errorf("at %v the lit row is %d, want %d", at, got, lyricsLead)
		}
	}
}

// The line being sung is in the artwork's accent, like everything else on the
// screen that matters.
func TestLitLineIsAccent(t *testing.T) {
	m := lyricsModel(120, 44)
	m.setProgress(20 * time.Second)

	want := m.styles.Accent
	if got := m.styles.LyricFade[0].GetForeground(); got != want {
		t.Errorf("the lit line is %v, want the accent %v", got, want)
	}
	if !m.styles.LyricFade[0].GetBold() {
		t.Error("the lit line is not set apart from the fade")
	}
}

// The fade falls away as a curve, not a straight line: nearly flat beside the
// line being sung and steep at the edges, which is the shading that makes a
// surface look like it is curving away.
func TestFadeCurvesLikeACylinder(t *testing.T) {
	m := lyricsModel(120, 44)
	fade := m.styles.LyricFade

	// One step per row between the centre of the window and its edge, so the
	// deepest shade is actually reached and none is wasted.
	if want := lyricsLead + 1; len(fade) != want {
		t.Fatalf("the fade has %d steps for a window of %d rows, want %d", len(fade), lyricsMaxRows, want)
	}

	lum := func(i int) float64 {
		r, g, b, _ := fade[i].GetForeground().RGBA()
		return (0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(b>>8)) / 255
	}

	// Monotonic from the first fade step to the last.
	for i := 2; i < len(fade); i++ {
		if lum(i) >= lum(i-1) {
			t.Errorf("step %d is no darker than %d", i, i-1)
		}
	}
	// And the fall starts at once: without that, three or four rows around the
	// middle sit at almost one strength and the block flattens out.
	if drop := lum(1) - lum(2); drop < 0.08 {
		t.Errorf("the first step away drops only %.3f, want the fall to start at once", drop)
	}
	// The furthest row has to be genuinely faint, or the window has no edge.
	if lum(len(fade)-1) > lum(1)/3 {
		t.Errorf("the outermost row is %.3f against %.3f at the centre, want it far darker",
			lum(len(fade)-1), lum(1))
	}
}

// The window is symmetric about the line being sung, or the block reads as a
// list with a tail rather than as a curve.
func TestWindowIsSymmetric(t *testing.T) {
	m := lyricsModel(120, 44)
	m.setProgress(24 * time.Second)

	rows := m.lyricsBlock(44, lyricsMaxRows)
	lit := -1
	for i, r := range rows {
		if strings.Contains(r, "\x1b[1;") {
			lit = i
		}
	}
	above, below := lit, len(rows)-1-lit
	if above != below {
		t.Errorf("%d rows above the lit line and %d below, want them equal", above, below)
	}
}

// Showing the words moves the column beside the picture and nothing else. A
// visualiser that makes the cover jump is worse than no visualiser, and the
// same holds here.
func TestLyricsDoNotMoveTheArtwork(t *testing.T) {
	m := lyricsModel(120, 44)
	l := m.layout()
	// The trace steps aside for the words, which is its own rule; this is about
	// the picture staying put.
	m.scope.mode = scopeOff

	m.lyrics.on = false
	off := strings.Split(plain(m.render()), "\n")
	m.lyrics.on = true
	on := strings.Split(plain(m.render()), "\n")

	if len(off) != len(on) {
		t.Fatalf("%d rows without the words, %d with", len(off), len(on))
	}

	// The artwork column is everything left of the gap.
	cut := leftMargin + l.artWidth
	for i := range off {
		a, b := []rune(off[i]), []rune(on[i])
		end := min(cut, min(len(a), len(b)))
		if string(a[:end]) != string(b[:end]) {
			t.Errorf("row %d moved in the artwork column:\n  off: %q\n  on:  %q",
				i, string(a[:end]), string(b[:end]))
		}
	}
}

// The words and the trace both have their place: the information moves to the
// top of the column and the words take what it leaves, which is above the band
// the trace runs across. Neither has to give way.
func TestWordsAndWaveformCoexist(t *testing.T) {
	m := lyricsModel(120, 44)
	m.scope.mode = scopeWave
	m.setProgress(24 * time.Second)
	m.scope.frame = []float32{0.8, -0.8, 0.5, -0.5}
	m.scope.follow(m.scope.frame)

	if !m.scopeVisible() || !m.lyricsVisible() {
		t.Fatalf("scope %v, lyrics %v — want both", m.scopeVisible(), m.lyricsVisible())
	}

	var lastWord, firstTrace = -1, -1
	for i, row := range strings.Split(plain(m.render()), "\n") {
		if strings.Contains(row, "xxxxxx") {
			lastWord = i
		}
		if firstTrace < 0 && strings.ContainsRune(row, '⠀'+0x12) {
			firstTrace = i
		}
	}
	if lastWord < 0 {
		t.Fatal("no words on screen")
	}
	if firstTrace >= 0 && firstTrace <= lastWord {
		t.Errorf("the trace starts at row %d and the words end at %d, want them clear of each other",
			firstTrace, lastWord)
	}
}

// The information rises only as far as the top of the picture. Above that the
// two columns stop reading as one screen.
func TestInfoRisesNoHigherThanTheArtwork(t *testing.T) {
	m := lyricsModel(120, 44)
	l := m.layout()

	col := m.infoWithLyrics(l, l.bodyHeight)
	first := -1
	for i, row := range col {
		if strings.TrimSpace(plain(row)) != "" {
			first = i
			break
		}
	}
	if want := m.artTop(l, l.bodyHeight); first != want {
		t.Errorf("the information starts at row %d, want %d — the top of the picture", first, want)
	}
	if first == 0 {
		t.Error("the information went to the very top of the body")
	}
}

// Songs open with a bar or two of music. Lighting the first line through it
// says it is being sung when it is not.
func TestNothingIsLitBeforeTheFirstLine(t *testing.T) {
	m := lyricsModel(120, 44)
	m.lyrics.lines[0].At = 4000

	m.setProgress(time.Second)
	if got := m.lyricsAt(); got != -1 {
		t.Errorf("line %d is lit before the first one is sung", got)
	}
	for _, row := range m.lyricsBlock(44, lyricsMaxRows) {
		if strings.Contains(row, "\x1b[1;") {
			t.Errorf("a line is lit before the first one is sung: %q", plain(row))
		}
	}

	m.setProgress(5 * time.Second)
	if got := m.lyricsAt(); got < 0 {
		t.Error("nothing is lit once the first line's time has come")
	}
}

// The line being sung is swept through as it goes. Only lines are timed — no
// track measured had per-syllable data — so this claims no more than a progress
// bar does about a track: not which word, but how far in.
func TestLyricSweep(t *testing.T) {
	m := lyricsModel(120, 44)
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "the first line"},
		{At: 4000, Words: "and here is the second one"},
		{At: 8000, Words: "then a third"},
	}
	const length = 26

	m.setProgress(4 * time.Second)
	if got := m.lyricsSweep(1, length); got != 0 {
		t.Errorf("at the line's start %d characters are swept, want none", got)
	}

	m.setProgress(6 * time.Second)
	if got := m.lyricsSweep(1, length); got < length/2-2 || got > length/2+2 {
		t.Errorf("halfway through the line %d of %d are swept, want about half", got, length)
	}

	// It never runs past the line, whatever the clock says.
	m.setProgress(30 * time.Second)
	if got := m.lyricsSweep(1, length); got != length {
		t.Errorf("past the line %d of %d are swept, want all of it", got, length)
	}

	// And an unsynced lyric has nothing to sweep against, so the line stands
	// whole rather than creeping at a made-up rate.
	m.lyrics.synced = false
	if got := m.lyricsSweep(1, length); got != length {
		t.Errorf("an unsynced line swept to %d, want it drawn whole", got)
	}
}

// The sweep runs across the whole line, not restarting on each wrapped row: it
// is one line however many rows it takes.
func TestSweepCarriesAcrossWrappedRows(t *testing.T) {
	m := lyricsModel(120, 44)
	words := strings.Fields(strings.Repeat("word ", 12))
	m.lyrics.lines = []player.Lyric{
		{At: 0, Words: "before"},
		{At: 4000, Words: strings.Join(words, " ")},
		{At: 12000, Words: "after"},
	}
	length := len(strings.Join(words, " "))

	// A quarter of the way in, the sweep is a quarter through the whole line —
	// which is well past the end of the first wrapped row.
	m.setProgress(6 * time.Second)
	at := m.lyricsSweep(1, length)
	if at < length/5 || at > length/3 {
		t.Errorf("a quarter of the way in %d of %d is swept, want about a quarter", at, length)
	}

	rowWidth := 24
	if rows := wrapWords(m.lyrics.lines[1].Words, rowWidth); len(rows) < 2 {
		t.Fatal("the line did not wrap, so there is nothing to carry across")
	} else if at <= len(rows[0]) {
		t.Errorf("the sweep reached %d, still inside the first row of %d", at, len(rows[0]))
	}
}
