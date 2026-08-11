package ui

import (
	"encoding/base64"
	"sort"
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// The marks a bar of music is set in, and how a row of them is put together.
//
// They used to be ♪ and ♫, set from the face the lyrics are — which is why they
// were those two and nothing else: it is what the face carries. Now they are
// drawings, and the row is built here rather than by the setter.
//
// The order is the whole idea. Every mark already rides its own slice of the
// spectrum, low at the left and high at the right, so a row of instruments in
// that order says what the row of notes only mapped: the kick jumps on the kick
// and the cymbals jump on the cymbals.
//
// What comes out is the same pair the setter hands back for a line of type — a
// field of dots and a layout saying which piece each dot belongs to — so
// everything downstream carries on unchanged: the ride, the lean, the figure
// walking through and knocking them over, the popping off one at a time.

// A set of marks, drawn rather than set from a face. See cmd/spindle-marks.
type markSet struct {
	from, licence string
	sizes         []markSize
}

// marksDrawn is whether a bar of marks may be dealt a set of drawings at all.
//
// On, and it was off, and what changed is not the drawings.
//
// They were switched off because a row of dancers did nothing the notes were not
// already doing: a bar of marks was watched for half a minute at a time while
// something else was what the screen was about, and a set that arrived because
// thirty seconds had passed is a set that arrived for no reason. The note then
// was that they would come back when there was a set with a reason of its own.
//
// The reason turned out not to be in the sets. It was in the deal. The row is
// now dealt at the record's own joins rather than off a clock — see joins.go —
// so a change of set is the record changing, and the same drawings that read as
// arbitrary read as the record turning over. Watched on Mindblow's "Don't Let Me
// Go", where it is unmistakable.
//
// Still one word, and it still turns everything off without taking anything
// out.
const marksDrawn = true

// markCastFor is which marks a bar of them is dealt: the notes the face carries,
// or one of the drawn sets.
//
// The notes are not being replaced. They are what a bar of music has looked like
// here from the start, and a row of instruments is another way of saying the
// same thing rather than a better one — so they take turns, dealt from the bar
// the way a visiting figure is. The empty string is the notes.
//
// Dealt from the record as well as from the bar, and that was watched rather
// than reasoned. A wordless bar is stamped at the top of the spell it is in, so
// the first half minute of every record in the world is stamped nought and every
// one of them was dealt the same set. Skipping through a list, that is the same
// row over and over — the deal only ever moved on for anybody who let a record
// run past thirty seconds.
func markCastFor(record string, starts int64) string {
	sets := make([]string, 0, len(markSets)+1)
	sets = append(sets, "") // the notes
	for name := range markSets {
		sets = append(sets, name)
	}
	sort.Strings(sets)

	h := uint64(starts)*0x94d049bb133111eb + 0xd6e8feb86659fd93
	for _, c := range []byte(record) {
		h = (h ^ uint64(c)) * 0x100000001b3
	}
	h ^= h >> 30
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 27
	return sets[h%uint64(len(sets))]
}

// markSize is the whole row at one dot height.
type markSize struct {
	tall  int
	marks []markDots
}

// markDots is one drawing: its own size in dots, and a bit per dot.
type markDots struct {
	name       string
	wide, tall int
	bits       string
}

// at reports whether a dot of the drawing is set.
func (m markDots) at(x, y int, bits []byte) bool {
	if x < 0 || y < 0 || x >= m.wide || y >= m.tall {
		return false
	}
	i := y*m.wide + x
	return i/8 < len(bits) && bits[i/8]&(1<<(i%8)) != 0
}

const (
	// markSpread is the air between two marks, as a share of how tall they are.
	// Enough that they read as separate players rather than as a frieze, little
	// enough that the row is a group.
	markSpread = 0.45

	// markLeast is the fewest that make a row. Under this the ends are all there
	// is, and the sound running along the row has nowhere to run.
	markLeast = 3
)

// markRowFor is the size to draw a row at and which of the set fit across it.
//
// The whole row at a smaller size beats part of it at a larger one, and that is
// the only reason there is a choice: the order is what the row says — the kick
// at one end and the cymbals at the other, and the sound running between them —
// so four of seven marks is not a smaller version of the picture, it is a
// different one. Every baked size is tried from the largest down, and the first
// that holds all of them wins.
//
// Only sizes at or under the room there is: a drawing scaled up is a drawing
// with its stroke pulled apart, and the sizes were baked so that the stroke
// comes out the same weight at each of them.
//
// If none of them holds the whole row, the smallest is taken and the middle is
// thinned — the ends are kept, because the ends are what the order is for.
func markRowFor(set markSet, tall, dotsX int) (markSize, []markDots, int, bool) {
	sizes := append([]markSize(nil), set.sizes...)
	sort.Slice(sizes, func(i, j int) bool { return sizes[i].tall > sizes[j].tall })

	var smallest markSize
	var found bool
	for _, size := range sizes {
		if size.tall > tall {
			continue
		}
		smallest = size
		if !found {
			found = true
		}
		if row, gap := markFit(size, dotsX, len(size.marks)); row != nil {
			return size, row, gap, true
		}
	}
	if !found {
		// Smaller than anything baked: the smallest there is, which is better
		// than nothing at all.
		smallest = sizes[len(sizes)-1]
	}

	row, gap := markFit(smallest, dotsX, markLeast)
	if row == nil {
		return markSize{}, nil, 0, false
	}
	return smallest, row, gap, true
}

// markFit thins a row until it fits, and gives up rather than going under least.
func markFit(size markSize, dotsX, least int) ([]markDots, int) {
	gap := max(int(markSpread*float64(size.tall)), 2)

	fits := func(marks []markDots) bool {
		wide := gap * (len(marks) - 1)
		for _, m := range marks {
			wide += m.wide
		}
		return wide <= dotsX
	}

	row := append([]markDots(nil), size.marks...)
	for len(row) > least && !fits(row) {
		row = append(row[:len(row)/2], row[len(row)/2+1:]...)
	}
	if !fits(row) {
		return nil, 0
	}
	return row, gap
}

// markPicture builds the field of dots a row of marks is drawn from, and the
// layout the rest of the screen reads it through.
func markPicture(name string, w, rows int) (cover.Grain, msg.WordLayout, bool) {
	set, ok := markSets[name]
	if !ok || w <= 0 || rows <= 0 {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	size, row, gap, ok := markRowFor(set, int(wordsMark*float64(dotsY)), dotsX)
	if !ok || len(row) == 0 {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	// Where the row stands: across the middle of the screen, as a line of type
	// would be, and centred in what it is given.
	wide := gap * (len(row) - 1)
	for _, m := range row {
		wide += m.wide
	}
	left := (dotsX - wide) / 2
	top := (dotsY - size.tall) / 2

	grain := cover.Grain{
		DotsX: dotsX, DotsY: dotsY,
		CellsX: w, CellsY: rows,
		Lum: make([]uint8, dotsX*dotsY),
	}
	layout := msg.WordLayout{
		Count:   len(row),
		DotsX:   dotsX,
		At:      make([]int16, dotsX),
		Tops:    []int{top},
		Bottoms: []int{top + size.tall - 1},
	}
	for i := range layout.At {
		layout.At[i] = -1
	}

	for i, m := range row {
		bits, err := base64.StdEncoding.DecodeString(m.bits)
		if err != nil {
			return cover.Grain{}, msg.WordLayout{}, false
		}

		// Each of them sits on the same floor rather than on its own middle: a
		// row of instruments standing on a line is a band, and the same row
		// centred one by one is a mobile.
		x0 := left
		for _, before := range row[:i] {
			x0 += before.wide + gap
		}
		y0 := top + size.tall - m.tall

		for y := range m.tall {
			for x := range m.wide {
				if !m.at(x, y, bits) {
					continue
				}
				if px, py := x0+x, y0+y; px >= 0 && py >= 0 && px < dotsX && py < dotsY {
					grain.Lum[py*dotsX+px] = 255
				}
			}
		}

		// Which piece a dot belongs to, by the column it is in: the marks stand
		// side by side and never overlap, so a column is one player's.
		for x := x0; x < x0+m.wide+gap && x < dotsX; x++ {
			if x >= 0 {
				layout.At[x] = int16(i)
			}
		}
	}
	return grain, layout, true
}

// marksWalk moves to the next set of marks by hand, and round to the deal again.
//
// The row is dealt at the record's own turns, which is right for listening and
// useless for looking: a set that has just been drawn cannot be seen without
// waiting for a join that may be a minute off. So the key walks the sets in the
// order they are named, and one more press hands the row back to the deal.
func (m *Model) marksWalk() {
	sets := make([]string, 0, len(markSets)+1)
	for name := range markSets {
		sets = append(sets, name)
	}
	sort.Strings(sets)
	sets = append(sets, "") // and back to whatever the record would have given

	at := len(sets) - 1
	for i, name := range sets {
		if name == m.words.picked {
			at = i
			break
		}
	}
	m.words.picked = sets[(at+1)%len(sets)]
	m.words.showed = time.Now()

	// Thrown away rather than left up: the picture on screen was made for the
	// set that was on, and wordsGrind only asks for a new one when what it holds
	// is not what it wants.
	m.words.cast = "\x00"
}

// marksForcing reports that a bar of marks is up because somebody asked for it.
//
// The same length the record's name gets: long enough to look at, short enough
// that the screen goes back to the music on its own rather than waiting to be
// let go.
func (m Model) marksForcing() bool {
	return !m.words.showed.IsZero() && time.Since(m.words.showed) < marksShows
}

// marksShows is how long a set stays when it is asked for.
const marksShows = 12 * time.Second
