package ui

import (
	"sync"
	"time"
)

// Somebody walks past with a placard.
//
// Shuffle and repeat are set and forgotten: you want to know what they are at
// the moment you press them, and after that you want the screen back. So they
// are not badges in a corner — one drawing crosses the picture with the answer
// held up, and takes it away with him.
//
// One drawing rather than one per state, which is why the sign is blank in the
// sheet and what goes on it is drawn here. Shuffled or in order, repeating all
// or one or nothing, and whatever the next switch turns out to be: the carrier
// does not change, only what he is carrying.
//
// He is not a visitor. The bunny and the dancer turn up because a bar dealt them
// and do something worth watching; this one is sent for and has nothing to do
// but walk. See figureErrand.

const (
	// signCrosses is how long he takes to walk the whole width.
	//
	// Long enough to read the sign without hurrying and short enough that a run
	// of presses does not queue up a parade: press again while he is out and he
	// turns round with the new sign rather than a second one setting off. See
	// signFlow.
	signCrosses = 3200 * time.Millisecond

	// signWalks is how tall he is drawn, as a share of the screen.
	//
	// Smaller than a visiting figure. He is an announcement rather than a
	// performance, and at the size the bunny comes on he would be the picture
	// for three seconds rather than something crossing it.
	signWalks = 0.42

	// signFoot is where his feet go, as a share of the screen from the top.
	//
	// Below the band the words are set in, in the water. Through the middle he
	// would cross the lyric, and a switch is not worth two seconds of somebody's
	// song.
	signFoot = 0.86

	// signInset is how far inside the sign's blank the drawing on it is kept,
	// in dots. One: enough that the mark never touches the frame and reads as
	// something written on the sign rather than as part of it.
	signInset = 1
)

// signWhat is what the placard says.
type signWhat int

const (
	signNothing signWhat = iota
	signShuffled
	signInOrder
	signRepeatAll
	signRepeatOne
	signRepeatOff
)

// signState is what he is carrying and when he set off.
type signState struct {
	what signWhat
	at   time.Time

	// shuffle and repeat are what the last look found, so a change can be told
	// from the state being what it already was.
	shuffle bool
	repeat  string
	seen    bool
}

// signFlow watches the two switches and sends him out when either moves.
//
// The reading rather than the key, so a hand on a phone or a line in a script
// gets the same answer as this keyboard — the same rule the head on the edge and
// the volume's lamps follow.
func (m *Model) signFlow() {
	if m.ps == nil {
		return
	}
	shuffle, repeat := m.ps.Shuffle, m.ps.Repeat

	if !m.sign.seen {
		m.sign.shuffle, m.sign.repeat, m.sign.seen = shuffle, repeat, true
		return
	}
	switch {
	case shuffle != m.sign.shuffle:
		m.sign.what = signInOrder
		if shuffle {
			m.sign.what = signShuffled
		}
	case repeat != m.sign.repeat:
		switch repeat {
		case "track", "one":
			m.sign.what = signRepeatOne
		case "context", "all":
			m.sign.what = signRepeatAll
		default:
			m.sign.what = signRepeatOff
		}
	default:
		return
	}
	// Pressed again while he is still out: he keeps walking and swaps the sign,
	// rather than a second one setting off behind him.
	m.sign.shuffle, m.sign.repeat = shuffle, repeat
	if !m.signWalking() {
		m.sign.at = time.Now()
	}
}

// signWalking is whether he is on screen.
func (m Model) signWalking() bool {
	return m.sign.what != signNothing && !m.sign.at.IsZero() && time.Since(m.sign.at) < signCrosses
}

// signGone is how far across he has got, nought to one.
func (m Model) signGone() float64 {
	if !m.signWalking() {
		return 0
	}
	return float64(time.Since(m.sign.at)) / float64(signCrosses)
}

