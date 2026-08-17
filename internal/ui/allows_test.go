package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
)

// tempState keeps a test's answers out of the state directory a real run
// writes to: a cached answer from one test is an answer another test never
// asked for.
func tempState(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

// narrow is a backend whose registration Spotify refuses the optional
// endpoints, the way an application registered after 2024 is refused.
type narrow struct {
	player.Player
	asked int
}

func (n *narrow) Allows(context.Context) (player.Allowances, error) {
	n.asked++
	return player.Allowances{}, nil
}

// unreachable is a backend that could not be asked at all — no network, a token
// that has expired, a rate limit.
type unreachable struct{ player.Player }

func (unreachable) Allows(context.Context) (player.Allowances, error) {
	return player.Allowances{}, errors.New("ask what this application may do: no route to host")
}

// Somebody else's application is asked what it may do, and until the answer
// arrives nothing optional is offered: a key that fails when it is pressed is
// worse than a key that is not there.
func TestAnOwnApplicationIsAskedWhatItMayDo(t *testing.T) {
	tempState(t)

	backend := &narrow{Player: player.NewMock()}
	m := New(backend, nil, defaultTestCell).WithApplication("1c227ccd43c64c89918ce162bfc38c7b", true)
	m.width, m.height = 120, 40
	m.resize()

	if m.canSave() {
		t.Error("a key was offered before anybody had asked whether it works")
	}

	cmd := m.askAllows()
	if cmd == nil {
		t.Fatal("an application nobody knows anything about was not asked about")
	}
	var tm tea.Model = m
	tm, _ = tm.Update(cmd())
	if backend.asked != 1 {
		t.Errorf("Spotify was asked %d times, want once", backend.asked)
	}
	if tm.(Model).canSave() {
		t.Error("the key is offered by an application Spotify refuses it to")
	}
}

// The one spindle ships with is allowed everything and is not asked: a request
// spent hearing what is already known is a request wasted.
func TestTheShippedApplicationIsNotAsked(t *testing.T) {
	tempState(t)

	backend := &narrow{Player: player.NewMock()}
	m := New(backend, nil, defaultTestCell).WithApplication("d420a117a32841c2b3474932e49fb54b", false)

	if !m.allows.Has(player.Collecting) {
		t.Error("the application spindle ships with was not allowed what it is allowed")
	}
	if m.askAllows() != nil {
		t.Error("it was asked about anyway")
	}
	if backend.asked != 0 {
		t.Error("a request went out for an answer already known")
	}
}

// A question that could not be put is not an answer. Nothing is written down and
// nothing is turned on, and the next run asks again.
func TestAQuestionThatFailedIsNotAnAnswer(t *testing.T) {
	tempState(t)

	m := New(unreachable{player.NewMock()}, nil, defaultTestCell).WithApplication("1c227ccd43c64c89918ce162bfc38c7b", true)

	cmd := m.askAllows()
	if cmd == nil {
		t.Fatal("nothing was asked")
	}
	took, ok := cmd().(allowsTook)
	if !ok || took.err == nil {
		t.Fatalf("the failure was not carried back: %#v", took)
	}

	m.tookAllows(took)
	if m.canSave() {
		t.Error("a failure to ask turned a feature on")
	}
}

// And the settings screen says which application is in use and what it costs,
// because the alternative is a listener wondering where a key went.
func TestTheSettingsScreenNamesTheApplication(t *testing.T) {
	shipped := New(player.NewMock(), nil, defaultTestCell).WithApplication("d420a117a32841c2b3474932e49fb54b", false)
	shipped.width, shipped.height = 120, 40
	shipped.tab = tabSettings
	shipped.resize()
	if got := ansi.Strip(shipped.render()); !strings.Contains(got, "ships with") {
		t.Error("the screen does not say which application is in use")
	}

	own := New(player.NewMock(), nil, defaultTestCell).WithApplication("1c227ccd43c64c89918ce162bfc38c7b", true)
	own.width, own.height = 120, 40
	own.tab = tabSettings
	own.settings.cursor.cursor = settingApplication
	own.resize()

	screen := ansi.Strip(own.render())
	if !strings.Contains(screen, "your own") {
		t.Error("the screen does not say the listener brought their own application")
	}
	if !strings.Contains(screen, "without") {
		t.Errorf("the screen does not say what is missing:\n%s", screen)
	}
}
