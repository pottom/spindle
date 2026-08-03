package ui

import (
	"charm.land/bubbles/v2/textinput"

	"github.com/pottom/spindle/internal/player"
)

// playlistPane is the library tab. It has two levels: the playlists themselves,
// and the tracks inside whichever one is open.
type playlistPane struct {
	items  []player.Playlist
	cursor listState

	open   *player.Playlist // nil while browsing the top level
	tracks []player.Track
	inner  listState
}

// selected returns the playlist under the cursor at the top level.
func (p playlistPane) selected() *player.Playlist {
	if p.cursor.cursor < 0 || p.cursor.cursor >= len(p.items) {
		return nil
	}
	return &p.items[p.cursor.cursor]
}

// cover is the artwork this pane wants shown: the open playlist's, or the one
// under the cursor. The playing track's cover is deliberately not used here —
// it already has a whole tab of its own.
func (p playlistPane) cover() string {
	if p.open != nil {
		return p.open.CoverURL
	}
	if sel := p.selected(); sel != nil {
		return sel.CoverURL
	}
	return ""
}

// searchPane is the search tab: a query and its results.
type searchPane struct {
	input   textinput.Model
	results []player.Track
	cursor  listState

	// seq rises with every query, so a slow search that lands after a newer one
	// can be thrown away.
	seq int
}

func newSearchPane() searchPane {
	in := textinput.New()
	in.Prompt = "⌕ "
	in.Placeholder = "title, artist or album"
	in.Focus()
	return searchPane{input: in}
}

// selected returns the result under the cursor.
func (s searchPane) selected() *player.Track {
	if s.cursor.cursor < 0 || s.cursor.cursor >= len(s.results) {
		return nil
	}
	return &s.results[s.cursor.cursor]
}

func (s searchPane) cover() string {
	if sel := s.selected(); sel != nil {
		return sel.CoverURL
	}
	return ""
}
