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

func trackFromFull(t *spotify.FullTrack) Track {
	out := Track{
		ID:       t.ID.String(),
		Title:    t.Name,
		Artists:  artistNames(t.Artists),
		Album:    t.Album.Name,
		CoverURL: bestImage(t.Album.Images),
		Duration: time.Duration(t.Duration) * time.Millisecond,
	}
	if t.LinkedFrom != nil {
		out.OriginID = t.LinkedFrom.ID.String()
	}
	return out
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
