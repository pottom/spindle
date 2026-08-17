package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// Silence costs nothing. A paused player is fed the last thing that was heard,
// over and over, and drawing it sixty times a second cost between a fifth and a
// third of a core — measured per picture on a paused player: wave 31%, ladder
// 28%, mirror 26%, bars 21%, and six tenths of one per cent with no picture.
//
// So the picture sinks to where silence leaves it, and then the loop stops.
func TestThePictureRestsWhileNothingPlays(t *testing.T) {
	m := playerModel()
	m.scope.modes[tabPlayer] = scopeWave
	m.scope.running = true
	m.ps.Playing = false

	loud := []float32{0.9, -0.8, 0.7, 0.6}
	m.scope.frame = append([]float32(nil), loud...)
	m.scope.bands = []float32{0.9, 0.5}
	m.scope.envelope = 0.9

	// While it still has somewhere to sink to, the frames keep coming.
	var tm tea.Model = m
	tm, cmd := tm.Update(msg.WaveformReady{Samples: loud})
	if cmd == nil {
		t.Fatal("the picture stopped before it had settled")
	}
	if got := tm.(Model); !got.scope.running {
		t.Error("the loop was marked stopped while the picture was still moving")
	}

	// Sunk: nothing moves, so nothing is drawn.
	for range 200 {
		tm, cmd = tm.Update(msg.WaveformReady{Samples: loud})
		if cmd == nil {
			break
		}
	}
	settled := tm.(Model)
	if cmd != nil {
		t.Fatal("the picture never came to rest")
	}
	if settled.scope.running {
		t.Error("the loop is still marked as running")
	}
	if !settled.scope.atRest() {
		t.Error("it stopped while something was still moving")
	}
}

// And the music starting again wakes it at once, rather than a second later
// when the tick comes round.
func TestMusicStartingWakesThePicture(t *testing.T) {
	m := playerModel()
	m.scope.modes[tabPlayer] = scopeWave
	m.scope.running = false
	m.ps.Playing = false

	var tm tea.Model = m
	_, cmd := tm.Update(msg.StateFetched{State: &player.State{
		TrackID: m.ps.TrackID, Title: m.ps.Title, Playing: true, Duration: m.ps.Duration,
	}})
	if cmd == nil {
		t.Fatal("the state came back playing and nothing was asked for")
	}
}
