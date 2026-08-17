package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
)

// tracksBy is a queue of one track each by the named artists.
func tracksBy(artists ...string) []player.Track {
	out := make([]player.Track, 0, len(artists))
	for i, who := range artists {
		out = append(out, player.Track{
			ID: string(rune('a' + i)), Title: who + " song", Artists: []string{who},
		})
	}
	return out
}

func ordering(tracks []player.Track) []string {
	out := make([]string, 0, len(tracks))
	for _, t := range tracks {
		out = append(out, t.Artists[0])
	}
	return out
}

// What is coming is put in an order that follows what is playing: the same
// artist first, then the ones Spotify compares them to, then the ones it would
// have played next, then everything else — each group in the order it was
// already in.
//
// Audio features would have been the obvious way to judge this, and Spotify
// closed them to everybody in 2024. What is left is artists.
func TestTheQueueIsOrderedByWhatGoesWithIt(t *testing.T) {
	queue := tracksBy("Stranger", "Queen", "Suggested Band", "Another Stranger", "Near Band", "Queen")
	near := map[string]bool{"near band": true}
	suggested := map[string]bool{"suggested band": true}

	got := ordering(fitOrder(queue, []string{"Queen"}, near, suggested))
	want := []string{"Queen", "Queen", "Near Band", "Suggested Band", "Stranger", "Another Stranger"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the order is %v,\nwant %v", got, want)
		}
	}
}

// Every track that was in the queue is still in it. A list somebody spent a
// minute building must never be damaged by asking a question about it.
func TestOrderingKeepsEveryTrack(t *testing.T) {
	queue := tracksBy("A", "B", "C", "D", "E")
	got := fitOrder(queue, []string{"C"}, map[string]bool{"e": true}, nil)

	if len(got) != len(queue) {
		t.Fatalf("%d tracks went in and %d came out", len(queue), len(got))
	}
	seen := map[string]int{}
	for _, track := range got {
		seen[track.ID]++
	}
	for _, track := range queue {
		if seen[track.ID] != 1 {
			t.Errorf("%q is in the new order %d times, want once", track.Title, seen[track.ID])
		}
	}
}

// The key is offered where it can act and nowhere else: a backend that cannot
// be told the new order, or an application Spotify will not answer, has no key.
func TestTheKeyIsOfferedOnlyWhereItWorks(t *testing.T) {
	m := queueModel(0, "b", "c", "d")
	m.width, m.height = 130, 40
	m.resize()
	if !m.fitAvailable() {
		t.Fatal("a mock that can do all of it was not offered the key")
	}
	if !strings.Contains(ansi.Strip(m.render()), "follow") {
		t.Error("the help bar does not offer it")
	}

	narrow := queueModel(0, "b", "c", "d")
	narrow.allows = player.Allowances{}
	if narrow.fitAvailable() {
		t.Error("an application Spotify will not answer was offered the key")
	}
	if narrow.fitQueue() != nil {
		t.Error("it asked anyway")
	}

	short := queueModel(0, "b")
	if short.fitAvailable() {
		t.Error("a queue of one was offered an order")
	}
}

// And it says what it did, because a list that rearranges itself with nothing
// said is a list that looks like it lost something.
func TestItSaysWhatItDid(t *testing.T) {
	m := queueModel(0, "b", "c", "d")
	m.queue = tracksBy("Stranger", "Queen", "Other")
	m.ps = &player.State{TrackID: "now", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}, Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(fitTook{near: map[string]bool{}, suggested: map[string]bool{}})
	after := tm.(Model)

	if after.queue[0].Artists[0] != "Queen" {
		t.Errorf("the queue was not put in order: %v", ordering(after.queue))
	}
	if !strings.Contains(after.said, "Bohemian Rhapsody") {
		t.Errorf("it said %q, which does not say what it followed", after.said)
	}
}

