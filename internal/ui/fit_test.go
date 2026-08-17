package ui

import (
	"strings"
	"testing"

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
