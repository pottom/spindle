package player

import (
	"context"
	"fmt"
	"slices"
)

// The library screens, answered from the same catalogue everything else in the
// mock is drawn from, and paged the way the real backends page — so --mock and
// the tests walk the offsets a live account would.

func (m *Mock) LikedTracks(ctx context.Context, offset int) (Page[Track], error) {
	if err := m.delay(ctx); err != nil {
		return Page[Track]{}, err
	}
	return mockPage(tracksByID(mockLikedIDs), offset), nil
}

func (m *Mock) SavedAlbums(ctx context.Context, offset int) (Page[Album], error) {
	if err := m.delay(ctx); err != nil {
		return Page[Album]{}, err
	}
	return mockPage(slices.Clone(mockAlbumList), offset), nil
}

func (m *Mock) FollowedArtists(ctx context.Context, offset int) (Page[Artist], error) {
	if err := m.delay(ctx); err != nil {
		return Page[Artist]{}, err
	}
	return mockPage(slices.Clone(mockArtists), offset), nil
}

// AlbumTracks lists an album in album order, which is the order a record is
// heard in and not the order the catalogue happens to hold it.
func (m *Mock) AlbumTracks(ctx context.Context, albumID string, offset int) (Page[Track], error) {
	if err := m.delay(ctx); err != nil {
		return Page[Track]{}, err
	}

	album := mockAlbumByID(albumID)
	if album == nil {
		return Page[Track]{}, fmt.Errorf("album tracks: unknown album %q", albumID)
	}

	return mockPage(mockAlbumTracks(album.Name), offset), nil
}

// mockAlbumTracks is a record's own track list, in the order it plays.
//
// Its own function because two things ask: the list, and whatever has to say
// where in the list a track sits. Answered twice, from the same catalogue, they
// came out in different orders — the list is sorted by track number and the
// other walked the catalogue as written — and a play that named a track landed
// on a different one.
func mockAlbumTracks(name string) []Track {
	var out []Track
	for _, t := range mockCatalogue {
		if t.Album == name {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b Track) int { return a.TrackNumber - b.TrackNumber })
	return out
}

func (m *Mock) ArtistAlbums(ctx context.Context, artistID string, offset int) (Page[Album], error) {
	if err := m.delay(ctx); err != nil {
		return Page[Album]{}, err
	}

	artist := mockArtistByID(artistID)
	if artist == nil {
		return Page[Album]{}, fmt.Errorf("artist albums: unknown artist %q", artistID)
	}

	var out []Album
	for _, a := range mockAlbumList {
		if slices.Contains(a.Artists, artist.Name) {
			out = append(out, a)
		}
	}
	return mockPage(out, offset), nil
}

// RecentlyPlayed answers from a fixed history rather than from what this session
// happened to play: a mock that has just started has heard nothing, and a screen
// cannot be looked at empty.
func (m *Mock) RecentlyPlayed(ctx context.Context, limit int) ([]Track, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}

	// tracksByID cannot serve here: the history holds the same track twice, and
	// both are meant to survive.
	out := make([]Track, 0, len(mockRecentIDs))
	for _, id := range mockRecentIDs {
		if t, ok := findMockTrack(id); ok {
			out = append(out, t)
		}
	}
	// Clamped the way the Web API clamps it, so a count that is out of range
	// gives the same answer here as it does live.
	return out[:min(max(limit, 1), len(out))], nil
}

// ArtistTopTracks implements the capability, so that --mock exercises the same
// path a daemon-backed run does rather than leaving the artist screen's best
// half untried until it meets a real device. The mock counts no plays, so what
// comes back is the artist's catalogue tracks in catalogue order.
func (m *Mock) ArtistTopTracks(ctx context.Context, artistID string) ([]Track, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}

	artist := mockArtistByID(artistID)
	if artist == nil {
		return nil, fmt.Errorf("artist top tracks: unknown artist %q", artistID)
	}

	var out []Track
	for _, t := range mockCatalogue {
		if slices.Contains(t.Artists, artist.Name) {
			out = append(out, t)
		}
	}
	return out[:min(mockTopTracks, len(out))], nil
}

// mockTopTracks is how many a top-tracks list holds. Spotify answers ten; the
// mock's catalogue is short enough that ten would be most of an artist's work.
const mockTopTracks = 5

func mockAlbumByID(id string) *Album {
	for i := range mockAlbumList {
		if mockAlbumList[i].ID == id {
			return &mockAlbumList[i]
		}
	}
	return nil
}

func mockArtistByID(id string) *Artist {
	for i := range mockArtists {
		if mockArtists[i].ID == id {
			return &mockArtists[i]
		}
	}
	return nil
}
