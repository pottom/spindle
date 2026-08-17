package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/notes"
	"github.com/pottom/spindle/internal/player"
)

// countsRelated answers who sounds like whom, and counts being asked.
type countsRelated struct {
	player.Player
	asked int
}

func (c *countsRelated) RelatedArtists(context.Context, string) ([]player.Artist, error) {
	c.asked++
	return []player.Artist{{ID: "a2", Name: "Halott Pénz"}, {ID: "a3", Name: "Wellhello"}}, nil
}

// refusesRelated is an application Spotify has clamped down on.
type refusesRelated struct{ player.Player }

func (refusesRelated) RelatedArtists(context.Context, string) ([]player.Artist, error) {
	return nil, errors.New("fetch related artists: refused")
}

func artistPage(p player.Player) Model {
	m := New(p, nil, defaultTestCell)
	m.width, m.height = 130, 40
	m.tab = tabLibrary
	m.resize()
	m.stack = append(m.stack, openPage{kind: openArtist, id: "a1", name: "Majka"})
	m.artists = map[string]notes.Artist{"a1": {Name: "Majka", Line: "Hungarian rapper"}}
	return m
}

// The panel's "who else" line has needed a last.fm key since it went in.
// Spotify answers the same question to an application it has not clamped down
// on, so the line now works for everybody.
func TestWhoElseSoundsLikeThisWithoutAKey(t *testing.T) {
	backend := &countsRelated{Player: player.NewMock()}
	m := artistPage(backend)

	if got := ansi.Strip(m.render()); strings.Contains(got, "Halott Pénz") {
		t.Fatal("the panel had an answer before anybody asked")
	}

	cmd := m.askRelated("a1")
	if cmd == nil {
		t.Fatal("nothing was asked about who else sounds like this")
	}
	var tm tea.Model = m
	tm, _ = tm.Update(cmd())

	if got := ansi.Strip(tm.(Model).render()); !strings.Contains(got, "Halott Pénz") {
		t.Errorf("the panel does not say who else sounds like this:\n%s", got)
	}

	// Asked once per artist: the panel is redrawn on every frame.
	after := tm.(Model)
	if after.askRelated("a1") != nil || backend.asked != 1 {
		t.Errorf("it was asked %d times, want once", backend.asked)
	}
}

// Last.fm answers about how a record is heard rather than about a catalogue, and
// it is the only one that answers at all for some artists — so where it has
// spoken, it keeps the line.
func TestLastFMKeepsTheLineWhereItHasOne(t *testing.T) {
	m := artistPage(&countsRelated{Player: player.NewMock()})
	m.artists["a1"] = notes.Artist{Name: "Majka", Similar: []string{"Ganxsta Zolee"}}
	m.related = map[string][]player.Artist{"a1": {{ID: "a2", Name: "Halott Pénz"}}}

	got := ansi.Strip(m.render())
	if !strings.Contains(got, "Ganxsta Zolee") {
		t.Error("what last.fm said was dropped")
	}
	if strings.Contains(got, "Halott Pénz") {
		t.Error("two sources were mixed into one line")
	}
}

// An application that may not ask is not asked, and one that is refused leaves
// the line as it was rather than showing an error where a name should be.
func TestARefusedApplicationAsksNobody(t *testing.T) {
	narrow := artistPage(&countsRelated{Player: player.NewMock()})
	narrow.allows = player.Allowances{}
	if narrow.askRelated("a1") != nil {
		t.Error("an application that may not ask asked anyway")
	}

	refused := artistPage(refusesRelated{player.NewMock()})
	cmd := refused.askRelated("a1")
	if cmd == nil {
		t.Fatal("nothing was asked")
	}
	var tm tea.Model = refused
	tm, _ = tm.Update(cmd())
	if got := ansi.Strip(tm.(Model).render()); strings.Contains(got, "refused") {
		t.Error("a refusal was drawn where a name should be")
	}
}

// The names Spotify gives are somewhere to go: the menu over an artist offers
// each of them, and choosing one opens their records.
func TestTheMenuOffersWhoTheySoundLike(t *testing.T) {
	m := artistPage(&countsRelated{Player: player.NewMock()})
	m.related = map[string][]player.Artist{"a1": {
		{ID: "a2", Name: "Halott Pénz"},
		{ID: "a3", Name: "Wellhello"},
		{ID: "a4", Name: "Punnany Massif"},
		{ID: "a5", Name: "Ganxsta Zolee"},
	}}

	verbs := m.actionsForArtist(player.Artist{ID: "a1", Name: "Majka"})
	var offered []string
	for _, v := range verbs {
		if strings.HasPrefix(v.label, "Sounds like") {
			offered = append(offered, v.label)
		}
	}
	if len(offered) != relatedVerbs {
		t.Errorf("the menu offers %d of them, want %d: %v", len(offered), relatedVerbs, offered)
	}
	if !strings.Contains(offered[0], "Halott Pénz") {
		t.Errorf("the first is %q", offered[0])
	}

	// And it goes there.
	for _, v := range verbs {
		if strings.Contains(v.label, "Halott Pénz") {
			v.do(&m)
		}
	}
	if page := m.open(); page == nil || page.id != "a2" {
		t.Error("choosing one did not open their records")
	}
}

// An artist nobody is compared to gets no such verbs, rather than an empty one.
func TestNoOneToSoundLikeIsNoVerb(t *testing.T) {
	m := artistPage(&countsRelated{Player: player.NewMock()})
	for _, v := range m.actionsForArtist(player.Artist{ID: "a1", Name: "Majka"}) {
		if strings.HasPrefix(v.label, "Sounds like") {
			t.Error("the menu offered somebody it has never heard of")
		}
	}
}
