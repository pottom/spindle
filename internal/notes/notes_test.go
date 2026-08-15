package notes

import (
	"context"
	"errors"
	"testing"
)

// A source that answers from a script rather than from the network. Everything
// worth testing here is the walking and the keeping; what the databases send
// back was measured against the live services and is written down in
// docs/MUSICBRAINZ.md.
type saying struct {
	name string
	fill func(k *Key, a *Artist)
	err  error
	// asked counts the walks, which is how the cache is caught not working.
	asked *int
}

func (s saying) Name() string { return s.name }

func (s saying) Artist(_ context.Context, k *Key, _ Artist) (Artist, error) {
	if s.asked != nil {
		*s.asked++
	}
	var a Artist
	if s.err != nil {
		return a, s.err
	}
	s.fill(k, &a)
	return a, nil
}

// The chain is a pipeline: each source adds what is still missing and hands on
// what the next one needs. The order they are asked in is the order they are
// trusted in.
func TestTheChainFillsWhatIsStillEmpty(t *testing.T) {
	first := saying{name: "first", fill: func(k *Key, a *Artist) {
		k.Wikidata = "Q1"
		a.Line = "the first answer"
	}}
	second := saying{name: "second", fill: func(k *Key, a *Artist) {
		if k.Wikidata != "Q1" {
			t.Errorf("the second source was asked without what the first learned")
		}
		a.Line = "the second answer"
		a.Note = "the paragraph"
	}}

	got, err := NewChain(first, second).Artist(context.Background(), Key{Name: "somebody"})
	if err != nil {
		t.Fatalf("Artist: %v", err)
	}
	if got.Line != "the first answer" {
		t.Errorf("line = %q, want the first source's — it answered first", got.Line)
	}
	if got.Note != "the paragraph" {
		t.Errorf("note = %q, want the second source's — nobody else had one", got.Note)
	}
}

// A source that is down, rate limited or simply ignorant all come to the same
// thing: it added nothing, and the walk carries on. There is no failure here
// that is worth a screen without a panel.
func TestASourceThatFailsDoesNotStopTheWalk(t *testing.T) {
	broken := saying{name: "broken", err: errors.New("503")}
	working := saying{name: "working", fill: func(_ *Key, a *Artist) { a.Note = "still here" }}

	got, err := NewChain(broken, working).Artist(context.Background(), Key{})
	if err != nil {
		t.Fatalf("Artist: %v", err)
	}
	if got.Note != "still here" {
		t.Error("a broken source took the answer down with it")
	}
}

// A source with no key is not registered, and nothing downstream has to keep
// asking whether it is there.
func TestASourceThatIsNotThereIsNotAsked(t *testing.T) {
	var missing Source
	c := NewChain(missing, saying{name: "there", fill: func(*Key, *Artist) {}})
	if c.Sources() != 1 {
		t.Errorf("the chain holds %d sources, want only the one that exists", c.Sources())
	}
	if _, err := c.Artist(context.Background(), Key{}); err != nil {
		t.Errorf("walking a chain with a missing source: %v", err)
	}
}

// Asked twice, answered once. Walking a list asks after the same few artists
// over and over, and one request a second cannot serve a screen.
func TestAnAnswerIsKeptAndNotAskedForTwice(t *testing.T) {
	asked := 0
	c := NewCached(NewChain(saying{
		name:  "counted",
		asked: &asked,
		fill:  func(_ *Key, a *Artist) { a.Note = "once" },
	}))
	c.dir = "" // memory only: a test may not write to the user's cache

	k := Key{SpotifyArtist: "abc", Name: "somebody"}
	for range 3 {
		if _, err := c.Artist(context.Background(), k); err != nil {
			t.Fatalf("Artist: %v", err)
		}
	}
	if asked != 1 {
		t.Errorf("the chain was walked %d times, want once", asked)
	}
}

// And a miss is kept too. The long tail of a real library is full of records
// nothing will ever be known about, and asking after them again every time the
// cursor passes is the whole cost of having this at all.
func TestNotKnowingIsAlsoAnAnswerWorthKeeping(t *testing.T) {
	asked := 0
	c := NewCached(NewChain(saying{name: "silent", asked: &asked, fill: func(*Key, *Artist) {}}))
	c.dir = ""

	k := Key{SpotifyArtist: "nobody"}
	first, _ := c.Artist(context.Background(), k)
	if first.Known() {
		t.Fatal("the silent source said something")
	}
	if _, err := c.Artist(context.Background(), k); err != nil {
		t.Fatalf("Artist: %v", err)
	}
	if asked != 1 {
		t.Errorf("an artist nobody knows was asked after %d times, want once", asked)
	}
}

// What an answer is filed under has to be a filename on every platform spindle
// builds for, and two artists must not land on one file.
func TestAnAnswerIsFiledUnderSomethingSafe(t *testing.T) {
	if got := cacheName(Key{SpotifyArtist: "0D8reSG6hzc5KEQWZPYGFB"}); got != "artist-0D8reSG6hzc5KEQWZPYGFB.json" {
		t.Errorf("cacheName = %q", got)
	}
	got := cacheName(Key{Name: "AC/DC · ?"})
	if got == cacheName(Key{Name: "AC-DC"}) {
		t.Error("two artists share a file")
	}
	for _, r := range got {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			t.Errorf("cacheName = %q, which holds %q", got, r)
		}
	}
}
