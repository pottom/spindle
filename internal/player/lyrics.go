package player

import "context"

// Lyric is one line of a song against the clock.
type Lyric struct {
	// At is when the line is sung, from the start of the track.
	At    int64 // milliseconds
	Words string
}

// Lyrics is what a track says, and whether it says it in time.
type Lyrics struct {
	// Synced says whether the lines carry timings. Without them there is
	// something to read but nothing to follow.
	Synced bool

	// Language is what the words are in, as the provider tags them: "en",
	// "hu", and so on. It is carried because a syllable is counted differently
	// in different languages, and how many syllables a word has is how long it
	// takes to sing — see the sweep in internal/ui.
	Language string

	Lines []Lyric
}

// LyricSource is implemented by backends that can fetch the words of a track.
// Spotify has no public endpoint for them; the local device reaches the one its
// own client uses, so only it can answer.
//
// Kept out of Player on purpose, like the waveform: widening the interface
// would force every backend to pretend it has words to give.
type LyricSource interface {
	// Lyrics returns the words for a track, or nil when it has none. A track
	// without lyrics is the ordinary case, not an error.
	Lyrics(ctx context.Context, trackID string) (*Lyrics, error)
}
