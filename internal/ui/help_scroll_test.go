package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// The help page scrolls under a head that does not.
//
// The keys come to 36 rows on a wide terminal and 68 on a narrow one, so on
// anything short they were simply cut off at the bottom with nothing to say so.
// The head stays put because there is a picture in it, and a picture put into
// the terminal by the kitty protocol is placed on the screen rather than in the
// text: it stays where it was put while whatever is under it moves.
func TestTheHelpPageScrollsUnderItsHead(t *testing.T) {
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.width, m.height = 120, 30
	m.tab = tabHelp
	m.splashFlow()

	page := func(mm Model) []string { return mm.helpPanel(mm.layout(), max(mm.layout().bodyHeight, 1)) }
	top := page(m)

	out, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	got, ok := out.(Model)
	if !ok {
		t.Fatal("the update did not hand back a model")
	}
	if got.helpAt == 0 {
		t.Fatal("page down did not move the page")
	}
	down := page(got)

	// The head is where it was, and the keys are not.
	head := got.helpHead(got.layout().interior - leftMargin - rightMargin)
	for i := range head {
		if i < len(top) && i < len(down) && top[i] != down[i] {
			t.Errorf("row %d of the head moved with the page", i)
		}
	}
	if strings.Join(top[len(head):], "") == strings.Join(down[len(head):], "") {
		t.Error("the keys did not move")
	}

	// And it comes back, and does not run off the top.
	out, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	got, _ = out.(Model)
	out, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	got, _ = out.(Model)
	if got.helpAt != 0 {
		t.Errorf("paging up past the start left it at %d", got.helpAt)
	}
}
