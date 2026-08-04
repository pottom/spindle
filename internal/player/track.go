package player

import "time"

// Track is one entry of a search result or a playlist. It is deliberately
// narrower than State: a list needs identity and labels, not playback flags.
type Track struct {
	ID       string
	Title    string
	Artists  []string
	Album    string
	CoverURL string
	Duration time.Duration

	// Queued marks a track put into the queue by hand, as opposed to one the
	// context supplied. Only the former can be reordered or removed.
	Queued bool

	// The rest is detail: worth showing when a track is being looked at, never
	// needed to list one. Everything here arrives with the track itself, so
	// none of it costs a request.
	Released    string // as Spotify reports it: a year, a month or a full date
	AlbumType   string // "album", "single" or "compilation"
	TrackNumber int
	DiscNumber  int
	TotalTracks int
	Explicit    bool

	// Popularity is Spotify's 0–100 rating, or nil when the backend does not
	// report it. Zero is a value, not an absence: an obscure catalogue track
	// really does score nothing.
	Popularity *int
}
