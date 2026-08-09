package ui

import (
	"strings"
	"unicode"
)

// The chorus, and what the screen makes of knowing that a line is one.
//
// Everything about how a line arrives is already dealt from the line itself —
// which way it comes in, whether it nods as one thing or a word at a time, which
// of its words lean and which way. So a chorus already arrives the same way
// every time it comes round, and has since the moves were written. What it does
// not do is show that it knows: a minute and a dozen lines separate one chorus
// from the next, and a rhyme nobody can hear is not a rhyme.
//
// Measured over thirty real sheets, half of what goes on this screen is a line
// that has been sung before — 44% of lines at the median, 85% at the top of the
// range, and only five of the thirty had no repeat at all. That is the largest
// thing the screen knows about the song and has never once used.
//
// Dealing the whole refrain one arrival was tried first and thrown out on the
// numbers: over the same thirty sheets it took the share of consecutive lines
// arriving in exactly the same way from 16% to 41%, which is a picture that has
// stopped surprising anybody by the second chorus.
//
// What is done instead is a shadow. A line that has been sung before is already
// faintly on the screen, in its place and complete, while its dots are still on
// their way to it — so the words burn up through their own imprint rather than
// arriving out of nothing. A verse comes out of the dark; the chorus comes back
// to where it was.

// refrainState is which lines of the sheet are returns, and the record they
// were worked out for.
type refrainState struct {
	forTrack string

	// again is keyed by the moment a line is sung, which is the one thing the
	// picture carries about the line it is showing. See wordsAgain.
	again map[int64]bool
}

// refrainFind works out which lines of the sheet have been sung before, once
// per record rather than once per frame.
//
// From the sheet rather than from what has been shown: a listener who drops the
// needle into the last chorus is in the last chorus, and the screen has no
// business pretending the song has not been there yet.
func (m *Model) refrainFind() {
	m.refrain = refrainState{forTrack: m.lyrics.forTrack}
	if len(m.lyrics.lines) == 0 {
		return
	}

	seen := make(map[string]bool, len(m.lyrics.lines))
	again := make(map[int64]bool)
	for _, line := range m.lyrics.lines {
		key := refrainKey(line.Words)
		if key == "" {
			continue
		}
		if seen[key] {
			again[line.At] = true
		}
		seen[key] = true
	}
	m.refrain.again = again
}

// refrainKey is what makes two lines the same line.
//
// Case and punctuation are dropped, because a sheet writes the chorus with a
// comma the second time as often as not, and the run of spaces with it. What is
// left is the words, which is what a listener hears come round again.
func refrainKey(line string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(line)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		case unicode.IsSpace(r):
			space = true
		}
	}
	return b.String()
}

// wordsAgain reports that what is on screen has been sung before in this
// record.
//
// The marks are never a return, whatever the sheet says about the bar they
// stand in: they are the same three notes every time by construction, and a
// shadow behind them would be on the screen for most of a record. Nor is the
// record's name, which is put up on purpose and has nothing to come back from.
func (m Model) wordsAgain() bool {
	if m.words.beats || m.words.telling || len(m.refrain.again) == 0 {
		return false
	}
	if m.ps == nil || m.refrain.forTrack != m.ps.TrackID {
		return false
	}
	return m.refrain.again[m.words.starts]
}
