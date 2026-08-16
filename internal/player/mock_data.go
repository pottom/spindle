package player

import "time"

// coverBase is the Spotify image CDN prefix for 640px album art. Cover URLs are
// public, so the mock can exercise the real download path without credentials.
const coverBase = "https://i.scdn.co/image/ab67616d0000b273"

// The four albums are deliberately unalike to look at: flat saturated blocks,
// fine line art on white, a photographic gradient and a dark high-contrast photo.
// Between them they expose banding, colour shifts and resampling artefacts, and
// they give the accent extractor four very different answers.
const (
	coverOpera    = coverBase + "fdab4a163ab9f6db72c952ee"
	coverHotSpace = coverBase + "44c0a9843fac69db4d56d14e"
	coverBowie    = coverBase + "98dc2963e511c2ff25475d03"
	coverBudapest = coverBase + "ebbe1174db29f8adfaf3dd62"
)

const (
	albumOpera    = "A Night at the Opera"
	albumHotSpace = "Hot Space"
	albumBowie    = "best of bowie"
	albumBudapest = "Hungarian Rhapsody: Queen Live in Budapest"
)

// The four albums the catalogue is drawn from, with real release dates and
// lengths so the detail panel has something to show offline. The mock is where
// the layout is judged; blank fields would flatter it.
//
// This is also what the saved-albums screen lists and what an artist's records
// are picked from, so the same four records answer every way in to them.
var mockAlbumList = []Album{
	{ID: "al1", Name: albumOpera, Artists: []string{"Queen"}, CoverURL: coverOpera, Released: "1975-11-21", Tracks: 12, AlbumType: "album"},
	{ID: "al2", Name: albumHotSpace, Artists: []string{"Queen"}, CoverURL: coverHotSpace, Released: "1982-05-21", Tracks: 11, AlbumType: "album"},
	{ID: "al3", Name: albumBowie, Artists: []string{"David Bowie"}, CoverURL: coverBowie, Released: "2002-10-14", Tracks: 20, AlbumType: "compilation"},
	{ID: "al4", Name: albumBudapest, Artists: []string{"Queen"}, CoverURL: coverBudapest, Released: "2012-11-05", Tracks: 22, AlbumType: "album"},
}

// The two artists behind the catalogue. Their pictures are their own album
// covers: artist photographs live at URLs the mock has no honest way to know,
// and a broken image would say less about the layout than a real cover does.
//
// Their ids are the real ones. Everything else here is invented, but an id is
// what the other databases are asked with — MusicBrainz matches on the Spotify
// address of an artist, exactly — and a made-up id would leave the whole of
// that path untravelled under --mock, which is where it is looked at. See
// internal/notes.
var mockArtists = []Artist{
	{ID: "1dfeR4HaWDbWqFHLkxsg1d", Name: "Queen", ImageURL: coverOpera, Genres: []string{"glam rock", "classic rock"}, Followers: 45_000_000},
	{ID: "0oSGxfWSnnOXhD2fKuz2Gy", Name: "David Bowie", ImageURL: coverBowie, Genres: []string{"art rock", "glam rock"}, Followers: 12_000_000},
}

// mockLikedIDs is what the saved-songs screen lists — enough of the catalogue,
// spread across the albums, to run past one page.
var mockLikedIDs = []string{"t01", "t03", "t05", "t08", "t11", "t12", "t14", "t16", "t18"}

// mockRecentIDs is the listening history, most recent first. One track is in it
// twice on purpose: a real history repeats, and a screen built on the assumption
// that it does not breaks the first time somebody plays a song again.
var mockRecentIDs = []string{"t03", "t01", "t12", "t01", "t16", "t05", "t09"}

// detail fills in what the Web API would have supplied along with the track.
func detail(t Track, number int) Track {
	if a := mockAlbumByName(t.Album); a != nil {
		t.Released, t.TotalTracks, t.AlbumType = a.Released, a.Tracks, a.AlbumType
	}
	t.TrackNumber = number
	t.DiscNumber = 1
	return t
}

