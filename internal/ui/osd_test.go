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
		if !strings.Contains(screen, osdSpeaker) {
			t.Errorf("%v: the card carries no glyph", tab)
		}
	}
}

// One at a time, and each of them says its own thing.
func TestEachKindOfCard(t *testing.T) {
	m := osdModel()

	m.showOSD(osdPlaying)
	m.ps.Playing = false
	if got := ansi.Strip(m.render()); !strings.Contains(got, "paused") || !strings.Contains(got, iconPause) {
		t.Error("the pause card does not say it is paused")
	}

	m.showOSD(osdSeeking)
	m.setProgress(90 * time.Second)
	if got := ansi.Strip(m.render()); !strings.Contains(got, "1:30 / 6:00") {
		t.Errorf("the seek card does not say where the track is:\n%s", got)
	}

	m.ps.Volume = 0
	m.showOSD(osdVolume)
	if got := ansi.Strip(m.render()); !strings.Contains(got, "muted") || !strings.Contains(got, osdMute) {
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
