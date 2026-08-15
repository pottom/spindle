package notes

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Wikipedia, by way of Wikidata: the paragraph.
//
// The only prose in the whole ecosystem that is reliably about the artist rather
// than about the filing. It is not reached directly — a name is not an
// identifier, and searching Wikipedia for "Queen" is an invitation to be told
// about a monarch — but through the Wikidata id MusicBrainz hands on, which is
// exact.
//
// Two requests: the Wikidata item says which article in which language, and the
// article's summary endpoint gives a lead paragraph already trimmed to two to
// four sentences. Neither is on MusicBrainz's budget and both are comfortable at
// far higher rates.
//
// Language matters here in a way it does not elsewhere. Asked in Hungarian, a
// Hungarian artist comes back in Hungarian — measured on Majka, where the
// Hungarian lead is one sentence and the English one is four. So the wanted
// language is asked for first and English is the fallback, and which one
// answered is recorded: a paragraph in the wrong language should say so rather
// than look like a translation.

// Wikipedia asks Wikidata and then Wikipedia. It needs no key.
type Wikipedia struct {
	// Langs is which Wikipedias to try, in order. The first that has an article
	// wins; there is no merging.
	Langs []string

	pace pace
}

// NewWikipedia makes the source. Languages are tried in the order given, and
// English is appended if it is not already there — an artist with no article in
// your language is the common case, not the exception.
func NewWikipedia(langs ...string) *Wikipedia {
	out := &Wikipedia{pace: pace{every: 200 * time.Millisecond}}
	seen := map[string]bool{}
	for _, l := range append(langs, "en") {
		if l != "" && !seen[l] {
			seen[l] = true
			out.Langs = append(out.Langs, l)
		}
	}
	return out
}

func (*Wikipedia) Name() string { return "Wikipedia" }

// Artist adds the lead paragraph, and the one-line description under it where
// nobody has written one yet.
func (w *Wikipedia) Artist(ctx context.Context, k *Key, have Artist) (Artist, error) {
	var a Artist
	if have.Note != "" {
		// Somebody has already written the paragraph. Two requests saved.
		return a, nil
	}
	if k.Wikidata == "" {
		return a, fmt.Errorf("wikipedia: nothing to look up")
	}

	var links map[string]struct {
		Title string `json:"title"`
	}
	u := "https://www.wikidata.org/w/rest.php/wikibase/v1/entities/items/" +
		url.PathEscape(k.Wikidata) + "/sitelinks"
	if err := fetch(ctx, &w.pace, u, &links); err != nil {
		return a, err
	}

	for _, lang := range w.Langs {
		link, ok := links[lang+"wiki"]
		if !ok || link.Title == "" {
			continue
		}

		var page struct {
			Description string `json:"description"`
			Extract     string `json:"extract"`
			Thumbnail   struct {
				Source string `json:"source"`
			} `json:"thumbnail"`
			Original struct {
				Source string `json:"source"`
			} `json:"originalimage"`
		}
		u := "https://" + lang + ".wikipedia.org/api/rest_v1/page/summary/" +
			url.PathEscape(strings.ReplaceAll(link.Title, " ", "_"))
		if err := fetch(ctx, &w.pace, u, &page); err != nil {
			continue
		}
		if strings.TrimSpace(page.Extract) == "" {
			continue
		}

		a.Note = strings.Join(strings.Fields(page.Extract), " ")
		a.NoteFrom, a.NoteLang = w.Name(), lang
		a.Line = page.Description
		// The picture comes back in the same answer as the words, which is why
		// there is no separate request for it anywhere. See the artist panel.
		a.ImageURL = page.Original.Source
		fill(&a.ImageURL, page.Thumbnail.Source)
		return a, nil
	}
	return a, fmt.Errorf("wikipedia: no article in %s", strings.Join(w.Langs, ", "))
}
