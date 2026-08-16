package main

import (
	"os"
	"strings"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/notes"
)

// The databases spindle asks about an artist, and the language it asks in.
//
// None of them is required, and only the last needs a key. MusicBrainz turns a
// Spotify id into an identifier the rest of the world uses; Wikipedia is the only
// place a paragraph about an artist reliably exists in a language other than
// English; Last.fm is the only one that has heard of everybody, and it is the
// one that says who else people listening to this listen to.
//
// A source that needs configuring and has not been configured is simply not in
// the chain, and every screen works without any of them. See internal/notes.

// artistNotes builds the chain.
//
// In the order they are trusted. MusicBrainz first because it turns a Spotify id
// into the identifiers the rest are keyed by; Wikipedia next because it is the
// only one that answers in the reader's own language; Last.fm last, and only
// where a key has been set, because what it adds is the artists nobody wrote an
// article about — and who else people listening to this listen to, which is not
// in any catalogue.
func artistNotes() *notes.Cached {
	return notes.NewCached(notes.NewChain(
		notes.NewMusicBrainz(),
		notes.NewWikipedia(readingLanguage()),
		notes.NewLastFM(auth.LastFMKey()),
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
