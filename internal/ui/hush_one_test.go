package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// The hush is one drawing, and it is drawn as large as the screen allows.
//
// A set can be smaller than a company. Asked for four of one the deal used to
// find one, call that a failure at every size and fall through to the smallest —
// so the one thing on screen was drawn as small as it can be drawn.
func TestTheHushIsOneAndItIsBig(t *testing.T) {
	set, ok := markSets[markHush]
	if !ok {
		t.Fatal("no hush")
	}
	const w, rows = 200, 44
	var share float64 = wordsMark
	band := int(share * float64(rows*dotsPerCellY))
	size, crowd, _, ok := markCrowdFor(set, band, w*dotsPerCellX, 7)
	if !ok {
		t.Fatal("no row")
	}
	if len(crowd) != 1 {
		t.Errorf("the hush came up as %d drawings", len(crowd))
	}
	// The largest baked size the band has room for, which is what one drawing
	// with the screen to itself should be given.
	var fits int
	for _, s := range set.sizes {
		if s.tall <= band {
			fits = max(fits, s.tall)
		}
	}
	if size.tall < fits {
		t.Errorf("the hush was drawn at %d dots with %d baked and the room for it", size.tall, fits)
	}
	t.Logf("one drawing, %d dots tall, %d wide", size.tall, crowd[0].wide)
}

// He cannot hear it, so he does not move to it.
//
// Everything else set on this screen answers the record — it rises with how loud
// the band is, leans on the beat and turns over on it. He has his fingers in his
// ears. Standing still while the water goes on behind him is the plainest way
// the picture can say the sound is reaching nobody.
func TestTheHushDoesNotDanceToWhatItCannotHear(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.stage.on = true
	m.stage.mode = scopeWords
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: 3 * time.Minute, Playing: true, Volume: 60}
	m.setProgress(40 * time.Second)
	m.words.forTrack = "one"
	m.wordsGrind()

	// Loud, and with a beat to lean on: a row that is listening moves.
	m.scope.beat.Loud, m.words.swellLow, m.words.swellHigh = -6, -40, -6
	m.words.beats = true

	if got := m.wordsRiding(4); got == nil {
		t.Fatal("nothing rides while the room can hear")
	}

	m.toggleMute()
	m.wordsGrind()

	if got := m.wordsRiding(1); got != nil {
		t.Errorf("he rose with the music: %v", got)
	}
	if tilt, _ := m.wordsTilting(1); tilt != nil {
		t.Errorf("he leant with the beat: %v", tilt)
	}
	if turned := m.wordsTurning(1); turned != nil {
		t.Errorf("he turned on the beat: %v", turned)
	}
}
