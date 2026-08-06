package ui

import (
	"strings"
	"time"
)

// A record with no words of its own.
//
// Most of what gets played is not in any lyric database — instrumentals, live
// takes, half of everything electronic, and anything the sheet was never
// written for. On those the lyric screen has nothing to set, and what it used
// to do was put the title up for five seconds and then hand the rest of the
// record to one picture and leave it there. Three minutes of the same meter.
//
// So it takes turns instead. The record is cut into spells; something is set at
// the top of each one for a few seconds and the music has the rest of it, drawn
// as the mirrored meter one spell and as the stack of lamps the next. What goes
// up is dealt from the track, never the same card twice running, so a record
// plays the same way twice and still does not repeat itself.

// wordsSpell is how long one turn lasts. Long enough that a card is an event
// rather than a caption, short enough that a record of any length gets several.
const wordsSpell = 30 * time.Second

// The cards a wordless record is dealt.
type wordsCard int

const (
	wordsCardNone   wordsCard = iota // the music, and nothing over it
	wordsCardTitle                   // the record saying its own name
	wordsCardArtist                  // and whose it is
	wordsCardAlbum                   // and what it came out on
	wordsCardNotes                   // and three notes, for a bar with nothing to say
)

// wordsCardPool is what a turn can be dealt after the record has said its name.
// The title is not in it: a record says what it is once. See soloCard for the
// same rule where there are words to get out of the way of.
var wordsCardPool = []wordsCard{wordsCardArtist, wordsCardAlbum, wordsCardNotes}

// wordsWordless reports that the lyric database has nothing for what is
// playing — or has not answered about it yet, which at the top of a record
// looks the same from here and is exactly when the title card is due.
func (m Model) wordsWordless() bool {
	if m.ps == nil {
		return false
	}
	if m.lyrics.forTrack != m.ps.TrackID {
		return true
	}
	return m.lyrics.missing || !m.lyrics.synced
}

// wordsSpells is which turn the record is in and how far into it, so the two
// halves of a spell — the card and the music — are worked out from one clock.
func (m Model) wordsSpells() (int, time.Duration) {
	gone := max(m.elapsed(), 0)
	return int(gone / wordsSpell), gone % wordsSpell
}

// wordsIdle is what a wordless record puts up where a sung line would go, and
// when that turn began — which is what the picture takes for the line's
// identity, and so what decides how it gathers, whether it keeps time and
// whether it leans.
func (m Model) wordsIdle() ([]string, int64) {
	if !m.wordsWordless() {
		return nil, 0
	}

	spell, into := m.wordsSpells()
	if into > wordsTitle {
		return m.wordsIdleMarks(spell) // the rest of the turn keeps time on its own
	}

	var lines []string
	switch m.wordsCardFor(spell) {
	case wordsCardNone:
	case wordsCardTitle:
		lines = m.soloName()
	case wordsCardArtist:
		lines = m.wordsCard(strings.Join(m.ps.Artists, ", "))
	case wordsCardAlbum:
		// A single is its own record, and setting the same name twice in a row
		// under two different headings is a screen with a bug in it.
		if m.ps.Album != m.ps.Title {
			lines = m.wordsCard(m.ps.Album)
		}
	case wordsCardNotes:
		lines = []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)}
	}

	if len(lines) == 0 || lines[0] == "" {
		return m.wordsIdleMarks(spell)
	}
	return lines, int64(spell) * wordsSpell.Milliseconds()
}

// wordsIdleMarks is what a wordless record has up the rest of the time: the
// three marks, the same as a bar of a song nobody is singing.
//
// Which is what a record with no words is, from here — one long solo. The cards
// take the marks' place for a few seconds and hand it back, exactly as the
// record's name does in the middle of a real one, and the screen is the same
// picture throughout instead of a different one every half minute.
func (m Model) wordsIdleMarks(spell int) ([]string, int64) {
	// Stamped after the card that opens the turn, so the marks arrive again
	// once it has gone rather than sitting through it.
	return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)},
		int64(spell)*wordsSpell.Milliseconds() + wordsTitle.Milliseconds()
}

// wordsCardFor is the card a turn gets.
//
// The first turn gets nothing: the top of a record is where it changed, and a
// caption there is a caption on something nobody has started listening to. The
// second is the record saying its own name, once and never again. After that
// they are dealt from what is left, each stepping on from the last by at least
// one, so the same card never comes up twice running.
func (m Model) wordsCardFor(spell int) wordsCard {
	switch spell {
	case 0:
		return wordsCardNone
	case 1:
		return wordsCardTitle
	}

	var at int
	for turn := 2; turn <= spell; turn++ {
		h := wordsDeal(m.ps.TrackID, turn)
		at = (at + 1 + int(h%uint64(len(wordsCardPool)-1))) % len(wordsCardPool)
	}
	return wordsCardPool[at]
}

// wordsDeal is the number a track's nth turn is dealt from.
func wordsDeal(track string, turn int) uint64 {
	var h uint64 = 0xcbf29ce484222325
	for _, r := range track {
		h = (h ^ uint64(r)) * 0x100000001b3
	}
	h ^= uint64(turn) * 0x9e3779b97f4a7c15

	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 29
	return h
}

// wordsIdleArt is what is drawn while nothing at all is set.
//
// The mirrored meter, and nothing else. It used to take turns with the stack of
// lamps, which was a different program on screen every half minute; and a
// wordless record hardly comes here any more anyway, because the marks have the
// screen when no card is up. What is left is the gap between two lines of a
// song, where the meter jumping to the middle and springing apart again as the
// next line lands is the whole of the effect.
func (m Model) wordsIdleArt(w, rows int) []string {
	return m.stageArt(w, rows)
}
