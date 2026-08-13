package ui

import (
	"fmt"
	"time"
)

// Following the singer along the line, on the big screen.
//
// This screen used to claim exactly this — a light travelling word by word —
// and it was taken out for a good reason, written down at the time: a lyric
// sheet says when a line starts and nothing else, so where the singer has got to
// inside it was a guess, and a guess that is right most of the time is worse
// than none, because the times it is wrong are the times somebody is looking
// straight at it.
//
// What has changed is that it is no longer a guess. Thirty-six lines were timed
// by ear against the playhead and the answer is in FINDINGS.md: a line is sung
// for 85% of its window and never more than three seconds, the light is spent on
// syllables rather than letters, and the stamp itself sits about 200 ms in front
// of the voice. What is left is a median error of a fifth of a second, which is
// small enough that a movement can be hung on it — provided the movement is one
// that a fifth of a second does not make a liar of.
//
// So the three below are deliberately different in how much they claim, and the
// screen rotates them line by line so that they can be watched against each
// other on the same record:
//
//   - ink, which fills the line in as it is sung and claims a position;
//   - heat, which glows around where the voice is and claims a neighbourhood;
//   - lift, which raises the word being sung and claims a moment.
//
// None of them may take a movement that is already spoken for. The height of the
// dots is the sound's, the shove round the colour wheel is the beat's; what was
// free was brightness, colour and — for one word at a time, briefly — a rise.

// syncMode is where the key has got to: off, one of the four held so it can be
// watched, or all of them in turn.
//
// Held rather than rotating was the second arrangement. The first rotated them
// line by line, on the reasoning that four effects on one record is the only way
// to compare them — and it made the loudest of them impossible to find, because
// a burst lands on one line in four and is over inside a third of a second. A
// thing being judged has to be holdable.
type syncMode int

const (
	syncOff syncMode = iota
	syncModeInk
	syncModeHeat
	syncModeLift
	syncModeBurst

	// syncRotate is the original arrangement, kept for the end: once each of
	// them has been seen on its own, the way they follow one another is the
	// last question.
	syncRotate

	syncModes
)

func (s syncMode) next() syncMode { return (s + 1) % syncModes }

func (s syncMode) String() string {
	switch s {
	case syncModeInk:
		return "ink"
	case syncModeHeat:
		return "heat"
	case syncModeLift:
		return "lift"
	case syncModeBurst:
		return "burst"
	case syncRotate:
		return "all of them, a line each"
	default:
		return "off"
	}
}

// syncEffect is which of the four a line is drawn with.
type syncEffect int

const (
	syncInk syncEffect = iota
	syncHeat
	syncLift

	// syncBurst is the loud one: the word shakes and throws sparks off itself.
	// See wordburst.go.
	syncBurst

	syncEffects
)

func (e syncEffect) String() string {
	switch e {
	case syncInk:
		return "ink"
	case syncHeat:
		return "heat"
	case syncLift:
		return "lift"
	default:
		return "burst"
	}
}

const (
	// syncGhostFaint and syncGhostDark are how much of its brightness a word
	// that has not been sung yet keeps, under the two modes.
	syncGhostFaint = 0.55
	syncGhostDark  = 0.18

	// syncHeatSpread is how far either side of the voice the glow reaches, in
	// words. Wider than a word on purpose: the timing is good to about a fifth
	// of a second, and a glow that spans the uncertainty tells the truth where a
	// spotlight would tell a lie.
	syncHeatSpread = 1.6

	// syncHeatGain is how much brighter the word under the voice is than the
	// line it stands in.
	syncHeatGain = 0.45

	// syncLiftDots is how far the word being sung rises, in dots, and
	// syncLiftFall how much of that is left a word later. It comes back down
	// under the same gravity everything else on this screen obeys.
	syncLiftDots = 3.0
	syncLiftFall = 0.35
)

