package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
)

// refusing is a backend that can be asked to save and always refuses the
// application, the way a registration Spotify made after 2024 is refused.
type refusing struct {
	player.Player
	asked int
}

func (r *refusing) Save(context.Context, string) error {
	r.asked++
	return fmt.Errorf("save track: %w", player.ErrNotPermitted)
}

func (r *refusing) Unsave(ctx context.Context, id string) error { return r.Save(ctx, id) }

// notCollecting is a backend with no library at all — the mock before this
// existed, and any future one that only plays.
type notCollecting struct{ player.Player }

// likeModel is a list of tracks with the cursor on one, and the saved tracks
// known.
func likeModel(t *testing.T, p player.Player) Model {
	t.Helper()

	m := New(p, nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.tab = tabQueue
	m.resize()
	m.queue = []player.Track{
		{ID: "t01", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}},
		{ID: "t02", Title: "Another", Artists: []string{"Somebody"}},
	}
	m.library.adoptLiked([]player.Track{m.queue[0]}, true)
	return m
}

// The heart fills in under the hand rather than half a second later, and the
// list of saved tracks follows it: they are two views of one collection, and a
// heart that disagreed with the row above it would be the screen contradicting
// itself.
func TestLikingFillsTheHeartAtOnce(t *testing.T) {
	m := likeModel(t, player.NewMock())
	m.queuePane.cursor.cursor = 1

	if m.library.saved("t02") {
		t.Fatal("the track was saved before anybody pressed anything")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	liked := tm.(Model)
	if !liked.library.saved("t02") {
		t.Error("the heart was not filled in")
	}
	if len(liked.library.liked) != 2 || liked.library.liked[0].ID != "t02" {
		t.Errorf("the saved tracks are %d and head with %q, want the new one first",
			len(liked.library.liked), liked.library.liked[0].ID)
	}
	if cmd == nil {
		t.Fatal("nothing was sent to Spotify")
	}

	// And the answer says so, quietly: the heart has already said which way it
	// went.
	back := cmd()
	took, ok := back.(savedTook)
	if !ok || took.err != nil || !took.saved {
		t.Fatalf("the backend answered %#v", back)
	}

	// Pressing it again takes it back out.
	var again tea.Model = liked
	again, _ = again.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if again.(Model).library.saved("t02") {
		t.Error("pressing it twice left the track liked")
	}
}

// A refusal of the application is a fact about the whole run: the heart goes
// back, the key stops being offered, and it says why once rather than failing
// under the hand every time.
func TestARefusedApplicationLosesTheKey(t *testing.T) {
	backend := &refusing{Player: player.NewMock()}
	m := likeModel(t, backend)
	m.queuePane.cursor.cursor = 1

	if !m.canSave() {
		t.Fatal("a backend that can be asked was not offered the key")
	}
	if !strings.Contains(ansi.Strip(m.render()), "like") {
		t.Error("the help bar does not offer the key")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if cmd == nil {
		t.Fatal("nothing was sent")
	}
	tm, _ = tm.Update(cmd())

	after := tm.(Model)
	if after.library.saved("t02") {
		t.Error("the heart stayed filled in after Spotify refused")
	}
	if after.canSave() {
		t.Error("the key is still offered after the application was refused")
	}
	if strings.Contains(ansi.Strip(after.render()), " like") {
		t.Error("the help bar still offers a key that cannot work")
	}
	if !strings.Contains(ansi.Strip(after.render()), "does not allow") {
		t.Error("nothing on the screen says why")
	}

	// And it is not asked again.
	was := backend.asked
	var more tea.Model = after
	if _, cmd := more.Update(tea.KeyPressMsg{Code: 'h', Text: "h"}); cmd != nil {
		cmd()
	}
	if backend.asked != was {
		t.Error("the key was pressed again and went out again")
	}
}

// A backend with no library is not asked at all, and offers nothing.
func TestABackendWithNoLibraryOffersNothing(t *testing.T) {
	m := likeModel(t, notCollecting{player.NewMock()})
	if m.canSave() {
		t.Error("a backend that cannot save was offered the key")
	}
	if m.toggleSaved() != nil {
		t.Error("something was sent to a backend that cannot save")
	}
	if strings.Contains(ansi.Strip(m.render()), " like") {
		t.Error("the help bar offers a key nothing is behind")
	}
}

// The player screen has no cursor and one record on it, so the key means the
// one that is sounding — and the heart beside its name is what answers.
//
// It was not offered there at all: an act with nowhere to show itself is an act
// nobody can tell happened, and there was no heart on that screen to show it.
func TestTheHeartOnThePlayerScreen(t *testing.T) {
	m := playerModel()
	m.tab = tabPlayer
	m.width, m.height = 120, 40
	m.resize()

	if strings.Contains(ansi.Strip(m.render()), likedMark) {
		t.Fatal("a heart was drawn for a track nobody has saved")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if cmd == nil {
		t.Fatal("the key did nothing on the player screen")
	}

	liked := tm.(Model)
	if !liked.library.saved(liked.ps.TrackID) {
		t.Error("what is playing was not taken into the collection")
	}
	if !strings.Contains(ansi.Strip(liked.render()), likedMark) {
		t.Error("nothing on the screen says it was")
	}
	if !strings.Contains(ansi.Strip(liked.render()), "like") {
		t.Error("the help bar does not offer the key")
	}
}
