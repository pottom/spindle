package ui

import "time"

// A record with no words of its own.
//
// Most of what gets played is not in any lyric database — instrumentals, live
// takes, half of everything electronic, and anything the sheet was never
// written for. On those the lyric screen has nothing to set, and what it used
// to do was put the title up for five seconds and then hand the rest of the
// record to one picture and leave it there. Three minutes of the same meter.
//
// What it did after that was take turns: the record cut into spells with the
// artist set at the top of one and the album at the top of the next. That is
// gone too, and the reason is worth keeping written down, because it was asked
// for twice in different words. Nothing goes up on this screen unless it is
// sent for. A card that arrives on a schedule of its own is a caption nobody
// asked for over a picture somebody was watching, and from a sofa there is no
// difference between the record's name, the artist's and the album's — they are
// all writing that appeared by itself. The name is on a key, and everything
// that is worth reading is on it with it. See soloName.
//
// So a wordless record is one long solo, all of it: the marks, keeping time
// with the music and dancing with each other. What stops that being three
// minutes of one thing is that the row performs — see crew.go — and that the
// bar under it is stamped afresh every spell, which deals it a new lean and a
// new chance of a visitor.

// wordsSpell is how long one turn lasts. Long enough to be a stretch of a
// record rather than a change of picture, short enough that a record of any
// length gets several.
const wordsSpell = 30 * time.Second

// wordsWordless reports that the lyric database has nothing for what is
// playing, or has not answered yet.
//
// The waiting used to count as neither, and the reasoning was that nothing
// should be put up on spec: at the top of a record whose sheet was a second
// late, the marks would arrive for that second and the first line would come
// straight over them, which is two changes of picture where the record made
// one.
//
// It was the wrong way round, and what was on the screen instead settled it.
// Nothing to put up meant the picture drawn when nothing is set — the meter
// above and below and an empty band across the middle — for as long as the
// answer took. Watched at the top of every record: the marks of the record
// before, then a hole, then the marks again. A hole is a change of picture too,
// and a worse one.
//
// And the marks are not a guess. Nearly every record opens with a stretch
// nobody sings — the intro is a gap like any other, and the marks keep it — so
// putting them up while the answer is on its way is putting up what is about to
// be there anyway.
func (m Model) wordsWordless() bool {
	if m.ps == nil {
		return false
	}
	if m.lyrics.forTrack != m.ps.TrackID {
		return true // no answer yet: the marks keep it until there is one
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

	spell, _ := m.wordsSpells()
	return m.wordsIdleMarks(spell)
}

// wordsIdleMarks is what a wordless record has up: the marks, the same as a bar
// of a song nobody is singing.
//
// Which is what a record with no words is, from here — one long solo. The bar
// is stamped at the top of each spell rather than once for the whole record,
// and that stamp is what the rest of the screen deals from: which way the marks
// lean, and whether a figure comes to walk through them. So a five minute
// instrumental is ten hands rather than one, without anything being written
// across it.
func (m Model) wordsIdleMarks(spell int) ([]string, int64) {
	return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)},
		int64(spell) * wordsSpell.Milliseconds()
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

	return m.drawCellsIn(w, rows, grid, paint, hue, m.styles.Words, m.styles.WordsSeq)
}
