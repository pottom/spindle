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

	var part [faceParts_]wordPaint
	for i := range part {
		part[i] = m.wordsBeatPaint(int(i), int(faceParts_), freqs, levels)
	}

	lightAt := func(x, y int, at facePart, burn float32) {
		burn *= figureBurn
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]

		s := part[at]
		if level := int8(float32(s.level) * burn); level > paint[cell] {
			paint[cell], hue[cell] = level, s.hue
		}
	}
	light := func(x, y int, at facePart) { lightAt(x, y, at, 1) }

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
	figureSpins                     // turning, and growing or shrinking as he turns
	figureDrops                     // in from over the top, fast, and a squash as he lands
	figureRises                     // up through the floor
	figureBursts                    // apart, every dot on its own line out
	figureCrumbles                  // to pieces, and the pieces fall
	figureWays
)

const (
	// figureTurns is how many times round a spin goes, figureFalls how far a
	// drop comes from as a share of the screen, and figureFlies how far a burst
	// throws a dot as a share of the figure's own size.
	figureTurns = 1.5
	figureFalls = 1.4
	figureFlies = 2.2

	// figureCrumb is how far the pieces fall as a share of the screen, and
	// figureGrit how much they wander sideways on the way down.
	figureCrumb = 1.1
	figureGrit  = 0.35

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
	return figureWay(h % uint64(figureWays))
}

func (m Model) figureGoesBy() figureWay {
	h := uint64(m.words.starts)*0x2545f4914f6cdd1d + 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27

	// Not the way he came in, so the two ends of a visit are two things.
	way := figureWay(h % uint64(figureWays-1))
	if way >= m.figureComesBy() {
		way++
	}
	return way
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

	cx, cy := float64(wide)/2, float64(tall)/2
	dx, dy := float64(x)-cx, float64(y)-cy

	switch way {
	case figureSpins:
		// Turning about himself, and small until he has finished turning.
		a := (1 - t) * figureTurns * 2 * math.Pi
		sin, cos := math.Sin(a), math.Cos(a)
		size := 0.25 + 0.75*t
		return int(cx + (dx*cos-dy*sin)*size), int(cy + (dx*sin+dy*cos)*size), burn, true

	case figureDrops:
		// From over the top, fast, and squashed for the moment he lands.
		fall := (1 - t) * (1 - t) * figureFalls * float64(dotsY)
		squash := 1.0
		if t > 0.82 {
			squash = 1 - 0.35*math.Sin(math.Pi*(t-0.82)/0.18)
		}
		return x, int(cy + dy*squash - fall), burn, true

	case figureRises:
		return x, y + int((1-t)*figureFalls*float64(dotsY)), burn, true

	case figureBursts:
		// Every dot on its own line out from the middle, and gone before it has
		// got far — what is left of a burst is a shape you have to remember.
		away := (1 - t) * figureFlies
		if figureSpeck(x, y)%100 < int(away*45) {
			return 0, 0, 0, false
		}
		// A piece that has flown further is further gone, the way a spark is.
		return int(float64(x) + dx*away), int(float64(y) + dy*away),
			burn * float32(1-0.7*away/figureFlies), true

	case figureCrumbles:
		// The top goes first, the way a wall does, and every piece takes its own
		// line down.
		gone := 1 - t
		high := 1 - float64(y)/float64(max(tall, 1))
		if high < gone-0.15 {
			return 0, 0, 0, false
		}
		drop := gone * gone * figureCrumb * float64(dotsY)
		drift := (float64(figureSpeck(x, y)%200)/100 - 1) * figureGrit * gone * float64(wide)

		// A piece that has fallen further has less of itself left.
		return int(float64(x) + drift), y + int(drop),
			burn * float32(1-0.75*drop/(figureCrumb*float64(dotsY))), true
	}

	return x, y, burn, true
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
