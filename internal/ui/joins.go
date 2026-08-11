package ui

import (
	"math"
	"time"
)

// Where the record changes, and the picture with it.
//
// The big screen used to turn over every thirty seconds. That is a clock on a
// screen where everything else answers the record: which marks come up, how the
// row leans, whether a figure visits — all of it landing wherever the half
// minute happened to fall.
//
// A record says where its own joins are, and it says it in the shape of its
// spectrum rather than in the level: the drums coming in, a chorus opening out,
// everything falling away in a breakdown. The bands are already normalised
// against the record's own loudness, so what is left when the loudness is taken
// out of them is the shape. How far the shape now stands from what has been
// playing is how much has changed.
//
// # What it is worth, measured
//
// Measured against ground truth rather than against itself, which is the only
// way this could be settled. Two keys went on the big screen and Emerald
// Princess was listened through by ear, marking every join: 31 marks. The score
// says 128 BPM, 4/4, 348 bars, so a bar is 1.875s and eight bars are 15.000s —
// and 26 of the 31 marks landed within a second of an eight-bar line, every one
// of them on a bar divisible by eight. The printed dynamics are those bars: mp
// at 33, mf at 48, f at 57, ff at 74, the fall back at 87. The ear found what
// the composer wrote, so the ear is the truth here.
//
// Against those 24 joins, tolerance one bar:
//
//	texture, as built here     15 of 24 found, 12 invented
//	harmony (the twelve notes)  8 of 24 found, 23 invented
//	the key changing            4 of 24 found, 12 invented
//	all of them together       16 of 24 found, 18 invented
//
// So the shape of the spectrum is the whole of it, and the notes are no help:
// harmony moves every couple of bars and the key holds for minutes, and neither
// is a section. That measurement is why there is no chroma in this file.
//
// At the figures below the row turns over 1.9 times a minute, against the two a
// minute the clock did, and 57% of those turns land within a bar of a join a
// listener marked. Against none of them, which is what a stopwatch scores.
const (
	// joinNear and joinFar are the two windows compared: what the record is
	// doing now against what it has been doing. Swept — nearer and the answer
	// is a transient, further and a section has to be half over before it says
	// so.
	joinNear = 4500 * time.Millisecond
	joinFar  = 8 * time.Second

	// joinWatch is how fast the typical novelty of this record is followed.
	// Slow: it is the scale everything is measured against, and a scale that
	// moves with what it is measuring cannot say anything is unusual.
	joinWatch = 0.002

	// joinEdge is how far above that a moment has to stand to be a join.
	joinEdge = 1.3

	// joinApart is the shortest a section may be. Twenty seconds — swept: at
	// ten the row turns over 2.8 times a minute, which is busier than the clock
	// it replaces, and only half of those land on anything.
	joinApart = 20 * time.Second

	// joinMost is the longest the picture will hold without one. A record that
	// never changes still has to turn over, or the screen has stopped.
	joinMost = 60 * time.Second

	// joinWarm is how much has to be heard before any of it means anything:
	// both windows full, and a little more so the typical is not set by the
	// first thing through it.
	joinWarm = 14 * time.Second
)

// joinsState is the two windows, kept as running sums over a ring of shapes.
type joinsState struct {
	ring  [][]float32 // the band shape, loudness taken out
	at    int
	fill  int
	near  []float32 // running sum of the newest joinNear of them
	far   []float32 // and of the joinFar behind those
	watch float32   // the typical novelty of this record
	seen  bool      // and whether it has been given its first value

	// nov is what the last frame measured, kept only so the bar on ctrl+shift+d
	// can put it beside the line it has to cross. Nothing draws from it.
	nov float32

	// begins is where the section on screen started, on the playback clock, and
	// forTrack the record it belongs to.
	begins   time.Duration
	forTrack string
	turns    int           // how many times this record has turned over
	heard    time.Duration // how much of this record has gone through
}

// joinsFlow takes this frame's bands and works out whether the record has just
// changed. Called once a frame, off the same clock everything else here uses.
func (m *Model) joinsFlow(fps int) {
	if fps <= 0 || m.ps == nil {
		return
	}
	bands := m.scope.bands
	if len(bands) == 0 {
		return
	}

	j := &m.joins
	if j.forTrack != m.ps.TrackID {
		*j = joinsState{forTrack: m.ps.TrackID, begins: 0}
	}

	near := max(int(joinNear/time.Second*time.Duration(fps)), 1)
	far := max(int(joinFar/time.Second*time.Duration(fps)), 1)
	if len(j.ring) != near+far {
		j.ring = make([][]float32, near+far)
		j.near, j.far = make([]float32, len(bands)), make([]float32, len(bands))
		j.at, j.fill = 0, 0
	}

	// The shape: the bands as a direction rather than a size, so a chorus that
	// is merely louder than the verse is not a change.
	var total float32
	for _, v := range bands {
		total += v
	}
	if total <= 0 {
		return
	}
	// The oldest entry of the ring is about to be dropped, so its slice is the
	// one to write into: a frame that allocates is thirty allocations a second
	// that the collector has to come back for, and the pauses it takes land
	// outside the update where nothing can account for them. Measured — this is
	// what put the dropped frames back after the daemon was fixed.
	shape := j.ring[j.at]
	if shape == nil {
		shape = make([]float32, len(bands))
	}
	for i, v := range bands {
		shape[i] = v / total
	}

	// Into the ring, and out of the sums at the other end.
	if j.fill == len(j.ring) {
		for i := range j.far {
			j.far[i] -= shape[i]
		}
	}
	j.ring[j.at] = shape
	for i := range j.near {
		j.near[i] += shape[i]
	}
	if out := j.ring[(j.at+len(j.ring)-near)%len(j.ring)]; out != nil {
		for i := range j.near {
			j.near[i] -= out[i]
			j.far[i] += out[i]
		}
	}
	j.at = (j.at + 1) % len(j.ring)
	if j.fill < len(j.ring) {
		j.fill++
	}

	j.heard = m.elapsed()
	if j.fill < len(j.ring) {
		return
	}

	var sum float64
	for i := range j.near {
		d := float64(j.near[i]/float32(near) - j.far[i]/float32(far))
		sum += d * d
	}
	nov := float32(math.Sqrt(sum))
	j.nov = nov

	// The scale starts at the first thing measured rather than at nothing.
	// Easing up from zero, the first novelty after the windows fill stands
	// enormously above a scale that is still climbing, so every record was
	// handed a join the moment it had heard enough to have one — measured, at
	// exactly the twentieth second, on a record whose shape had not moved at
	// all.
	if !j.seen {
		j.watch, j.seen = nov, true
	}
	j.watch += (nov - j.watch) * joinWatch

	if j.heard < joinWarm || j.heard-j.begins < joinApart {
		return
	}
	if (j.watch > 0 && nov > j.watch*joinEdge) || j.heard-j.begins > joinMost {
		j.begins = j.heard
		j.turns++
	}
}

// joinsAt is when the stretch of record on screen began, on the playback clock.
//
// It is what the wordless screen is stamped with, in place of the thirty second
// spell: everything dealt from that stamp — the cast, the lean, the chance of a
// visitor — now lands where the record turned rather than where the clock did.
func (m Model) joinsAt() time.Duration {
	if m.ps == nil || m.joins.forTrack != m.ps.TrackID {
		return 0
	}
	return m.joins.begins
}

// joinsTurns is how many times the record on screen has turned over.
func (m Model) joinsTurns() int {
	if m.ps == nil || m.joins.forTrack != m.ps.TrackID {
		return 0
	}
	return m.joins.turns
}
