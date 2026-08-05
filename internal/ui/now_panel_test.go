package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// libraryPlaying is the library tab with something playing, at a given size.
func libraryPlaying(t *testing.T, w, h int) Model {
	t.Helper()

	m := likedModel(t)
	m.ps = &player.State{
		TrackID: "now", Title: "Heroes", Artists: []string{"David Bowie"},
		Album: "\"Heroes\"", CoverURL: "https://example.test/heroes.jpg",
		Playing: true, DeviceName: "spindle",
		Progress: 30 * time.Second, Duration: 3 * time.Minute,
	}
	m.width, m.height = w, h
	m.resize()
	return m
}

// The library's band is in two halves: the row under the cursor on the left,
// what is playing on the right. Half a band of nothing was what it had before.
func TestTheLibraryBandShowsWhatIsPlaying(t *testing.T) {
	m := libraryPlaying(t, 200, 50)
	if !m.showsNowPanel() {
		t.Fatal("a wide library shows no panel for what is playing")
	}

	band := strings.Join(m.libraryPaneView(m.layout(), m.layout().bodyHeight)[:10], "\n")
	for _, want := range []string{"Heroes", "David Bowie", "0:30"} {
		if !strings.Contains(plain(band), want) {
			t.Errorf("the band does not say %q:\n%s", want, plain(band))
		}
	}
}

// Only there, and only with room for it: the queue spends that half on the
// trace, the player screen is nothing else, and a narrow one needs its width
// for the list.
func TestTheNowPanelKeepsToItsScreen(t *testing.T) {
	m := libraryPlaying(t, 200, 50)

	for _, tab := range []tabID{tabPlayer, tabQueue, tabSearch} {
		m.tab = tab
		if m.showsNowPanel() {
			t.Errorf("the %v tab draws the panel for what is playing", tab)
		}
	}

	narrow := libraryPlaying(t, 100, 40)
	if narrow.showsNowPanel() {
		t.Error("a narrow library gives half its band away rather than its list")
	}
}

// The two pictures are kept apart all the way through: they are asked for in
// their own slots, and an answer for one must not land in the other.
func TestTheTwoPicturesDoNotCrossOver(t *testing.T) {
	m := libraryPlaying(t, 200, 50)
	m.covers = cover.NewLoader(nil, nil)

	if cmd := m.syncNowCover(); cmd == nil {
		t.Fatal("the picture of what is playing was never asked for")
	}
	if m.nowCover.url != m.ps.CoverURL {
		t.Fatalf("the second picture is of %q, want what is playing", m.nowCover.url)
	}
	if m.cover.url == m.nowCover.url && m.cover.url != "" {
		t.Error("both pictures are of the same thing, want the cursor's and the playing one's")
	}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.CoverReady{
		URL: m.nowCover.url, Width: m.nowCover.width, Height: m.nowCover.height,
		Slot: nowSlot, Art: cover.Art{Cells: "picture"},
	})
	got := tm.(Model)
	if got.nowCover.art != "picture" {
		t.Error("the answer for the second slot never reached the second picture")
	}
	if got.cover.art == "picture" {
		t.Error("the answer for the second slot landed in the first picture")
	}
}
