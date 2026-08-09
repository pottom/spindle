package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// wordless is a model playing a track the lyric database has nothing for.
func wordless(t *testing.T) Model {
	t.Helper()

	m := scopeModel(100, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.TrackID, m.ps.Title, m.ps.Album = "instrumental", "Windowlicker", "Windowlicker EP"
	m.ps.Artists = []string{"Aphex Twin"}
	m.ps.Duration = 6 * time.Minute
	m.lyrics.forTrack, m.lyrics.missing = m.ps.TrackID, true
	return m
}

// A record with no words is not given one picture and left with it. It takes
// turns: a card at the top of each one, the music for the rest, and never the
// same card twice running.
func TestAWordlessRecordTakesTurns(t *testing.T) {
	m := wordless(t)

	// The top of a record is where it changed, so nothing is put over it.
	if got := m.wordsCardFor(0); got != wordsCardNone {
		t.Errorf("the record opens on card %d, want the music alone", got)
	}

	seen := map[wordsCard]int{}
	was := m.wordsCardFor(0)
	for spell := 1; spell < 14; spell++ {
		card := m.wordsCardFor(spell)
		if card == was {
			t.Errorf("turn %d deals %d again, want a card it did not just show", spell, card)
		}
		seen[card]++
		was = card
	}

	if len(seen) < 3 {
		t.Errorf("twelve turns dealt %d different cards, want the record to keep changing", len(seen))
	}
	t.Logf("twelve turns dealt %v", seen)

	// And the same record deals the same way twice, so a song looked at again
	// is the song that was looked at.
	again := wordless(t)
	for spell := range 14 {
		if a, b := m.wordsCardFor(spell), again.wordsCardFor(spell); a != b {
			t.Fatalf("turn %d dealt %d one time and %d the next", spell, a, b)
		}
	}
}

// A wordless record is one long solo, so the marks have the screen whenever a
// card does not — never another visualiser. Switching between two of those put
// a different program up every half minute, and the join was what you saw.
func TestAWordlessRecordKeepsTheMarksUp(t *testing.T) {
	m := wordless(t)

	for _, at := range []time.Duration{0, wordsTitle + time.Second, wordsSpell + 10*time.Second} {
		m.setProgress(at)
		lines, _ := m.wordsIdle()
		if len(lines) != 1 || !wordsBeats(lines[0]) {
			t.Errorf("%s in, the screen has %q, want the marks", at, lines)
		}
	}

	// And a card takes their place for its few seconds, then hands it back.
	// Whichever turn is dealt one — the record's own name is not among them any
	// more, so what comes up is whose it is or what it came out on.
	spell := 0
	for at := 1; at < 12; at++ {
		if m.wordsCardFor(at) != wordsCardNone {
			spell = at
			break
		}
	}
	if spell == 0 {
		t.Fatal("no turn in twelve was dealt a card")
	}

	m.setProgress(time.Duration(spell)*wordsSpell + time.Second)
	lines, card := m.wordsIdle()
	if len(lines) == 0 || wordsBeats(lines[0]) {
		t.Fatalf("the turn dealt a card has %q", lines)
	}

	m.setProgress(time.Duration(spell)*wordsSpell + wordsTitle + time.Second)
	back, marks := m.wordsIdle()
	if len(back) != 1 || !wordsBeats(back[0]) {
		t.Errorf("after the card the screen has %q, want the marks back", back)
	}
	if marks <= card {
		t.Errorf("the marks are stamped %d against the card's %d, want them arriving after it", marks, card)
	}
}

// wordy is a backend that has words to give, which is what makes fetchLyrics
// worth calling at all.
type wordy struct{ *player.Mock }

func (wordy) Lyrics(context.Context, string) (*player.Lyrics, error) { return nil, nil }

// The words are sent for by whatever is going to show them. The pane on the
// player is one; the lyric picture on the big screen is the other, and it has
// its own key — before this, turning it on without the pane left it waiting for
// words nobody had asked for, and every record looked as though it had none.
func TestTheBigScreenSendsForTheWords(t *testing.T) {
	m := scopeModel(100, 44)
	m.player = wordy{player.NewMock()}
	m.lyrics.on = false

	if cmd := m.fetchLyrics(); cmd != nil {
		t.Fatal("the words were sent for with nothing on screen to show them")
	}

	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	if cmd := m.fetchLyrics(); cmd == nil {
		t.Error("the lyric picture is on the big screen and the words were never sent for")
	}
}

// Nothing is put up while the database has still to answer.
//
// A record whose sheet is a second late used to be taken for wordless the
// moment it started: the marks arrived for that second, and the first line came
// straight over them. Two changes of picture where the record had made one, and
// the first of them lasted a second.
func TestNothingIsPutUpOnSpec(t *testing.T) {
	m := scopeModel(100, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.TrackID, m.ps.Title = "new", "A Song"
	m.ps.Artists = []string{"The Band"}
	m.ps.Duration = 3 * time.Minute

	// The record has just started and nothing has answered about it yet.
	for _, at := range []time.Duration{time.Second, wordsSpell, 2 * wordsSpell} {
		m.setProgress(at)
		if lines, _ := m.wordsIdle(); len(lines) != 0 {
			t.Errorf("%s in, with no answer yet, the screen was given %q", at, lines)
		}
		if lines, _ := m.wordsComing(); len(lines) != 0 {
			t.Errorf("%s in, with no answer yet, the picture was asked for %q", at, lines)
		}
	}

	// The sheet lands, a second before the first line: the line is what goes up,
	// and the marks never had their second.
	m.lyrics.forTrack, m.lyrics.synced = "new", true
	m.lyrics.lines = []player.Lyric{{At: 61_000, Words: "the first line"}}
	m.setProgress(61*time.Second - wordsGather/2)

	lines, _ := m.wordsComing()
	if len(lines) != 1 || lines[0] != "the first line" {
		t.Errorf("with the sheet in and the singer due, the picture is %q, want the line", lines)
	}

	// And a record it has answered about and has nothing for takes its turns.
	m.lyrics.synced, m.lyrics.missing = false, true
	m.setProgress(wordsSpell + time.Second)
	if lines, _ := m.wordsIdle(); len(lines) == 0 {
		t.Error("a record known to have no words was left with nothing")
	}
}

// The screen has one picture, and the moment nothing is set is not an excuse
// for another one.
//
// It used to be the mirrored spectrum there — a different picture from every
// other one this screen draws, filling exactly the moments a record changes
// over. And it hid the change it was covering: its columns run to white
// wherever the music is loud, so the accent everything else is coloured by had
// nowhere to show.
func TestNothingSetIsStillTheSamePicture(t *testing.T) {
	const w, rows = 100, 40

	m := scopeModel(w, rows)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.9
	}
	m.scope.bands = bands

	idle := m.wordsIdleArt(w, rows)
	if len(idle) != rows {
		t.Fatalf("the idle picture is %d rows, want %d", len(idle), rows)
	}

	// The meter stands on the floor and hangs from the ceiling, which is what
	// every picture here does.
	if strings.TrimSpace(ansiOff(idle[0])) == "" {
		t.Error("nothing hangs from the ceiling")
	}
	if strings.TrimSpace(ansiOff(idle[rows-1])) == "" {
		t.Error("nothing stands on the floor")
	}

	// And the middle is empty, because there is nothing to put in it.
	var middle int
	for _, line := range idle[rows/2-2 : rows/2+2] {
		middle += len(strings.TrimSpace(ansiOff(line)))
	}
	t.Logf("the middle four rows carry %d cells", middle)

	// It is not the mirrored spectrum, which fills the middle and leaves the
	// edges bare — the other way round from this.
	mirrored := m.stageArt(w, rows)
	var was int
	for _, line := range mirrored[rows/2-2 : rows/2+2] {
		was += len(strings.TrimSpace(ansiOff(line)))
	}
	if middle >= was {
		t.Errorf("the middle carries %d cells against the mirrored picture's %d, want it left for what goes there", middle, was)
	}
}
