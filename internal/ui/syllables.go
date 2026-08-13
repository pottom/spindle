package ui

import (
	"strings"
	"unicode"
)

// Counting syllables, because time is spent on syllables and not on letters.
//
// The sweep across a line used to be laid out by character: a line of forty
// letters was a third lit when the clock was a third through it. That is the
// wrong ruler, and it is wrong by a measurable amount — on lines from a record
// that had been tapped by ear, words landed a median of 210 ms away from where
// the syllables put them, and the worst was 369 ms. It is worst in the middle
// of a line, because the two rulers agree at both ends, and that is exactly
// where it was heard: "a word lights slightly before the singer says it, in the
// middle of a sentence".
//
// The reason is words like "through" and "dangerous". Seven letters and one
// syllable, nine letters and three: by the letter the first is slow and the
// second is quick, and by the mouth it is the other way round.
//
// This is not the syllable model that failed. That one asked how *long* a
// syllable takes, and the answer varied threefold inside one record — 442 ms in
// the verse and 141 in the rap. This asks only how a line divides between its
// own words, which needs no rate at all: within one line the singer holds one
// pace, whatever that pace is.

// syllables counts them in a word, in the language the words are in.
//
// Hungarian is exact: one vowel, one syllable, and the digraphs that look like
// vowels to a stranger — cs, sz, gy — are consonants and stay uncounted. English
// is a heuristic, vowel runs less the silent e, which is right about nine times
// in ten. Anything else is counted as vowel runs, which is the safer half of the
// English rule and holds for most languages written in this alphabet.
func syllables(word, lang string) int {
	// Hungarian counts every vowel and English counts runs of them. The
	// difference is diphthongs: English writes one sound with two letters —
	// "beautiful" has three syllables and five vowels in a row — and Hungarian
	// has none of them, so "fiatalság" is fi-a-tal-ság, four vowels and four
	// syllables. Counting runs there loses one on every second word.
	each := lang == "hu"

	runs := 0
	inVowel := false
	for _, r := range strings.ToLower(word) {
		// y is a vowel in English and never one in Hungarian, where it only
		// ever ends a digraph: gy, ly, ny, ty. "Gyász" is one syllable, and
		// counting the y makes it two.
		if vowel(r) && !(each && r == 'y') {
			if !inVowel || each {
				runs++
			}
			inVowel = true
			continue
		}
		inVowel = false
	}
	if runs == 0 {
		// A word with no vowel in it is still a word: "hmm", "rhythm", the
		// numeral in "1991". It is worth one beat rather than none, or the line
		// would sweep straight past it.
		if strings.IndexFunc(word, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0 {
			return 0
		}
		return 1
	}

	if lang == "en" && runs > 1 {
		w := strings.ToLower(strings.TrimFunc(word, func(r rune) bool { return !unicode.IsLetter(r) }))
		// The e that is not sounded: "times", "changed", "guard". Not after a
		// consonant and l ("little"), and not in a doubled vowel ("agree").
		if strings.HasSuffix(w, "e") && !strings.HasSuffix(w, "le") &&
			!strings.HasSuffix(w, "ee") && !strings.HasSuffix(w, "ye") {
			runs--
		}
		if strings.HasSuffix(w, "es") || strings.HasSuffix(w, "ed") {
			if len(w) > 3 && !vowel(rune(w[len(w)-3])) && !strings.HasSuffix(w, "ted") &&
				!strings.HasSuffix(w, "ded") && !strings.HasSuffix(w, "ses") && !strings.HasSuffix(w, "zes") {
				runs--
			}
		}
	}
	return max(runs, 1)
}

// lyricsSyllables is how many a whole line has, which is what a line has to sing
// rather than how long it has to sing it in.
func lyricsSyllables(line, lang string) int {
	total := 0
	for _, word := range strings.Fields(line) {
		total += syllables(word, lang)
	}
	return total
}

// vowel says whether a rune is one, accents and all. The Hungarian long vowels
// are here because they are vowels, and because a line of Hungarian counted
// without them comes out half the length it is.
func vowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'y',
		'á', 'é', 'í', 'ó', 'ö', 'ő', 'ú', 'ü', 'ű',
		'à', 'è', 'ì', 'ò', 'ù', 'â', 'ê', 'î', 'ô', 'û', 'ä', 'ë', 'ï', 'ã', 'õ', 'å', 'æ', 'ø':
		return true
	}
	return false
}

// sweepTo is how much of a line has been sung, given how far through its
// singing the voice is, rounded out to a whole word.
//
// The share is spent on syllables rather than on characters — see above — and a
// word lights as its turn comes rather than as it finishes, which is the same
// claim a progress bar makes and is what the eye follows anyway.
func sweepTo(line, lang string, frac float64) int {
	runes := []rune(line)
	if frac >= 1 {
		return len(runes)
	}

	type word struct{ from, to, syl int }
	var words []word
	total := 0
	for i := 0; i < len(runes); {
		for i < len(runes) && unicode.IsSpace(runes[i]) {
			i++
		}
		from := i
		for i < len(runes) && !unicode.IsSpace(runes[i]) {
			i++
		}
		if from == i {
			break
		}
		s := syllables(string(runes[from:i]), lang)
		words = append(words, word{from, i, s})
		total += s
	}
	if len(words) == 0 || total == 0 {
		return len(runes)
	}

	want := frac * float64(total)
	lit, seen := 0, 0
	for _, w := range words {
		if float64(seen) > want {
			break
		}
		lit = w.to
		seen += w.syl
	}
	return lit
}
