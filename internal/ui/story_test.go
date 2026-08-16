package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/notes"
	"github.com/pottom/spindle/internal/player"
)

// storyModel is the player with something playing and something written about
// it.
func storyModel(note string) Model {
	m := playerModel()
	m.songs = map[string]notes.TrackNote{
		m.ps.TrackID: {Note: note, NoteFrom: "Last.fm", Listeners: 2291235, Plays: 16593050},
	}
	return m
}

// The key is offered where there is something behind it and nowhere else. A box
// that opens on nothing is worse than a key that is not there.
func TestTheStoryKeyIsOfferedOnlyWhereThereIsOne(t *testing.T) {
	quiet := playerModel()
	if quiet.storyAvailable() {
		t.Error("a record nobody has written about offered its story")
	}
	if strings.Contains(ansi.Strip(quiet.render()), " about") {
		t.Error("the help bar offered a key that does nothing")
	}

	told := storyModel("Written by Freddie Mercury for the 1975 album.")
	if !told.storyAvailable() {
		t.Fatal("a record with a paragraph about it offered nothing")
	}
	if !strings.Contains(ansi.Strip(told.render()), "about") {
		t.Error("the help bar did not offer the key")
	}
}

// It opens on the key and shuts on the next thing anybody does, whatever that
// is: it is read rather than worked in, which is the bargain the big screen
// makes too.
func TestTheStoryOpensAndTheNextKeyPutsItAway(t *testing.T) {
	m := storyModel("It was recorded in 1975 and nobody expected it to work.")

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	open := tm.(Model)
	if !open.story {
		t.Fatal("the key did not open it")
	}

	screen := ansi.Strip(open.render())
	if !strings.Contains(screen, "nobody expected it to work") {
		t.Error("the box does not hold what was written")
	}
	if !strings.Contains(screen, m.ps.Title) {
		t.Error("the box does not say which record it is about")
	}

	// Anything at all, and it is spent on nothing else.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	shut := tm.(Model)
	if shut.story {
		t.Error("a key left it up")
	}
	if shut.ps.Shuffle != m.ps.Shuffle {
		t.Error("the key that put it away also did what it usually does")
	}

	// And a press of the pointer, likewise.
	if away, _ := open.mouseClick(clickAt(leftMargin+4, tabBarHeight+3)); away.story {
		t.Error("a press inside it left it up")
	}
}

// A paragraph is longer than a terminal. The box takes what it has room for and
// says that there was more; a box that cannot be drawn is worse than one that
// ends in a mark.
func TestALongStoryIsCutToTheRoom(t *testing.T) {
	long := strings.Repeat("Every word of this is filler and there are many of them. ", 60)
	m := storyModel(long)
	m.story = true

	l := m.layout()
	p, ok := m.openPopup()
	if !ok {
		t.Fatal("nothing is up")
	}
	box := m.menuShape(l, p)
	if box.h > l.bodyHeight {
		t.Errorf("the box is %d rows tall in a body of %d", box.h, l.bodyHeight)
	}

	screen := ansi.Strip(m.render())
	if !strings.Contains(screen, "…") {
		t.Error("a story that did not fit did not say so")
	}
	if !strings.Contains(screen, "Every word of this is filler") {
		t.Error("the box holds none of it")
	}
}

// What is playing is asked after once, and an answer of nothing is an answer.
func TestASongIsAskedAfterOnce(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.notes = &notes.Cached{}
	m.ps = &player.State{TrackID: "t1", Title: "one", Artists: []string{"somebody"}}

	if m.syncSong() == nil {
		t.Fatal("nothing was asked about the record playing")
	}
	if m.syncSong() != nil {
		t.Error("it was asked again while the first answer was still coming")
	}

	m.tookSong(songTook{track: "t1"})
	if m.syncSong() != nil {
		t.Error("a record nobody has written about was asked after twice")
	}
}

// A paragraph is longer than a terminal, so it scrolls — by the same keys as
// everything else that scrolls, and by the wheel. Anything that is not one of
// those still puts it away.
func TestTheStoryScrolls(t *testing.T) {
	long := strings.Repeat("Every word of this is filler and there are a great many of them. ", 40)
	m := storyModel(long)
	m.story = true

	l := m.layout()
	if m.storyLast(l) <= 0 {
		t.Fatal("a story of forty sentences fits on one screen")
	}

	head := ansi.Strip(m.render())

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	down := tm.(Model)
	if down.storyAt == 0 {
		t.Fatal("a page down did not move it")
	}
	if !down.story {
		t.Fatal("a page down put it away")
	}
	if ansi.Strip(down.render()) == head {
		t.Error("it moved and the screen did not")
	}

	// It stops at the end rather than scrolling into empty rows.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	end := tm.(Model)
	if end.storyAt != end.storyLast(l) {
		t.Errorf("the end of it is at %d, want %d", end.storyAt, end.storyLast(l))
	}
	for range 5 {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	if got := tm.(Model).storyAt; got != end.storyLast(l) {
		t.Errorf("it scrolled past the end to %d", got)
	}

	// At the end there is nothing more, and it does not say there is.
	tail := ansi.Strip(end.render())
	if strings.Contains(tail, "…") {
		t.Error("the last screen of it still says there is more")
	}
	if !strings.Contains(tail, "listeners") {
		t.Error("what is under the words never came into view")
	}

	// The wheel reads on too.
	turned, _ := end.mouseWheel(wheelAt(leftMargin+4, tabBarHeight+3, tea.MouseWheelUp))
	if turned.storyAt >= end.storyAt {
		t.Error("the wheel over the box did not move it back")
	}

	// And something that is not a movement is still the way out.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if shut := tm.(Model); shut.story || shut.storyAt != 0 {
		t.Error("a key that means nothing here left it up, or left it scrolled")
	}
}
