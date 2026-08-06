package ui

import (
	"context"
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

	// The top of a record is where it changed, so nothing is put over it; the
	// turn after that is the one time it says its own name.
	if got := m.wordsCardFor(0); got != wordsCardNone {
		t.Errorf("the record opens on card %d, want the music alone", got)
	}
	if got := m.wordsCardFor(1); got != wordsCardTitle {
		t.Errorf("the second turn deals %d, want the record's own name", got)
	}

	seen := map[wordsCard]int{}
	was := wordsCardTitle
	for spell := 2; spell < 14; spell++ {
		card := m.wordsCardFor(spell)
		if card == was {
			t.Errorf("turn %d deals %d again, want a card it did not just show", spell, card)
		}
		if card == wordsCardTitle {
			t.Errorf("turn %d says the record's name again, want it said once", spell)
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
		if len(lines) != 1 || lines[0] != wordsNotes {
			t.Errorf("%s in, the screen has %q, want the marks", at, lines)
		}
	}

	// And the card takes their place for its few seconds, then hands it back.
	m.setProgress(wordsSpell + time.Second)
	lines, card := m.wordsIdle()
	if len(lines) != 2 || lines[0] != m.ps.Title {
		t.Fatalf("the turn that says the record's name has %q", lines)
	}

	m.setProgress(wordsSpell + wordsTitle + time.Second)
	back, marks := m.wordsIdle()
	if len(back) != 1 || back[0] != wordsNotes {
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
