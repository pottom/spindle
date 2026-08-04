package player

import (
	"net/http"
	"testing"
)

// An album read through the daemon comes back named — cover, artists and length
// on every track — where the Web API's album listing sends the track alone. The
// daemon is therefore asked first, exactly as it is for a playlist.
func TestAlbumTracksAskTheDaemonFirst(t *testing.T) {
	l, daemon, web := newDaemonStub(t, http.StatusOK, daemonContext(3))

	page, err := l.AlbumTracks(t.Context(), "al1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 3 {
		t.Errorf("%d tracks, want the 3 the daemon named", len(page.Items))
	}
	if got := daemon.Get("uri"); got != "spotify:album:al1" {
		t.Errorf("uri = %q, want the album's", got)
	}
	if *web != nil {
		t.Error("the Web API was asked as well, and it knows less about the album")
	}
}

// Without a daemon there is only the Web API, which answers this one. The offset
// has to survive the fall-through, or a request for the third page of a long
// album is answered with the first.
func TestAlbumTracksFallBackToTheWebApi(t *testing.T) {
	l, _, web := newDaemonStub(t, http.StatusInternalServerError, "")

	if _, err := l.AlbumTracks(t.Context(), "al1", 100); err != nil {
		t.Fatal(err)
	}
	if web.Get("offset") != "100" {
		t.Errorf("the Web API was asked for offset %q, want 100", web.Get("offset"))
	}
}

// An artist's top tracks are the daemon's to answer or nobody's: the Web API
// refuses them to this client id. Falling back to it would turn a plain failure
// into a 403 with a misleading message on it.
func TestArtistTopTracksHaveNoWebFallback(t *testing.T) {
	l, daemon, _ := newDaemonStub(t, http.StatusOK, daemonContext(4))

	tracks, err := l.ArtistTopTracks(t.Context(), "ar1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 4 {
		t.Errorf("%d tracks, want the 4 the daemon named", len(tracks))
	}
	if got := daemon.Get("uri"); got != "spotify:artist:ar1" {
		t.Errorf("uri = %q, want the artist's", got)
	}

	broken, _, web := newDaemonStub(t, http.StatusInternalServerError, "")
	if _, err := broken.ArtistTopTracks(t.Context(), "ar1"); err == nil {
		t.Error("a daemon that cannot answer must fail rather than reach for the Web API")
	}
	if *web != nil {
		t.Error("the Web API was asked for top tracks, which it refuses")
	}
}

// Everything else in the library belongs to the account rather than to the
// device, and the daemon knows none of it. Local has to pass those straight
// through, offset and all.
func TestLibraryListsGoStraightToTheWebApi(t *testing.T) {
	cases := []struct {
		name string
		call func(*Local) error
	}{
		{"liked tracks", func(l *Local) error { _, err := l.LikedTracks(t.Context(), 50); return err }},
		{"saved albums", func(l *Local) error { _, err := l.SavedAlbums(t.Context(), 50); return err }},
		{"artist albums", func(l *Local) error { _, err := l.ArtistAlbums(t.Context(), "ar1", 50); return err }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, daemon, web := newDaemonStub(t, http.StatusOK, daemonContext(0))
			if err := c.call(l); err != nil {
				t.Fatal(err)
			}
			if *daemon != nil {
				t.Error("the daemon was asked about the library, which it knows nothing about")
			}
			if web.Get("offset") != "50" {
				t.Errorf("the Web API was asked for offset %q, want 50", web.Get("offset"))
			}
		})
	}
}

// The two lists that are not paged by offset — the followed artists, which
// Spotify walks by cursor, and the history, which it walks by timestamp — still
// belong to the account and still go out through the Web API.
func TestTheCursoredListsAlsoGoToTheWebApi(t *testing.T) {
	cases := []struct {
		name string
		call func(*Local) error
	}{
		{"followed artists", func(l *Local) error { _, err := l.FollowedArtists(t.Context(), 0); return err }},
		{"recently played", func(l *Local) error { _, err := l.RecentlyPlayed(t.Context(), 20); return err }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, daemon, web := newDaemonStub(t, http.StatusOK, daemonContext(0))
			if err := c.call(l); err != nil {
				t.Fatal(err)
			}
			if *daemon != nil {
				t.Error("the daemon was asked, and it keeps no library of its own")
			}
			if *web == nil {
				t.Error("nobody was asked at all")
			}
		})
	}
}
