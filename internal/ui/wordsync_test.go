package ui

import (
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
