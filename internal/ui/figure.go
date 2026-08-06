package ui

import (
	"encoding/base64"
	"math"
	"sort"
	"sync"
	"time"
)

// The figures: small line drawings that walk about the screen.
//
// They are not drawn by this code. They come from pictures somebody else drew,
// converted to dots by cmd/spindle-figures and written into figures_gen.go —
// so what happens at playback time is a lookup and a blit, and adding another
// figure is a folder of drawings and a ten line manifest.
//
// What the drawing does not carry is a face. A still picture cannot blink, and
// the one thing this screen is for is answering the music, so the head is left
// hollow by the generator and the face is drawn into it — the same eyes, brows
// and mouth that were there before, following the same sound.

// figureDrawing is one figure, at every size it was made for.
type figureDrawing struct {
	from    string
	licence string
	sizes   []figureSize
}

// figureSize is a figure at one height.
type figureSize struct {
	tall  int
	poses map[string]figurePose
}

// figurePose is one drawing: the dots, and where its head is.
//
// The dots are packed a bit each and written as base64, because a drawing
// spelled out a character a dot is six times the size in the source and this
// file already holds every pose of every figure. Nothing is unpacked: drawing
// reads the bit it wants.
type figurePose struct {
	wide, tall                 int
	headX, headY, headW, headH int
	bits                       string
}

// figureFor is the drawing of a given name, and whether there is one.
func figureFor(name string) (figureDrawing, bool) {
	d, ok := figures[name]
	return d, ok
}

// at is the size of a figure closest to a wanted height, and the pose from it.
func (d figureDrawing) at(tall int, pose string) (figurePose, bool) {
	if len(d.sizes) == 0 {
		return figurePose{}, false
	}

	best := d.sizes[0]
	for _, size := range d.sizes[1:] {
		if abs(size.tall-tall) < abs(best.tall-tall) {
			best = size
		}
	}
	p, ok := best.poses[pose]
	return p, ok
}

// draw lights the pose's dots. The head is not drawn: it was cleared when the
// figure was made, and what goes in it is the caller's business.
func (p figurePose) draw(light func(x, y int)) {
	packed := figureDots(p.bits)
	stride := (p.wide + 7) / 8
	for y := range p.tall {
		for x := range p.wide {
			if at := y*stride + x/8; at < len(packed) && packed[at]&(1<<(x%8)) != 0 {
				light(x, y)
			}
		}
	}
}

// figureDots unpacks a drawing, once. Every frame asks for the same handful of
// them, and unpacking a thousand bytes thirty times a second to look at the
// same picture is work nobody asked for.
var figureUnpacked sync.Map

