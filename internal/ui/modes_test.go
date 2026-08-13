package ui

import (
	"context"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// atThePlayer is a model on the player tab with something playing, which is what
// both of these keys need before they will do anything.
func atThePlayer(t *testing.T, p player.Player, st *player.State) Model {
	t.Helper()
	return Model{
		keys:       newKeyMap(),
		tab:        tabPlayer,
		player:     p,
		ps:         st,
		progressAt: time.Now(),
	}
}

// pressAndSettle sends a key and runs whatever command came back, so the test
// sees what the player was actually asked for rather than only what the screen
// shows. Both halves are the point: only the first and the icon lies, only the
// second and it lags a poll behind.
func pressAndSettle(t *testing.T, m Model, key string) Model {
	t.Helper()
	next, cmd := m.Update(press(key))
	m = next.(Model)
	if cmd != nil {
		cmd()
	}
	return m
}

// The shuffle key flips what is drawn straight away and asks the player for the
// same thing. Both halves matter: only the first and the icon lies, only the
// second and it lags a poll behind.
func TestTheShuffleKeyFlipsBothTheScreenAndThePlayer(t *testing.T) {
	p := player.NewMock()
	m := atThePlayer(t, p, &player.State{Title: "something", Playing: true})

	m = pressAndSettle(t, m, "s")
	if !m.ps.Shuffle {
		t.Error("the screen still says shuffle is off")
	}
	if st, _ := p.State(context.Background()); !st.Shuffle {
		t.Error("the player was not asked to shuffle")
	}

	m = pressAndSettle(t, m, "s")
	if m.ps.Shuffle {
		t.Error("pressing it again did not turn shuffle off on the screen")
	}
	if st, _ := p.State(context.Background()); st.Shuffle {
		t.Error("the player was not asked to stop shuffling")
	}
}

// The repeat key walks one cycle: off, then the whole list, then the one track,
// then off again. The order is the one every other player uses, and going round
// has to land back where it started or the key is a trap.
func TestTheRepeatKeyWalksTheWholeCycle(t *testing.T) {
	p := player.NewMock()
	// Starting from off rather than from the zero value, because that is what
	// arrives: both backends normalise to one of the three modes before the
	// interface ever sees it.
	m := atThePlayer(t, p, &player.State{Title: "something", Playing: true, Repeat: player.RepeatOff})

	// Twice round, because the bug this guards against is a cycle that works
	// once and then sticks.
	for _, want := range []string{
		player.RepeatContext, player.RepeatTrack, player.RepeatOff,
		player.RepeatContext, player.RepeatTrack, player.RepeatOff,
	} {
		m = pressAndSettle(t, m, "r")
		if m.ps.Repeat != want {
			t.Fatalf("the screen says %q, want %q", m.ps.Repeat, want)
		}
		st, _ := p.State(context.Background())
		if st.Repeat != want {
			t.Fatalf("the player was asked for %q, want %q", st.Repeat, want)
		}
	}
}

// Anything that is not one of the three lands on off, which is the mode that
// leaves the player alone. Nothing should ever hand this a fourth value —
// both backends normalise first — but the argument is a string, and the cost of
// being wrong is a player that repeats a track nobody asked it to.
func TestTheRepeatCycleFromSomethingUnexpected(t *testing.T) {
	for _, odd := range []string{"", "sideways", "REPEAT_TRACK"} {
		if got := nextRepeat(odd); got != player.RepeatOff {
			t.Errorf("%q led to %q, want %q", odd, got, player.RepeatOff)
		}
	}
}

// A poll already in flight when the key was pressed carries the old flags. It
// must not put them back on the screen, or the icon turns on and flicks off
// again a moment later — which reads exactly like the key not working.
func TestAPollInFlightDoesNotUndoTheModes(t *testing.T) {
	m := Model{
		ps:              &player.State{Shuffle: true, Repeat: player.RepeatTrack, Title: "old"},
		progressAt:      time.Now(),
		optimisticUntil: time.Now().Add(optimisticWindow),
	}

	m.adopt(&player.State{Shuffle: false, Repeat: player.RepeatOff, Title: "new"})

	if !m.ps.Shuffle {
		t.Error("a stale poll turned shuffle back off")
	}
	if m.ps.Repeat != player.RepeatTrack {
		t.Errorf("a stale poll put repeat back to %q", m.ps.Repeat)
	}
	if m.ps.Title != "new" {
		t.Error("metadata should still come through")
	}
}

// And once the window has passed, what the daemon says is the truth — including
// when somebody changed it from their phone.
func TestAfterTheWindowTheDaemonsModesWin(t *testing.T) {
	m := Model{
		ps:              &player.State{Shuffle: true, Repeat: player.RepeatTrack},
		optimisticUntil: time.Now().Add(-time.Second),
	}

	m.adopt(&player.State{Shuffle: false, Repeat: player.RepeatContext})

	if m.ps.Shuffle {
		t.Error("shuffle stayed on after the window, ignoring the daemon")
	}
	if m.ps.Repeat != player.RepeatContext {
		t.Errorf("repeat is %q, want the daemon's %q", m.ps.Repeat, player.RepeatContext)
	}
}
