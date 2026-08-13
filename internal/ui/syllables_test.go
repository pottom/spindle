package ui

import "testing"

// The counter, on the words that made the letter ruler wrong.
func TestSyllablesAreCounted(t *testing.T) {
	for _, c := range []struct {
		word, lang string
		want       int
	}{
		// English: the ones where letters and mouths disagree most.
		{"through", "en", 1}, {"dangerous", "en", 3}, {"times", "en", 1},
		{"changed", "en", 1}, {"godforsaken", "en", 4}, {"borderlines", "en", 3},
		{"destiny", "en", 3}, {"little", "en", 2}, {"agree", "en", 2},
		{"the", "en", 1}, {"I", "en", 1}, {"rhythm", "en", 1},

		// Hungarian: one vowel, one syllable, exactly. The digraphs are
		// consonants — "gyász" is one, not two.
		{"emlék", "hu", 2}, {"fiatalság", "hu", 4}, {"gyász", "hu", 1},
		{"szívemben", "hu", 3}, {"elevenen", "hu", 4}, {"őrült", "hu", 2},

		// And a word with nothing to say is worth nothing.
		{"—", "en", 0}, {"...", "hu", 0},
	} {
		if got := syllables(c.word, c.lang); got != c.want {
			t.Errorf("syllables(%q, %q) = %d, want %d", c.word, c.lang, got, c.want)
		}
	}
}

// The sweep divides a line by its syllables, so a long word with one of them
// does not race the light past the words after it.
//
// Measured on a line that was tapped by ear: by the letter, "times" lights 243
// ms before its turn, because "dangerous" is nine letters and three syllables
// and the light spends the letters rather than the mouthfuls.
func TestTheSweepIsSpentOnSyllables(t *testing.T) {
	const line = "They say these are dangerous times"

	// Halfway through the singing, the voice is on "dangerous": four syllables
	// of eight have gone. By the letter it would already be past it.
	at := sweepTo(line, "en", 0.5)
	if got := line[:at]; got != "They say these are dangerous" {
		t.Errorf("halfway the line is lit to %q, want it on the long word", got)
	}

	// A word lights as its turn comes, not as it ends: at nothing at all, the
	// first word is lit and no more.
	if got := line[:sweepTo(line, "en", 0)]; got != "They" {
		t.Errorf("at the start the line is lit to %q, want the first word", got)
	}
	// And by the end, all of it.
	if got := sweepTo(line, "en", 1); got != len([]rune(line)) {
		t.Errorf("at the end %d of %d is lit, want all of it", got, len([]rune(line)))
	}
}

// It never stops inside a word, whatever fraction it is asked for.
func TestTheSweepLandsOnWholeWords(t *testing.T) {
	for _, line := range []string{
		"They say these are dangerous times",
		"Fiatalság, mennyi balhé",
		"one",
	} {
		runes := []rune(line)
		for i := range 101 {
			at := sweepTo(line, "hu", float64(i)/100)
			if at < 0 || at > len(runes) {
				t.Fatalf("%q at %d%% swept to %d, which is off the line", line, i, at)
			}
			if at > 0 && at < len(runes) && runes[at] != ' ' {
				t.Errorf("%q at %d%% stops inside a word: %q", line, i, string(runes[:at]))
			}
		}
	}
}
