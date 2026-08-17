package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
)

// osdModel is a library screen with something playing, which is the case the
// card exists for: the keys work here and nothing on the screen answers them.
func osdModel() Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 120, 34
	m.tab = tabLibrary
	m.ps = &player.State{
		TrackID: "t1", Title: "Bohemian Rhapsody", Artists: []string{"Queen"},
		Playing: true, Volume: 60, Duration: 6 * time.Minute,
	}
	m.resize()
	return m
}

// The volume, the transport and the playhead all work from every tab, and on
// every tab but one there is nothing to see them on. Pressing volume-up while
// reading a playlist changed a number on a screen that was not drawn, and the
// only way to know it had worked was to hear it — which is exactly what
// somebody adjusting the volume cannot do yet.
func TestTheCardSaysWhatAKeyDidOnAnyTab(t *testing.T) {
	for _, tab := range []tabID{tabPlayer, tabQueue, tabLibrary, tabSearch} {
		m := osdModel()
		m.tab = tab
		m.resize()

		if strings.Contains(ansi.Strip(m.render()), "%") && tab != tabPlayer {
			t.Fatalf("%v: a reading was on screen before anything was pressed", tab)
		}

		cmd := m.setVolume(75)
		if cmd == nil {
			t.Fatalf("%v: the volume did not move", tab)
		}
		screen := ansi.Strip(m.render())
		if !strings.Contains(screen, "75%") {
			t.Errorf("%v: the card does not say the new volume:\n%s", tab, screen)
		}
		if !strings.Contains(screen, "volume") {
			t.Errorf("%v: the card does not say what was changed", tab)
		}
	}
}

// One at a time, and each of them says its own thing.
func TestEachKindOfCard(t *testing.T) {
	m := osdModel()

	m.showOSD(osdPlaying)
	m.ps.Playing = false
	if got := ansi.Strip(m.render()); !strings.Contains(got, "paused") {
		t.Error("the pause card does not say it is paused")
	}

	m.showOSD(osdSeeking)
	m.setProgress(90 * time.Second)
	if got := ansi.Strip(m.render()); !strings.Contains(got, "1:30") || !strings.Contains(got, "6:00") {
		t.Errorf("the seek card does not say where the track is:\n%s", got)
	}

	m.ps.Volume = 0
	m.showOSD(osdVolume)
	if got := ansi.Strip(m.render()); !strings.Contains(got, "muted") {
		t.Error("silence is not drawn as silence")
	}
}

// It takes itself down. A card that waited for the next redraw would sit there
// until the poll came round, which on a quiet library tab is a minute.
func TestTheCardTakesItselfDown(t *testing.T) {
	m := osdModel()
	if cmd := m.showOSD(osdVolume); cmd == nil {
		t.Fatal("nothing was scheduled to take it down")
	}
	if !m.osdUp() {
		t.Fatal("it never went up")
	}

	m.osd.at = time.Now().Add(-2 * osdFor)
	if m.osdUp() {
		t.Error("it outstayed its welcome")
	}
	if strings.Contains(ansi.Strip(m.render()), "%") {
		t.Error("it is still on the screen")
	}
}

// A run of presses is one card rather than a flicker of them: each press starts
// the clock again, so it goes a moment after the hand stops.
func TestARunOfPressesIsOneCard(t *testing.T) {
	m := osdModel()
	m.setVolume(65)
	first := m.osd.at

	m.osd.at = time.Now().Add(-osdFor / 2)
	m.setVolume(70)
	if !m.osd.at.After(first) {
		t.Error("the second press did not keep the card up")
	}
	if !m.osdUp() {
		t.Error("the card went down under the hand")
	}
}

// It is drawn over the screen rather than in it: nothing underneath may move,
// because the reader was looking at that a moment ago and will be again.
func TestTheCardMovesNothingUnderIt(t *testing.T) {
	m := osdModel()
	before := strings.Split(ansi.Strip(m.render()), "\n")

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
	after := strings.Split(ansi.Strip(tm.(Model).render()), "\n")

	if len(before) != len(after) {
		t.Fatalf("the screen is %d rows and became %d", len(before), len(after))
	}

	// Nothing either side of it moved. The card is a quarter of the way in from
	// the left, so the first columns of every row are what they were.
	const margin = 20
	for i := range before {
		if leftOf(before[i], margin) != leftOf(after[i], margin) {
			t.Errorf("row %d moved under the card:\n  %q\n  %q",
				i, leftOf(before[i], margin), leftOf(after[i], margin))
		}
	}

	// And nothing was pushed past the edge of the terminal.
	for i, row := range after {
		if w := len([]rune(strings.TrimRight(row, " "))); w > 120 {
			t.Errorf("row %d is %d cells wide on a terminal of 120", i, w)
		}
	}
}

func leftOf(row string, cols int) string {
	runes := []rune(row)
	if len(runes) < cols {
		return string(runes)
	}
	return string(runes[:cols])
}

// Words rather than pictograms: the first draft wore a speaker and a stopwatch,
// and a terminal with an ordinary font drew nothing at all where they should
// have been. Nothing on this card may need a font nobody has.
func TestTheCardIsWordsAndRules(t *testing.T) {
	m := osdModel()
	m.showOSD(osdVolume)

	for _, row := range m.osdCard() {
		for _, r := range ansi.Strip(row) {
			if r > 0x2600 {
				t.Errorf("the card carries %q, which is a code point a plain font may not have", r)
			}
		}
	}
}

// The arrows belong to whatever list is on screen — the wall walks by them, and
// so does the queue — which left the transport holding the shift key on every
// tab but one, for the two things somebody reaches for most often. So there is
// a set that needs no arrow: plus and minus for the level, the chevrons for
// winding through a track.
func TestTheTransportWithoutArrows(t *testing.T) {
	// A fresh screen for each: the state they act on is shared between a model
	// and its copies, so a reading taken after one press is not what the next
	// one started from.
	on := func(tab tabID) Model {
		m := osdModel()
		m.tab = tab
		m.resize()
		m.setProgress(time.Minute)
		return m
	}

	for _, tab := range []tabID{tabPlayer, tabQueue, tabLibrary} {
		if got := pressed(t, on(tab), "+").ps.Volume; got <= 60 {
			t.Errorf("%v: + left the volume at %d, want louder than 60", tab, got)
		}
		if got := pressed(t, on(tab), "-").ps.Volume; got >= 60 {
			t.Errorf("%v: - left the volume at %d, want quieter than 60", tab, got)
		}
		if got := pressed(t, on(tab), ">").playhead(); got <= time.Minute {
			t.Errorf("%v: > wound the track to %s, want further on", tab, got)
		}
		if got := pressed(t, on(tab), "<").playhead(); got >= time.Minute {
			t.Errorf("%v: < wound the track to %s, want further back", tab, got)
		}
	}
}

// And they are letters on the one screen that is a field to type in.
func TestTheSearchFieldKeepsItsCharacters(t *testing.T) {
	m := osdModel()
	m.tab = tabSearch
	m.search.typing = true
	m.resize()

	typed := pressed(t, m, "+")
	if typed.ps.Volume != 60 {
		t.Errorf("a character typed into the search field moved the volume to %d", typed.ps.Volume)
	}
	if !strings.Contains(typed.search.input.Value(), "+") {
		t.Errorf("the character never reached the query: %q", typed.search.input.Value())
	}
}

// pressed is press with a model to press it on: what the screen became.
func pressed(t *testing.T, m Model, key string) Model {
	t.Helper()
	var tm tea.Model = m
	tm, _ = tm.Update(press(key))
	return tm.(Model)
}