func figureDots(bits string) []byte {
	if got, ok := figureUnpacked.Load(bits); ok {
		return got.([]byte)
	}
	packed, err := base64.StdEncoding.DecodeString(bits)
	if err != nil {
		return nil
	}
	figureUnpacked.Store(bits, packed)
	return packed
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// figureTall is how much of the screen a figure stands in, as a share of its
// height. Taller than the marks he stands in for: he is a whole person rather
// than a line of type, and the meters take what he leaves.
const figureTall = 0.55

// figureLines draws the figure who is on, with the face drawn into his head.
//
// The drawing gives the body, the arms, the legs and the walk; it cannot give
// an expression, because it is a still picture. So the head comes back hollow
// from the generator and the face goes in it — the same eyes and mouth that
// answer the music everywhere else on this screen.
func (m Model) figureLines(w, rows int) []string {
	who, ok := figureFor(m.faceWho())
	if !ok || w <= 0 || rows <= 0 || len(m.styles.Words) == 0 {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	levels, freqs := len(m.styles.Words[0]), len(m.styles.Words)

	pose, ok := who.at(int(figureTall*float64(dotsY)), m.figurePose())
	if !ok {
		return nil
	}

	// Where he stands: his own walk across the screen, and his feet on the foot
	// of the band the marks are set in, so the meter below him is the meter a
	// bar of notes leaves.
	room := (dotsX - pose.wide) / 2
	left := room + int(m.faceWalk()*float64(room+pose.wide))
	top := (dotsY-int(wordsMark*float64(dotsY)))/2 + int(wordsMark*float64(dotsY)) - pose.tall

	// And how he is coming or going, if he is in the middle of either.
	way, t := m.figureWaying()

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}

	lightAt := func(x, y int, _ facePart, burn float32) {
		burn *= figureBurn
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]

		// He takes the colour of the picture he is standing in rather than one
		// of his own: the hue of his column, the same way the meter hues its
		// columns, and the brightness of whatever that column of the spectrum
		// is doing. So he glows where the music is loud beside him, and he is
		// never a colour the screen is not already using.
		his := m.stageLevel(x, dotsX)
		if level := int8(float32(levels-1) * burn * (0.35 + 0.65*his)); level > paint[cell] {
			paint[cell], hue[cell] = level, int8(min(x/dotsPerCellX*freqs/w, freqs-1))
		}
	}
	light := func(x, y int, at facePart) { lightAt(x, y, at, 1) }

	// The marks he is walking through, if this is one of the visits where he
	// does. What he has reached is not there any more; what he is reaching is
	// coming apart. See figureSweeps.
	if m.figureSweeps() {
		count := max(m.words.where.Count, 1)
		m.figureThrough(w, rows, func(x, y, piece int, burn float32) {
			if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
				return
			}
			cell := (y/dotsPerCellY)*w + x/dotsPerCellX
			grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]

			s := m.wordsBeatPaint(piece, count, freqs, levels)
			if level := int8(float32(s.level) * burn); level > paint[cell] {
				paint[cell], hue[cell] = level, s.hue
			}
		})
	}

	pose.draw(func(x, y int) {
		x, y, burn, on := figureWarp(way, t, x, y, pose.wide, pose.tall, dotsY)
		if on {
			lightAt(x+left, y+top, facePartBody, burn)
		}
	})

	// The face only while he is whole: a head coming apart does not blink.
	if t < 1 {
		if tall := max((dotsY-(top+pose.tall))/dotsPerCellY, 0); tall >= wordsBand {
			m.wordsUnder(grid, paint, hue, w, rows, tall, max(top-dotsPerCellY, 0))
		}
		return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
	}

	// And a face, if the figure left a hole for one. A figure drawn with a face
	// of its own does not want a second pair of eyes over it — what changes his
	// expression then is the pose, not something drawn on top.
	if p, ok := faceLayout(pose.headW, pose.headH); ok && pose.headW > 0 {
		p.reach = 0 // his own arms are in the drawing; he does not want two pairs
		p.draw(m.faceNow(), func(x, y int, at facePart) {
			light(x+left+pose.headX, y+top+pose.headY, at)
		})
	}

	if tall := max((dotsY-(top+pose.tall))/dotsPerCellY, 0); tall >= wordsBand {
		m.wordsUnder(grid, paint, hue, w, rows, tall, max(top-dotsPerCellY, 0))
	}
	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}

// figureFrame is one drawing held for a while.
type figureFrame struct {
	pose string
	held time.Duration
}

// figureActs are the things a figure does: a run of drawings rather than one.
//
// A still picture swapped in for a second is a figure holding a sign. Two or
// three of them in a row is somebody doing something — which is the whole
// reason he walks on. The drawings come with forty-four poses; these are the
// runs worth making out of them.
var figureActs = map[string][]figureFrame{
	// Both feet off the ground, twice. The one that reads from across a room.
	"cheer": {{"cheer0", 170}, {"cheer1", 200}, {"cheer0", 170}, {"cheer1", 260}, {"idle", 120}},

	// Down, up, and a moment to land.
	"jump": {{"duck", 130}, {"jump", 260}, {"jump", 120}, {"duck", 110}, {"idle", 120}},

	// Three frames of a swing, which at this size is a dance move.
	"punch": {{"attack0", 110}, {"attack1", 110}, {"attack2", 190}, {"idle", 130}},

	// One leg out, held long enough to be seen, then back.
	"kick": {{"attackKick", 300}, {"idle", 140}},

	// Talking is the mouth moving, and the mouth is a whole drawing here, so
	// talking is the drawing moving.
	"talk": {{"talk", 150}, {"idle", 130}, {"talk", 170}, {"idle", 120}, {"talk", 160}, {"idle", 130}},

	// A pose held: he has stopped to consider something.
	"think": {{"think", 900}, {"idle", 150}},

	// Look at this.
	"show": {{"show", 700}, {"idle", 150}},

	// Knocked about by the music, and back on his feet.
	"hurt": {{"hit", 140}, {"hurt", 280}, {"idle", 160}},

	// Arms out wide, the way somebody stands in front of a speaker.
	"wide": {{"wide", 520}, {"idle", 140}},
}

