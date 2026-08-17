package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// refusesLists is a backend whose registration may not read a playlist somebody
// else owns — which is every application registered since Spotify's 2024
// clampdown, whoever owns it.
type refusesLists struct{ player.Player }

func (refusesLists) PlaylistTracksPage(context.Context, string, int) (player.Page[player.Track], error) {
	return player.Page[player.Track]{}, fmt.Errorf("fetch playlist tracks: %w", player.ErrNotPermitted)
}

// A list this application may not read says so where its tracks would be.
//
// It used to say "Nothing here." — which about a playlist with eighty tracks in
// it is the program telling a lie it could have avoided, and leaves somebody
// wondering whether the list is empty, whether spindle is broken, or whether
// Spotify is.
func TestARefusedPlaylistSaysWhyRatherThanNothing(t *testing.T) {
	m := New(refusesLists{player.NewMock()}, nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.tab = tabLibrary
	m.resize()
	m.stack = append(m.stack, openPage{
		kind: openPlaylist, id: "p9", name: "Ultimate Party Classics", subtitle: "Spotify · 80 tracks",
	})
	m.stack[0].pages.loading = true

	cmd := fetchOpenCmd(m.player, openPlaylist, "p9", 0)
	var tm tea.Model = m
	tm, _ = tm.Update(cmd())

	after := tm.(Model)
	if !after.open().refused {
		t.Fatal("the page does not know it was refused")
	}
	if after.err != nil {
		t.Error("a refusal of the application was flashed as an error instead")
	}
	if after.allows.Has(player.Elsewhere) {
		t.Error("the ability was left on after Spotify said no")
	}

	screen := ansi.Strip(after.render())
	if strings.Contains(screen, "Nothing here") {
		t.Error("the screen still claims the list is empty")
	}
	if !strings.Contains(screen, "will not hand this list") {
		t.Errorf("the screen does not say why:\n%s", screen)
	}
	if !strings.Contains(screen, "settings") {
		t.Error("the screen does not say where to find out more")
	}
}

// And a list that really is empty still says so.
func TestAnEmptyPlaylistStillSaysNothingHere(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.tab = tabLibrary
	m.resize()
	m.stack = append(m.stack, openPage{kind: openPlaylist, id: "p8", name: "Empty"})

	var tm tea.Model = m
	tm, _ = tm.Update(msg.OpenedFetched{ID: "p8"})

	if got := ansi.Strip(tm.(Model).render()); !strings.Contains(got, "Nothing here") {
		t.Error("an empty list no longer says it is empty")
	}
}
