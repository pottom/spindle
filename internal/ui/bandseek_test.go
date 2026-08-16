package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// bandModel is a playlist opened from the library, with the cursor resting on
// the track that is sounding — which is when the band over the list carries a
// playhead. See trackDetailAt.
func bandModel() Model {
	m := playerModel()
	m.tab = tabLibrary
	m.stack = append(m.stack, openPage{
		kind: openPlaylist, id: "p1", name: "Deep Cuts",
		tracks: []player.Track{
			{ID: m.ps.TrackID, Title: m.ps.Title, Artists: m.ps.Artists, Duration: m.ps.Duration},
			{ID: "t2", Title: "Another", Artists: []string{"Somebody"}, Duration: 3 * time.Minute},
		},
	})
	m.resize()
	return m
}

// The band over a list is a player, and its playhead is a bar like any other.
//
// It was drawn and could not be pressed: the rows over the list answered "the
// list, and no row of it", so the one thing up there worth pointing at was the
// one thing the pointer went through.
func TestTheBandOverAListCanBeSeeked(t *testing.T) {
	m := bandModel()

	x, y := wordAt(t, m, knob)
	at := m.spotAt(x, y)
	if at.kind != spotSeek {
		t.Fatalf("the playhead in the band is at column %d of row %d, and the pointer calls it %v",
			x, y, at.kind)
	}

	// Where it is drawn is where it says it is.
	bar := barCells(m.bandDetailWidth(m.layout()))
	want := atFraction(at.at, bar, m.ps.Duration)
	if got := m.elapsed(); absDuration(got-want) > m.ps.Duration/time.Duration(bar) {
		t.Errorf("the playhead is drawn at %s and pressing it asks for %s", got, want)
	}

	// A press takes hold of it rather than acting at once, as on the player's
	// own bar, so that a drag scrubs.
	held, _ := m.mouseClick(clickAt(x, y))
	if !held.drag.on || held.drag.kind != spotSeek {
		t.Error("pressing the bar in the band did not take hold of it")
	}
	if held.drag.w != bar {
		t.Errorf("the drag thinks the bar is %d cells wide, want %d", held.drag.w, bar)
	}
}

// The rest of the band belongs to the list under it, so a wheel anywhere up
// there still moves the list rather than stopping dead.
func TestTheRestOfTheBandIsStillTheList(t *testing.T) {
	m := bandModel()

	x, y := wordAt(t, m, knob)
	if at := m.spotAt(x, y-1); at.kind != spotList || at.at != -1 {
		t.Errorf("the row over the playhead is %v/%d, want the list and no row", at.kind, at.at)
	}
}
