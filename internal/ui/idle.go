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
	wordsCardArtist                  // whose the record is
	wordsCardAlbum                   // and what it came out on
)

// wordsCardPool is what a turn can be dealt.
//
// The record's own name is not in it. It goes up when it is sent for and at no
// other time — a name that appears on its own schedule appears twice as often
// as anybody wants it to, and over things you were reading. See soloTelling.
//
// Neither are the marks. They have the rest of every turn anyway, so dealing
// them as the thing that opens one put the same picture up twice in a row —
// marks arriving, and then the very same marks arriving again a moment later
// by another road. A turn dealt nothing is the music with the marks over it for
// the whole of it, which is what that card was reaching for.
var wordsCardPool = []wordsCard{wordsCardArtist, wordsCardAlbum, wordsCardNone}

// wordsWordless reports that the lyric database has nothing for what is
// playing.
//
// A record it has not answered about yet is not one of them. It used to be
// counted as wordless the moment it started, on the grounds that the two look
// the same from here — and they do, right up until the answer lands. What that
// produced at the top of a record whose sheet was a second late was the marks
// arriving for that second and the first line coming straight over them: two
// changes of picture where the record had made one. So nothing is put up on
// spec; whatever the screen was showing keeps it until there is an answer to
// act on.
func (m Model) wordsWordless() bool {
	if m.ps == nil || m.lyrics.forTrack != m.ps.TrackID {
		return false
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

	starts := int64(spell) * wordsSpell.Milliseconds()

	var lines []string
	switch m.wordsCardFor(spell) {
	case wordsCardNone:
	case wordsCardArtist:
		lines = m.wordsCard(strings.Join(m.ps.Artists, ", "))
	case wordsCardAlbum:
		// A single is its own record, and setting the same name twice in a row
		// under two different headings is a screen with a bug in it.
		if m.ps.Album != m.ps.Title {
			lines = m.wordsCard(m.ps.Album)
		}
	}

	if len(lines) == 0 || lines[0] == "" {
		return m.wordsIdleMarks(spell)
	}
	return lines, starts
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
// caption there is a caption on something nobody has started listening to.
// After that they are dealt from the pool, each stepping on from the last by at
// least one, so the same card never comes up twice running.
func (m Model) wordsCardFor(spell int) wordsCard {
	if spell <= 0 {
		return wordsCardNone
	}

	var at int
	for turn := 1; turn <= spell; turn++ {
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

// wordsIdleArt is what is drawn while nothing at all is set: the meter and the
// water, with nothing in the middle for them to make room for.
//
// It used to be the mirrored spectrum, which is a different picture from every
// other one this screen draws — a second idiom on a screen that is supposed to
// have one, and the moments it filled were exactly the moments a record changes
// over. It also hid the change: its columns run to white wherever the music is
// loud, so the accent that the rest of the screen is coloured by, and that the
// next record's colour arrives in, had nowhere to show.
//
// What is left is the picture underneath every other picture here. The middle
// is empty because there is nothing to put in it, and when there is, it lands
// in a screen that was already its own.
func (m Model) wordsIdleArt(w, rows int) []string {
	if w <= 0 || rows <= 0 || len(m.styles.Words) == 0 {
		return nil
	}

	freqs := len(m.styles.Words)
	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}
	for r := range rows {
		for c := range w {
			hue[r*w+c] = int8(min(c*freqs/w, freqs-1))
		}
	}

	// The room a line would have left, though there is no line: the middle band
	// stays empty rather than being filled because nothing is using it. What
	// lands there later lands in a screen that was already its own, instead of
	// pushing a picture out of the way.
	tall, head := m.wordsBandNow(w, rows)
	if tall < wordsBand {
		dotsY := rows * dotsPerCellY
		band := int(wordsMark * float64(dotsY))
		tall = max((dotsY-((dotsY-band)/2+band))/dotsPerCellY, wordsBand)
		head = max((dotsY-band)/2-dotsPerCellY, 0)
	}
	m.wordsUnder(grid, paint, hue, w, rows, tall, head)

	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}
