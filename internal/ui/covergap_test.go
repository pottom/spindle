package ui

import (
	"testing"
	"time"
)

// A picture that would not come is asked for again.
//
// Two covers on a real library's wall were blank: one was served as WebP, which
// nothing here could read until it could, and the other failed once for a
// moment. Both were then never asked for again — a tile that has failed matches
// what it was asked for, so the wall skipped it for as long as its size stayed
// the same, and only a resize mended the hole.
func TestAPictureThatFailedIsAskedForAgain(t *testing.T) {
	m := wallModel(t, 6)
	m.syncGridCovers()

	id := m.libraryTiles()[0].id
	tile := m.tiles[id]
	tile.failed, tile.failedAt = true, time.Now()
	m.tiles[id] = tile

	// Not at once: a wall of missing covers must not be a wall of requests.
	if cmd := m.syncGridCovers(); cmd != nil {
		t.Error("a picture that failed a moment ago was asked for again straight away")
	}

	// And after a while, it is.
	tile.failedAt = time.Now().Add(-2 * coverRetryAfter)
	m.tiles[id] = tile
	if cmd := m.syncGridCovers(); cmd == nil {
		t.Error("a picture that failed long ago was never asked for again")
	}

	// The slot it holds is its own either way: asking again must not shuffle
	// the wall's pictures about. See freeSlot.
	if m.tiles[id].slot != tile.slot {
		t.Errorf("asking again moved it from slot %d to %d", tile.slot, m.tiles[id].slot)
	}
}
