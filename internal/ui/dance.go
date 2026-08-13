package ui

import (
	"slices"
	"time"

	"github.com/pottom/spindle/internal/ui/bones"
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
	// is dealt.
	//
	// One to four, which is one bar to four. Longer was the first arrangement
	// and it was wrong for the same reason a dancer does not do one thing for
	// half a minute: what he is doing here is a routine, and a routine is made
	// of moves that follow each other. A wordless bar runs half a minute, so at
	// this length he gets through four or five of them.
	danceRoundsLeast = 1
	danceRoundsMost  = 4

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

	// nth is how far into the routine he is: a bar of his own is not one move
	// but a run of them, each dealt as the one before it finishes. See
	// danceCarryOn.
	nth int

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

// danceNames is every move he knows, in one order, so a deal is dealt from a
// list that does not change between runs.
func danceNames() []string {
	out := make([]string, 0, len(boneDances))
	for name := range boneDances {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// danceFor is the move of a given name, and whether he knows it.
func danceFor(name string) (bones.Dance, bool) {
	d, ok := boneDances[name]
	return d, ok
}

// danceCastFor says whether this bar is his, dealt from the record and from
// where in it the bar falls — so a record dances the same way twice, which is
// the rule every other deal on this screen keeps.
func danceCastFor(record string, starts int64) bool {
	if len(boneDances) == 0 {
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
func (m *Model) danceDeal(record string, starts int64, nth int) {
	names := danceNames()
	if len(names) == 0 {
		return
	}

	h := uint64(starts)*0xd6e8feb86659fd93 + uint64(nth+1)*0x9e3779b97f4a7c15
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

		// Never the one he has just done. Dealt freely, a move follows itself
		// about one time in seven, and a man who drops to the floor, stands up
		// and drops to the floor again reads as a picture that has stuck rather
		// than as a routine — which is the same reason a line never arrives the
		// way the line before it did. See wordsMoveFor.
		if name == m.dance.move && len(names) > 1 {
			weights[i] = 0
		}
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
	if _, ok := danceFor(m.dance.picked); ok {
		pick = m.dance.picked
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
func (m Model) danceUp() bool {
	if !m.words.dancing {
		return false
	}
	_, ok := danceFor(m.dance.move)
	return ok
}

// danceEach is how long one keyframe lasts.
//
// A turn of the move is a bar, however many keyframes it was written in: the
// same sixteen poses take two seconds on a record at 120 and three at 80, so the
// move is danced to the record rather than beside it. Where there is no beat to
// keep, nothing lasts any time at all and he holds still — the row of marks
// rests the same way when the drums do, and resting is the picture rather than a
// fault in it.
func (m Model) danceEach() (time.Duration, bool) {
	d, ok := danceFor(m.dance.move)
	period := m.scope.beat.Period
	if !ok || !m.beatKeeping() || period <= 0 || len(d.Frames) == 0 {
		return 0, false
	}
	each := time.Duration(float64(period) * danceBeatsPerLoop / float64(len(d.Frames)))
	return each, each > 0
}

// danceAt is how far into the move he is, in keyframes, and not in whole ones:
// what is between two of them is worked out rather than waited for, so the dance
// is as smooth as the screen is quick. See bones.Tween.
func (m Model) danceAt() float64 {
	each, ok := m.danceEach()
	if !ok {
		return 0
	}
	return float64(time.Since(m.dance.since)) / float64(each)
}

// danceDone reports that the move has been round as many times as it was dealt,
// so the bar can deal the next one.
func (m Model) danceDone() bool {
	d, ok := danceFor(m.dance.move)
	if !ok || len(d.Frames) == 0 {
		return true
	}
	return m.danceAt() >= float64(m.dance.rounds*len(d.Frames))
}

// danceSpent is how long the move that has just finished took, to the keyframe.
func (m Model) danceSpent() time.Duration {
	each, ok := m.danceEach()
	d, known := danceFor(m.dance.move)
	if !ok || !known {
		return 0
	}
	return time.Duration(m.dance.rounds*len(d.Frames)) * each
}

// danceTall is the height he is drawn at, in dots, at the size of the room.
func (m Model) danceTall() int {
	return int(float64(m.height*dotsPerCellY) * danceHigh)
}

// danceStands is where he is drawn: how much a dot is worth, where his box
// begins across the screen, and the row his feet are on.
func (m Model) danceStands(dotsX, dotsY int) (scale float64, left, ground int, ok bool) {
	d, known := danceFor(m.dance.move)
	if !known {
		return 0, 0, 0, false
	}
	from, to, top := d.Reach()
	if d.Floor <= top || to <= from {
		return 0, 0, 0, false
	}

	scale = float64(m.danceTall()) / (d.Floor - top)
	wide := int((to - from) * scale)
	return scale, (dotsX-wide)/2 - int(from*scale), (dotsY + m.danceTall()) / 2, true
}

// dancePose is the figure as he stands this instant.
func (m Model) dancePose() (bones.Pose, bones.Dance, bool) {
	d, ok := danceFor(m.dance.move)
	if !ok {
		return bones.Pose{}, bones.Dance{}, false
	}
	return d.At(m.danceAt()), d, true
}

// dancePicture is the pose he arrives in as a picture, so that the bar has
// something to arrive with and the meter under it has something to measure
// against.
//
// Only the one: what is drawn every frame after it is worked out from the
// joints. This is the still the screen adopts — see wordsAdopt, and danceDraw
// for what is actually on the screen.
func (m Model) dancePicture(w, rows int) (cover.Grain, msg.WordLayout, bool) {
	pose, d, ok := m.dancePose()
	if !ok || w <= 0 || rows <= 0 {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	scale, left, ground, ok := m.danceStands(dotsX, dotsY)
	if !ok {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	g := cover.Grain{DotsX: dotsX, DotsY: dotsY, CellsX: w, CellsY: rows, Lum: make([]uint8, dotsX*dotsY)}
	lo, hi, top := dotsX, 0, dotsY
	bones.Draw(pose, d.Box, d.Floor, scale, left, ground, bones.Pen(m.danceTall()), func(x, y int) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		g.Lum[y*dotsX+x] = 255
		lo, hi, top = min(lo, x), max(hi, x), min(top, y)
	})
	if hi < lo {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	// One piece, and it covers him: the rest of this screen asks the layout
	// which word a dot belongs to, and the answer for a dancer is always the
	// same one.
	layout := msg.WordLayout{
		Count: 1, DotsX: dotsX,
		At:      make([]int16, dotsX),
		Tops:    []int{top},
		Bottoms: []int{min(ground, dotsY-1)},
		Lefts:   []int{lo},
		Rights:  []int{hi},
	}
	for x := range layout.At {
		layout.At[x] = -1
		if x >= lo && x <= hi {
			layout.At[x] = 0
		}
	}
	layout.Settle()
	return g, layout, true
}

// danceDraw puts him on the screen as he stands this instant.
//
// Drawn straight into the grid rather than through a picture of his own: a
// figure is a few thousand lit dots and the screen is a quarter of a million, so
// building him a canvas every frame would be most of the work for none of the
// answer.
//
// The gathering is the same one everything else here arrives by — he assembles
// out of the air like a line of type does — because the bar he stands in is the
// same bar, and a picture that arrives differently reads as a different screen
// rather than as a different picture.
func (m Model) danceDraw(grid []uint8, paint, hue []int8, w, rows int, gather float32, freqs, levels int) {
	pose, d, ok := m.dancePose()
	if !ok {
		return
	}
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	scale, left, ground, ok := m.danceStands(dotsX, dotsY)
	if !ok {
		return
	}

	// One colour for the whole of him, and one brightness: the dance is the
	// answer to the music here, so the light is not also answering it. The
	// middle band, so he is neither the bass nor the cymbals but the record.
	tone := int8(min(freqs/2, freqs-1))
	burn := int8(levels - 1)
	if gather < 1 {
		burn = int8(float32(levels-1) * gather)
	}
	if burn < 0 {
		return
	}

	bones.Draw(pose, d.Box, d.Floor, scale, left, ground, bones.Pen(m.danceTall()), func(x, y int) {
		at, to := x, y

		// Arriving: the same movement the line before him left by. See
		// wordsAlong and wordsFrom.
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

// danceCarryOn keeps the routine going: when a move runs out, the next one is
// dealt and begins where it left off.
//
// A bar of his own is half a minute, and one move is a bar or four of it. What
// he does with the rest is the point of the whole thing — he is not showing a
// move, he is dancing, and dancing is one thing after another. The moves are
// written to make this free: every one of them begins and ends in the same
// standing pose, so any of them follows any other without a join.
//
// The clock is not restarted between them. A move is stepped off the beat from
// when it began, and beginning the next one at the instant a frame happened to
// be drawn would shift the whole routine off the grid by however far into a
// frame that was. So what has been spent is added on instead, and the keyframes
// keep falling where the beats do.
func (m *Model) danceCarryOn(record string, starts int64) {
	if m.dance.move == "" || m.words.cast != danceCast {
		m.dance.nth = 0
		m.danceDeal(record, starts, 0)
		return
	}
	if !m.danceDone() {
		return
	}
	m.dance.since = m.dance.since.Add(m.danceSpent())
	m.dance.nth++
	m.danceDeal(record, starts, m.dance.nth)
}