func mockAlbumByName(name string) *Album {
	for i := range mockAlbumList {
		if mockAlbumList[i].Name == name {
			return &mockAlbumList[i]
		}
	}
	return nil
}

func secs(m, s int) time.Duration {
	return time.Duration(m)*time.Minute + time.Duration(s)*time.Second
}

// mockCatalogue is everything the mock backend knows about. Playback rotates
// through the first four entries; search and the playlists draw on all of them.
var mockCatalogue = []Track{
	detail(Track{ID: "t01", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(5, 55)}, 11),
	detail(Track{ID: "t02", Title: "Under Pressure", Artists: []string{"Queen", "David Bowie"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 8)}, 3),
	detail(Track{ID: "t03", Title: "Ashes to Ashes", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(4, 23)}, 4),
	detail(Track{ID: "t04", Title: "Is This the World We Created...? - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(2, 56)}, 10),

	detail(Track{ID: "t05", Title: "Love of My Life", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(3, 39)}, 5),
	detail(Track{ID: "t06", Title: "You're My Best Friend", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(2, 52)}, 8),
	detail(Track{ID: "t07", Title: "Death on Two Legs", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(3, 43)}, 1),

	detail(Track{ID: "t08", Title: "Body Language", Artists: []string{"Queen"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 31)}, 4),
	detail(Track{ID: "t09", Title: "Las Palabras de Amor", Artists: []string{"Queen"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 30)}, 6),
	detail(Track{ID: "t10", Title: "Staying Power", Artists: []string{"Queen"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 12)}, 2),

	detail(Track{ID: "t11", Title: "Life on Mars?", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(3, 53)}, 4),
	detail(Track{ID: "t12", Title: "Heroes", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(6, 11)}, 9),
	detail(Track{ID: "t13", Title: "Let's Dance", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(4, 8)}, 1),
	detail(Track{ID: "t14", Title: "Rebel Rebel", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(4, 30)}, 3),

	detail(Track{ID: "t15", Title: "Tavaszi Szél Vizet Áraszt - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(1, 12)}, 2),
	detail(Track{ID: "t16", Title: "Now I'm Here - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(5, 8)}, 5),
	detail(Track{ID: "t17", Title: "Friends Will Be Friends - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(4, 2)}, 9),
	detail(Track{ID: "t18", Title: "Radio Ga Ga - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(6, 6)}, 12),
}

// mockPlaylistDef ties a playlist's metadata to the tracks it holds.
type mockPlaylistDef struct {
	Playlist
	trackIDs []string
}

// Their snapshots are made up, and they are made up the same way every run: a
// mock whose playlists changed on every launch would make the cache that
// compares them useless exactly where it is looked at. See listcache.go.
var mockPlaylists = []mockPlaylistDef{
	{
		Playlist: Playlist{ID: "p1", Snapshot: "snap-p1", Name: "Deep Cuts", Owner: "you", CoverURL: coverOpera},
		trackIDs: []string{"t05", "t07", "t09", "t10", "t14", "t06"},
	},
	{
		Playlist: Playlist{ID: "p2", Snapshot: "snap-p2", Name: "Bowie Essentials", Owner: "Spotify", CoverURL: coverBowie},
		trackIDs: []string{"t03", "t11", "t12", "t13", "t14", "t02"},
	},
	{
		Playlist: Playlist{ID: "p3", Snapshot: "snap-p3", Name: "Live & Loud", Owner: "you", CoverURL: coverBudapest},
		trackIDs: []string{"t16", "t18", "t17", "t15", "t04"},
	},
	{
		Playlist: Playlist{ID: "p4", Snapshot: "snap-p4", Name: "Queen Forever", Owner: "you", CoverURL: coverHotSpace},
		trackIDs: []string{"t01", "t02", "t05", "t06", "t07", "t08", "t09", "t10", "t16"},
	},
}

func mockDevices() []Device {
	return []Device{
		{ID: "mock-macbook", Name: "MacBook Pro", Type: "computer", Active: true},
		{ID: "mock-iphone", Name: "iPhone", Type: "smartphone"},
		{ID: "mock-speaker", Name: "Kitchen speaker", Type: "speaker"},
	}
}