// signDraw puts him into the picture, walking right to left with the sign up.
func (m Model) signDraw(w, rows int, grid []uint8, paint, hue []int8, levels, freqs int) {
	if !m.signWalking() {
		return
	}
	who, ok := figureFor(figureSigner)
	if !ok {
		return
	}
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 || levels <= 0 {
		return
	}

	// A step on every beat where there is one to step on, and on his own clock
	// where there is not — the same choice the visiting figures make.
	frame := int(m.signGone() * 16)
	if beats, ok := m.beatsRun(); ok {
		frame = beats
	}
	pose, ok := who.at(int(signWalks*float64(dotsY)), "walk"+string(rune('0'+frame%4)))
	if !ok {
		return
	}

	// Right to left, all the way off at both ends so he is never cut in half by
	// the edge of the screen.
	left := int((1-m.signGone())*float64(dotsX+2*pose.wide)) - pose.wide
	top := int(signFoot*float64(dotsY)) - pose.tall

	light := func(x, y int) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		// The colour of the column he is in, like everything else that crosses
		// this picture: he is never a colour the screen is not already using.
		if level := int8(levels - 1); level > paint[cell] {
			paint[cell] = level
			hue[cell] = int8(min(x/dotsPerCellX*freqs/w, freqs-1))
		}
	}

	// Not turned. He was drawn walking the way he goes: in walk3 the left leg is
	// out in front with the heel down and the right is behind pushing off, and
	// the placard is up on the leading side. Mirroring him kept the travel and
	// reversed the stride, which is a moonwalk — and nothing else in the drawing
	// argues with it, because the face is round and looks straight out. See
	// TestTheSignerWalksTheWayHeGoes.
	pose.draw(false, func(x, y int) { light(x+left, y+top) })

	// And what he is carrying, written into the blank where the drawing has it.
	slot := signSlotFor(pose)
	if slot.wide <= 0 {
		return
	}
	x0 := left + slot.left
	signMark(m.sign.what, x0+signInset, top+slot.top+signInset,
		slot.wide-2*signInset, slot.tall-2*signInset, light)
}

// signSlot is where the blank sits inside a pose.
type signSlot struct{ left, top, wide, tall int }

var (
	signSlots  sync.Map // keyed by pose size
	signSlotMu sync.Mutex
)

// signSlotFor finds the blank the figure is holding: the largest run of empty
// the ink closes in.
//
// Found rather than written down, because it is a property of the drawing and
// the drawing is the thing that changes. Measured once per size and kept — it
// is derived from the baked dots, which do not move.
func signSlotFor(p figurePose) signSlot {
	key := [2]int{p.wide, p.tall}
	if got, ok := signSlots.Load(key); ok {
		return got.(signSlot)
	}
	signSlotMu.Lock()
	defer signSlotMu.Unlock()
	if got, ok := signSlots.Load(key); ok {
		return got.(signSlot)
	}

	w, h := p.wide, p.tall
	ink := make([]bool, w*h)
	p.draw(false, func(x, y int) {
		if x >= 0 && y >= 0 && x < w && y < h {
			ink[y*w+x] = true
		}
	})

	// Whatever the outside can reach is not enclosed by anything.
	out := make([]bool, w*h)
	stack := make([][2]int, 0, 2*(w+h))
	for x := range w {
		stack = append(stack, [2]int{x, 0}, [2]int{x, h - 1})
	}
	for y := range h {
		stack = append(stack, [2]int{0, y}, [2]int{w - 1, y})
	}
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := q[0], q[1]
		if x < 0 || y < 0 || x >= w || y >= h || out[y*w+x] || ink[y*w+x] {
			continue
		}
		out[y*w+x] = true
		stack = append(stack, [2]int{x + 1, y}, [2]int{x - 1, y}, [2]int{x, y + 1}, [2]int{x, y - 1})
	}

	// The biggest of what is left. The head is a hole too — it is cleared when
	// a figure is made — so it is the sign only because the sign is bigger,
	// which is what the drawing was asked for.
	seen := make([]bool, w*h)
	best, bestN := signSlot{}, 0
	for y := range h {
		for x := range w {
			if ink[y*w+x] || out[y*w+x] || seen[y*w+x] {
				continue
			}
			l, t, r, b, n := x, y, x, y, 0
			st := [][2]int{{x, y}}
			seen[y*w+x] = true
			for len(st) > 0 {
				q := st[len(st)-1]
				st = st[:len(st)-1]
				n++
				l, t = min(l, q[0]), min(t, q[1])
				r, b = max(r, q[0]), max(b, q[1])
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := q[0]+d[0], q[1]+d[1]
					if nx >= 0 && ny >= 0 && nx < w && ny < h && !ink[ny*w+nx] && !out[ny*w+nx] && !seen[ny*w+nx] {
						seen[ny*w+nx] = true
						st = append(st, [2]int{nx, ny})
					}
				}
			}
			if n > bestN {
				bestN, best = n, signSlot{left: l, top: t, wide: r - l + 1, tall: b - t + 1}
			}
		}
	}
	signSlots.Store(key, best)
	return best
}
