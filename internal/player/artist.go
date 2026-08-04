package player

import "context"

// Artist is one entry of a followed-artists list. Like Album it carries only
// what arrives with the list itself, so drawing one costs nothing extra.
type Artist struct {
	ID   string
	Name string

	// ImageURL is a photograph of the artist rather than cover art, which is why
	// it is not called CoverURL: an artist row that borrowed an album's sleeve
	// would be showing something that is not the artist.
	ImageURL string

	// Genres and Followers are the whole of what Spotify says about an artist
	// beyond the name. They arrive with the followed list, so a detail panel has
	// them without asking again.
	Genres    []string
	Followers int
}

// ArtistTopTracks is implemented by backends that can say what an artist is
// most listened to for.
//
// It is kept out of Player because only a local device can answer it. Measured
// against a live account, the Web API refuses /artists/{id}/top-tracks to this
// client id outright; the daemon resolves an artist the way pressing play on
// one does, and what comes back is that list. A backend that cannot ask has no
// way to pretend, which is the point of leaving it here rather than widening
// the interface.
type ArtistTopTracks interface {
	// ArtistTopTracks is the artist's best-known recordings, most listened to
	// first. It is a single list rather than a page: what is asked for is the
	// top of it, and there is no rest to scroll to.
	ArtistTopTracks(ctx context.Context, artistID string) ([]Track, error)
}
