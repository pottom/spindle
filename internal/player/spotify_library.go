package player

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify/v2"
)

// LikedTracks reads the saved songs, most recently saved first.
func (s *Spotify) LikedTracks(ctx context.Context, offset int) (Page[Track], error) {
	start := max(offset, 0)
	page, err := s.client.CurrentUsersTracks(ctx, spotify.Limit(pageLimit), spotify.Offset(start))
	if err != nil {
		return Page[Track]{}, fmt.Errorf("fetch liked tracks: %w", err)
	}
	if page == nil {
		return Page[Track]{}, nil
	}

	out := make([]Track, 0, len(page.Tracks))
	for i := range page.Tracks {
		out = append(out, trackFromFull(&page.Tracks[i].FullTrack))
	}
	return Page[Track]{Items: out, More: page.Next != "", Next: start + pageLimit}, nil
}

// AlbumTracks reads one album's own track list, in album order.
func (s *Spotify) AlbumTracks(ctx context.Context, albumID string, offset int) (Page[Track], error) {
	start := max(offset, 0)
	page, err := s.client.GetAlbumTracks(ctx, spotify.ID(albumID),
		spotify.Limit(pageLimit), spotify.Offset(start))
	if err != nil {
		return Page[Track]{}, fmt.Errorf("fetch album tracks: %w", err)
	}
	if page == nil {
		return Page[Track]{}, nil
	}

	out := make([]Track, 0, len(page.Tracks))
	for i := range page.Tracks {
		out = append(out, trackFromSimple(&page.Tracks[i]))
	}
	return Page[Track]{Items: out, More: page.Next != "", Next: start + pageLimit}, nil
}

// artistGroups is what counts as an artist's own discography: the records they
// released, and not the hundreds of compilations they appear on one track of.
// Left unsaid, Spotify includes the latter, and a well-covered artist's page
// becomes a list of other people's albums.
var artistGroups = []spotify.AlbumType{
	spotify.AlbumTypeAlbum,
	spotify.AlbumTypeSingle,
	spotify.AlbumTypeCompilation,
}

// ArtistAlbums reads what an artist has released.
func (s *Spotify) ArtistAlbums(ctx context.Context, artistID string, offset int) (Page[Album], error) {
	start := max(offset, 0)
	page, err := s.client.GetArtistAlbums(ctx, spotify.ID(artistID), artistGroups,
		spotify.Limit(pageLimit), spotify.Offset(start))
	if err != nil {
		return Page[Album]{}, fmt.Errorf("fetch artist albums: %w", err)
	}
	if page == nil {
		return Page[Album]{}, nil
	}

	out := make([]Album, 0, len(page.Albums))
	for i := range page.Albums {
		out = append(out, albumFromSimple(&page.Albums[i]))
	}
	return Page[Album]{Items: out, More: page.Next != "", Next: start + pageLimit}, nil
}

// SavedAlbums reads the albums in the user's own library.
func (s *Spotify) SavedAlbums(ctx context.Context, offset int) (Page[Album], error) {
	start := max(offset, 0)
	page, err := s.client.CurrentUsersAlbums(ctx, spotify.Limit(pageLimit), spotify.Offset(start))
	if err != nil {
		return Page[Album]{}, fmt.Errorf("fetch saved albums: %w", err)
	}
	if page == nil {
		return Page[Album]{}, nil
	}

	out := make([]Album, 0, len(page.Albums))
	for i := range page.Albums {
		out = append(out, albumFromSimple(&page.Albums[i].SimpleAlbum))
	}
	return Page[Album]{Items: out, More: page.Next != "", Next: start + pageLimit}, nil
}

// FollowedArtists reads the artists the user follows.
//
// This is the one library list Spotify does not page by offset: /v1/me/following
// takes `after`, the id of the last artist already handed over, and an offset
// sent with it is ignored. The offset stays in the signature all the same —
// making the UI hold a cursor for this one list and an offset for the other five
// would be carrying the Web API's shape into a screen that has no use for it —
// so the cursor that reaches an offset is remembered as the pages are read.
// Reading a list front to back, which is the only way a list is read, therefore
// costs one request per page; only a jump into a stretch never read before has
// to walk there from the start.
func (s *Spotify) FollowedArtists(ctx context.Context, offset int) (Page[Artist], error) {
	start := max(offset, 0)
	at, after := s.followedFrom(start)

	for {
		opts := []spotify.RequestOption{spotify.Limit(pageLimit)}
		if after != "" {
			opts = append(opts, spotify.After(after))
		}
		page, err := s.client.CurrentUsersFollowedArtists(ctx, opts...)
		if err != nil {
			return Page[Artist]{}, fmt.Errorf("fetch followed artists: %w", err)
		}
		if page == nil {
			return Page[Artist]{}, nil
		}

		next := at + len(page.Artists)
		s.rememberFollowed(next, page.Cursor.After)

		// Still short of what was asked for: keep walking, as long as there is
		// somewhere to walk to.
		if next <= start && page.Cursor.After != "" && len(page.Artists) > 0 {
			at, after = next, page.Cursor.After
			continue
		}

		// The walk can only land on a page boundary, so a page it lands on may
		// begin before the offset asked for. Drop that part rather than hand
		// back artists the caller has already seen.
		items := page.Artists
		if start > at {
			items = items[min(start-at, len(items)):]
		}

		out := make([]Artist, 0, len(items))
		for i := range items {
			out = append(out, artistFromFull(&items[i]))
		}
		// Next counts the whole page, including anything trimmed off its front:
		// it is the offset the next request starts at, not the number returned.
		return Page[Artist]{Items: out, More: page.Next != "", Next: next}, nil
	}
}

// followedFrom is the furthest point in the followed list that has been read
// before and does not lie past offset, with the cursor that reaches it.
func (s *Spotify) followedFrom(offset int) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	best, after := 0, ""
	for at, cursor := range s.followed {
		if at <= offset && at > best {
			best, after = at, cursor
		}
	}
	return best, after
}

func (s *Spotify) rememberFollowed(at int, cursor string) {
	if cursor == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.followed == nil {
		s.followed = make(map[int]string)
	}
	s.followed[at] = cursor
}

// RecentlyPlayed is what was listened to last, most recent first.
//
// Spotify keeps around fifty entries and walks them by timestamp rather than by
// offset, so this asks for a number of them and there is no page to follow. The
// repeats are left in: a track played three times this morning was played three
// times, and that is the whole of what a history has to say.
func (s *Spotify) RecentlyPlayed(ctx context.Context, limit int) ([]Track, error) {
	items, err := s.client.PlayerRecentlyPlayedOpt(ctx, &spotify.RecentlyPlayedOptions{
		Limit: spotify.Numeric(min(max(limit, 1), pageLimit)),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch recently played: %w", err)
	}

	out := make([]Track, 0, len(items))
	for i := range items {
		out = append(out, trackFromSimple(&items[i].Track))
	}
	return out, nil
}
