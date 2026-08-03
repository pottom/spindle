package player

import "time"

// mockTrack is one entry of the fixed playlist the mock backend loops over.
type mockTrack struct {
	id       string
	title    string
	artists  []string
	album    string
	duration time.Duration
}

var mockTracks = []mockTrack{
	{
		id:       "mock-track-1",
		title:    "Bohemian Rhapsody",
		artists:  []string{"Queen"},
		album:    "A Night at the Opera",
		duration: 5*time.Minute + 55*time.Second,
	},
	{
		id:       "mock-track-2",
		title:    "Under Pressure",
		artists:  []string{"Queen", "David Bowie"},
		album:    "Hot Space",
		duration: 4*time.Minute + 8*time.Second,
	},
	{
		id:       "mock-track-3",
		title:    "Sing About Me, I'm Dying Of Thirst",
		artists:  []string{"Kendrick Lamar"},
		album:    "good kid, m.A.A.d city",
		duration: 12*time.Minute + 3*time.Second,
	},
	{
		id:       "mock-track-4",
		title:    "Teardrop",
		artists:  []string{"Massive Attack"},
		album:    "Mezzanine",
		duration: 5*time.Minute + 29*time.Second,
	},
}

func mockDevices() []Device {
	return []Device{
		{ID: "mock-macbook", Name: "MacBook Pro", Type: "computer", Active: true},
		{ID: "mock-iphone", Name: "iPhone", Type: "smartphone"},
		{ID: "mock-speaker", Name: "Kitchen speaker", Type: "speaker"},
	}
}
