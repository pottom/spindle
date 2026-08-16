package notes

import (
	"strings"
	"testing"
)

// A key nobody has set is not a source that fails: it is not in the chain, and
// no request is made that was going to be refused.
//
// The interface rather than the pointer is the whole of it. A nil *LastFM handed
// to something taking a Source is not nil — it is an interface holding a nil
// pointer, and it would be asked, and it would answer by crashing.
func TestWithoutAKeyThereIsNoSource(t *testing.T) {
	if s := NewLastFM(""); s != nil {
		t.Errorf("no key gave a source: %#v", s)
	}
	if s := NewLastFM("   "); s != nil {
		t.Error("a key of spaces gave a source")
	}
	if NewLastFM("abc") == nil {
		t.Error("a key gave nothing")
	}

	c := NewChain(NewMusicBrainz(), NewLastFM(""))
	if c.Sources() != 1 || c.Has("Last.fm") {
		t.Errorf("the chain holds %v, want only the one that is configured", c.Names())
	}
}

// A biography arrives as HTML with a link to Last.fm's own site on the end of it
// and the licence notice after that. Neither is prose about the artist, and a
// paragraph that trails off into "User-contributed text is available under…"
// reads as a page that failed to load.
func TestABiographyIsCleanedOfWhatIsNotProse(t *testing.T) {
	raw := `Majoros P&#233;ter, known as <b>Majka</b>, is a hungarian rapper` +
		` &amp; anchorman. <a href="https://www.last.fm/music/Majka">Read more on Last.fm</a>.` +
		` User-contributed text is available under the Creative Commons By-SA License.`

	got := plainBio(raw)
	switch {
	case strings.Contains(got, "<"):
		t.Errorf("markup survived: %q", got)
	case strings.Contains(got, "Read more"), strings.Contains(got, "Creative Commons"):
		t.Errorf("the tail survived: %q", got)
	case !strings.Contains(got, "hungarian rapper & anchorman"):
		t.Errorf("the prose did not: %q", got)
	}

	if plainBio("") != "" {
		t.Error("nothing became something")
	}
	if got := plainBio("<a>Read more on Last.fm</a>."); got != "" {
		t.Errorf("a biography that is only a link came back as %q", got)
	}
}
