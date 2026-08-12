package ui

import (
	"strings"
	"testing"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The screen says why the cover looks the way it does, and where it looks
// better.
//
// It already said which of the two renderers was in use. What it did not say was
// why, or that there is anything better: somebody running spindle in a terminal
// without the protocol saw a blurry approximation of the one thing the program
// is built around, with no way of knowing a sharp version exists.
//
// On this screen rather than in a banner. A banner on every start is nagging
// about something that will not change, and a line in the help is invisible;
// this is where somebody already is when they wonder why the cover looks like
// that.
func TestTheSettingsSayWhyTheCoverLooksLikeThat(t *testing.T) {
	for _, c := range []struct {
		what  string
		found cover.Graphics
		wants []string
		not   []string
	}{{
		what:  "a terminal that draws it",
		found: cover.Graphics{Kitty: true, Placeholders: true, Name: "ghostty 1.3.1"},
		wants: []string{"the picture itself"},
		not:   []string{"coloured blocks", "kitty and Ghostty"},
	}, {
		what:  "one with the protocol but not the placeholders",
		found: cover.Graphics{Kitty: true, Name: "WezTerm 20240203"},
		// Named, because "no graphics protocol" would be untrue here and would
		// send somebody looking for a setting that would not help.
		wants: []string{"WezTerm", "placeholders", "coloured blocks", "kitty and Ghostty"},
		not:   []string{"no graphics protocol"},
	}, {
		what:  "one with no protocol at all",
		found: cover.Graphics{},
		wants: []string{"no graphics protocol", "coloured blocks", "kitty and Ghostty"},
		not:   []string{"placeholders"},
	}} {
		says := artworkSays(c.found)
		for _, want := range c.wants {
			if !strings.Contains(says, want) {
				t.Errorf("%s: %q does not mention %q", c.what, says, want)
			}
		}
		for _, no := range c.not {
			if strings.Contains(says, no) {
				t.Errorf("%s: %q should not mention %q", c.what, says, no)
			}
		}
	}
}
