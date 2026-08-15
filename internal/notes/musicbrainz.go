package notes

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// MusicBrainz: the key ring rather than the library.
//
// What it is unmatched at is saying which record this is. It stores the Spotify
// address of an artist as a relation, so a Spotify id can be turned into an MBID
// by exact string match — no name search, no scoring, no false positives: it
// either hits or it does not. Measured: about four artists in five of the ones
// anybody has heard of, and near all of the famous.
//
// What it is not good at is prose. There is no biography field, and the
// annotation that sometimes stands in for one is as likely to be a note between
// editors about how to file a reissue. So this source hands on the identifiers
// and the facts — where somebody is from, when they began, who was in the band
// and what they played — and leaves the paragraph to whoever comes next.
//
// Its rate limit is the tightest of any of them and it is enforced against
// bursts rather than rate: fifteen requests at once all failed, while eight a
// second in a row did not. Everything here goes through one lock, spaced.

// MusicBrainz asks musicbrainz.org. It needs no key.
type MusicBrainz struct{ pace pace }

// NewMusicBrainz makes the source. It is never nil: there is nothing to
// configure and nothing that can be missing.
func NewMusicBrainz() *MusicBrainz {
	return &MusicBrainz{pace: pace{every: 1100 * time.Millisecond}}
}

func (*MusicBrainz) Name() string { return "MusicBrainz" }

const mbBase = "https://musicbrainz.org/ws/2/"

// Artist fills in what MusicBrainz knows, and the identifiers the sources after
// it are keyed by.
func (m *MusicBrainz) Artist(ctx context.Context, k *Key, _ Artist) (Artist, error) {
	var a Artist
	if k.MBID == "" {
		if err := m.match(ctx, k); err != nil {
			return a, err
		}
	}
	if k.MBID == "" {
		return a, fmt.Errorf("musicbrainz: no artist for %q", k.Name)
	}

	var got mbArtist
	u := mbBase + "artist/" + url.PathEscape(k.MBID) + "?inc=url-rels+artist-rels+aliases&fmt=json"
	if err := fetch(ctx, &m.pace, u, &got); err != nil {
		return a, err
	}

	a.Name = got.Name
	a.Sort = got.SortName
	a.Kind = got.Type
	a.Area = got.Area.Name
	a.Line = got.Disambiguation
	a.Began = got.LifeSpan.Begin
	a.Ended = got.LifeSpan.End

	// The names the same person is filed under. On a stage name this is the one
	// place the given name appears at all, and it is the sort of thing a
	// listener actually wonders about.
	//
	// In a script the reader can read, though. MusicBrainz files the big artists
	// under their name in every alphabet there is, and a line reading
	// "クイーン · 皇后乐队" under Queen tells a Hungarian nothing at all. Latin
	// letters are the test rather than a list of languages: it is the alphabet
	// the rest of this screen is written in.
	for _, al := range got.Aliases {
		if al.Name == "" || strings.EqualFold(al.Name, got.Name) || !readable(al.Name) {
			continue
		}
		a.Aliases = append(a.Aliases, al.Name)
		if len(a.Aliases) == mostAliases {
			break
		}
	}

	for _, rel := range got.Relations {
		switch {
		case rel.Type == "wikidata" && k.Wikidata == "":
			k.Wikidata = lastPath(rel.URL.Resource)
		case rel.Type == "member of band" && rel.Artist.Name != "":
			a.Members = append(a.Members, Member{
				Name:        rel.Artist.Name,
				Instruments: rel.Attributes,
				From:        rel.Begin,
				To:          rel.End,
			})
		}
	}
	return a, nil
}

// match turns a Spotify artist id into an MBID.
//
// By the address rather than by the name. MusicBrainz keeps the streaming
// services' own links as relations, so this is an exact lookup of a string it
// either has or has not — and a lookup that can only 404 is worth a great deal
// more than a search that always returns something. A name search on "Majka"
// offers a Polish singer at ninety-one out of a hundred.
func (m *MusicBrainz) match(ctx context.Context, k *Key) error {
	if k.SpotifyArtist == "" {
		return fmt.Errorf("musicbrainz: nothing to match on")
	}
	link := "https://open.spotify.com/artist/" + k.SpotifyArtist

	var got mbURL
	u := mbBase + "url?resource=" + url.QueryEscape(link) + "&inc=artist-rels&fmt=json"
	if err := fetch(ctx, &m.pace, u, &got); err != nil {
		return err
	}
	for _, rel := range got.Relations {
		if rel.Artist.ID != "" {
			k.MBID = rel.Artist.ID
			if k.Name == "" {
				k.Name = rel.Artist.Name
			}
			return nil
		}
	}
	return fmt.Errorf("musicbrainz: %s is not linked to an artist", link)
}

// What is read out of the answers. Only the fields that are used: the rest of
// what MusicBrainz sends is large and none of spindle's business.
type mbArtist struct {
	Name           string `json:"name"`
	SortName       string `json:"sort-name"`
	Type           string `json:"type"`
	Disambiguation string `json:"disambiguation"`
	Area           struct {
		Name string `json:"name"`
	} `json:"area"`
	LifeSpan struct {
		Begin string `json:"begin"`
		End   string `json:"end"`
	} `json:"life-span"`
	Aliases   []struct{ Name string } `json:"aliases"`
	Relations []mbRelation            `json:"relations"`
}

type mbRelation struct {
	Type       string   `json:"type"`
	Attributes []string `json:"attributes"`
	Begin      string   `json:"begin"`
	End        string   `json:"end"`
	URL        struct {
		Resource string `json:"resource"`
	} `json:"url"`
	Artist struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
}

type mbURL struct {
	Relations []mbRelation `json:"relations"`
}

// mostAliases is how many other names are worth a line. Three: the given name
// behind a stage name, and a spelling or two of it.
const mostAliases = 3

// readable reports whether a name is written in the alphabet the rest of the
// screen is.
func readable(s string) bool {
	letters, latin := 0, 0
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.Is(unicode.Latin, r) {
			latin++
		}
	}
	return letters > 0 && latin*2 > letters
}

// fill writes a value only where there is not one already: the chain trusts
// whoever answered first.
func fill(at *string, with string) {
	if *at == "" {
		*at = with
	}
}

// lastPath is the identifier at the end of a URL, which is how both Wikidata and
// MusicBrainz name things.
func lastPath(s string) string {
	s = strings.TrimSuffix(s, "/")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
