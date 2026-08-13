package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

func syncModel(t *testing.T) Model {
	t.Helper()
	m := stageModel(120, 44)
	m.stage.mode = scopeWords
	m.lyrics.synced = true
	m.lyrics.lines = []player.Lyric{
		{At: 10000, Words: "one two three four"},
		{At: 14000, Words: "five six seven eight"},
	}
	m.words.starts, m.words.ends = 10000, 14000
	m.words.where.Count = 4
	m.words.sync = syncModeInk
	return m
}

// The two screens put the voice in the same place on the same line.
//
// They did not: the player screen spends a line's singing on its syllables and
// the big screen spent it on its pieces, giving a comma as long as the word in
// front of it. On real lines the two disagreed about when the last word of a
// line starts by up to 478 ms — which is what was seen, from across a room, as
// the end of the line lagging the singer.
func TestBothScreensPutTheVoiceInTheSamePlace(t *testing.T) {
	lines := []struct{ lang, text string }{
		{"en", "And all along the borderlines, of everything we knew"},
		{"en", "They say the times are changing, on the other side."},
		{"en", "Ain't nothin' gonna be the same again."},
		{"hu", "Olyan jó volt fiatalnak lenni, akkor"},
		{"hu", "Pedig már eltelt jó pár év."},
		{"hu", "Sose voltunk ilyen büszkék,"},
	}

	const window = 3500 * time.Millisecond
	for _, l := range lines {
		m := stageModel(120, 44)
		m.stage.mode = scopeWords
		m.lyrics.synced, m.lyrics.language = true, l.lang
		m.lyrics.lines = []player.Lyric{{At: 10000, Words: l.text}}
		m.words.starts, m.words.ends = 10000, 10000+window.Milliseconds()
		m.words.text, m.words.sync = l.text, syncModeInk

		pieces := wordsPieces(l.text)
		m.words.where.Count = len(pieces)

		// The last piece that is a word rather than a mark, and where it ends in
		// the line — which is what the player screen's sweep counts up to.
		last, ends := -1, 0
		for i, p := range pieces {
			if strings.TrimFunc(l.text[p.from:p.to], wordsPunct) != "" {
				last, ends = i, len([]rune(l.text[:p.to]))
			}
		}

		reach := func(lit func(at time.Duration) bool) time.Duration {
			for at := time.Duration(0); at < 2*window; at += 10 * time.Millisecond {
				if lit(at) {
					return at
				}
			}
			return -1
		}

		stage := reach(func(at time.Duration) bool {
			m.setProgress(10*time.Second + at)
			pos, ok := m.wordsSyncAt()
			return ok && pos > float32(last)
		})
		sung := lyricsSung(window)
		player := reach(func(at time.Duration) bool {
			frac := float64(at-lyricsStampsEarly) / float64(sung)
			return sweepTo(l.text, l.lang, min(max(frac, 0), 1)) >= ends
		})

		if stage < 0 || player < 0 {
			t.Errorf("the last word of %q never lit: stage %v, player %v", l.text, stage, player)
			continue
		}
		if apart := max(stage-player, player-stage); apart > 50*time.Millisecond {
			t.Errorf("the last word of %q lights at %v on the big screen and %v on the player screen, %v apart",
				l.text, stage, player, apart)
		}
	}
}

// A mark is not sung, so it is not worth any of the singing: it lights with the
// word beside it rather than holding the line up for a slice of a second.
func TestPunctuationTakesNoTimeFromTheLine(t *testing.T) {
	// Two words and a mark hanging off each: "sooner, now." — two syllables and
	// one, and the two marks worth nothing.
	shares := []float32{2, 0, 1, 0}

	for _, at := range []struct {
		frac float32
		want float32
	}{
		{0, 0},
		{0.25, 0.375}, // a quarter of the way through a word worth two thirds
		{0.5, 0.75},   // still inside the first word, which is most of the line
		{0.667, 2},    // the first word is done, and its comma with it
		{0.833, 2.5},  // halfway through the second
		{1, 4},        // all of it, marks and all
	} {
		if got := wordsSyncWalk(shares, at.frac); got < at.want-0.02 || got > at.want+0.02 {
			t.Errorf("%.1f%% through the line the voice is at piece %.2f, want %.2f", at.frac*100, got, at.want)
		}
	}

	// The mark is never where the voice is: it is behind the word it hangs off
	// the moment that word is finished, and never a stop of its own.
	if before, after := wordsSyncWalk(shares, 0.66), wordsSyncWalk(shares, 0.68); before > 1 || after < 2 {
		t.Errorf("the comma held the line up: the voice went %.2f -> %.2f across the end of the first word", before, after)
	}

	// And with nothing to weigh — a row of marks, a line with no letters in it —
	// the pieces share it out evenly, which is what it did before.
	if got := wordsSyncWalk([]float32{0, 0, 0, 0}, 0.5); got != 2 {
		t.Errorf("with nothing to weigh the voice is at piece %.2f of 4, want it halfway", got)
	}
}

// Where the voice is, in words: nothing before the line's own stamp, and the
// whole line by the time the singing is done. The span is the measured one,
// which is 85% of the window rather than the window.
func TestTheSyncFollowsTheMeasuredSpan(t *testing.T) {
	m := syncModel(t)
	sung := lyricsSung(4 * time.Second)

	m.setProgress(10 * time.Second)
	if at, ok := m.wordsSyncAt(); !ok || at != 0 {
		t.Errorf("at the stamp the voice is at word %.2f (ok=%v), want the first", at, ok)
	}

	m.setProgress(10*time.Second + lyricsStampsEarly + sung/2)
	at, _ := m.wordsSyncAt()
	if at < 1.7 || at > 2.3 {
		t.Errorf("halfway through the singing the voice is at word %.2f of 4, want about two", at)
	}

	m.setProgress(10*time.Second + lyricsStampsEarly + sung)
	if at, _ := m.wordsSyncAt(); at < 4 {
		t.Errorf("at the end of the singing the voice is at word %.2f, want all four", at)
	}
}

