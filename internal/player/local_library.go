package player

import "context"

// The library lives in the account, not on the device: the daemon plays what it
// is told and knows nothing about what has been saved, followed or listened to.
// All of it comes from the Web API, which answers these for this client id.

func (l *Local) LikedTracks(ctx context.Context, offset int) (Page[Track], error) {
	return l.web.LikedTracks(ctx, offset)
}

func (l *Local) SavedAlbums(ctx context.Context, offset int) (Page[Album], error) {
	return l.web.SavedAlbums(ctx, offset)
}

func (l *Local) FollowedArtists(ctx context.Context, offset int) (Page[Artist], error) {
	return l.web.FollowedArtists(ctx, offset)
}

func (l *Local) ArtistAlbums(ctx context.Context, artistID string, offset int) (Page[Album], error) {
	return l.web.ArtistAlbums(ctx, artistID, offset)
}

func (l *Local) RecentlyPlayed(ctx context.Context, limit int) ([]Track, error) {
	return l.web.RecentlyPlayed(ctx, limit)
}

// AlbumTracks asks the daemon what an album holds, for the reason
// PlaylistTracksPage does: it resolves the context the way playing the album
// would, and what comes back is named — title, artists, cover and length — where
// the Web API's own album listing repeats none of the album on its tracks. The
// Web API is still the fallback, since without a daemon it is all there is.
func (l *Local) AlbumTracks(ctx context.Context, albumID string, offset int) (Page[Track], error) {
	page, err := l.contextTracks(ctx, "spotify:album:"+albumID, offset)
	if err == nil {
		return page, nil
	}
	return l.web.AlbumTracks(ctx, albumID, offset)
}

// ArtistTopTracks implements the capability of the same name. There is no
// fallback: the Web API refuses /artists/{id}/top-tracks to this client id, so
// when the daemon cannot answer, nobody can.
func (l *Local) ArtistTopTracks(ctx context.Context, artistID string) ([]Track, error) {
	page, err := l.contextTracks(ctx, "spotify:artist:"+artistID, 0)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}
