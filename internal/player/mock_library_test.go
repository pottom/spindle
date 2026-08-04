package player

import (
	"slices"
	"testing"
)

// walkMock reads a mock list the way a screen does — from the top, following
// Page.Next for as long as Page.More — and returns everything it was given.
func walkMock[T any](t *testing.T, read func(offset int) (Page[T], error)) []T {
	t.Helper()

	var all []T
	for offset, guard := 0, 0; ; guard++ {
		if guard > 100 {
			t.Fatal("the walk never reached the end of the list")
		}
		page, err := read(offset)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, page.Items...)
		if !page.More {
			return all
		}
		offset = page.Next
	}
}

// The saved songs are the longest list the mock holds, and the one where a page
// boundary that repeated or skipped an entry would show. Walking it to the end
// has to give every liked track, each of them once, in order.
func TestMockLikedTracksPageToTheEnd(t *testing.T) {
	m, _ := newTestMock()

	if len(mockLikedIDs) <= mockPageLimit {
		t.Fatalf("%d liked tracks fit in one page, so the walk proves nothing", len(mockLikedIDs))
	}

	var walked []string
	for _, track := range walkMock(t, func(offset int) (Page[Track], error) {
		return m.LikedTracks(t.Context(), offset)
	}) {
		walked = append(walked, track.ID)
	}
	if !slices.Equal(walked, mockLikedIDs) {
		t.Errorf("walking the liked songs gave %v, want %v", walked, mockLikedIDs)
	}
}

// Every library list has to end. A list that reports More on its last page
// leaves the screen fetching the same empty page for as long as the user keeps
// scrolling.
func TestMockLibraryListsEnd(t *testing.T) {
	m, _ := newTestMock()

	albums := walkMock(t, func(offset int) (Page[Album], error) {
		return m.SavedAlbums(t.Context(), offset)
	})
	if len(albums) != len(mockAlbumList) {
		t.Errorf("%d saved albums, want the %d the mock holds", len(albums), len(mockAlbumList))
	}

	artists := walkMock(t, func(offset int) (Page[Artist], error) {
		return m.FollowedArtists(t.Context(), offset)
	})
	if len(artists) != len(mockArtists) {
		t.Errorf("%d followed artists, want the %d the mock holds", len(artists), len(mockArtists))
	}
}

// An album is heard in album order. The catalogue holds its tracks in the order
// they were written down, which for the mock is not the same thing.
func TestMockAlbumTracksAreInAlbumOrder(t *testing.T) {
	m, _ := newTestMock()

	tracks := walkMock(t, func(offset int) (Page[Track], error) {
		return m.AlbumTracks(t.Context(), "al1", offset)
	})
	if len(tracks) < 2 {
		t.Fatalf("%d tracks on the album, want enough to have an order", len(tracks))
	}

	for i := 1; i < len(tracks); i++ {
		if tracks[i-1].TrackNumber > tracks[i].TrackNumber {
			t.Errorf("track %d follows track %d", tracks[i].TrackNumber, tracks[i-1].TrackNumber)
		}
	}
	for _, track := range tracks {
		if track.Album != albumOpera {
			t.Errorf("%q is on %q, not on the album asked for", track.Title, track.Album)
		}
	}
}

// An artist's page shows their own records and nobody else's.
func TestMockArtistAlbumsAreTheirs(t *testing.T) {
	m, _ := newTestMock()

	albums := walkMock(t, func(offset int) (Page[Album], error) {
		return m.ArtistAlbums(t.Context(), "ar2", offset)
	})
	if len(albums) == 0 {
		t.Fatal("David Bowie has no albums in the mock")
	}
	for _, album := range albums {
		if !slices.Contains(album.Artists, "David Bowie") {
			t.Errorf("%q is by %v, and was listed under David Bowie", album.Name, album.Artists)
		}
	}
}

// An unknown id is a bug in the caller, not an empty screen. Answering with no
// tracks would look like an album that has none.
func TestMockLibraryRefusesUnknownIDs(t *testing.T) {
	m, _ := newTestMock()

	if _, err := m.AlbumTracks(t.Context(), "nope", 0); err == nil {
		t.Error("AlbumTracks accepted an album that does not exist")
	}
	if _, err := m.ArtistAlbums(t.Context(), "nope", 0); err == nil {
		t.Error("ArtistAlbums accepted an artist that does not exist")
	}
	if _, err := m.ArtistTopTracks(t.Context(), "nope"); err == nil {
		t.Error("ArtistTopTracks accepted an artist that does not exist")
	}
}

// The history repeats where the listening did, and stops at the count asked
// for. Both are what the Web API does, and the mock is where the screen built
// on them is worked out.
func TestMockRecentlyPlayedKeepsRepeatsAndHonoursTheLimit(t *testing.T) {
	m, _ := newTestMock()

	all, err := m.RecentlyPlayed(t.Context(), len(mockRecentIDs)+10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(mockRecentIDs) {
		t.Fatalf("%d tracks, want the %d in the history", len(all), len(mockRecentIDs))
	}

	seen := map[string]int{}
	for _, track := range all {
		seen[track.ID]++
	}
	if !slices.ContainsFunc(mockRecentIDs, func(id string) bool { return seen[id] > 1 }) {
		t.Error("nothing in the history was played twice, so the repeat is never exercised")
	}

	few, err := m.RecentlyPlayed(t.Context(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(few) != 3 {
		t.Errorf("%d tracks for a limit of 3, want 3", len(few))
	}
}

// The mock answers ArtistTopTracks so that --mock exercises the capability
// rather than leaving the artist screen half untried until a daemon is running.
func TestMockAnswersTheTopTracksCapability(t *testing.T) {
	m, _ := newTestMock()

	var backend any = m
	top, ok := backend.(ArtistTopTracks)
	if !ok {
		t.Fatal("the mock does not implement ArtistTopTracks")
	}

	tracks, err := top.ArtistTopTracks(t.Context(), "ar1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) == 0 || len(tracks) > mockTopTracks {
		t.Fatalf("%d top tracks, want between 1 and %d", len(tracks), mockTopTracks)
	}
	for _, track := range tracks {
		if !slices.Contains(track.Artists, "Queen") {
			t.Errorf("%q is by %v, and was listed under Queen", track.Title, track.Artists)
		}
	}
}
