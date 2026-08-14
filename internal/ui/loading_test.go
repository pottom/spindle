package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// A list that has not arrived is not an empty list. The pages are small —
// Spotify answers ten search results at a time and refuses eleven — so the
// difference between a slow answer and a wrong one is on screen often.
func TestAListStillReadingDoesNotClaimToBeEmpty(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabLibrary
	m.width, m.height = 120, 40
	m.resize()
	m.library.pages[libraryPlaylists].loading = true

	screen := plain(strings.Join(m.libraryPaneView(m.layout(), m.layout().bodyHeight), "\n"))
	if strings.Contains(screen, "Nothing saved yet") {
		t.Errorf("a library still reading says it is empty:\n%s", screen)
	}
	if !strings.Contains(screen, "Reading your library") {
		t.Errorf("a library still reading says nothing about it:\n%s", screen)
	}
}

// The search waits more than any other screen, and a query in flight looks
// exactly like one that matched nothing.
func TestAQueryInFlightSaysSo(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSearch
	m.width, m.height = 120, 40
	m.resize()
	m.search.input.SetValue("bowie")
	m.search.current().pages.loading = true

	screen := plain(strings.Join(m.searchPaneView(m.layout(), m.layout().bodyHeight), "\n"))
	if strings.Contains(screen, "Nothing matched") {
		t.Errorf("a query in flight says it matched nothing:\n%s", screen)
	}
	if !strings.Contains(screen, "Asking Spotify") {
		t.Errorf("a query in flight says nothing about it:\n%s", screen)
	}
}

// The rows already on screen are not always all of them, and nothing else says
// so: the spinner beside the heading does.
func TestMoreOnTheWayTurnsTheSpinner(t *testing.T) {
	m := likedModel(t)
	m.width, m.height = 120, 40
	m.resize()

	quiet := plain(strings.Join(m.libraryPaneView(m.layout(), m.layout().bodyHeight), "\n"))
	m.library.pages[libraryPlaylists].loading = true
	busy := plain(strings.Join(m.libraryPaneView(m.layout(), m.layout().bodyHeight), "\n"))

	if quiet == busy {
		t.Error("a page on its way changes nothing on the screen")
	}
}

// The spinner only turns while something is waited for: a redraw every hundred
// milliseconds is the whole cost of it.
func TestTheSpinnerStopsWhenNothingIsWaitedFor(t *testing.T) {
	m := likedModel(t)
	if m.listLoading() {
		t.Error("a library that has arrived still says it is reading")
	}

	m.library.pages[libraryPlaylists].loading = true
	if !m.listLoading() {
		t.Error("a library reading a page says it is not")
	}

	// And a page of an opened list counts the same way.
	m.library.pages[libraryPlaylists].loading = false
	showOpen(&m, player.Playlist{ID: "p1", Name: "one"}, nil)
	m.openMut().pages.loading = true
	if !m.listLoading() {
		t.Error("an open list reading a page says it is not")
	}
}

// Asking for a page starts the spinner turning, or it would sit still through
// the wait it exists for.
func TestAskingForAPageStartsTheSpinner(t *testing.T) {
	m := likedModel(t)
	m.library.pages[libraryAlbums].loading = false

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("switching to a list never read asked for nothing")
	}
	if !tm.(Model).listLoading() {
		t.Error("the new list is not marked as reading")
	}
}

// What is on screen is asked for again while it is on screen, because nothing in
// the Web API says a playlist was edited from a phone.
func TestWhatIsOnScreenIsAskedForAgain(t *testing.T) {
	m := likedModel(t)
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()

	// Fresh: nothing to do.
	m.library.pages[m.library.kind].at = time.Now()
	if cmd := m.refreshOnScreen(); cmd != nil {
		t.Error("a list read a moment ago was asked for again")
	}

	// Old: asked for.
	m.library.pages[m.library.kind].at = time.Now().Add(-staleAfter - time.Second)
	if cmd := m.refreshOnScreen(); cmd == nil {
		t.Error("a list older than the stale mark was not asked for again")
	}
	if !m.library.pages[m.library.kind].loading {
		t.Error("the refresh did not mark the page as being fetched")
	}

	// And a fetch already in flight is not asked for twice.
	m.library.pages[m.library.kind].at = time.Now().Add(-staleAfter - time.Second)
	if cmd := m.refreshOnScreen(); cmd != nil {
		t.Error("a page already being fetched was asked for again")
	}

	// The queue is the daemon's rather than Spotify's, and is asked far more
	// often for that reason.
	m.tab = tabQueue
	m.queueAt = time.Now()
	if cmd := m.refreshOnScreen(); cmd != nil {
		t.Error("the queue was asked for again a moment after it arrived")
	}
	m.queueAt = time.Now().Add(-queueStaleAfter - time.Second)
	if cmd := m.refreshOnScreen(); cmd == nil {
		t.Error("the queue was not asked for again")
	}
	if queueStaleAfter >= staleAfter {
		t.Errorf("the queue is asked every %s and a list every %s, want the queue oftener",
			queueStaleAfter, staleAfter)
	}
}
