package ui

import (
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// Somebody dancing in the bar between the lines.
//
// The wordless bar has always been a row of marks: eight small drawings, each
// riding its own slice of the spectrum, leaning on the beat and turning round on
// its own count. This is the other thing that can stand there — one figure, at
// the size of the room, doing a move.
//
// He is dealt like the marks are, from the record and from where in it the bar
// falls, so a record dances the same way twice. What is different is that he
// has his own clock: a move is frames, and the frames step. Which is why he is
// the one picture here that does not ride the spectrum — see moves.go for the
// argument, and the rule it comes from.

const (
	// danceEvery is how often the bar is given to the dancer rather than to the
	// marks, as one in this many.
	//
	// One in three. The row of marks is what this screen has rested on since the
	// beginning and a dancer every time would make a novelty of the record's own
	// quiet passages; one in three is often enough that a wordless record has
	// him up two or three times and rare enough that he is still an event.
	danceEvery = 3

	// danceBeatsPerLoop is how long one turn of a move takes, in beats.
	//
	// Four, which is a bar of most music. A backspin that goes round in a bar
	// reads as a backspin; the same twelve frames in one beat is a blur, and in
	// eight it is a man thinking about it.
	danceBeatsPerLoop = 4

	// danceRoundsLeast and danceRoundsMost are how many turns of the loop a move
	// is dealt. Between them a move lasts two bars and eight, which is the
	// length of a passage anybody would call a passage.
	danceRoundsLeast = 2
	danceRoundsMost  = 8

	// danceHigh is how much of the screen's height the figure stands in.
	//
	// Not all of it: the meter and its water need the foot of the screen, and a
	// figure drawn to the ceiling has his head in the record's name.
	danceHigh = 0.62
)

// danceState is what the dancer carries between the bars.
type danceState struct {
	// move is what he is doing, empty when he is not up, and rounds how many
	// turns of its loop he was dealt.
	move   string
	rounds int

	// since is when the move began. The frames are counted off the beat rather
	// than off this, and this is where the counting starts from.
	since time.Time

	// picked is a move asked for by hand. A move dealt one bar in three, on a
	// record with four wordless bars in it, is a move nobody can judge; the key
	// walks them so one can be watched. See marksWalk.
	picked string
}

// danceCast is what the bar is called when it is his. It goes where a set of
// marks would be named, so the one field says what is standing there.
const danceCast = "dance"

// danceSet is the company the dancer is drawn from. One for now, and named
// rather than dealt: another company would be another character, and which
// character is up is not something to leave to chance in the middle of a record.
const danceSet = "break"

// danceCastFor says whether this bar is his, dealt from the record and from
// where in it the bar falls — so a record dances the same way twice, which is
// the rule every other deal on this screen keeps.
func danceCastFor(record string, starts int64) bool {
	if _, ok := moveSetFor(danceSet); !ok {
		return false
	}

	h := uint64(starts)*0x9e3779b97f4a7c15 + 0x94d049bb133111eb
	for _, c := range []byte(record) {
		h = (h ^ uint64(c)) * 0x100000001b3
	}
	h ^= h >> 29
	h *= 0xff51afd7ed558ccd
	h ^= h >> 32
	return h%danceEvery == 0
}

// danceDeal picks the move and how long it goes on for.
//
// The deal is from the record and the bar, as everything here is, but it is
// weighted by what the music is doing: the big moves want a loud passage. Not a
// rule, a lean — a hush can still throw a backspin, only not often.
//
// Loudness here is the swell rather than the decibels: where the record is
// inside the range it has been moving through lately, which is the one reading
// that can tell a chorus from a verse on any record. See swell.go.
func (m *Model) danceDeal(record string, starts int64) {
	set, ok := moveSetFor(danceSet)
	if !ok {
		return
	}
	names := set.names()
	if len(names) == 0 {
		return
	}

	h := uint64(starts)*0xd6e8feb86659fd93 + 0x9e3779b97f4a7c15
	for _, c := range []byte(record) {
		h = (h ^ uint64(c)) * 0x100000001b3
	}
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 29

	// How much the record is giving, nought to one: the swell where there is one
	// and the drive beside it, which is how hard the hits are landing. Two
	// readings of the same question, and either alone is quiet on some records.
	give := (min(max(float64(m.swell()), 0), 1.35)/1.35 + min(max(float64(m.words.drive), 0), 1)) / 2

	// Each move's weight is its size against what the record is giving: at a
	// hush the small ones are worth several times the big ones, and at full
	// tilt it is the other way round.
	weights := make([]float64, len(names))
	var total float64
	for i, name := range names {
		size := danceSizes[name]
		weights[i] = 0.25 + (1-size)*(1-give) + size*give
		total += weights[i]
	}

	want := float64(h%1000) / 1000 * total
	pick := names[len(names)-1]
	for i, name := range names {
		if want -= weights[i]; want < 0 {
			pick = name
			break
		}
	}

	// Unless one was asked for by name, which is how a move gets watched rather
	// than waited for. See marksWalk.
	if m.dance.picked != "" {
		if _, _, ok := set.at(m.danceTall(), m.dance.picked); ok {
			pick = m.dance.picked
		}
	}

	m.dance.move = pick
	m.dance.rounds = danceRoundsLeast + int((h>>32)%uint64(danceRoundsMost-danceRoundsLeast+1))
	m.dance.since = time.Now()
}

// danceSizes is how big a thing each move is, nought to one.
//
// Written down rather than measured off the drawings: how much of the room a
// move takes is not the same question as how much of an event it is. The bounce
// is what he does while nothing is happening and the backspin is what a record
// builds to, and no measurement of their ink says so.
var danceSizes = map[string]float64{
	"bounce":    0.0,
	"sweep":     0.3,
	"drop":      0.35,
	"sidekick":  0.5,
	"sixstep":   0.65,
	"headstand": 0.85,
	"backspin":  1.0,
}

// danceUp reports that the dancer is the picture in the wordless bar.
func (m Model) danceUp() bool { return m.words.dancing && m.dance.move != "" }

// danceStep is how far into the move he is, in frames.
//
// Counted off the beat and not off a clock: a move that takes the same number of
// seconds on every record is a move that is not dancing to any of them. Where
// there is no beat to count — the first seconds of a record, a hush, a stream
// that has stopped — he holds where he is. The row of marks rests the same way
// when the drums do, and resting is the picture rather than a fault in it.
func (m Model) danceStep() int {
	period := m.scope.beat.Period
	if !m.beatKeeping() || period <= 0 {
		return 0
	}

	set, ok := moveSetFor(danceSet)
	if !ok {
		return 0
	}
	_, d, ok := set.at(m.danceTall(), m.dance.move)
	if !ok {
		return 0
	}
	loop := d.span(d.loopFrom, d.loopTo)
	if loop <= 0 {
		return 0
	}

	// One turn of the loop is a bar, however many frames that turn was drawn in.
	each := time.Duration(float64(period) * danceBeatsPerLoop / float64(loop))
	if each <= 0 {
		return 0
	}
	return int(time.Since(m.dance.since) / each)
}

// danceTall is the height he is drawn at, in dots, at the size of the room.
func (m Model) danceTall() int {
	return int(float64(m.height*dotsPerCellY) * danceHigh)
}

// danceFrame is the drawing that is up, and where it stands.
func (m Model) danceFrame() (moveFrame, moveSize, bool) {
	set, ok := moveSetFor(danceSet)
	if !ok {
		return moveFrame{}, moveSize{}, false
	}
	size, d, ok := set.at(m.danceTall(), m.dance.move)
	if !ok {
		return moveFrame{}, moveSize{}, false
	}

	f, going := d.frameAt(m.danceStep(), m.dance.rounds)
	if !going {
		// The move is over and the bar is not: he stands there, which is what
		// the last frame of every sheet is drawn for.
		f = d.frames[len(d.frames)-1]
	}
	return f, size, true
}

// danceDone reports that the move has run out, so the bar can deal another.
func (m Model) danceDone() bool {
	set, ok := moveSetFor(danceSet)
	if !ok {
		return true
	}
	_, d, ok := set.at(m.danceTall(), m.dance.move)
	if !ok {
		return true
	}
	_, going := d.frameAt(m.danceStep(), m.dance.rounds)
	return !going
}

// dancePicture is the first frame as a picture, so that the bar has something
// to arrive with and the meter under it has something to measure against.
//
// Only the first: what is drawn every frame after it comes from the baked
// frames rather than from here. This is the still the screen adopts — see
// wordsAdopt, and danceDraw for what is actually on the screen.
func (m Model) dancePicture(w, rows int) (cover.Grain, msg.WordLayout, bool) {
	f, size, ok := m.danceFrame()
	if !ok || w <= 0 || rows <= 0 {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	left, floor := danceStands(size, dotsX, dotsY)

	g := cover.Grain{DotsX: dotsX, DotsY: dotsY, CellsX: w, CellsY: rows, Lum: make([]uint8, dotsX*dotsY)}
	f.draw(0, size.wide, func(x, y int) {
		px, py := left+x, floor-f.tall+y
		if px >= 0 && py >= 0 && px < dotsX && py < dotsY {
			g.Lum[py*dotsX+px] = 255
		}
	})

	// One piece, and it covers him: the rest of this screen asks the layout
	// which word a dot belongs to, and the answer for a dancer is always the
	// same one.
	layout := msg.WordLayout{
		Count: 1, DotsX: dotsX,
		At:      make([]int16, dotsX),
		Tops:    []int{max(floor-f.tall, 0)},
		Bottoms: []int{min(floor, dotsY-1)},
		Lefts:   []int{left},
		Rights:  []int{left + size.wide - 1},
	}
	for x := range layout.At {
		layout.At[x] = -1
		if x >= left && x < left+size.wide {
			layout.At[x] = 0
		}
	}
	layout.Settle()
	return g, layout, true
}

// danceStands is where he stands: across the middle of the screen, and on a
// floor low enough to leave the meter its band.
func danceStands(size moveSize, dotsX, dotsY int) (left, floor int) {
	return (dotsX - size.wide) / 2, (dotsY + size.tall) / 2
}

// danceDraw puts the frame that is up on the screen.
//
// Drawn straight into the grid rather than through a picture of his own: a
// figure is a few thousand lit dots and the screen is a quarter of a million, so
// grinding him into a canvas every frame would be most of the work for none of
// the answer.
//
// The gathering is the same one everything else here arrives by — he assembles
// out of the air like a line of type does — because the bar he stands in is the
// same bar, and a picture that arrives differently reads as a different screen
// rather than as a different picture.
func (m Model) danceDraw(grid []uint8, paint, hue []int8, w, rows int, gather float32, freqs, levels int) {
	f, size, ok := m.danceFrame()
	if !ok {
		return
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	left, floor := danceStands(size, dotsX, dotsY)

	// One colour for the whole of him, and one brightness: the dance is the
	// answer to the music here, so the light is not also answering it. Which
	// band he takes his colour from is the middle one, so he is neither the bass
	// nor the cymbals but the record.
	tone := int8(min(freqs/2, freqs-1))
	burn := int8(levels - 1)
	if gather < 1 {
		burn = int8(float32(levels-1) * gather)
	}
	if burn < 0 {
		return
	}

	f.draw(0, size.wide, func(x, y int) {
		at, to := left+x, floor-f.tall+y

		// Arriving: the same movement the line before him left by, run
		// backwards. See wordsAlong and wordsFrom.
		if gather < 1 {
			p := wordsAlong(m.words.move, gather, at, to, dotsX, dotsY)
			if p < 1 {
				dx, dy := wordsFrom(m.words.move, at, to, dotsX, dotsY)
				at += int(dx * (1 - p))
				to += int(dy * (1 - p))
			}
		}
		if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
			return
		}

		cell := (to/dotsPerCellY)*w + at/dotsPerCellX
		grid[cell] |= 1 << brailleBit[at%dotsPerCellX][to%dotsPerCellY]
		if burn > paint[cell] {
			paint[cell], hue[cell] = burn, tone
		}
	})
}
