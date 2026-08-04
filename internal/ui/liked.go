package ui

import (
	"github.com/pottom/spindle/internal/player"
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
// The count is only filled in once the whole list has been read, and the cover
// is the most recently saved track's. Spotify draws a collage for this list and
// serves it to nobody; one of the covers it is made of says more than an empty
// box, and it changes when the list does, which is the truth about it.
func likedPlaylist(tracks []player.Track, all bool) player.Playlist {
	pl := player.Playlist{
		ID:    likedID,
		Name:  "Liked Songs",
		Owner: "saved by you",
	}
	if len(tracks) > 0 {
		pl.CoverURL = tracks[0].CoverURL
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
	return likedPlaylist(m.playlists.liked, m.playlists.likedAll)
}

// refreshLikedRow puts a freshly read first page onto the row already on
// screen. The library and the saved tracks are two requests and either can
// answer first, so the row is built from whichever arrives and mended by the
// other.
func (m *Model) refreshLikedRow() {
	for i, pl := range m.playlists.items {
		if isLiked(pl.ID) {
			m.playlists.items[i] = m.likedRow()
			return
		}
	}
}