// figureActNames is the acts in a settled order, so a record deals the same one
// twice. Maps in Go do not have an order; this does.
var figureActNames = []string{"cheer", "jump", "punch", "kick", "talk", "think", "show", "hurt", "wide"}

// figureActFor is the act a bar's nth turn gets.
func figureActFor(starts int64, turn int) string {
	h := uint64(starts)*0xff51afd7ed558ccd + uint64(turn+1)*0xbf58476d1ce4e5b9
	h ^= h >> 33
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 29
	return figureActNames[h%uint64(len(figureActNames))]
}

// figureActLong is how long an act runs, end to end.
func figureActLong(act string) time.Duration {
	var all time.Duration
	for _, f := range figureActs[act] {
		all += f.held * time.Millisecond
	}
	return all
}

// figureWaying is the way he is coming or going, and how far through it he is:
// one when he is whole and standing about, which is most of a visit.
func (m Model) figureWaying() (figureWay, float64) {
	gone := m.faceGone()
	switch {
	case gone < faceWalkIn:
		return m.figureComesBy(), gone / faceWalkIn
	case gone > 1-faceWalkOut:
		return m.figureGoesBy(), (1 - gone) / faceWalkOut
	}
	return figureWalks, 1
}

// figurePose is the drawing he is in this frame: the act he is in the middle
// of, the walk cycle while he is moving, and standing about otherwise.
func (m Model) figurePose() string {
	if act, ok := figureActs[m.face.act]; ok {
		since := time.Since(m.face.actAt)
		for _, f := range act {
			if since < f.held*time.Millisecond {
				return f.pose
			}
			since -= f.held * time.Millisecond
		}
	}

	if _, moving := m.faceGoing(); moving {
		at := int(math.Abs(math.Sin(math.Pi*faceSteps*m.faceGone()))*4) % 4
		if m.faceWalk() > 0 {
			at += 4
		}
		return "walk" + string(rune('0'+at))
	}
	return "idle"
}

// faceWho is which figure is on: one of the drawn ones, or nothing at all,
// which is the one this code draws itself.
//
// Dealt from the bar, so a record turns up the same twice, and so that the two
// kinds are both seen — the drawn figure has a body and a walk the geometry
// cannot match, and the geometry has a face and a pair of hands that answer the
// music in a way a still drawing never will.
func (m Model) faceWho() string {
	if len(figures) == 0 {
		return ""
	}

	// One of the drawn ones, or the one this code draws itself. He is in the
	// set with the rest of them: he has a face that blinks and a pair of hands
	// that answer the music, which no still drawing does, and the drawings have
	// a body and a walk that no formula does. Both are worth turning up.
	h := uint64(m.words.starts)*0x94d049bb133111eb + 0xd6e8feb86659fd93
	h ^= h >> 30
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 27

	if h%uint64(len(figures)+1) == 0 {
		return ""
	}

	names := make([]string, 0, len(figures))
	for name := range figures {
		names = append(names, name)
	}
	sort.Strings(names)
	return names[int(h>>8)%len(names)]
}

// How he comes and goes.
//
// Walking on from the side is the plainest thing a figure can do, and doing
// only that is what made him predictable. These are the others: they move his
// dots rather than him, which is the one thing this screen can do that a
// cartoon cannot — every one of him is a dot, and a dot can be sent anywhere.
type figureWay int

const (
	figureWalks    figureWay = iota // on from the side, off by one
	figureGathers                   // out of specks, from the floor up
	figureCrumbles                  // to pieces, and the pieces are handed to the water
	figureWays
)

// Spinning, dropping in from over the top, rising through the floor and
// bursting apart were all built and all thrown out. They moved him about the
// screen, which is a thing a sprite does; these two do something to what he is
// made of, which is the thing only this screen can do.

