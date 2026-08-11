package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

func signModel(t *testing.T) Model {
	t.Helper()
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.stage.on = true
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: 3 * time.Minute, Playing: true, Volume: 60}
	m.signFlow() // adopt whatever the switches already were
	return m
}

// A switch sends somebody across with a placard, and he takes it away with him.
func TestASwitchWalksPast(t *testing.T) {
	m := signModel(t)
	if m.signWalking() {
		t.Fatal("somebody was already walking before a switch moved")
	}

	m.ps.Shuffle = true
	m.signFlow()
	if !m.signWalking() {
		t.Fatal("turning shuffle on sent nobody")
	}
	if m.sign.what != signShuffled {
		t.Errorf("the sign said %v", m.sign.what)
	}

	// He gets all the way across, and then the screen is the screen again.
	m.sign.at = time.Now().Add(-signCrosses - time.Millisecond)
	if m.signWalking() {
		t.Error("he was still there after the crossing was over")
	}

	// Off says so too, rather than saying nothing.
	m.ps.Shuffle = false
	m.signFlow()
	if m.sign.what != signInOrder {
		t.Errorf("turning shuffle off said %v", m.sign.what)
	}

	// All three of the repeat's states, each its own sign.
	for repeat, want := range map[string]signWhat{
		"context": signRepeatAll, "track": signRepeatOne, "off": signRepeatOff,
	} {
		m.ps.Repeat = repeat
		m.signFlow()
		if m.sign.what != want {
			t.Errorf("repeat %q showed %v, want %v", repeat, m.sign.what, want)
		}
	}
}

// Pressed again while he is out, he swaps the sign rather than a second one
// setting off behind him.
func TestASecondPressSwapsTheSign(t *testing.T) {
	m := signModel(t)
	m.ps.Shuffle = true
	m.signFlow()
	set := m.sign.at

	m.ps.Repeat = "track"
	m.signFlow()
	if m.sign.what != signRepeatOne {
		t.Errorf("the second press did not change the sign: %v", m.sign.what)
	}
	if !m.sign.at.Equal(set) {
		t.Error("the second press started him over rather than swapping what he carries")
	}
}

// The five signs each draw something, and none of it leaves the placard.
func TestEverySignFitsTheBoard(t *testing.T) {
	who, ok := figureFor(figureSigner)
	if !ok {
		t.Fatal("no signer")
	}
	for _, tall := range []int{62, 100, 140} {
		p, ok := who.at(tall, "walk0")
		if !ok {
			continue
		}
		slot := signSlotFor(p)
		w, h := slot.wide-2*signInset, slot.tall-2*signInset
		if w < 10 || h < 6 {
			t.Errorf("at %d dots the placard is only %dx%d", tall, w, h)
		}
		for _, what := range []signWhat{signShuffled, signInOrder, signRepeatAll, signRepeatOne, signRepeatOff} {
			var lit, out int
			signMark(what, 0, 0, w, h, func(x, y int) {
				lit++
				if x < 0 || y < 0 || x >= w || y >= h {
					out++
				}
			})
			if lit == 0 {
				t.Errorf("at %d dots, sign %v drew nothing", tall, what)
			}
			if out > 0 {
				t.Errorf("at %d dots, sign %v put %d dots outside the placard", tall, what, out)
			}
		}
	}
}
