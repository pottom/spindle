// Package notes is what the other databases know about a record that Spotify
// does not say.
//
// Spotify will tell you a name, a cover and a popularity score. It will not tell
// you who wrote the song, who was in the band and when, or a sentence about who
// this is — and those are the things somebody reading a music player in a
// terminal actually wants. Several free databases will, and none of them alone
// covers a whole library.
//
// So this is a chain rather than a client. Each source is asked in turn; each
// may fill in what is still empty and hand on an identifier the next one needs —
// MusicBrainz turns a Spotify id into an MBID and a Wikidata id, and Wikipedia
// turns the Wikidata id into a paragraph. Every one of them is allowed to be
// missing, silent or slow, and what comes out is whatever was learned.
//
// Nothing here is required for spindle to work. A source with no key is not
// registered; a source that fails is a source that knew nothing. There is no
// error state on screen and there is no such thing as a half-drawn panel: what
// arrived is drawn and what did not is absent.
package notes

import (
	"context"
	"time"
)

// Key is what is known about the thing being asked after, and what has been
// learned about it on the way through the chain.
//
// It gains identifiers as it goes: the Spotify id is what spindle starts with,
// and it is the source that can match exactly on it — MusicBrainz stores Spotify
// links — that turns it into the ids everything else is keyed by.
type Key struct {
	// SpotifyArtist is the id spindle holds. Name is the fallback for sources
	// that can only be asked by name, and for matching where no link exists.
	SpotifyArtist string
	Name          string

	// MBID and Wikidata are filled in as they are learned.
	MBID     string
	Wikidata string
}

// Artist is what the chain learned. Every field may be empty; nothing here is
// promised by anybody.
type Artist struct {
	Name    string
	Sort    string
	Aliases []string

	// Kind is "Person" or "Group" as MusicBrainz classes them, Area where they
	// are from, and Began and Ended the years — as reported, which can be a
	// year, a month or a full date.
	Kind  string
	Area  string
	Began string
	Ended string

	// Line is the one-line description a database keeps for telling two artists
	// of the same name apart. It is short by design and often the best sentence
	// there is.
	Line string

	// Note is the paragraph, and NoteFrom which database wrote it — shown,
	// because a paragraph from somewhere has to say where.
	Note     string
	NoteFrom string
	NoteLang string

	// Members is who was in the band, with what they played and when.
	Members []Member

	// Similar is who else people who listen to this listen to, and Listeners
	// how many of them there are. Tags is what those people call this music.
	//
	// Not from the same place as the rest: this is what a scrobbling service
	// knows and a catalogue does not, and it is the one thing here that says
	// something about how a record is heard rather than about how it was made.
	Similar   []string
	Tags      []string
	Listeners int

	// ImageURL is a photograph of the artist rather than of a record.
	ImageURL string
}

// Member is one person in a band, for as long as they were in it.
type Member struct {
	Name        string
	Instruments []string
	From, To    string
}

// Known reports whether anything at all was learned. A panel with nothing in it
// is not drawn.
func (a Artist) Known() bool {
	return a.Note != "" || a.Line != "" || len(a.Members) > 0 || a.Area != "" ||
		a.ImageURL != "" || len(a.Similar) > 0
}

// Source is one database.
//
// It is handed what has been learned so far and gives back only what it knows
// itself; the chain does the merging, and the merging is first-come. A source
// cannot overwrite an earlier one even by accident, which is what makes the
// order they are asked in the order they are trusted in — rather than a
// convention every source has to remember.
//
// The Key is a pointer because identifiers travel forwards: MusicBrainz turns a
// Spotify id into an MBID and a Wikidata id, and the sources after it are keyed
// by those.
type Source interface {
	// Name is what the panel credits, where it credits anything.
	Name() string

	// Artist is what this database knows. `have` is what is already known, so a
	// source can decline to spend a request on something already answered.
	// Returning an error means it knew nothing — the chain carries on, because a
	// source that is down is not a reason to have no panel.
	Artist(ctx context.Context, k *Key, have Artist) (Artist, error)
}

// merge writes everything in `from` that `into` does not already have.
func merge(into *Artist, from Artist) {
	fill(&into.Name, from.Name)
	fill(&into.Sort, from.Sort)
	fill(&into.Kind, from.Kind)
	fill(&into.Area, from.Area)
	fill(&into.Began, from.Began)
	fill(&into.Ended, from.Ended)
	fill(&into.Line, from.Line)
	fill(&into.ImageURL, from.ImageURL)

	if into.Note == "" && from.Note != "" {
		into.Note, into.NoteFrom, into.NoteLang = from.Note, from.NoteFrom, from.NoteLang
	}
	if len(into.Aliases) == 0 {
		into.Aliases = from.Aliases
	}
	if len(into.Members) == 0 {
		into.Members = from.Members
	}
	if len(into.Similar) == 0 {
		into.Similar = from.Similar
	}
	if len(into.Tags) == 0 {
		into.Tags = from.Tags
	}
	if into.Listeners == 0 {
		into.Listeners = from.Listeners
	}
}

// Chain asks its sources in order.
type Chain struct {
	sources []Source
}

// NewChain makes one. Sources that are nil — a source with no key, typically —
// are dropped here, so nothing downstream has to keep asking whether they are
// there.
func NewChain(sources ...Source) *Chain {
	out := &Chain{}
	for _, s := range sources {
		if s != nil {
			out.sources = append(out.sources, s)
		}
	}
	return out
}

// Sources is how many are in the chain, which is what a program says when it
// wants to explain that a key would add one.
func (c *Chain) Sources() int { return len(c.sources) }

// Names is what they are called, in the order they are asked. A screen that says
// what spindle is asking on somebody's behalf is a screen they can decide about.
func (c *Chain) Names() []string {
	out := make([]string, 0, len(c.sources))
	for _, s := range c.sources {
		out = append(out, s.Name())
	}
	return out
}

// Has reports whether one of them is in the chain by name.
func (c *Chain) Has(name string) bool {
	for _, s := range c.sources {
		if s.Name() == name {
			return true
		}
	}
	return false
}

// Artist walks the chain and hands back what was learned.
//
// The context bounds the whole walk rather than each source: what this is for is
// a panel on a screen, and a panel that arrives late is a panel nobody is
// looking at any more.
func (c *Chain) Artist(ctx context.Context, k Key) (Artist, error) {
	var a Artist
	a.Name = k.Name

	for _, s := range c.sources {
		if ctx.Err() != nil {
			break
		}
		// An error is not fatal and is not reported. A source that is down, rate
		// limited or simply ignorant all come to the same thing here: it knew
		// nothing. See the package comment.
		got, err := s.Artist(ctx, &k, a)
		if err != nil {
			continue
		}
		merge(&a, got)
	}
	return a, ctx.Err()
}

// Fetched is how long an answer is good for.
//
// Weeks, because none of this changes: a band's line-up in 1971 is not news, and
// a Wikipedia lead paragraph is rewritten a few times a decade. The cache is
// what keeps a wall of covers from being a wall of requests.
const Fetched = 14 * 24 * time.Hour