// wordsSyncOn reports whether the singer is being followed at all.
func (m Model) wordsSyncOn() bool {
	return m.words.sync != syncOff && m.lyrics.synced && m.words.where.Count > 0
}

// wordsSyncEffect is which of the four this line is drawn with: whichever is
// being held, or the next in turn where they are rotating.
func (m Model) wordsSyncEffect() syncEffect {
	if m.words.sync == syncRotate {
		return syncEffect(m.words.line % int(syncEffects))
	}
	return syncEffect(int(m.words.sync) - int(syncModeInk))
}

// wordsSyncSpan is how long this line is sung for, and how much of that has
// gone. It reports false where there is nothing to follow.
//
// The model is the one the player screen's sweep uses, measured the same way —
// see lyricsSung and lyricsStampsEarly. This screen reads it against its own
// clock, which runs with the music rather than ahead of it.
func (m Model) wordsSyncSpan() (gone, sung time.Duration, ok bool) {
	if !m.wordsSyncOn() || m.words.starts <= 0 {
		return 0, 0, false
	}
	window := lyricsDefaultLine
	if m.words.ends > m.words.starts {
		window = time.Duration(m.words.ends-m.words.starts) * time.Millisecond
	}
	if sung = lyricsSung(window); sung <= 0 {
		return 0, 0, false
	}
	return max(time.Duration(m.wordsClock()-m.words.starts)*time.Millisecond-lyricsStampsEarly, 0), sung, true
}

// wordsSyncShares is what each piece of the line is worth in time, in syllables.
//
// The two screens shared a line out by two different rulers, and it was noticed
// before it was measured: the player screen spends the singing on syllables, and
// this one spent it on pieces — every word and every comma an equal slice of it.
// The picture is cut into pieces because a comma has to be able to move on its
// own, and that is worth keeping; what it is not is worth a share of the voice.
// Measured on real lines against the player screen's ruler, the two disagreed on
// where the last word of a line starts by up to 478 ms — the same line, the same
// second, one screen above the other.
//
// So the pieces stay and the ruler goes: a piece is worth its syllables, a mark
// is worth none and lights with the word beside it, and both screens now put the
// voice in the same place.
func (m Model) wordsSyncShares() []float32 {
	count := m.words.where.Count
	if count <= 0 || m.words.text == "" || m.words.beats {
		return nil
	}

	// Cut the same way the picture was cut, from the same text. Anything else
	// and the shares would belong to pieces that are not on the screen, so a
	// count that does not match is a count that is not followed.
	pieces := wordsPieces(m.words.text)
	if len(pieces) != count {
		return nil
	}

	out := make([]float32, count)
	var total float32
	for i, p := range pieces {
		out[i] = float32(syllables(m.words.text[p.from:p.to], m.lyrics.language))
		total += out[i]
	}
	if total <= 0 {
		return nil
	}
	return out
}

// wordsSyncWalk is where the voice has got to, in pieces, when it is this far
// through the line's singing.
//
// A piece worth nothing is stepped straight over, which is what a comma should
// be: the voice does not stop at it, so it lights as its neighbour does rather
// than holding the line up for a slice of a second.
func wordsSyncWalk(shares []float32, frac float32) float32 {
	var total float32
	for _, share := range shares {
		total += share
	}
	if total <= 0 {
		return float32(len(shares)) * min(frac, 1)
	}

	want := min(frac, 1) * total
	var seen float32
	for i, share := range shares {
		if share > 0 && want < seen+share {
			return float32(i) + (want-seen)/share
		}
		seen += share
	}
	return float32(len(shares))
}

// wordsSyncAt is how far through its singing the line is, in pieces: 0 as the
// first word begins, count as the last has finished. It reports false where
// there is nothing to follow.
func (m Model) wordsSyncAt() (float32, bool) {
	gone, sung, ok := m.wordsSyncSpan()
	if !ok {
		return 0, false
	}
	frac := float32(gone) / float32(sung)
	if shares := m.wordsSyncShares(); shares != nil {
		return wordsSyncWalk(shares, frac), true
	}
	return min(frac, 1) * float32(m.words.where.Count), true
}

