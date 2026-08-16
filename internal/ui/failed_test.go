package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A play that failed says why, and goes on saying it long enough to be read.
//
// The polls arrive every few seconds and every one of them used to wipe it: a
// track that would not play explained itself for an instant and then went quiet,
// leaving somebody pressing enter at a list that would not play and nothing on
// the screen saying why.
func TestAFailureIsReadableBeforeItIsWiped(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 100, 40
	m.resize()
	m.ps = &player.State{TrackID: "a", Title: "one", Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.Error{Err: errors.New("play track: rate limited")})
	if got := tm.(Model); got.err == nil {
		t.Fatal("the failure was not taken up at all")
	}

	// A poll lands a moment later, as one does.
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "a", Title: "one", Playing: true}})
	shown := tm.(Model)
	if shown.err == nil {
		t.Fatal("a successful poll wiped the failure before anybody could read it")
	}
	if !strings.Contains(ansi.Strip(shown.render()), "rate limited") {
		t.Error("the screen does not say what went wrong")
	}

	// And once it has been up long enough, the next answer may have it.
	shown.errAt = time.Now().Add(-2 * saidWindow)
	var back tea.Model = shown
	back, _ = back.Update(msg.StateFetched{State: &player.State{TrackID: "a", Title: "one", Playing: true}})
	if back.(Model).err != nil {
		t.Error("the failure is still on screen long after it was news")
	}
}
