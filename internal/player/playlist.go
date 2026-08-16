package player

import (
	"html"
	"strings"
	"time"
)

// Playlist is a named collection of tracks, with enough metadata to list it
// without fetching its contents.
type Playlist struct {
	ID       string
	Name     string
	Owner    string
	CoverURL string
	Tracks   int
	Duration time.Duration

	// Description is what the owner wrote about it, which for the playlists
	// Spotify makes is the only place the reason for the list is written down.
	// It arrives as HTML entities and, sometimes, as markup.
	Description string

	// Snapshot is Spotify's own name for this version of the list, changed by
	// every edit anybody makes to it.
	//
	// It is the cheapest true answer to "is what I read yesterday still right".
	// A list read to its end is dozens of requests, and comparing the first page
	// only catches an edit that touched the first page; this catches every edit
	// there is, and it arrives with the list of playlists spindle already asks
	// for. Empty where the backend does not report one, and then the first page
	// is compared instead. See listcache.go.
	Snapshot string
}

// plainText turns a playlist description into something a terminal can show.
// Spotify stores them as HTML: entities for anything punctuated, and anchor
// tags around the playlists and artists it names itself. The markup is dropped
// rather than rendered — there is nothing to click here — and the words inside
// it are kept, because that is where the name of the thing linked to is.
func plainText(s string) string {
	var out strings.Builder
	out.Grow(len(s))

	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>' && depth > 0:
			depth--
		case depth == 0:
			out.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(html.UnescapeString(out.String())), " ")
}
