package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// likedID is the name the library gives the account's saved tracks.
//
// Liked songs is a list like any other to read and unlike any other to name:
// Spotify keeps it per account and gives it no id and no uri a third party can
// use. So it is carried as a playlist with an id of our own, and the two places
// that cannot pretend — where its tracks are read, and how it is played — ask
// whether this is that id. It is not a valid Spotify id, so nothing can collide
// with it.
const likedID = "spindle:liked"

// isLiked reports whether an id names the saved tracks rather than a playlist.
func isLiked(id string) bool { return id == likedID }

// likedPlaylist is the row at the top of the library.
//
// It is first because it is the list nearly everyone opens first, and because
// it is the only one that is not a playlist: a row that belongs to a different
// kind of thing reads as a heading where it sits above the rest.
//
// Its cover is drawn rather than fetched. Spotify makes a tile for this list in
// its own clients and gives it to nobody, and the alternative — showing the
// newest saved track's sleeve — says the collection is that record and changes
// every time a song is saved.
//
// The count is only filled in once the whole list has been read.
func likedPlaylist(tracks []player.Track, all bool) player.Playlist {
	pl := player.Playlist{
		ID:       likedID,
		Name:     "Liked Songs",
		Owner:    "saved by you",
		CoverURL: cover.LikedURL,
	}
	if all {
		pl.Tracks = len(tracks)
		for _, t := range tracks {
			pl.Duration += t.Duration
		}
	}
	return pl
}

// likedRow is the saved tracks as the library shows them, from whatever has
// been read of them so far.
func (m Model) likedRow() player.Playlist {
	return likedPlaylist(m.library.liked, m.library.likedAll)
}

// readSaved asks for the first page of the saved tracks, once, for the sake of
// the hearts every track list draws.
//
// The library reads the same page when it opens, for the row at the top of it.
// This is the other way in: somebody who never opens the library would otherwise
// be shown a column that is blank because nothing was ever read, which is
// indistinguishable from a list of nothing saved.
//
// It was asked for only where the glance was open, which is where the hearts
// were. They are a column of the table now, on the queue and in a playlist and
// wherever else tracks are listed, so the page is read whether the glance is or
// not.
func (m Model) readSaved() tea.Cmd {
	if m.library.likedIDs != nil {
		return nil
	}
	return fetchOpenCmd(m.player, openPlaylist, likedID, 0)
}

// refreshLikedRow puts a freshly read first page onto the row already on
// screen. The library and the saved tracks are two requests and either can
// answer first, so the row is built from whichever arrives and mended by the
// other.
func (m *Model) refreshLikedRow() {
	for i, pl := range m.library.playlists {
		if isLiked(pl.ID) {
			m.library.playlists[i] = m.likedRow()
			return
		}
	}
}
