package notes

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Last.fm: what people who listen to this also listen to, and how many of them
// there are.
//
// It is the only one of these that needs a key, and the only one that had heard
// of every artist tried against it. Measured on six records from this library:
// MusicBrainz knew four of them, Wikipedia four, TheAudioDB five — Last.fm knew
// all six, including two that nothing else has a row for. It is also the only
// place a *Hungarian* artist has neighbours: ListenBrainz returned an empty list
// for Majka, Last.fm returned Halott Pénz and Wellhello.
//
// What it does not have is a picture — every artist comes back with the same
// grey placeholder, and has for years — or a word of Hungarian: asked with
// lang=hu, all six answered in English or not at all. So it stands after
// Wikipedia in the chain: the paragraph in the reader's own language wins, and
// this fills in for the artists nobody wrote an article about.
//
// Free for non-commercial use, about five requests a second, and the key is
// asked for by name — an installation without one simply does not have this
// source. See internal/auth for where it is kept.

// LastFM asks ws.audioscrobbler.com.
type LastFM struct {
	key  string
	pace pace
}

// NewLastFM makes the source, or nothing where there is no key.
//
// Nothing rather than a source that fails: the chain drops what is not there, so
// nothing downstream has to keep asking whether this one is configured, and no
// request is ever made that was going to be refused.
//
// It gives back the interface rather than the type on purpose. A nil *LastFM
// handed to something that takes a Source is not nil — it is an interface
// holding a nil pointer, and it would be asked, and it would answer by crashing.
func NewLastFM(key string) Source {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	return &LastFM{key: key, pace: pace{every: 200 * time.Millisecond}}
}

func (*LastFM) Name() string { return "Last.fm" }

// Artist adds what Last.fm knows: the paragraph where nobody else had one, who
// else people listening to this listen to, and how many are listening.
func (l *LastFM) Artist(ctx context.Context, k *Key, have Artist) (Artist, error) {
	var a Artist

	got, err := l.ask(ctx, k)
	if err != nil {
		return a, err
	}

	// The paragraph, only where the chain has none. Last.fm writes in English
	// whatever it is asked in — measured on all six — so a Hungarian lead from
	// Wikipedia is worth more than a longer English one from here.
	if have.Note == "" {
		if bio := plainBio(got.Bio.Content); bio != "" {
			a.Note, a.NoteFrom, a.NoteLang = bio, l.Name(), "en"
		}
	}

	for _, s := range got.Similar.Artist {
		if s.Name != "" {
			a.Similar = append(a.Similar, s.Name)
		}
	}
	for _, t := range got.Tags.Tag {
		if t.Name != "" {
			a.Tags = append(a.Tags, t.Name)
		}
	}
	a.Listeners, _ = strconv.Atoi(got.Stats.Listeners)

	// And the identifier, where this is the one that found the artist: an MBID
	// learned here is an MBID everything else can be asked with.
	if k.MBID == "" && got.MBID != "" {
		k.MBID = got.MBID
	}
	return a, nil
}

// ask reads the artist, by identifier where there is one and by name otherwise.
//
// By name matters here rather than being a fallback of last resort: the artists
// nothing else has heard of are exactly the ones with no MusicBrainz id, and
// they are why this source is worth having. Last.fm's own matching is what finds
// them.
func (l *LastFM) ask(ctx context.Context, k *Key) (lfmArtist, error) {
	var out struct {
		Artist lfmArtist `json:"artist"`
		Error  int       `json:"error"`
		Msg    string    `json:"message"`
	}

	q := url.Values{"method": {"artist.getinfo"}, "api_key": {l.key}, "format": {"json"}}
	switch {
	case k.MBID != "":
		q.Set("mbid", k.MBID)
	case k.Name != "":
		q.Set("artist", k.Name)
	default:
		return lfmArtist{}, fmt.Errorf("last.fm: nothing to look up")
	}

	if err := fetch(ctx, &l.pace, lfmBase+q.Encode(), &out); err != nil {
		return lfmArtist{}, err
	}
	if out.Error != 0 {
		// An identifier it does not know is worth trying by name: the two
		// databases disagree about which artists exist, which is the whole
		// reason this one is in the chain. Measured: Two Steps From Hell is not
		// found by its MBID and is found by its name.
		if k.MBID != "" && k.Name != "" {
			k.MBID = ""
			defer func() { k.MBID = "" }()
			return l.ask(ctx, &Key{Name: k.Name})
		}
		return lfmArtist{}, fmt.Errorf("last.fm: %s", out.Msg)
	}
	return out.Artist, nil
}

