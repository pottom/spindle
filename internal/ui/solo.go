package ui

import (
	"strings"
	"time"
)

// The bars nobody sings, and the one moment a record says its own name.
//
// A lyric sheet is a list of moments with words against them, and the gaps
// between those moments are the song playing on its own: the intro, the solo,
// the middle eight, the two bars before the last chorus. Until now the screen
// held the last line up through all of it, because a line was only ever given
// up when the next one arrived — so a forty second solo was spent looking at
// something somebody sang a minute ago.
//
// The gaps are worth having for themselves. In a long one the words give way to
// the marks, which keep time with the music the way the words do, and in exactly
// one of them — the longest the record has — the screen says what is playing.
//
// Once, and only if there is room. A title held up at the top of every track is
// a caption on a picture nobody is looking at yet; a title that appears halfway
// through a solo, stands still for four seconds while everything around it
// moves, and gives the bar back to the marks, is a title card. Records without
// a solo do not get one, which is what makes it worth seeing on the ones that
// do.

const (
	// soloHold is the longest a sung line is left up once the singer has left
	// it. Ordinary lines follow each other well inside this, so nothing changes
	// for them; it is what stops the last line of a verse standing over a solo.
	soloHold = 6 * time.Second

	// soloLeast is the shortest bar of nothing that counts as a solo. Under
	// this, putting the marks up and taking them down again is a flicker rather
	// than a change, and there is no room for a title card between them.
	soloLeast = 12 * time.Second

	// soloTells is how long the record's name stands there.
	soloTells = 4500 * time.Millisecond
)

// soloGap is a stretch of a record with nothing sung in it.
type soloGap struct{ from, to int64 } // milliseconds

func (g soloGap) long() int64 { return g.to - g.from }

// soloGaps is every stretch of the record nobody sings in.
//
// Taken from the moments that do have words rather than from the ones that do
// not: sheets disagree about how a rest is written down — some leave a line
// empty, some put a mark on it, most write nothing at all and leave a minute
// between two verses — and the times somebody sings are the one thing all of
// them agree on.
func (m Model) soloGaps() []soloGap {
	if !m.lyrics.synced {
		return nil
	}

	var sung []int64
	for _, line := range m.lyrics.lines {
		if words := strings.TrimSpace(line.Words); words != "" && !wordsBeats(words) {
			sung = append(sung, line.At)
		}
	}
	if len(sung) == 0 {
		return nil
	}

	// The intro, and then whatever is left between each line and the next once
	// the line has had its time.
	out := []soloGap{{0, sung[0]}}
	for i := 1; i < len(sung); i++ {
		out = append(out, soloGap{sung[i-1] + soloHold.Milliseconds(), sung[i]})
	}

	// And the run-out, which is a gap like any other and was missing.
	//
	// A sheet stops at the last line somebody sings, and plenty of records go on
	// for a minute after it — an outro, a fade, a whole instrumental coda. With
	// nothing after the last line to make a gap against, the screen had nothing
	// to put up and fell through to the picture it draws when nothing at all is
	// set: the meter above and below and an empty band across the middle, for
	// the rest of the record. Watched on Mike Mana's "Never The Same", that is
	// everything from 1:55 to the end, every time it is played.
	if m.ps != nil {
		// Where the singing stopped. Most sheets write that down: an entry with
		// no words in it, after the last line anybody sings. Measured on Mike
		// Mana's "Never The Same", the last line is at 1:42.9 and the sheet's
		// own full stop at 1:44.5 — and taking the line's six seconds instead
		// left four and a half seconds that were neither a line nor a gap, with
		// the empty picture in them.
		last := sung[len(sung)-1] + soloHold.Milliseconds()
		for _, line := range m.lyrics.lines {
			if line.At <= sung[len(sung)-1] {
				continue
			}
			if strings.TrimSpace(line.Words) == "" {
				last = min(last, line.At)
				break
			}
		}
		if end := m.ps.Duration.Milliseconds(); end > last {
			out = append(out, soloGap{last, end})
		}
	}
	return out
}

// soloNow is the bar of nothing the record is in, if it is a long one.
//
// It closes a gathering early: the last moment of a gap belongs to the line
// coming after it, which starts pulling itself together before it is sung.
func (m Model) soloNow() (soloGap, bool) {
	clock := m.wordsClock()
	for _, gap := range m.soloGaps() {
		if gap.long() < soloLeast.Milliseconds() {
			continue
		}
		if clock >= gap.from && clock < gap.to-wordsGather.Milliseconds() {
			return gap, true
		}
	}
	return soloGap{}, false
}

// soloForcing reports that the card was asked for by hand, which is the only
// way it is ever put up.
func (m Model) soloForcing() bool {
	return !m.words.forced.IsZero() && time.Since(m.words.forced) < soloTells
}

// soloTelling reports that the record's name is what is on screen.
//
// Only ever because somebody asked for it. It used to go up by itself as well,
// once a record, in the middle of the longest solo — and by itself is exactly
// where it went wrong: a record can be taken for wordless while its sheet is on
// its way, say its name for that, and then say it again when the sheet turns up
// with a solo in the middle of it. A name that appears on its own schedule is
// also a name that appears over something you were reading.
//
// So it does not appear at all unless it is sent for. See stageKey.
func (m Model) soloTelling() bool { return m.soloForcing() }

// soloName is what the record is called, as it is set: its own name and whose
// it is, one under the other, each broken where it is too long for the room.
func (m Model) soloName() []string {
	if m.ps == nil || m.ps.Title == "" {
		return nil
	}
	return m.wordsCard(m.ps.Title, strings.Join(m.ps.Artists, ", "))
}