const (
	// figureGrain is how much of a dot's own moment is decided by where it is
	// rather than by the dot itself, and figureGrainOver how far through the
	// movement the last of them has turned up — short of the end, so he stands
	// there whole for a moment before he is done arriving.
	figureGrain     = 0.45
	figureGrainOver = 1.35

	// figureCrumbFall is how much of coming apart is decided by how high a dot
	// is and how much by the dot itself. All height and he unzips in rows; all
	// chance and he dissolves evenly, which is a fade. Between them he comes
	// apart from the top, raggedly.
	figureCrumbFall = 0.55

	// figureSprayEvery is one drop for every this many dots that come loose,
	// and figureSprayThrow how hard they are thrown. Not every dot: a thousand
	// drops is the whole of the water's room, and the meter wants some of it.
	figureSprayEvery = 3
	figureSprayThrow = 3.5

	// figureBurn is how brightly he is drawn at all — under the water and the
	// sparks that cross the same screen, never mind the type.
	//
	// He is a thing that wanders through the picture, not the picture. Drawn at
	// full strength he is a cut-out laid over the music; drawn under everything
	// else he is in the room with it, and the eye goes to him because he moves
	// rather than because he is the brightest thing there.
	figureBurn = 0.5

	// figureFaint is how much of that he has as he arrives or leaves. What
	// crosses this screen the rest of the time is water and sparks, and he
	// comes and goes out of the same stuff before he is anybody.
	figureFaint = 0.22
)

// figureComesBy and figureGoesBy are how this visit starts and ends.
//
// Dealt from the bar like everything else, and never the same way twice in one
// visit: coming and going by the same trick is a figure with one idea.
func (m Model) figureComesBy() figureWay {
	h := uint64(m.words.starts)*0xd6e8feb86659fd93 + 0x94d049bb133111eb
	h ^= h >> 31
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 29

	if h%2 == 0 {
		return figureWalks
	}
	return figureGathers
}

func (m Model) figureGoesBy() figureWay {
	h := uint64(m.words.starts)*0x2545f4914f6cdd1d + 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27

	if h%2 == 0 {
		return figureWalks
	}
	return figureCrumbles
}

// figureSliding reports that a way is one he does with his feet, so the walk
// carries him on or off rather than something happening to his dots.
func figureSliding(way figureWay) bool { return way == figureWalks }

// figureWarp is where a dot of his goes, how brightly it burns, and whether it
// is drawn at all.
//
// t is how far the movement has run: nought as it begins, one when he is whole
// and standing. Everything is worked out from where the dot belongs, so a
// thousand of them cost nothing to remember and come apart the same way twice.
//
// The brightness is what makes it read. A figure who slams in at full strength
// is a picture being switched on; the same dots coming up out of the dark, or
// going out as they fall, are the water and the sparks that cross this screen
// all the time — the same stuff, doing something else.
func figureWarp(way figureWay, t float64, x, y, wide, tall, dotsY int) (int, int, float32, bool) {
	if t >= 1 {
		return x, y, 1, true
	}
	t = min64(max64(t, 0), 1)

	// Faint while he is on his way, and only at full strength once he is whole.
	burn := float32(figureFaint + (1-figureFaint)*t*t)

	// And not all of him at once. Every dot has its own moment, so he comes
	// together out of specks and goes back to them — which is what the water
	// does, and what a shape being faded up and down is not.
	if way != figureCrumbles && figureNotYet(way, x, y, tall, t) {
		return 0, 0, 0, false
	}

	switch way {
	case figureCrumbles:
		// He does not fall apart on screen: he is handed to the water. Every
		// dot that has come loose has left him and is a drop now, arcing and
		// fading on the same physics the meter throws — so what you watch is
		// not a dimmer figure, it is a figure turning into sparks. See
		// figureLoose and Model.figureSpray.
		if figureLoose(x, y, tall, t) {
			return 0, 0, 0, false
		}
		return x, y, burn, true
	}

	return x, y, burn, true
}

// figureNotYet reports that a dot has not arrived yet, or has already gone.
//
// Its moment is its own, nudged by where it is: what rises comes up from the
// feet, what falls lands from the head down, and the rest of them turn up in no
// order at all. A shape that arrives all at once is a slide changing.
func figureNotYet(way figureWay, x, y, tall int, t float64) bool {
	own := float64(figureSpeck(x, y)%1000) / 1000
	high := 1 - float64(y)/float64(max(tall, 1))

	var by float64
	if way == figureGathers {
		by = high // out of the floor, the feet first
	}
	return own*(1-figureGrain)+by*figureGrain > t*figureGrainOver
}