const lfmBase = "https://ws.audioscrobbler.com/2.0/?"

type lfmArtist struct {
	Name  string `json:"name"`
	MBID  string `json:"mbid"`
	Stats struct {
		Listeners string `json:"listeners"`
		Playcount string `json:"playcount"`
	} `json:"stats"`
	Similar struct {
		Artist []struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"similar"`
	Tags struct {
		Tag []struct {
			Name string `json:"name"`
		} `json:"tag"`
	} `json:"tags"`
	Bio struct {
		Content string `json:"content"`
	} `json:"bio"`
}

// tags and the tail Last.fm signs every biography with.
var (
	lfmTags = regexp.MustCompile(`<[^>]*>`)
	lfmTail = regexp.MustCompile(`(?s)\s*(Read more on Last\.fm|User-contributed text).*$`)
)

// plainBio turns a Last.fm biography into something a terminal can show.
//
// It arrives as HTML with a link to their own site on the end of it, and the
// licence notice after that. Neither is prose about the artist, and a paragraph
// that trails off into "User-contributed text is available under…" reads as a
// page that failed to load.
func plainBio(s string) string {
	s = lfmTail.ReplaceAllString(s, "")
	s = lfmTags.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	return strings.Join(strings.Fields(s), " ")
}

// Track is what Last.fm has to say about one song.
//
// The one thing in this whole layer that is about a recording rather than about
// who made it: a paragraph somebody wrote about the song, and how many people
// have heard it. It is empty for most of a real library and worth having for the
// rest — the story of a song is read once and remembered.
//
// Matched by name, because that is what spindle holds. It is Last.fm's own
// matching rather than an exact key, so a wrong answer is possible; what makes
// that tolerable is that the answer is prose about a title, and prose about the
// wrong song reads wrong immediately.
func (l *LastFM) Track(ctx context.Context, k *TrackKey, have TrackNote) (TrackNote, error) {
	var t TrackNote
	if have.Note != "" {
		return t, nil
	}
	if k.Artist == "" || k.Title == "" {
		return t, fmt.Errorf("last.fm: nothing to look up")
	}

	var out struct {
		Track struct {
			Listeners string `json:"listeners"`
			Playcount string `json:"playcount"`
			TopTags   struct {
				Tag []struct {
					Name string `json:"name"`
				} `json:"tag"`
			} `json:"toptags"`
			Wiki struct {
				Content string `json:"content"`
			} `json:"wiki"`
		} `json:"track"`
		Error int    `json:"error"`
		Msg   string `json:"message"`
	}

	q := url.Values{
		"method":  {"track.getinfo"},
		"api_key": {l.key},
		"format":  {"json"},
		"artist":  {k.Artist},
		"track":   {k.Title},
	}
	if err := fetch(ctx, &l.pace, lfmBase+q.Encode(), &out); err != nil {
		return t, err
	}
	if out.Error != 0 {
		return t, fmt.Errorf("last.fm: %s", out.Msg)
	}

	if bio := plainBio(out.Track.Wiki.Content); bio != "" {
		t.Note, t.NoteFrom = bio, l.Name()
	}
	t.Listeners, _ = strconv.Atoi(out.Track.Listeners)
	t.Plays, _ = strconv.Atoi(out.Track.Playcount)
	for _, tag := range out.Track.TopTags.Tag {
		if tag.Name != "" {
			t.Tags = append(t.Tags, tag.Name)
		}
	}
	return t, nil
}
