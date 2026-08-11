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

	// turns says the drawings have a front, so a mark may be turned round on
	// the beat. A drum seen head on has no front and turning it does nothing;
	// anything with feet does. See wordsTurning.
	turns bool
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
	// The notes, or everybody. Not one set at a time any more: every drawing
	// dances, stands on the same floor and is drawn with the same hand, so a
	// company dealt across the lot is a different crowd on every record where
	// picking a room only ever had five answers. The rooms are still there for
	// the m key — see marksWalk.
	sets := []string{"", markMixed}

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

// markMixed is the cast that is everybody at once, rather than one set.
//
// Every drawing on the screen now dances, stands on the same floor and is drawn
// with the same hand — so keeping them in the rooms they were drawn in buys
// nothing. A company of a bear, a television, a smiley and a guitar reads as one
// crowd, and there are more crowds in fifty drawings dealt five at a time than
// there are sets to walk through.
//
// The sets are still there and the m key still walks them, for anyone who wants
// a pure one.
const markMixed = "everyone"

// markEveryone is the whole pool at a height: every set's drawings, minus the
// ones that do not survive being baked that small.
//
// The filter is the whole reason a drawing carries a least of its own. Dealt
// blind, a row at 36 dots would put a bear beside a smiley, and only one of them
// would still be a drawing — see the measurements beside each set's manifest.
func markEveryone(tall int) []markDots {
	names := make([]string, 0, len(markSets))
	for name := range markSets {
		names = append(names, name)
	}
	// Sorted, so the pool is the same pool on every run and a seed means the
	// same company twice.
	sort.Strings(names)

	var out []markDots
	for _, name := range names {
		for _, size := range markSets[name].sizes {
			if size.tall != tall {
				continue
			}
			for _, one := range size.marks {
				if one.least > 0 && tall < one.least {
					continue
				}
				out = append(out, one)
			}
		}
	}
	return out
}

// markCastSet is the set a cast names, or the whole crowd assembled for
// markMixed. Assembled here rather than baked, because which drawings are
// allowed at a height is a filter and a filter belongs next to the reason for
// it.
func markCastSet(name string) (markSet, bool) {
	if name != markMixed {
		set, ok := markSets[name]
		return set, ok
	}
	out := markSet{from: "every set", licence: "see each set", turns: true}
	for _, tall := range markHeights() {
		if pool := markEveryone(tall); len(pool) > 0 {
			out.sizes = append(out.sizes, markSize{tall: tall, marks: pool})
		}
	}
	return out, len(out.sizes) > 0
}

