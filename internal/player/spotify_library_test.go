package player

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The library lists are as long as the account is old — a saved-songs list runs
// to thousands — so every one of them has to carry the offset it was asked for
// and the page size settled on. Asking for the first page and stopping is how a
// library of two thousand songs became one of fifty.
func TestLibraryPagingSendsOffsetAndLimit(t *testing.T) {
	cases := []struct {
		name string
		body string
		call func(*Spotify) error
	}{
		{"liked tracks", `{"items":[]}`, func(s *Spotify) error {
			_, err := s.LikedTracks(t.Context(), 150)
			return err
		}},
		{"saved albums", `{"items":[]}`, func(s *Spotify) error {
			_, err := s.SavedAlbums(t.Context(), 150)
			return err
		}},
		{"album tracks", `{"items":[]}`, func(s *Spotify) error {
			_, err := s.AlbumTracks(t.Context(), "al1", 150)
			return err
		}},
		{"artist albums", `{"items":[]}`, func(s *Spotify) error {
			_, err := s.ArtistAlbums(t.Context(), "ar1", 150)
			return err
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, got := newQueryStub(t, c.body)
			if err := c.call(s); err != nil {
				t.Fatal(err)
			}
			if got.Get("offset") != "150" {
				t.Errorf("offset = %q, want 150", got.Get("offset"))
			}
			if want := strconv.Itoa(pageLimit); got.Get("limit") != want {
				t.Errorf("limit = %q, want %q", got.Get("limit"), want)
			}
		})
	}
}

