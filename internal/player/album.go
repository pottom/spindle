package player

// Album is one entry of a saved-albums list or of an artist's records, with
// enough on it to draw a row and a detail panel without fetching its tracks.
//
// Deliberately as thin as Playlist: no track list and no total duration, both
// of which would cost a request per album and neither of which a list shows.
type Album struct {
	ID       string
	Name     string
	Artists  []string
	CoverURL string

	// Released is as Spotify reports it: a year, a year and month, or a full
	// date, depending on what the label recorded.
	Released string

	// Tracks is how many the album holds, which is what a row says instead of a
	// running time.
	Tracks int

	// AlbumType is "album", "single" or "compilation" — the difference a
	// discography has to show, since a single and an album read alike otherwise.
	AlbumType string
}
