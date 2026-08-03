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

func secs(m, s int) time.Duration {
	return time.Duration(m)*time.Minute + time.Duration(s)*time.Second
}

// mockCatalogue is everything the mock backend knows about. Playback rotates
// through the first four entries; search and the playlists draw on all of them.
var mockCatalogue = []Track{
	{ID: "t01", Title: "Bohemian Rhapsody", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(5, 55)},
	{ID: "t02", Title: "Under Pressure", Artists: []string{"Queen", "David Bowie"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 8)},
	{ID: "t03", Title: "Ashes to Ashes", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(4, 23)},
	{ID: "t04", Title: "Is This the World We Created...? - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(2, 56)},

	{ID: "t05", Title: "Love of My Life", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(3, 39)},
	{ID: "t06", Title: "You're My Best Friend", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(2, 52)},
	{ID: "t07", Title: "Death on Two Legs", Artists: []string{"Queen"}, Album: albumOpera, CoverURL: coverOpera, Duration: secs(3, 43)},

	{ID: "t08", Title: "Body Language", Artists: []string{"Queen"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 31)},
	{ID: "t09", Title: "Las Palabras de Amor", Artists: []string{"Queen"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 30)},
	{ID: "t10", Title: "Staying Power", Artists: []string{"Queen"}, Album: albumHotSpace, CoverURL: coverHotSpace, Duration: secs(4, 12)},

	{ID: "t11", Title: "Life on Mars?", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(3, 53)},
	{ID: "t12", Title: "Heroes", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(6, 11)},
	{ID: "t13", Title: "Let's Dance", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(4, 8)},
	{ID: "t14", Title: "Rebel Rebel", Artists: []string{"David Bowie"}, Album: albumBowie, CoverURL: coverBowie, Duration: secs(4, 30)},

	{ID: "t15", Title: "Tavaszi Szél Vizet Áraszt - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(1, 12)},
	{ID: "t16", Title: "Now I'm Here - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(5, 8)},
	{ID: "t17", Title: "Friends Will Be Friends - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(4, 2)},
	{ID: "t18", Title: "Radio Ga Ga - Live", Artists: []string{"Queen"}, Album: albumBudapest, CoverURL: coverBudapest, Duration: secs(6, 6)},
}

// mockPlaylistDef ties a playlist's metadata to the tracks it holds.
type mockPlaylistDef struct {
	Playlist
	trackIDs []string
}

var mockPlaylists = []mockPlaylistDef{
	{
		Playlist: Playlist{ID: "p1", Name: "Deep Cuts", Owner: "you", CoverURL: coverOpera},
		trackIDs: []string{"t05", "t07", "t09", "t10", "t14", "t06"},
	},
	{
		Playlist: Playlist{ID: "p2", Name: "Bowie Essentials", Owner: "Spotify", CoverURL: coverBowie},
		trackIDs: []string{"t03", "t11", "t12", "t13", "t14", "t02"},
	},
	{
		Playlist: Playlist{ID: "p3", Name: "Live & Loud", Owner: "you", CoverURL: coverBudapest},
		trackIDs: []string{"t16", "t18", "t17", "t15", "t04"},
	},
	{
		Playlist: Playlist{ID: "p4", Name: "Queen Forever", Owner: "you", CoverURL: coverHotSpace},
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
