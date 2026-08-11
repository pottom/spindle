package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// A stopped record looks stopped.
//
// The other half of the hush: a bar of music with no music in it should look
// like something, so one unimpressed figure holds the pause up and stands there
// until the record goes on again.
func TestAStoppedRecordLooksStopped(t *testing.T) {
	if set, ok := markSets[markHeld]; !ok {
		t.Fatal("there is no held set")
	} else if !set.apart {
		t.Error("the held set is in the deal, so it can arrive while a record plays")
	}
	for _, tall := range markHeights() {
		for _, one := range markEveryone(tall) {
			if one.set == markHeld {
				t.Fatalf("%q was in the pool at %d dots", one.name, tall)
			}
		}
	}

	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.stage.on = true
	m.scope.modes[m.tab] = scopeWords
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: 3 * time.Minute, Playing: true, Volume: 60}
	m.setProgress(40 * time.Second)
	m.words.forTrack = "one"

	m.wordsGrind()
	if m.words.cast == markHeld {
		t.Fatal("a playing record was drawn as stopped")
	}

	m.ps.Playing = false
	m.wordsGrind()
	if m.words.cast != markHeld {
		t.Errorf("a stopped record was drawn as %q", m.words.cast)
	}
	if !m.wordsStill() {
		t.Error("he moved to a record that is not playing")
	}

	// Stopped beats silenced: a record nobody is playing is not playing to
	// anybody, silenced or not.
	m.toggleMute()
	m.wordsGrind()
	if m.words.cast != markHeld {
		t.Errorf("stopped and silenced at once was drawn as %q", m.words.cast)
	}

	// And a device that has said nothing yet is not a stopped record.
	m.ps = &player.State{}
	if m.held() {
		t.Error("a status with no track in it read as stopped")
	}
}