// wordsSyncPaint adjusts a word's brightness for where the voice is.
//
// level is what the sound gave it and levels how many steps there are. What
// comes back is the level after the effect has had its say, which is all the ink
// and the heat are: neither moves anything, and a picture that only changes
// brightness cannot be made to jitter by a timing that is a fifth of a second
// out.
func (m Model) wordsSyncPaint(word int, level int8, levels int) int8 {
	at, ok := m.wordsSyncAt()
	if !ok {
		return level
	}

	ghost := float32(syncGhostFaint)
	if m.words.syncDark {
		ghost = syncGhostDark
	}

	var scale float32
	switch m.wordsSyncEffect() {
	case syncInk:
		// Filled in behind the voice and a ghost ahead of it. The word being
		// sung is the first one at full strength, so the line reads as ink
		// running into it.
		if float32(word) < at {
			scale = 1
		} else {
			scale = ghost
		}

	case syncHeat:
		// A glow around where the voice is, falling away either side. The line
		// behind it does not stay lit: what this claims is a neighbourhood, not
		// a position, and a lit tail would be the position again.
		d := float32(word) + 0.5 - at
		if d < 0 {
			d = -d
		}
		near := max(1-d/syncHeatSpread, 0)
		scale = ghost + (1-ghost)*near + syncHeatGain*near*near

	default:
		// The lift and the burst move the word instead of lighting it, so the
		// brightness is only the ghost and the plain line behind it.
		if float32(word) < at {
			scale = 1
		} else {
			scale = ghost
		}
	}

	out := int8(min(float32(level)*scale, float32(levels-1)))
	if out < 0 {
		out = 0
	}
	return out
}

// wordsSyncLifts is how far each word is raised for being the one in the
// singer's mouth, in dots. Nil where nothing is being raised.
//
// Up, and then down under its own weight: the word the voice has just left is
// still on its way back while the next is rising, which is what makes a line
// read as being spoken along rather than as a row of lamps being switched.
//
// A whole line at a time, once a frame, because the drawer asks about every lit
// dot on the screen — thirty thousand of them at a decent size, thirty times a
// second. Asked one dot at a time this cost 12 ms a frame against 4.8 with it
// off, which is the difference between a picture and a slideshow; the same
// arithmetic done per word is free. The riding and the tilt are arrays for
// exactly this reason.
func (m Model) wordsSyncLifts(count int) []int {
	if count <= 0 || m.wordsSyncEffect() != syncLift {
		return nil
	}
	at, ok := m.wordsSyncAt()
	if !ok {
		return nil
	}

	out := make([]int, count)
	for word := range out {
		d := at - float32(word) - 0.5
		switch {
		case d < -1: // not reached yet
		case d < 0: // rising into it
			out[word] = -int(syncLiftDots * (1 + d))
		default: // falling back out of it
			out[word] = -int(syncLiftDots * pow(syncLiftFall, d))
		}
	}
	return out
}

// wordsSyncLabel names what is on this line, at the foot of the screen.
func (m Model) wordsSyncLabel(w int) string {
	ahead := "ahead faint"
	if m.words.syncDark {
		ahead = "ahead dark"
	}
	at, _ := m.wordsSyncAt()
	held := m.words.sync.String()
	if m.words.sync == syncRotate {
		held = "rotating · " + m.wordsSyncEffect().String()
	}
	return fit(m.styles.Empty.Render(fmt.Sprintf("  %s · %s · word %.1f of %d · y next, Y darker",
		held, ahead, at, m.words.where.Count)), w)
}

// pow is x to the y for the small non-integer powers the fall needs, without
// pulling in the whole of math for one call site.
func pow(x, y float32) float32 {
	out := float32(1)
	for range int(y) {
		out *= x
	}
	if f := y - float32(int(y)); f > 0 {
		out *= 1 - f*(1-x)
	}
	return out
}
