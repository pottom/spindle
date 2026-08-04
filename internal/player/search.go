package player

// SearchKind is what a query is looking for. The empty kind asks for all of
// them, which is what a fresh query wants: Spotify answers every kind in one
// request, so learning that an artist matched costs nothing extra.
type SearchKind string

const (
	SearchTracks    SearchKind = "track"
	SearchAlbums    SearchKind = "album"
	SearchArtists   SearchKind = "artist"
	SearchPlaylists SearchKind = "playlist"
)

// SearchKinds is the order they are offered in. Tracks first, because that is
// what nearly every search is for.
var SearchKinds = []SearchKind{SearchTracks, SearchAlbums, SearchArtists, SearchPlaylists}

// String is what a heading calls the kind.
func (k SearchKind) String() string {
	switch k {
	case SearchAlbums:
		return "albums"
	case SearchArtists:
		return "artists"
	case SearchPlaylists:
		return "playlists"
	default:
		return "tracks"
	}
}

// Results is what a query matched: a page of each kind that was asked for.
//
// All four together rather than one at a time, because that is how Spotify
// answers — one request carries every kind, each with its own paging. Reading
// further into one kind is then a request for that kind alone, which is why
// each page keeps its own More and Next.
type Results struct {
	Tracks    Page[Track]
	Albums    Page[Album]
	Artists   Page[Artist]
	Playlists Page[Playlist]
}
