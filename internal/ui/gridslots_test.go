package ui

import (
	"fmt"
	"testing"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// wallModel is a library wall with n playlists on it, each with a picture of
// its own.
func wallModel(t *testing.T, n int) Model {
	t.Helper()

	m := New(player.NewMock(), cover.NewLoader(nil, nil), defaultTestCell)
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	for i := range n {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID:       fmt.Sprintf("p%d", i),
			Name:     fmt.Sprintf("List %d", i),
			Owner:    "pottom",
			CoverURL: fmt.Sprintf("https://i.scdn.co/image/%d", i),
		})
	}
	return m
}

// held reports the slot each tile is drawing in, and complains where two tiles
// share one: a slot is one picture to the terminal, so two tiles holding the
// same one show the same sleeve.
func slotsAreDistinct(t *testing.T, m Model, when string) {
	t.Helper()

	at := map[int]string{}
	for id, tile := range m.tiles {
		if tile.slot == 0 {
			t.Errorf("%s: %s was never given a slot to draw in", when, id)
			continue
		}
		if other, taken := at[tile.slot]; taken {
			t.Errorf("%s: %s and %s both draw in slot %d — one sleeve, two names",
				when, other, id, tile.slot)
		}
		at[tile.slot] = id
	}
}

// The wall shifts under its own tiles, and every cover drew the one before it.
//
// The row of saved tracks that heads the playlists is built from a request of
// its own and arrives after them, moving every playlist along one. A tile that
// already had its picture kept the slot it was given before the shift, while its
// neighbour uploaded into that same slot — and on a real account seven of
// twenty-one tiles showed a neighbour's sleeve.
func TestAWallThatShiftsKeepsItsPicturesApart(t *testing.T) {
	m := wallModel(t, 20)
	m.syncGridCovers()
	slotsAreDistinct(t, m, "before the shift")

	before := map[string]int{}
	for id, tile := range m.tiles {
		before[id] = tile.slot
	}

	// The saved tracks arrive and take the head of the list, moving everything
	// along one.
	m.library.playlists = append([]player.Playlist{{
		ID: "liked", Name: "Liked Songs", CoverURL: "https://i.scdn.co/image/liked",
	}}, m.library.playlists...)
	m.syncGridCovers()

	slotsAreDistinct(t, m, "after the shift")

	// And a tile that only moved keeps the slot it had: re-sending a picture
	// already on the terminal is what scrolling must not cost.
	for id, was := range before {
		if now, still := m.tiles[id]; still && now.slot != was {
			t.Errorf("%s only moved along one and was re-sent, from slot %d to %d", id, was, now.slot)
		}
	}
}