// figureLoose reports that a dot has come away from him: the top goes first,
// the way a wall does, and each one on its own moment rather than a whole row
// at once — a shape that comes apart in rows is a shape being wiped.
func figureLoose(x, y, tall int, t float64) bool {
	high := 1 - float64(y)/float64(max(tall, 1))
	own := float64(figureSpeck(x, y)%1000) / 1000
	return high*figureCrumbFall+own*(1-figureCrumbFall) > t
}

// figureSpeck is a number of a dot's own, so a piece of him wanders the same
// way every time the same record plays.
func figureSpeck(x, y int) int {
	h := uint32(x)*2654435761 + uint32(y)*40503
	h ^= h >> 13
	h *= 2246822519
	h ^= h >> 16
	return int(h & 0x7fffffff)
}

// figureSweep hands the marks he has just walked into to the water.
//
// The same list the meter throws its drops into, so what he knocks over falls
// through the picture on the picture's own physics. This is the whole point of
// him sharing a frame with them: not that he is drawn beside the marks, but
// that he does something to them.
func (m *Model) figureSweep(w, rows int) {
	if !m.figureSweeps() {
		m.face.sweptLow, m.face.sweptHigh = figureUnswept, -figureUnswept
		return
	}

	who, ok := figureFor(m.faceWho())
	if !ok {
		return
	}
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	pose, ok := who.at(int(figureTall*float64(dotsY)), m.figurePose())
	if !ok {
		return
	}

	room := (dotsX - pose.wide) / 2
	left := room + int(m.faceWalk()*float64(room+pose.wide))
	edge := m.figureEdge(left, pose.wide)

	// The stretch he has walked, from one end of it to the other. It only ever
	// grows, so a mark he has knocked over stays knocked over.
	wasLow, wasHigh := m.face.sweptLow, m.face.sweptHigh
	m.face.sweptLow, m.face.sweptHigh = min(wasLow, edge), max(wasHigh, edge)

	g, where := m.words.have, m.words.where
	if g.DotsX != dotsX || where.Count == 0 {
		return
	}

	// Only the marks that came apart between the last frame and this one.
	span := dotsX / max(where.Count, 1)
	var n int
	for y := range dotsY {
		for x := range dotsX {
			if g.Lum[y*dotsX+x] < wordsLit {
				continue
			}
			piece := where.WordAt(x, y)
			if piece < 0 {
				continue
			}
			cx, _ := where.Middle(piece)
			if figureBroken(m.face.sweptLow, m.face.sweptHigh, cx, span) < 0.5 ||
				figureBroken(wasLow, wasHigh, cx, span) >= 0.5 {
				continue
			}
			if n++; n%figureShards != 0 || len(m.stage.drops) >= stageDrops {
				continue
			}
			m.stage.drops = append(m.stage.drops, stageDrop{
				col:    x,
				at:     float32(dotsY - 1 - y),
				speed:  figureSprayThrow * (m.scope.roll() + 0.15),
				bright: 0.6 + 0.4*m.scope.roll(),
			})
		}
	}
}

