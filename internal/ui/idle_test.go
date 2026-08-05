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

// The card is worth a few seconds of the turn. The rest of it belongs to the
// music, the same as a title has ever been.
func TestTheMusicHasMostOfEachTurn(t *testing.T) {
	m := wordless(t)

	m.setProgress(wordsSpell)
	if lines, _ := m.wordsIdle(); len(lines) == 0 {
		t.Fatal("the turn that says the record's name says nothing")
	}

	m.setProgress(wordsSpell + wordsTitle + time.Second)
	if lines, _ := m.wordsIdle(); len(lines) != 0 {
		t.Errorf("the card is still up %s into the turn, want the music to have it: %q",
			wordsTitle+time.Second, lines)
	}

	// And the turn after that brings one back.
	m.setProgress(2*wordsSpell + time.Second)
	if lines, _ := m.wordsIdle(); len(lines) == 0 {
		t.Error("the next turn deals nothing")
	}
}

// While the music has the screen it is not drawn the same way every time: the
// mirrored meter one turn, the stack of lamps the next.
func TestTheLampsTakeEveryOtherTurn(t *testing.T) {
	m := wordless(t)

	var lamps int
	for spell := range 6 {
		m.setProgress(time.Duration(spell)*wordsSpell + wordsTitle + time.Second)
		if m.wordsIdleLadder() {
			lamps++
		}
	}
	if lamps == 0 || lamps == 6 {
		t.Errorf("the lamps took %d of six turns, want them to take turns", lamps)
	}

	// A record with words of its own has no turns to take: the gap between two
	// lines is a song playing, not an empty screen wanting something on it.
	m.lyrics.forTrack, m.lyrics.missing, m.lyrics.synced = m.ps.TrackID, false, true
	for spell := range 6 {
		m.setProgress(time.Duration(spell)*wordsSpell + wordsTitle + time.Second)
		if m.wordsIdleLadder() {
			t.Fatal("a song with words was given the lamps between two lines")
		}
	}
	if lines, _ := m.wordsIdle(); len(lines) != 0 {
		t.Errorf("a song with words was dealt %q", lines)
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