// One press of a key must not rearrange a list somebody spent a minute
// building. Nothing is lost either way — every track stays — but the order they
// were in is not recoverable once it has gone, which is the test of whether
// something should ask first.
func TestTheKeyAsksBeforeItOrders(t *testing.T) {
	m := queueModel(0, "b", "c", "d")
	m.width, m.height = 130, 40
	m.resize()
	before := ids(m.queue)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	asked := tm.(Model)

	if cmd != nil {
		t.Error("the key went straight out and asked Spotify")
	}
	if !asked.actions.open {
		t.Fatal("nothing was asked")
	}
	if !strings.Contains(ansi.Strip(asked.render()), "Order what is coming?") {
		t.Error("the question is not on the screen")
	}
	if !strings.Contains(ansi.Strip(asked.render()), "Leave it as it is") {
		t.Error("there is no way out of the question")
	}
	if !slices.Equal(ids(asked.queue), before) {
		t.Error("the queue was rearranged before anybody answered")
	}

	// The way out leaves it alone.
	var away tea.Model = asked
	away, _ = away.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := away.(Model); got.actions.open || !slices.Equal(ids(got.queue), before) {
		t.Error("escaping the question did not leave the queue alone")
	}

	// And answering it acts.
	var yes tea.Model = asked
	yes, cmd = yes.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("answering the question did nothing")
	}
	if yes.(Model).actions.open {
		t.Error("the box stayed up after it was answered")
	}
}

// Choosing it from the menu does not ask again: opening a menu and picking a
// line is already two deliberate acts.
func TestTheMenuDoesNotAskTwice(t *testing.T) {
	m := queueModel(0, "b", "c", "d")
	m.queuePane.cursor.cursor = queueRowOf(0)

	var found *verb
	for _, v := range m.actionsFor(m.queue[0]) {
		if strings.Contains(v.label, "Order what is coming") {
			held := v
			found = &held
		}
	}
	if found == nil {
		t.Fatal("the queue's menu does not offer it")
	}
	if cmd := found.do(&m); cmd == nil {
		t.Error("choosing it from the menu asked again instead of acting")
	}
}

// A list that rearranges itself under the eye is a list nobody can check, so
// the rows that came up are marked for a few seconds.
func TestTheRowsThatCameUpAreMarked(t *testing.T) {
	m := queueModel(0, "b", "c", "d")
	m.width, m.height = 130, 40
	m.tab = tabQueue
	m.resize()
	m.queue = tracksBy("Stranger", "Queen", "Other")
	m.ps = &player.State{TrackID: "now", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}, Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(fitTook{near: map[string]bool{}, suggested: map[string]bool{}})
	after := tm.(Model)

	moved := after.queue[0]
	if moved.Artists[0] != "Queen" {
		t.Fatalf("the wrong track came up: %v", ordering(after.queue))
	}
	if !after.justMoved(moved.ID) {
		t.Error("the row that came up is not marked")
	}
	if after.justMoved(after.queue[1].ID) {
		t.Error("a row that only got pushed down is marked as though it moved up")
	}

	// The mark stands where the number would be, and it is drawn as a glyph: a
	// ground a shade off the screen's own is invisible in a terminal that will
	// not say what colour it draws on, which is most of them under tmux.
	row := after.trackRow(moved, 100, false, 1)
	if !strings.Contains(ansi.Strip(row), movedMark) {
		t.Errorf("the row that came up carries no mark: %q", ansi.Strip(row))
	}
	if strings.Contains(ansi.Strip(after.trackRow(after.queue[1], 100, false, 2)), movedMark) {
		t.Error("a row that did not come up is marked")
	}

	// It blinks first: four flashes, and the dark half of each is the row as it
	// always was.
	if !after.blinking() {
		t.Error("the marks went straight to standing still")
	}
	after.fitMovedAt = time.Now().Add(-fitBlink) // the first dark half
	if after.justMoved(moved.ID) {
		t.Error("the mark is on through the dark half of a flash")
	}

	// Then it stays, so the column can be read down at leisure.
	after.fitMovedAt = time.Now().Add(-time.Minute)
	if after.blinking() {
		t.Error("it is still flashing a minute later")
	}
	if !after.justMoved(moved.ID) {
		t.Error("the mark did not stay after the flashing")
	}

	// And it goes when what it describes does.
	after.forgetMoved()
	if after.justMoved(moved.ID) {
		t.Error("the mark survived the arrangement it was about")
	}
	if back := ansi.Strip(after.trackRow(moved, 100, false, 1)); !strings.Contains(back, "1 ") {
		t.Errorf("the number did not come back: %q", back)
	}
}