// figureSpray hands the dots that have come loose to the water.
//
// This is the whole of what makes it read as sparks rather than as a picture
// being dimmed: a drop is not drawn by this code at all. It is thrown into the
// same list the meter throws into, and from then on it arcs, falls and fades on
// the physics that has been crossing this screen all along — because it is the
// same water.
func (m *Model) figureSpray(w, rows int) {
	who, ok := figureFor(m.faceWho())
	if !ok {
		return
	}

	way, t := m.figureWaying()
	if way != figureCrumbles || t >= 1 {
		m.face.crumbled = 1
		return
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	pose, ok := who.at(int(figureTall*float64(dotsY)), m.figurePose())
	if !ok {
		return
	}

	// Where he is standing, the same arithmetic the drawing uses.
	room := (dotsX - pose.wide) / 2
	left := room + int(m.faceWalk()*float64(room+pose.wide))
	top := (dotsY-int(wordsMark*float64(dotsY)))/2 + int(wordsMark*float64(dotsY)) - pose.tall

	// Only what has come away since the last frame, so he sheds himself over
	// the whole of his going rather than all at once.
	was := m.face.crumbled
	m.face.crumbled = t
	if t >= was {
		return
	}

	var n int
	pose.draw(func(x, y int) {
		if !figureLoose(x, y, pose.tall, t) || figureLoose(x, y, pose.tall, was) {
			return
		}
		if n++; n%figureSprayEvery != 0 || len(m.stage.drops) >= stageDrops {
			return
		}

		at := dotsY - 1 - (y + top)
		if at < 0 || x+left < 0 || x+left >= dotsX {
			return
		}
		m.stage.drops = append(m.stage.drops, stageDrop{
			col:    x + left,
			at:     float32(at),
			speed:  figureSprayThrow * (m.scope.roll() - 0.35),
			bright: 0.5 + 0.5*m.scope.roll(),
		})
	})
}

// Walking through the marks.
//
// A bar of music is a row of marks across the screen, and a figure who walks on
// while they are up walks into them. He does not step round them and he does
// not stand politely beside them: what he reaches comes apart, and the pieces
// go into the water — the same water the meter throws, so what he knocks over
// falls through the picture and is gone.
//
// It is the one thing on this screen where two of its machines touch each
// other, and it is the only reason he shares a frame with anything.
const (
	// figureReach is how far in front of himself he clears, as a share of how
	// wide he is: a figure who only breaks what he is standing on looks like a
	// figure something is happening to.
	figureReach = 0.35

	// figureBreaks is how far he has to travel past a mark to have finished
	// breaking it, in dots. Short: a mark that takes a second to come apart is
	// a mark being dissolved, not one being walked into.
	figureBreaks = 26

	// figureShards is one drop for every this many dots of a mark he breaks.
	figureShards = 4

	// figureUnswept is the stretch of a visit he has not walked any of yet: an
	// empty span rather than a point, so the first frame does not read as the
	// whole screen having been walked through.
	figureUnswept = 1 << 30
)

// figureSweeps reports that this visit walks through the marks.
//
// The ones where he comes in on his feet: if he is walking on from the side
// then the row is in his way, and going round it is not something a figure does.
// The ones where he gathers out of specks are visits where he was never coming
// through anything.
func (m Model) figureSweeps() bool {
	return m.words.beats && m.figureComesBy() == figureWalks
}

// figureEdge is the front of him, in dots across the screen.
func (m Model) figureEdge(left, wide int) int {
	if way, _ := m.faceGoing(); way < 0 {
		return left
	}
	return left + wide
}

// figureSwept is the stretch he has walked through so far this visit.
func (m Model) figureSwept() (int, int) { return m.face.sweptLow, m.face.sweptHigh }

// figureBroken is how far a mark has come apart: nought while it is still
// standing in front of him, one once he is well past it.
//
// Measured against the whole stretch he has walked, from one end of it to the
// other, rather than against where he is standing this moment. Against the
// moment, everything on both sides of him counted as broken before he had taken
// a step — and worse, a mark he had already knocked over stood back up as he
// wandered away from it. Broken is broken.
func figureBroken(low, high, at, wide int) float64 {
	past := min64(float64(at-low), float64(high-at)) - float64(wide)*figureReach
	return min64(max64(past/figureBreaks, 0), 1)
}

// figureThrough draws what is left of the row he is walking into. The marks
// keep their own light: they are the picture and he is the visitor.
func (m Model) figureThrough(w, rows int, light func(x, y, piece int, burn float32)) {
	low, high := m.figureSwept()
	g, where := m.words.have, m.words.where
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if g.DotsX != dotsX || g.DotsY != dotsY || where.Count == 0 {
		return
	}

	for y := range dotsY {
		for x := range dotsX {
			if g.Lum[y*dotsX+x] < wordsLit {
				continue
			}
			piece := where.WordAt(x, y)
			if piece < 0 {
				continue
			}

			cx, _ := where.Middle(piece)
			broken := figureBroken(low, high, cx, dotsX/max(where.Count, 1))
			if broken >= 1 {
				continue
			}

			// Standing still until he arrives, then out from its own middle.
			at, to := x, y
			burn := float32(1)
			if broken > 0 {
				mx, my := where.Middle(piece)
				at += int(float64(x-mx) * broken * broken * wordsPopFlies)
				to += int(float64(y-my) * broken * broken * wordsPopFlies)
				burn = float32(1 - broken)
			}
			light(at, to, piece, burn)
		}
	}
}
