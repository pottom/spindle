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

	// soloOpen is how far into a record the title may first appear, and soloTail
	// how close to the end. The top of a track is where it was not wanted, and
	// the end of one is where the next record is already coming up underneath.
	soloOpen = 20 * time.Second
	soloTail = 10 * time.Second

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

// soloCard is when the record says its own name: the middle of the longest bar
// it has room for one in.
//
// The longest rather than the first, because the first might be twelve seconds
// and the second forty. The longest is the one that reads as deliberate — it is
// the solo, and it is where a film would put the title.
func (m Model) soloCard() (soloGap, bool) {
	var best soloGap
	var found bool

	for _, gap := range m.soloGaps() {
		if gap.long() < soloLeast.Milliseconds() {
			continue
		}
		if gap.from < soloOpen.Milliseconds() {
			continue
		}
		if m.ps != nil && m.ps.Duration > 0 && gap.to > (m.ps.Duration-soloTail).Milliseconds() {
			continue
		}
		if !found || gap.long() > best.long() {
			best, found = gap, true
		}
	}
	if !found {
		return soloGap{}, false
	}

	middle, half := (best.from+best.to)/2, soloTells.Milliseconds()/2
	return soloGap{middle - half, middle + half}, true
}

// soloForcing reports that the card was asked for by hand.
//
// A record's own moment comes once and may not come at all, which is the point
// of it — but it does mean there is no way to look at the thing on purpose. The
// key puts the same card up, drawn the same way, wherever the record is.
func (m Model) soloForcing() bool {
	return !m.words.forced.IsZero() && time.Since(m.words.forced) < soloTells
}

// soloTelling reports that the moment is now.
func (m Model) soloTelling() bool {
	if m.soloForcing() {
		return true
	}
	if _, in := m.soloNow(); !in {
		return false
	}
	card, ok := m.soloCard()
	if !ok {
		return false
	}
	clock := m.wordsClock()
	return clock >= card.from && clock < card.to
}

// soloName is what the record is called, as it is set: its own name and whose
// it is, one under the other.
func (m Model) soloName() []string {
	if m.ps == nil || m.ps.Title == "" {
		return nil
	}
	lines := []string{m.ps.Title}
	if len(m.ps.Artists) > 0 {
		lines = append(lines, strings.Join(m.ps.Artists, ", "))
	}
	return lines
}