// A saved track is wrapped in an envelope carrying the date it was saved; the
// track itself is a field inside it. Reading the envelope as the track gives a
// list of blank rows that all look like a failure somewhere else.
func TestLikedTracksReadTheTrackInsideTheEnvelope(t *testing.T) {
	s, _ := newQueryStub(t, `{"items":[
	  {"added_at":"2024-01-02T03:04:05Z","track":{"id":"t1","name":"Heroes","duration_ms":361000,
	    "artists":[{"name":"David Bowie"}],
	    "album":{"name":"best of bowie","release_date":"2002-10-14","images":[{"url":"https://img/300","width":300,"height":300}]}}}
	]}`)

	page, err := s.LikedTracks(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%d tracks, want 1", len(page.Items))
	}

	track := page.Items[0]
	checks := []struct {
		name      string
		got, want any
	}{
		{"id", track.ID, "t1"},
		{"title", track.Title, "Heroes"},
		{"album", track.Album, "best of bowie"},
		{"cover", track.CoverURL, "https://img/300"},
		{"released", track.Released, "2002-10-14"},
		{"duration", track.Duration, 361 * time.Second},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// An album does not repeat its own name and cover on each of its tracks, so a
// row from this list has the track's own detail and nothing about the record.
// The screen that asked for one album is the one that has those.
func TestAlbumTracksCarryTheTrackAndNotTheAlbum(t *testing.T) {
	s, _ := newQueryStub(t, `{"items":[
	  {"id":"t1","name":"Death on Two Legs","duration_ms":223000,"track_number":1,"disc_number":1,
	   "artists":[{"name":"Queen"}]}
	]}`)

	page, err := s.AlbumTracks(t.Context(), "al1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%d tracks, want 1", len(page.Items))
	}

	track := page.Items[0]
	if track.Title != "Death on Two Legs" || track.TrackNumber != 1 || track.Duration != 223*time.Second {
		t.Errorf("track = %+v, want it named, numbered and timed", track)
	}
	if track.Album != "" || track.CoverURL != "" {
		t.Errorf("album = %q, cover = %q — want both empty, the album list does not send them", track.Album, track.CoverURL)
	}
}

// An artist page is the artist's own records. Left to itself Spotify also sends
// every compilation they appear on one track of, which for anyone well covered
// is most of the list and none of it theirs.
func TestArtistAlbumsAsksForTheArtistsOwnRecords(t *testing.T) {
	s, got := newQueryStub(t, `{"items":[]}`)

	if _, err := s.ArtistAlbums(t.Context(), "ar1", 0); err != nil {
		t.Fatal(err)
	}

	groups := got.Get("include_groups")
	for _, want := range []string{"album", "single", "compilation"} {
		if !strings.Contains(groups, want) {
			t.Errorf("include_groups = %q, want it to hold %q", groups, want)
		}
	}
	if strings.Contains(groups, "appears_on") {
		t.Errorf("include_groups = %q, want the appearances left out", groups)
	}
}

func TestSavedAlbumsReadTheAlbumInsideTheEnvelope(t *testing.T) {
	s, _ := newQueryStub(t, `{"items":[
	  {"added_at":"2024-01-02T03:04:05Z","album":{"id":"al1","name":"A Night at the Opera",
	    "album_type":"album","release_date":"1975-11-21","total_tracks":12,
	    "artists":[{"name":"Queen"}],
	    "images":[{"url":"https://img/640","width":640,"height":640}]}}
	]}`)

	page, err := s.SavedAlbums(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%d albums, want 1", len(page.Items))
	}

	album := page.Items[0]
	checks := []struct {
		name      string
		got, want any
	}{
		{"id", album.ID, "al1"},
		{"name", album.Name, "A Night at the Opera"},
		{"cover", album.CoverURL, "https://img/640"},
		{"released", album.Released, "1975-11-21"},
		{"tracks", album.Tracks, 12},
		{"type", album.AlbumType, "album"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if len(album.Artists) != 1 || album.Artists[0] != "Queen" {
		t.Errorf("artists = %v, want [Queen]", album.Artists)
	}
}

// followedStub answers /me/following with pages of n artists, each page handing
// out the cursor for the one after it, and keeps every query it was asked with.
func followedStub(t *testing.T, pages int) (*Spotify, *[]url.Values) {
	t.Helper()
	var asked []url.Values

	s := newStub(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		asked = append(asked, query)

		// Which page this is, worked out from the cursor asked for.
		at := 0
		if after := query.Get("after"); after != "" {
			at, _ = strconv.Atoi(strings.TrimPrefix(after, "ar"))
			at++
		}

		items := make([]string, 0, pageLimit)
		for i := at; i < min(at+pageLimit, pages*pageLimit); i++ {
			items = append(items, fmt.Sprintf(`{"id":"ar%d","name":"artist %d","followers":{"total":%d}}`, i, i, i))
		}

		last := at+pageLimit >= pages*pageLimit
		next := `"https://api.spotify.com/v1/me/following?after=x"`
		if last {
			next = "null"
		}
		fmt.Fprintf(w, `{"artists":{"items":[%s],"next":%s,"cursors":{"after":"ar%d"}}}`,
			strings.Join(items, ","), next, at+len(items)-1)
	})
	return s, &asked
}

// Followed artists is the one library list Spotify pages by cursor. Sending it
// an offset does nothing at all — the answer is the top of the list again — so
// reading page two means holding on to what page one ended with.
func TestFollowedArtistsWalkTheCursor(t *testing.T) {
	s, asked := followedStub(t, 3)

	first, err := s.FollowedArtists(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != pageLimit || !first.More || first.Next != pageLimit {
		t.Fatalf("first page = %d artists, More %v, Next %d", len(first.Items), first.More, first.Next)
	}
	if got := (*asked)[0].Get("after"); got != "" {
		t.Errorf("the first request carried after=%q, want none", got)
	}

	second, err := s.FollowedArtists(t.Context(), first.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(*asked) != 2 {
		t.Fatalf("%d requests for two pages, want 2 — the cursor from the first was not kept", len(*asked))
	}
	if got, want := (*asked)[1].Get("after"), fmt.Sprintf("ar%d", pageLimit-1); got != want {
		t.Errorf("after = %q, want %q", got, want)
	}
	if second.Items[0].ID != fmt.Sprintf("ar%d", pageLimit) {
		t.Errorf("the second page starts at %s, want the artist after the first page", second.Items[0].ID)
	}
}

// A caller that jumps into a stretch of the list it has never read cannot be
// answered with the top of the list. The cursors have to be walked until the
// offset asked for is reached, and the part of the landing page that lies
// before it dropped.
func TestFollowedArtistsWalkToAnOffsetNeverRead(t *testing.T) {
	s, asked := followedStub(t, 3)

	want := pageLimit + 10
	page, err := s.FollowedArtists(t.Context(), want)
	if err != nil {
		t.Fatal(err)
	}
	if len(*asked) != 2 {
		t.Fatalf("%d requests to reach offset %d, want 2", len(*asked), want)
	}
	if page.Items[0].ID != fmt.Sprintf("ar%d", want) {
		t.Errorf("the page starts at %s, want ar%d — the artists before the offset were handed back", page.Items[0].ID, want)
	}
	if got := len(page.Items); got != pageLimit-10 {
		t.Errorf("%d artists, want the %d left of the page the walk landed on", got, pageLimit-10)
	}
	if page.Next != 2*pageLimit {
		t.Errorf("Next = %d, want %d — where the next request starts, not how many came back", page.Next, 2*pageLimit)
	}
}

func TestFollowedArtistsEndAtTheEndOfTheList(t *testing.T) {
	s, _ := followedStub(t, 2)

	last, err := s.FollowedArtists(t.Context(), pageLimit)
	if err != nil {
		t.Fatal(err)
	}
	if last.More {
		t.Error("More = true on the last page of the followed list")
	}
}

func TestFollowedArtistsCarryWhatArrivesWithThem(t *testing.T) {
	s, _ := newQueryStub(t, `{"artists":{"items":[
	  {"id":"ar1","name":"Queen","genres":["glam rock"],"followers":{"total":45000000},
	   "images":[{"url":"https://img/320","width":320,"height":320}]}
	],"next":null}}`)

	page, err := s.FollowedArtists(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("%d artists, want 1", len(page.Items))
	}

	artist := page.Items[0]
	if artist.ID != "ar1" || artist.Name != "Queen" || artist.ImageURL != "https://img/320" {
		t.Errorf("artist = %+v, want it named and pictured", artist)
	}
	if artist.Followers != 45_000_000 || len(artist.Genres) != 1 {
		t.Errorf("followers = %d, genres = %v — want what the list already said", artist.Followers, artist.Genres)
	}
}

// Recently played is asked for by count, not by offset: Spotify walks it by
// timestamp and keeps only the last fifty, so a limit larger than that is a
// request it would refuse.
func TestRecentlyPlayedAsksForACount(t *testing.T) {
	for _, c := range []struct{ in, want int }{
		{10, 10},
		{0, 1},
		{500, pageLimit},
	} {
		s, got := newQueryStub(t, `{"items":[]}`)
		if _, err := s.RecentlyPlayed(t.Context(), c.in); err != nil {
			t.Fatal(err)
		}
		if want := strconv.Itoa(c.want); got.Get("limit") != want {
			t.Errorf("limit for %d = %q, want %q", c.in, got.Get("limit"), want)
		}
		if got.Get("offset") != "" {
			t.Errorf("offset = %q, want none — this list has none", got.Get("offset"))
		}
	}
}

// A history repeats: playing a song twice is two entries. Collapsing them would
// answer a question nobody asked, and would make the list shorter than the count
// it was asked for.
func TestRecentlyPlayedKeepsRepeats(t *testing.T) {
	s, _ := newQueryStub(t, `{"items":[
	  {"played_at":"2024-01-02T10:00:00Z","track":{"id":"t1","name":"Heroes","artists":[{"name":"David Bowie"}]}},
	  {"played_at":"2024-01-02T09:00:00Z","track":{"id":"t2","name":"Life on Mars?","artists":[{"name":"David Bowie"}]}},
	  {"played_at":"2024-01-02T08:00:00Z","track":{"id":"t1","name":"Heroes","artists":[{"name":"David Bowie"}]}}
	]}`)

	tracks, err := s.RecentlyPlayed(t.Context(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 {
		t.Fatalf("%d tracks, want the 3 that were played", len(tracks))
	}
	if tracks[0].ID != "t1" || tracks[2].ID != "t1" {
		t.Errorf("history = %s, %s, %s — want the repeat kept, most recent first",
			tracks[0].ID, tracks[1].ID, tracks[2].ID)
	}
}
