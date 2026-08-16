package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A list opened is a list read through. The cursor used to pull the next page
// in, which meant searching a list that was a tenth loaded searched a tenth of
// it and said nothing about the rest.
func TestAListReadsItselfToTheEnd(t *testing.T) {
	m := likedModel(t)
	page := &m.stack

	// Open something with more to come.
	*page = append(*page, openPage{
		kind: openPlaylist, id: "p1", name: "long one",
		pages: paging{more: true, next: 50, loading: true},
	})

	var tm tea.Model = m
	tm, cmd := tm.Update(msg.OpenedFetched{
		ID: "p1", Offset: 50, More: true, Next: 100,
		Tracks: []player.Track{{ID: "t1", Title: "one"}},
	})
	if cmd == nil {
		t.Fatal("a page landed with more to come and nothing was sent for")
	}
	got := tm.(Model)
	if got.open().pages.next != 100 {
		t.Errorf("the next page starts at %d, want where the answer said", got.open().pages.next)
	}
	if !got.open().pages.loading {
		t.Error("nothing is in flight, so the list stopped reading itself")
	}

	// And the last page stops it: nothing more to ask for.
	tm, _ = tm.Update(msg.OpenedFetched{
		ID: "p1", Offset: 100, More: false,
		Tracks: []player.Track{{ID: "t2", Title: "two"}},
	})
	end := tm.(Model)
	if end.open().pages.loading || end.open().pages.more {
		t.Error("the last page left the list still reading")
	}
	if end.readOn() != nil {
		t.Error("a list that has been read through asked for more")
	}
}

// Spotify asking to be left alone stops it. Reading a list through is worth a
// burst of requests and worth none at all against a throttle.
func TestAThrottleStopsTheReading(t *testing.T) {
	m := likedModel(t)
	m.stack = append(m.stack, openPage{
		kind: openPlaylist, id: "p1", pages: paging{more: true, next: 50},
	})

	if m.readOn() == nil {
		t.Fatal("a list with more to come asked for nothing")
	}

	m.rateLimitedUntil = time.Now().Add(time.Minute)
	if m.readOn() != nil {
		t.Error("it went on reading through a throttle")
	}
}

// And a backend that says "there is more" for ever cannot spend a day's quota on
// one list.
func TestReadingOneListHasAnEnd(t *testing.T) {
	m := likedModel(t)
	m.stack = append(m.stack, openPage{
		kind: openPlaylist, id: "p1",
		pages: paging{more: true, next: 50, pages: walkMost},
	})

	if m.readOn() != nil {
		t.Errorf("a list read %d pages deep asked for another", walkMost)
	}
}
