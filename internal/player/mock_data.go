package player

import "time"

// coverBase is the Spotify image CDN prefix for 640px album art. Cover URLs are
// public, so the mock can exercise the real download path without credentials.
const coverBase = "https://i.scdn.co/image/ab67616d0000b273"

// mockTrack is one entry of the fixed playlist the mock backend loops over.
type mockTrack struct {
	id       string
	title    string
	artists  []string
	album    string
	cover    string
	duration time.Duration
}

// The four covers are deliberately unalike: flat saturated blocks, fine line art
// on white, a photographic gradient and a dark high-contrast photo. Between them
// they expose banding, colour shifts and resampling artefacts.
var mockTracks = []mockTrack{
	{
		id:       "mock-track-1",
		title:    "Bohemian Rhapsody",
		artists:  []string{"Queen"},
		album:    "A Night at the Opera",
		cover:    coverBase + "fdab4a163ab9f6db72c952ee",
		duration: 5*time.Minute + 55*time.Second,
	},
	{
		id:       "mock-track-2",
		title:    "Under Pressure",
		artists:  []string{"Queen", "David Bowie"},
		album:    "Hot Space",
		cover:    coverBase + "44c0a9843fac69db4d56d14e",
		duration: 4*time.Minute + 8*time.Second,
	},
	{
		id:       "mock-track-3",
		title:    "Ashes to Ashes",
		artists:  []string{"David Bowie"},
		album:    "best of bowie",
		cover:    coverBase + "98dc2963e511c2ff25475d03",
		duration: 4*time.Minute + 23*time.Second,
	},
	{
		id:       "mock-track-4",
		title:    "Is This the World We Created...? - Live",
		artists:  []string{"Queen"},
		album:    "Hungarian Rhapsody: Queen Live in Budapest",
		cover:    coverBase + "ebbe1174db29f8adfaf3dd62",
		duration: 2*time.Minute + 56*time.Second,
	},
}

func mockDevices() []Device {
	return []Device{
		{ID: "mock-macbook", Name: "MacBook Pro", Type: "computer", Active: true},
		{ID: "mock-iphone", Name: "iPhone", Type: "smartphone"},
		{ID: "mock-speaker", Name: "Kitchen speaker", Type: "speaker"},
	}
}
