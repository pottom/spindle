package main

import (
	"os"
	"strings"

	"github.com/pottom/spindle/internal/notes"
)

// The databases spindle asks about an artist, and the language it asks in.
//
// None of them is required and none of them needs a key today: what is here is
// MusicBrainz, which turns a Spotify id into an identifier the rest of the world
// uses, and Wikipedia, which is the only place a paragraph about an artist
// reliably exists. A source that needs configuring and has not been configured
// is simply not in the chain — see internal/notes.

// artistNotes builds the chain.
func artistNotes() *notes.Cached {
	return notes.NewCached(notes.NewChain(
		notes.NewMusicBrainz(),
		notes.NewWikipedia(readingLanguage()),
	))
}

// readingLanguage is the language to ask Wikipedia in, taken from the
// environment the terminal is already running in.
//
// Because it changes the answer rather than only its wording: asked in
// Hungarian, a Hungarian artist comes back in Hungarian, and asked in English
// the same artist comes back with an article somebody else wrote about them.
// English is the fallback wherever there is no article, which the source adds
// for itself.
//
// LC_ALL outranks LANG, as it does everywhere else; a locale is a language, a
// territory and an encoding, and only the first of those is a Wikipedia.
func readingLanguage() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := os.Getenv(name)
		if value == "" || strings.HasPrefix(value, "C") || strings.HasPrefix(value, "POSIX") {
			continue
		}
		lang, _, _ := strings.Cut(value, "_")
		lang, _, _ = strings.Cut(lang, ".")
		if len(lang) == 2 {
			return strings.ToLower(lang)
		}
	}
	return ""
}