// The three ways of showing it are three different pictures, and each does what
// it says: ink fills in behind the voice, heat glows around it and fades either
// side, and lift moves a word instead of lighting it.
func TestEachEffectDoesItsOwnThing(t *testing.T) {
	m := syncModel(t)
	m.setProgress(10*time.Second + lyricsStampsEarly + lyricsSung(4*time.Second)/2)

	// Line 0 is the ink: the words behind the voice are at full strength and
	// the ones ahead are a ghost.
	m.words.sync = syncModeInk
	if behind, ahead := m.wordsSyncPaint(0, 8, 10), m.wordsSyncPaint(3, 8, 10); behind <= ahead {
		t.Errorf("ink: behind the voice %d, ahead of it %d — want the sung part brighter", behind, ahead)
	}
	if m.wordsSyncLifts(4) != nil {
		t.Error("ink moved a word, which is the lift's job")
	}

	// Line 1 is the heat: brightest where the voice is, falling away on both
	// sides — the word behind it is dimmer than the word under it.
	m.words.sync = syncModeHeat
	under, back, far := m.wordsSyncPaint(2, 8, 10), m.wordsSyncPaint(0, 8, 10), m.wordsSyncPaint(3, 8, 10)
	if under <= back || under <= far {
		t.Errorf("heat: under the voice %d, behind %d, ahead %d — want the middle brightest", under, back, far)
	}

	// Line 2 is the lift: the word under the voice is raised, the others are
	// where they were.
	m.words.sync = syncModeLift
	lifts := m.wordsSyncLifts(4)
	if len(lifts) != 4 {
		t.Fatalf("lift: %d words were given a height, want 4", len(lifts))
	}
	if lifts[2] >= 0 {
		t.Errorf("lift: the word under the voice moved by %d dots, want it raised", lifts[2])
	}
	if lifts[3] != 0 {
		t.Errorf("lift: a word the voice has not reached moved by %d dots", lifts[3])
	}
}

// Off is off: with the key untouched nothing about the picture changes, which
// is what makes the two comparable at all.
func TestOffChangesNothing(t *testing.T) {
	m := syncModel(t)
	m.words.sync = syncOff
	m.setProgress(11 * time.Second)

	if _, ok := m.wordsSyncAt(); ok {
		t.Error("the voice was followed with the key off")
	}
	if got := m.wordsSyncPaint(0, 7, 10); got != 7 {
		t.Errorf("a word came out at %d with the key off, want the %d the sound gave it", got, 7)
	}
	if m.wordsSyncLifts(4) != nil {
		t.Error("a word moved with the key off")
	}
}

// The key cycles off, faint, dark and back — three stops, so both ways of
// treating the words ahead can be seen and then put away.
func TestTheKeyCyclesTheWays(t *testing.T) {
	got := syncOff
	var seen []syncMode
	for range 3 {
		got = got.next()
		seen = append(seen, got)
	}
	if seen[0] != syncModeInk || seen[1] != syncModeHeat || seen[2] != syncModeLift {
		t.Errorf("the key went %v, want ink, heat, lift", seen)
	}
}

// A line of type has no Lefts and Rights — those belong to a row of marks — and
// the burst has to find a word's ink without them. It read them anyway, at
// minus one, and took the program down in front of somebody watching the
// screen.
func TestTheBurstFindsAWordWithoutTheMarksLayout(t *testing.T) {
	where := msg.WordLayout{DotsX: 10, Count: 3, Tops: []int{0}, Bottoms: []int{3}}
	where.At = make([]int16, 10)
	for x := range where.At {
		switch {
		case x < 3:
			where.At[x] = 0
		case x < 4:
			where.At[x] = -1 // the space between two words
		case x < 7:
			where.At[x] = 1
		default:
			where.At[x] = 2
		}
	}

	spans := wordsSyncSpans(where, where.Count)
	if len(spans) != 3 {
		t.Fatalf("%d words were measured, want 3", len(spans))
	}
	for i, want := range [][2]int{{0, 2}, {4, 6}, {7, 9}} {
		if spans[i] != want {
			t.Errorf("word %d covers %v, want %v", i, spans[i], want)
		}
	}

	// And with no map at all there is nothing to burst, rather than a panic.
	if got := wordsSyncSpans(msg.WordLayout{}, 3); got != nil {
		t.Errorf("an empty layout measured %v, want nothing", got)
	}
}

// The whole picture is drawn with the burst on, at a size the screen actually
// is, without going down. This is the test the panic asked for: the effects run
// inside View, and a picture that panics takes the program with it.
func TestTheBurstDrawsAWholeScreen(t *testing.T) {
	m := stageModel(120, 40)
	m.stage.mode = scopeWords
	m.lyrics.synced = true
	m.words.sync = syncModeInk
	m.words.line = int(syncBurst)
	m.words.starts, m.words.ends = 10000, 14000
	m.setProgress(11 * time.Second)

	line := "one two three four five"
	if img, layout, ok := wordsImage([]string{line}, 120*dotsPerCellX, 40*dotsPerCellY); ok {
		m.words.have = cover.Grind(grayToImage(img), 120, 40, dotsPerCellX, dotsPerCellY)
		m.words.cellsX, m.words.cellsY, m.words.where = 120, 40, layout
		m.words.text = line
		m.words.since = time.Now().Add(-time.Second)
	}
	if got := m.wordsLines(120, 40); len(got) != 40 {
		t.Errorf("the picture came out %d rows, want 40", len(got))
	}
}