// markHeights is every height anything is baked at, largest first.
func markHeights() []int {
	seen := map[int]bool{}
	var out []int
	for _, set := range markSets {
		for _, size := range set.sizes {
			if !seen[size.tall] {
				seen[size.tall] = true
				out = append(out, size.tall)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

// markDots is one drawing: its own size in dots, and a bit per dot.
type markDots struct {
	// pitch is where this one stands between the low end of the room and the
	// top of it. It is what keeps the row meaning something now that the crowd
	// is dealt rather than listed: whoever is picked, they line up by it.
	pitch float64

	// least is the smallest baked height this drawing still reads at, and turns
	// whether it has a front to turn round. Both belong to the drawing rather
	// than to the set it arrived on, because a company is dealt across sets:
	// see markEveryone.
	least int
	turns bool

	// set is the sheet it was drawn on. It is not used to draw anything — it is
	// how a dealt company keeps from being all one kind. See pick.
	set string

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
)

// markCrowdFor is who is on this time, and how large they are drawn.
//
// The row used to be a list: a set was seven or eight drawings in a fixed order,
// every appearance the same, and the only question was whether they fitted. When
// they did not — and eight of anything usually does not — the whole row dropped
// a size and everybody shrank with it.
//
// It is a pool now, and the question is turned round: take the largest size the
// room allows, and see who fits at it. What comes up is a handful of the pool
// rather than all of it, dealt from the bar, so the same record puts up the same
// company twice and the next record puts up another.
//
// Two things are kept from the list it replaces. The order is the sound — the
// low end at the left, the top of the range at the right — which is why a
// drawing carries a pitch and the company is sorted by it however it was dealt.
// And the ends matter more than the middle, which is why a size that holds only
// two is passed over: a row needs enough of a spread for the sound to run along.
func markCrowdFor(set markSet, tall, dotsX int, seed int64) (markSize, []markDots, int, bool) {
	sizes := append([]markSize(nil), set.sizes...)
	sort.Slice(sizes, func(i, j int) bool { return sizes[i].tall > sizes[j].tall })

	var smallest markSize
	var found bool
	for _, size := range sizes {
		if size.tall > tall {
			continue
		}
		if !found {
			smallest, found = size, true
		}
		if len(size.marks) == 0 {
			continue
		}
		gap := max(int(markSpread*float64(size.tall)), 1)
		if crowd := pick(size.marks, dotsX, gap, seed); len(crowd) >= markCrowdLeast {
			return size, crowd, gap, true
		}
		smallest = size
	}
	if !found || len(smallest.marks) == 0 {
		return markSize{}, nil, 0, false
	}

	// Nothing held a company at any size, which is a very narrow screen. The
	// smallest, and whoever fits on it.
	gap := max(int(markSpread*float64(smallest.tall)), 1)
	crowd := pick(smallest.marks, dotsX, gap, seed)
	return smallest, crowd, gap, len(crowd) > 0
}

// pick takes a company from the pool that fits across the room.
//
// Not at random out of the whole pool, which was the first way and skewed: the
// narrow drawings fit more often than the wide ones, so a dealt company came out
// as four things from the top of the range and nothing from the bottom, and the
// row stopped meaning anything. The range is cut into as many bands as there are
// places, and each band sends one — so the company spans the room however it is
// dealt, and the low end is always somebody's.
//
// Within a band the choice is the bar's, and a band with nobody who fits sends
// nobody: the pool has a piano in it half again as wide as a note, and holding a
// place open for it would keep the row short.
func pick(pool []markDots, dotsX, gap int, seed int64) []markDots {
	h := uint64(seed)*0x9e3779b97f4a7c15 + 0xd6e8feb86659fd93
	roll := func() uint64 {
		h ^= h >> 30
		h *= 0xbf58476d1ce4e5b9
		h ^= h >> 27
		return h
	}

	for places := markCrowdMost; places >= markCrowdLeast; places-- {
		var crowd []markDots
		var wide int
		already := map[string]bool{}
		for band := range places {
			low, high := float64(band)/float64(places), float64(band+1)/float64(places)

			// Everybody in this stretch of the range, walked in a dealt order.
			var in []markDots
			for _, one := range pool {
				if one.pitch >= low && (one.pitch < high || band == places-1) {
					in = append(in, one)
				}
			}
			for i := len(in) - 1; i > 0; i-- {
				j := int(roll() % uint64(i+1))
				in[i], in[j] = in[j], in[i]
			}

			// Whoever fits, and a sheet nobody in the row has come off yet
			// before one that is already represented.
			//
			// Without the preference the deal is fair to the drawings and unfair
			// to the sheets: the instruments are nineteen of fifty-seven and the
			// faces fourteen, so a third of every company was instruments and
			// four hundred deals put up four rows that were all one sheet.
			room := func(one markDots) (int, bool) {
				want := one.wide
				if len(crowd) > 0 {
					want += gap
				}
				return want, wide+want <= dotsX
			}

			// A leaning, not a rule. Held absolutely it stopped being a mix and
			// became a quota: five places and five sheets means one of each,
			// every time, which is one animal and one face and one instrument
			// for ever. Overruled a quarter of the time, a company is usually
			// four or five sheets and sometimes three, which is a crowd.
			mind := roll()%markMixEvery != 0

			spare, hasSpare, took := markDots{}, false, false
			for _, one := range in {
				want, fits := room(one)
				if !fits {
					continue
				}
				if !mind || !already[one.set] {
					crowd, wide, took = append(crowd, one), wide+want, true
					already[one.set] = true
					break
				}
				if !hasSpare {
					spare, hasSpare = one, true
				}
			}
			// Nobody new fitted. A second off a sheet already in the row beats a
			// shorter row: the spread along the sound is what the row says, and
			// a missing band is a hole in it.
			if !took && hasSpare {
				if want, fits := room(spare); fits {
					crowd, wide = append(crowd, spare), wide+want
				}
			}
		}
		if len(crowd) >= markCrowdLeast {
			sort.SliceStable(crowd, func(i, j int) bool { return crowd[i].pitch < crowd[j].pitch })
			return crowd
		}
	}
	return nil
}

// markCrowdLeast and markCrowdMost are how many make a company.
//
// Four to six. Under four the row is a pair of ends with nothing between them
// and the sound has nowhere to run; over six they are small again, which is the
// thing this was built to stop.
const (
	markCrowdLeast = 4
	markCrowdMost  = 6

	// markMixEvery is how often the leaning toward a sheet not yet in the row is
	// overruled: one band in this many takes whoever comes first.
	//
	// Measured over four hundred deals against the five sheets in the pool, as
	// how many sheets a company came off:
	//
	//	no leaning   3.46 — and four rows that were all one sheet
	//	every 6      4.93 — 375 of 400 rows held all five
	//	every 4      4.79
	//	every 3      4.76
	//	every 2      4.44 — 216 held five, 146 four, 36 three, 2 two
	//	absolute     5.00 — one of each, every time
	//
	// Two. Anything weaker and the row is a quota with a rounding error: five
	// places and five sheets means one animal, one face, one instrument, for
	// ever. At two the usual company is four or five sheets and now and then it
	// is three, which is a crowd rather than a delegation.
	markMixEvery = 2
)

// markPicture builds the field of dots a row of marks is drawn from, and the
// layout the rest of the screen reads it through.
func markPicture(name string, w, rows int, seed int64) (cover.Grain, msg.WordLayout, bool) {
	set, ok := markCastSet(name)
	if !ok || w <= 0 || rows <= 0 {
		return cover.Grain{}, msg.WordLayout{}, false
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	size, row, gap, ok := markCrowdFor(set, int(wordsMark*float64(dotsY)), dotsX, seed)
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
		Lefts:   make([]int, len(row)),
		Rights:  make([]int, len(row)),
		Turns:   make([]bool, len(row)),
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
		layout.Lefts[i], layout.Rights[i] = x0, x0+m.wide-1
		layout.Turns[i] = m.turns

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

// wordsTurning is which marks of a row are facing the other way this frame.
//
// A mark with feet that never turns round is a mark standing on a stage; one
// that turns is somebody dancing on it. So each of them turns on its own count
// of beats, dealt when the row arrives, and the whole row is dealt again the
// next time one comes up: the same record twice over is the same dance twice
// over, and two records are not.
//
// Only on the beat, and only where there is a beat to take it from. A turn that
// happens on a clock of its own is the one movement on this screen that would
// not be answering the music, and there are already enough of those elsewhere.
func (m Model) wordsTurning(count int) []bool {
	if count == 0 || !m.words.beats || len(m.words.where.Lefts) < count {
		return nil
	}

	// Whether a mark turns belongs to the drawing — a hi-hat has no front — and
	// travels with it, because a company may be dealt across sets and hold a
	// dancer and a drum at once.
	turns := m.words.where.Turns
	if len(turns) < count {
		return nil
	}
	var any bool
	for _, one := range turns[:count] {
		any = any || one
	}
	if !any {
		return nil
	}

	// Counted from the record rather than from the row, which is the difference
	// between a dance and a set of statues. The row is dealt again at every one
	// of the record's own turns — see joins.go — and counted from there, a
	// record with a lot of build in it puts everybody back on their starting
	// side every time. Watched on Sandstorm against Cuban Pete: the first has a
	// join every twenty seconds and the row barely moved, the second holds a
	// stretch and it danced.
	//
	// What the deal still decides is who turns how often and which way up they
	// start. That is dealt from the bar, so a new row is a new arrangement — it
	// is only the counting that runs on.
	beats, ok := m.beatsIn(m.elapsed())
	if !ok {
		return nil
	}

	out := make([]bool, count)
	for i := range out {
		// Its own count, from the bar it arrived under: two beats is a fidget
		// and sixteen is a statue, so the deal is somewhere between.
		h := uint64(m.words.leanAt)*0x9e3779b97f4a7c15 + uint64(i)*0xbf58476d1ce4e5b9
		h ^= h >> 31
		every := marksTurnLeast + int(h%uint64(marksTurnMost-marksTurnLeast+1))

		// And its own starting side and its own place in the count. Both were
		// missing and both showed. Without a side, every mark starts facing the
		// same way and the ones dealt a long count stay there — twenty beats in,
		// two of eight had turned, which was arithmetic rather than luck.
		// Without a shift, two marks dealt the same count turn on the same beat
		// for as long as they are up, and six of eight going round together is
		// not eight people dancing, it is a row being flipped.
		side := int(h >> 8 & 1)
		shift := int(h >> 16 % uint64(every))
		out[i] = turns[i] && ((beats+shift)/every+side)%2 == 1
	}
	return out
}

// marksTurnLeast and marksTurnMost are the fewest and the most beats a mark
// keeps facing one way.
//
// Two to six, and it was three to ten. At the top of that range a mark held
// still for seven seconds at the tempo of an ordinary record, which is longer
// than anybody watches one mark for — and with eight of them the row looked
// like it was doing nothing while half of it was mid-turn. Short enough that
// something is always going round, long enough that no one of them is a
// flicker.
const (
	marksTurnLeast = 2
	marksTurnMost  = 6
)
