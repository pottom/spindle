package player

import (
	"time"

	"github.com/zmb3/spotify/v2"
)

// bestImage picks the largest artwork that is still worth downloading. Spotify
// lists images largest first, but says nothing binding about the order, so the
// choice is made on the numbers.
func bestImage(images []spotify.Image) string {
	best := -1
	for i, img := range images {
		w := int(img.Width)
		if w > maxCoverPixels {
			continue
		}
		if best < 0 || w > int(images[best].Width) {
			best = i
		}
	}
	if best >= 0 {
		return images[best].URL
	}

	// Everything on offer is oversized: take the smallest of them rather than
	// showing nothing.
	for i, img := range images {
		if best < 0 || img.Width < images[best].Width {
			best = i
		}
	}
	if best >= 0 {
		return images[best].URL
	}
	return ""
}

func artistNames(artists []spotify.SimpleArtist) []string {
	out := make([]string, 0, len(artists))
	for _, a := range artists {
		out = append(out, a.Name)
	}
	return out
}

// artistIDs is the same for the ids, which is what a screen needs to go to one
// of them: a name would have to be searched for, and a search finds the wrong
// artist as often as the right one.
func artistIDs(artists []spotify.SimpleArtist) []string {
	out := make([]string, 0, len(artists))
	for _, a := range artists {
		out = append(out, a.ID.String())
	}
	return out
}

func trackFromFull(t *spotify.FullTrack) Track {
	out := Track{
		ID:       t.ID.String(),
		Title:    t.Name,
		Artists:  artistNames(t.Artists),
		Album:    t.Album.Name,
		CoverURL: bestImage(t.Album.Images),
		Duration: time.Duration(t.Duration) * time.Millisecond,
	}
	out.AlbumID = t.Album.ID.String()
	out.ArtistIDs = artistIDs(t.Artists)
	out.Released = t.Album.ReleaseDate
	out.AlbumType = t.Album.AlbumType
	out.TrackNumber = int(t.TrackNumber)
	out.DiscNumber = int(t.DiscNumber)
	out.TotalTracks = int(t.Album.TotalTracks)
	out.Explicit = t.Explicit
	return out
}

// trackFromSimple is trackFromFull for the endpoints that answer with the
// slimmer object: an album's own track list, and the listening history.
//
// An album's tracks carry no album of their own — Spotify does not repeat the
// record's name and cover on each of its tracks — so those fields stay empty
// unless the source did send them, as the history does.
func trackFromSimple(t *spotify.SimpleTrack) Track {
	return Track{
		ID:          t.ID.String(),
		Title:       t.Name,
		Artists:     artistNames(t.Artists),
		Album:       t.Album.Name,
		AlbumID:     t.Album.ID.String(),
		ArtistIDs:   artistIDs(t.Artists),
		CoverURL:    bestImage(t.Album.Images),
		Duration:    time.Duration(t.Duration) * time.Millisecond,
		Released:    t.Album.ReleaseDate,
		AlbumType:   t.Album.AlbumType,
		TrackNumber: int(t.TrackNumber),
		DiscNumber:  int(t.DiscNumber),
		TotalTracks: int(t.Album.TotalTracks),
		Explicit:    t.Explicit,
	}
}

func albumFromSimple(a *spotify.SimpleAlbum) Album {
	return Album{
		ID:        a.ID.String(),
		Name:      a.Name,
		Artists:   artistNames(a.Artists),
		CoverURL:  bestImage(a.Images),
		Released:  a.ReleaseDate,
		Tracks:    int(a.TotalTracks),
		AlbumType: a.AlbumType,
	}
}

func artistFromFull(a *spotify.FullArtist) Artist {
	return Artist{
		ID:        a.ID.String(),
		Name:      a.Name,
		ImageURL:  bestImage(a.Images),
		Genres:    a.Genres,
		Followers: int(a.Followers.Count),
	}
}

func ownerName(u spotify.User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.ID
}

// repeatFromSpotify normalises the repeat mode. Spotify uses the same three
// words, but an unknown value should read as "off" rather than propagate.
func repeatFromSpotify(mode string) string {
	switch mode {
	case RepeatContext, RepeatTrack:
		return mode
	default:
		return RepeatOff
	}
}

// playlistFromSimple is a playlist as every list of them reports it: the
// library, and a search.
//
// Spotify does not report a playlist's total duration, and adding it up would
// mean fetching every track, so it is left at zero and the UI omits it.
func playlistFromSimple(p *spotify.SimplePlaylist) Playlist {
	return Playlist{
		ID:          p.ID.String(),
		Name:        p.Name,
		Owner:       ownerName(p.Owner),
		CoverURL:    bestImage(p.Images),
		Tracks:      int(p.Tracks.Total),
		Description: plainText(p.Description),
	}
}
