package ui

import "slices"

// A dance, which is a figure with his own clock.
//
// Everything else that stands in the wordless bar is a still: a row of marks
// rides the spectrum, leans on the beat and arrives by gathering, and the
// movement in it belongs to the sound. A dance is the first thing here that
// moves of its own accord — it has frames, and they step whether the band is
// loud or quiet.
//
// Which is why it may not do both. The rule this screen is built on is one
// question, one movement: the height of the dots answers the sound, the lean
// answers the beat. A figure that dances and rides and leans is three movements
// on one body, and the one that gets lost is the dance. So while he is up, the
// dance is the answer, and what is left to the sound is the room he is drawn in.
//
// The drawings are baked by cmd/spindle-moves — see moves_gen.go, and
// docs/MOVES.md for how they are asked for.

// moveSet is one company of moves: a character, at every size he was baked for.
type moveSet struct {
	from    string
	licence string
	sizes   []moveSize
}

// moveSize is a set at one height, in dots.
//
// tall is what the standing pose measures, and wide the cell every frame stands
// in — a frame keeps where it stood across that cell, so a leg thrown out to the
// side is a leg thrown out rather than the whole figure sliding.
type moveSize struct {
	tall, wide int
	moves      map[string]moveDance
}

// moveDance is one move: from standing, into the thing, round and round for as
// long as the music wants, and back up.
//
// The three ranges are frame numbers, both ends inclusive. in and out may be
// empty — the bounce is done standing and has nothing to go into — and are
// empty when their first index is past their last.
type moveDance struct {
	inFrom, inTo     int
	loopFrom, loopTo int
	outFrom, outTo   int
	frames           []moveFrame
}

// moveFrame is one drawing: the dots, how big they are, and where they stand
// across the cell.
//
// There is no height above the floor, because there is no such thing here: every
// frame was sat on its own lowest ink when it was baked. That is the floor in
// every pose a dance has — feet, palms, a shoulder, the crown of the head — and
// it is what stops the figure walking up and down the screen as the sheet's own
// rows drift. See cmd/spindle-moves.
type moveFrame struct {
	x, wide, tall int
	bits          string
}

// moveSetFor is the company of a given name, and whether there is one.
func moveSetFor(name string) (moveSet, bool) {
	s, ok := moveSets[name]
	return s, ok
}

// at is the size closest to a wanted height, and the move from it.
func (s moveSet) at(tall int, move string) (moveSize, moveDance, bool) {
	if len(s.sizes) == 0 {
		return moveSize{}, moveDance{}, false
	}

	best := s.sizes[0]
	for _, size := range s.sizes[1:] {
		if abs(size.tall-tall) < abs(best.tall-tall) {
			best = size
		}
	}
	d, ok := best.moves[move]
	return best, d, ok
}

// names is every move the set has, in the order they are named, so a deal is
// dealt from a list that does not change between runs.
func (s moveSet) names() []string {
	if len(s.sizes) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.sizes[0].moves))
	for name := range s.sizes[0].moves {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// steps is how many frames the move is, when its loop is gone round this many
// times: in, then the loop over and over, then out.
func (d moveDance) steps(rounds int) int {
	return d.span(d.inFrom, d.inTo) + rounds*d.span(d.loopFrom, d.loopTo) + d.span(d.outFrom, d.outTo)
}

// span is how many frames a range holds, and none where it holds none.
func (d moveDance) span(from, to int) int {
	if to < from {
		return 0
	}
	return to - from + 1
}

// frameAt is which drawing is up this many steps into the move, and whether the
// move is still going.
//
// The loop is where the time is spent: a move is asked for a number of rounds,
// and everything else about it — how long the entry takes, how long it holds —
// falls out of the frames it was drawn in.
func (d moveDance) frameAt(step, rounds int) (moveFrame, bool) {
	if len(d.frames) == 0 || step < 0 {
		return moveFrame{}, false
	}

	if in := d.span(d.inFrom, d.inTo); step < in {
		return d.frames[d.inFrom+step], true
	} else {
		step -= in
	}

	if loop := d.span(d.loopFrom, d.loopTo); loop > 0 && step < rounds*loop {
		return d.frames[d.loopFrom+step%loop], true
	} else {
		step -= rounds * loop
	}

	if out := d.span(d.outFrom, d.outTo); step < out {
		return d.frames[d.outFrom+step], true
	}
	return moveFrame{}, false
}

// draw lights the frame's dots, turned around or as drawn, measured from the
// left of the cell and up from the floor.
func (f moveFrame) draw(turned, wide int, light func(x, y int)) {
	packed := figureDots(f.bits)
	stride := (f.wide + 7) / 8
	for y := range f.tall {
		for x := range f.wide {
			if at := y*stride + x/8; at < len(packed) && packed[at]&(1<<(x%8)) != 0 {
				if turned != 0 {
					light(wide-1-(f.x+x), y)
					continue
				}
				light(f.x+x, y)
			}
		}
	}
}
