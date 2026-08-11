package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// The words, in dots: what is being sung, big enough to read across a room.
//
// This is the one picture here that is not drawn from the sound at all. It is
// drawn from the line of the song that is sounding now, taken from the same
// synced lyrics the player screen shows in a window, and set in dots across the
// whole terminal. When the line changes the dots scatter and gather again into
// the next one, so the moment the song moves on is the moment the picture does.
//
// With no lyrics to show it falls back to the record and the artist, which is
// worth having on a screen in a room, and with neither it is not offered at all.

// wordsFont is the face the words are cut from, parsed once.
//
// Bold, because a thin stroke drawn in dots at this size is a dotted line
// rather than a letter. It is the Go font, which comes with the imaging library
// already in use here: measured, it carries the whole of Latin including the
// Hungarian long vowels, Central European and Cyrillic — and no CJK, which is
// what decides whether this picture can be offered for a given song.
var wordsFont = func() *sfnt.Font {
	f, err := sfnt.Parse(gobold.TTF)
	if err != nil {
		return nil
	}
	return f
}()

const (
	// wordsReadable is the fewest dots a letter may be given before a line has
	// to be broken instead.
	//
	// Under this the strokes of a letter fall between the dots and the word
	// stops being a word: measured on the Go bold face, an "a" needs about
	// eight dots across its bowl before the hole in the middle survives being
	// dithered.
	wordsReadable = 9

	// wordsMostLines is how many lines a lyric may be broken into. More than
	// this and the letters are too small anyway, and the picture is a page
	// rather than a phrase.
	wordsMostLines = 4

	// wordsMargin is the share of the width and the height left clear around
	// the words.
	wordsMargin = 0.08

	// wordsLead is the line spacing, as a share of the size of the type.
	wordsLead = 1.35

	// wordsOver is how many times over each dot is drawn before it is decided,
	// and wordsCover how much of a dot the type has to cover to light it.
	//
	// Three, and a third. Drawn straight onto the dot grid a stem about one dot
	// wide is a coin toss per dot, so the letters came out dotted with stripes
	// through them — and which dots went missing changed with every step of the
	// terminal's font size, which is how it was noticed.
	wordsOver  = 3
	wordsCover = 128
)

// wordsImage draws the lines, white on black, filling w by h dots as far as the
// type will go. It reports false when the face has no glyph for something in
// them — a Japanese lyric on a Latin font is a row of empty boxes, and an empty
// box is worse than not offering the picture.
func wordsImage(lines []string, w, h int) (*image.Gray, msg.WordLayout, bool) {
	if wordsFont == nil || w <= 0 || h <= 0 || len(lines) == 0 {
		return nil, msg.WordLayout{}, false
	}
	if !wordsDrawable(lines) {
		return nil, msg.WordLayout{}, false
	}

	// A mark is set smaller than a line would be. One character given the whole
	// height is a note two hundred dots tall with nothing else on the screen —
	// which is what this looked like the first time, and what was wrong with it.
	// Held to a third, the meter has its band above and below, and the picture
	// is the same shape as it is when there are words.
	fit := h
	if len(lines) == 1 && wordsBeats(lines[0]) {
		fit = int(float64(h) * wordsMark)
	}

	size := wordsSize(lines, w, fit)
	if size < 4 {
		return nil, msg.WordLayout{}, false
	}

	face, err := opentype.NewFace(wordsFont, &opentype.FaceOptions{
		Size: float64(size * wordsOver), DPI: 72, Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, msg.WordLayout{}, false
	}
	defer face.Close() //nolint:errcheck // nothing to do about a face that will not close

	// Set larger than the dots it will be shown in, and brought down by
	// coverage. Drawn straight onto the dot grid, a stem about one dot wide is
	// a coin toss per dot — over the threshold here, under it there — and the
	// letters come out as a dotted texture with stripes through them, which
	// changes every time the terminal's font size does. Averaged from a larger
	// drawing, the same stem is a decision made once and made the same all the
	// way along it.
	big := image.NewGray(image.Rect(0, 0, w*wordsOver, h*wordsOver))
	d := &font.Drawer{Dst: big, Src: image.NewUniform(color.Gray{Y: 255}), Face: face}

	metrics := face.Metrics()
	lead := int(float64(size*wordsOver) * wordsLead)
	block := lead * (len(lines) - 1)
	top := (h*wordsOver-block)/2 + metrics.Ascent.Round()/2

	// Where every word lands is worked out here rather than afterwards: this is
	// the only place the widths are known, because a proportional face gives no
	// formula for how wide a word is, only a measurement.
	layout := msg.WordLayout{DotsX: w, At: make([]int16, w*len(lines))}
	for i := range layout.At {
		layout.At[i] = -1
	}

	for i, line := range lines {
		width := d.MeasureString(line).Round()
		left := (w*wordsOver - width) / 2
		baseline := top + i*lead

		d.Dot = fixed.P(left, baseline)
		d.DrawString(line)

		// The line of type covers from the top of its tallest letter to the
		// bottom of its lowest, with a little either side so that a dot on the
		// edge of a stroke still counts as part of the word it belongs to.
		layout.Tops = append(layout.Tops, max((baseline-metrics.Ascent.Round())/wordsOver, 0))
		layout.Bottoms = append(layout.Bottoms, min((baseline+metrics.Descent.Round())/wordsOver, h-1))

		// Measured as prefixes of the line rather than piece by piece, so that
		// where a piece is taken to be is where the drawer actually put it: a
		// face kerns its pairs, and a run of widths added up separately walks
		// away from the line under it.
		for _, piece := range wordsPieces(line) {
			from := (left + d.MeasureString(line[:piece.from]).Round()) / wordsOver
			to := (left + d.MeasureString(line[:piece.to]).Round()) / wordsOver
			for x := max(from, 0); x < to && x < w; x++ {
				layout.At[i*w+x] = int16(layout.Count)
			}
			layout.Count++
		}
	}

	// Down to the dots, by how much of each one the type covers. The threshold
	// is well under half on purpose: a stem of this face comes out around a dot
	// wide at the sizes a lyric is set at, so asking for half of a dot would
	// throw away every stroke that happens to straddle two of them — which is
	// the dotted, striped look this replaces.
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			var sum int
			for dy := range wordsOver {
				for dx := range wordsOver {
					sum += int(big.GrayAt(x*wordsOver+dx, y*wordsOver+dy).Y)
				}
			}
			if sum/(wordsOver*wordsOver) >= wordsCover {
				img.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return img, layout, true
}

// wordsPiece is a run of a line that moves as one thing, as byte offsets into
// it.
type wordsPiece struct{ from, to int }

// wordsPieces cuts a line into what keeps its own time: its words, and the
// punctuation hanging off either end of them.
//
// A comma is not part of the word in front of it. It is a mark of its own with
// its own shape, and on this screen — where everything answers a band of the
// spectrum and rides it — giving it to the word means it can only ever do what
// the word does. Cut loose it bounces on its own, which is what a full stop at
// the end of a line ought to be doing while the line is being sung.
func wordsPieces(line string) []wordsPiece {
	var out []wordsPiece

	from := -1
	cut := func(to int) {
		if from >= 0 {
			out = append(out, wordsCut(line, from, to)...)
			from = -1
		}
	}
	for i, r := range line {
		if unicode.IsSpace(r) {
			cut(i)
			continue
		}
		if from < 0 {
			from = i
		}
	}
	cut(len(line))
	return out
}

// wordsCut takes one word off the line and separates the marks at its ends
// from the word itself. What is between them stays where it is: a hyphen holds
// two halves of a word together and an apostrophe is part of one.
func wordsCut(line string, from, to int) []wordsPiece {
	field := line[from:to]

	head := 0
	for _, r := range field {
		if !wordsPunct(r) {
			break
		}
		head += utf8.RuneLen(r)
	}
	if head == len(field) {
		return []wordsPiece{{from, to}} // all marks, and so one thing
	}

	tail := len(field)
	for tail > head {
		r, size := utf8.DecodeLastRuneInString(field[:tail])
		if size == 0 || !wordsPunct(r) {
			break
		}
		tail -= size
	}

	var out []wordsPiece
	if head > 0 {
		out = append(out, wordsPiece{from, from + head})
	}
	out = append(out, wordsPiece{from + head, from + tail})
	if tail < len(field) {
		out = append(out, wordsPiece{from + tail, to})
	}
	return out
}

// wordsPunct reports that a rune is a mark that lives its own life.
//
// Every mark there is, quotes and apostrophes with the rest. Only the ends of a
// word are ever cut, which is what keeps "don't" and "you're" whole while a
// dropped g's tick in "singin'" and the one that opens "'cause" come away — and
// those are marks doing a mark's job, hanging off the end of a word.
func wordsPunct(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// wordsDrawable reports whether the face has every glyph the lines need.
func wordsDrawable(lines []string) bool {
	var b sfnt.Buffer
	for _, line := range lines {
		for _, r := range line {
			if r == ' ' {
				continue
			}
			i, err := wordsFont.GlyphIndex(&b, r)
			if err != nil || i == 0 {
				return false
			}
		}
	}
	return true
}

// wordsSize is the largest type the lines can be set in and still fit.
//
// Found by measuring rather than by arithmetic: the width of a string in a
// proportional face is the sum of what its letters happen to be, which no
// formula gives.
func wordsSize(lines []string, w, h int) int {
	longest := 0
	for _, line := range lines {
		longest = max(longest, len([]rune(line)))
	}
	if longest == 0 {
		return 0
	}

	// A starting guess from the space each letter would have if they were all
	// the same width, then walked down until it fits.
	size := min(int(float64(w)*(1-wordsMargin)/float64(longest)*1.9),
		int(float64(h)*(1-wordsMargin)/(float64(len(lines))*wordsLead)))

	for ; size >= 4; size-- {
		face, err := opentype.NewFace(wordsFont, &opentype.FaceOptions{
			Size: float64(size), DPI: 72, Hinting: font.HintingFull,
		})
		if err != nil {
			return 0
		}

		d := &font.Drawer{Face: face}
		widest := 0
		for _, line := range lines {
			widest = max(widest, d.MeasureString(line).Round())
		}
		tall := int(float64(size)*wordsLead) * len(lines)
		face.Close() //nolint:errcheck // measuring only

		if widest <= int(float64(w)*(1-wordsMargin)) && tall <= int(float64(h)*(1-wordsMargin)) {
			return size
		}
	}
	return 0
}

// wordsWrap breaks a line up until its letters are big enough to read.
//
// A lyric is written as one line however long it is, and one long line across a
// screen is a row of specks. Breaking it is what buys the letters their dots
// back: two lines of half the length are twice the size.
func wordsWrap(line string, w, h int) []string {
	return wordsWrapTo(line, w, h, wordsMostLines)
}

// wordsWrapTo is the same, held to a given number of lines. A lyric is given
// four; a part of a card is given two, because past that a name is a paragraph.
func wordsWrapTo(line string, w, h, most int) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	for n := 1; n <= most; n++ {
		lines := wordsSplit(line, n)
		if len(lines) < n {
			return lines // it will not break any further
		}
		if size := wordsSize(lines, w, h); size > 0 {
			// How many dots the average letter is getting, which is the whole
			// question: below wordsReadable it is worth another line.
			longest := 0
			for _, l := range lines {
				longest = max(longest, len([]rune(l)))
			}
			if int(float64(w)*(1-wordsMargin))/max(longest, 1) >= wordsReadable || n == most {
				return lines
			}
		}
	}
	return wordsSplit(line, most)
}

// wordsCardMost is the most lines one part of a card may be broken into, and
// wordsCardTries how many times a part is cut back when even that many leaves
// it too small to read.
const (
	wordsCardMost  = 2
	wordsCardTries = 6
)

// wordsCard sets out a card — what the record is called, whose it is, what it
// came out on — breaking any part of it that is too long for the room.
//
// Every other line on this screen goes through wordsWrap before it is set.
// These did not, and a long title was left to be set at whatever size fitted it
// on one line: on a narrow terminal a row of specks, and past a point nothing
// at all, because the type ran under the smallest the face can be cut at and
// the picture was refused outright.
func (m Model) wordsCard(parts ...string) []string {
	dotsX, dotsY := m.width*dotsPerCellX, m.height*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 {
		return nil
	}

	var kept []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	if len(kept) == 0 {
		return nil
	}

	// Each part is measured against its own share of the height, so that a
	// title broken in two is not sized as though it had the screen to itself
	// with the artist still to come under it.
	share := max(dotsY/len(kept), 1)

	var out []string
	for _, part := range kept {
		out = append(out, m.wordsCardPart(part, dotsX, share)...)
	}
	return out
}

// wordsCardPart is one part of a card, broken and if need be cut back until
// what is left can be read.
//
// Cut back by measuring rather than by arithmetic: how many letters a line ends
// up with is a question about a proportional face and a greedy line breaker
// together, and the only way to know is to break it and look at what came out.
func (m Model) wordsCardPart(part string, dotsX, share int) []string {
	lines := wordsWrapTo(part, dotsX, share, wordsCardMost)
	if wordsBigEnough(lines, dotsX) {
		return lines
	}

	// Start from what those lines could hold at a readable size, then keep
	// taking a share off until what comes back out of the breaker fits.
	most := wordsCardMost * int(float64(dotsX)*(1-wordsMargin)) / wordsReadable
	for range wordsCardTries {
		lines = wordsWrapTo(wordsShorten(part, most), dotsX, share, wordsCardMost)
		if wordsBigEnough(lines, dotsX) {
			break
		}
		most = most * 6 / 7
	}
	return lines
}

// wordsBigEnough reports that every line of a part gets its letters enough dots
// to be a word rather than a texture. The same threshold a lyric is broken to.
func wordsBigEnough(lines []string, dotsX int) bool {
	longest := 0
	for _, line := range lines {
		longest = max(longest, len([]rune(line)))
	}
	return int(float64(dotsX)*(1-wordsMargin))/max(longest, 1) >= wordsReadable
}

// wordsShorten cuts a part of a card down to a number of letters, on a word,
// and marks the cut.
//
// Breaking a long name in two is only half an answer. A podcast episode runs to
// ninety letters, and ninety letters over two lines is forty-five to a line —
// seven dots each on a wide terminal, which is a texture rather than a word.
// Past the point where another line would not save it, the rest is cut off.
//
// A card is a name, not a paragraph. Whoever is looking at it wants to know
// what is playing, and the first few words of a title say that.
func wordsShorten(part string, most int) string {
	if most < 4 || len([]rune(part)) <= most {
		return part
	}

	var kept []string
	var run int
	for _, word := range strings.Fields(part) {
		if run+len([]rune(word)) > most-1 { // room for the mark
			break
		}
		kept = append(kept, word)
		run += len([]rune(word)) + 1
	}
	if len(kept) == 0 {
		// One word longer than the whole card: cut it wherever it has got to.
		return string([]rune(part)[:max(most-1, 1)]) + "…"
	}
	return strings.Join(kept, " ") + "…"
}

// wordsSplit breaks a line into n roughly equal pieces, at spaces.
func wordsSplit(line string, n int) []string {
	words := strings.Fields(line)
	if n <= 1 || len(words) <= 1 {
		return []string{line}
	}
	n = min(n, len(words))

	// Greedy by length: each line takes words until it has its share of the
	// letters, which keeps the block square rather than leaving a stray word.
	total := len([]rune(line))
	want := total / n

	var out []string
	var cur []string
	var run int
	for i, word := range words {
		cur = append(cur, word)
		run += len([]rune(word)) + 1

		left := len(words) - i - 1
		if run >= want && len(out) < n-1 && left >= n-len(out)-1 {
			out = append(out, strings.Join(cur, " "))
			cur, run = nil, 0
		}
	}
	if len(cur) > 0 {
		out = append(out, strings.Join(cur, " "))
	}
	return out
}

// grayToImage is the words as something the grinder can take.
func grayToImage(g *image.Gray) image.Image {
	out := image.NewRGBA(g.Bounds())
	draw.Draw(out, out.Bounds(), g, g.Bounds().Min, draw.Src)
	return out
}

// wordsNow is what the picture should be showing: the line being sung, or the
// record and the artist where there is no synced lyric to show.
//
// The two are one picture rather than two modes because they answer the same
// question — what is this, right now — and because a song with no words in the
// database would otherwise leave the screen empty for its whole three minutes.
func (m Model) wordsNow() []string {
	lines, _ := m.wordsComing()
	return lines
}

// wordsComing is the line to show and when it is sung, on the playback clock.
//
// It looks a little ahead: a line takes a moment to gather, and a picture that
// starts gathering when the line starts is a picture that is complete half a
// second after the singer got there. Asked for early and timed against the
// line's own timestamp, the words finish arriving exactly as it begins.
func (m Model) wordsComing() ([]string, int64) {
	// Asked for by hand, and so before everything: whatever the record is doing
	// it can wait four seconds. See soloForcing.
	if m.soloForcing() {
		if name := m.soloName(); len(name) > 0 {
			return name, m.words.forced.UnixMilli()
		}
	}

	// And the marks, when they are asked for by hand. A set of them arrives on
	// the record's own turns, which is right for listening and useless for
	// looking: on a record with words they may not arrive at all. Asked for,
	// they come up over whatever is being sung, for as long as the name would.
	if m.marksForcing() {
		return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)}, m.words.showed.UnixMilli()
	}

	if m.lyrics.synced && m.ps != nil && m.lyrics.forTrack == m.ps.TrackID {
		at, clock := m.wordsAt(), m.wordsClock()

		// A bar nobody sings, and long enough to be worth something of its own:
		// the marks keep it. See solo.go.
		if gap, in := m.soloNow(); in {
			return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)}, gap.from
		}

		// The next line, if it is close enough that its gathering has begun.
		if next := at + 1; next < len(m.lyrics.lines) {
			if m.lyrics.lines[next].At-clock <= int64(wordsGather/time.Millisecond) {
				at = next
			}
		}

		if at >= 0 && at < len(m.lyrics.lines) {
			// A rest the sheet writes down: an entry with no words in it, where
			// the singer stops. It used to set nothing and be worth nothing —
			// the screen went blank on it, for however long the sheet said
			// nobody was singing.
			//
			// It is not a licence to draw nothing. A gap the sheet simply leaves
			// blank keeps the line before it up until the next one is due, and a
			// gap long enough to be worth a change of picture is already the
			// marks' — see soloNow, which is asked before this. A rest is the
			// same bar written down, so it reads the same way: step back to the
			// last line anybody sang and carry on.
			if strings.TrimSpace(m.lyrics.lines[at].Words) == "" {
				for back := at - 1; back >= 0; back-- {
					if strings.TrimSpace(m.lyrics.lines[back].Words) != "" {
						at = back
						break
					}
				}
			}

			if line := strings.TrimSpace(m.lyrics.lines[at].Words); line != "" {
				// A bar the sheet marks rather than writes: three notes rather
				// than the one it wrote, so the picture has the width of a line
				// and each of them can answer a different part of the sound.
				if wordsBeats(line) {
					return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)},
						m.lyrics.lines[at].At
				}
				return wordsWrap(line, m.width*dotsPerCellX, m.height*dotsPerCellY),
					m.lyrics.lines[at].At
			}

			// A rest with nothing sung before it at all — some sheets open with
			// one. There is no line to step back to, so it is the intro by
			// another name and the marks keep it.
			return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)}, m.lyrics.lines[at].At
		}

		// The intro. A gap long enough to be worth a change of picture is
		// already the marks' — see soloNow — and this is the same bar when it is
		// shorter than that: the record has begun and nobody has sung yet.
		//
		// Stamped at nought, which is where the intro starts and what soloNow
		// stamps it with too, so a record whose singer comes in at fourteen
		// seconds and one whose singer comes in at seven put up the same picture
		// in the same place rather than one of them putting up nothing.
		return []string{wordsMarks(m.width*dotsPerCellX, m.height*dotsPerCellY)}, 0
	}

	// A status that names no record.
	//
	// The daemon carries the track it is playing and carries nothing at all
	// while it loads the next one — no id, no title. Held on the skip key that
	// is where the player spends most of its time, and the screen spent it with
	// an empty band across the middle. It is not the end of the music, it is the
	// gap between two records, so the marks keep it like any other gap.
	//
	// Only before anything has played is there nothing to keep.
	if m.ps == nil || (m.ps.TrackID == "" && m.words.forTrack == "") {
		return nil, 0
	}

	// A record the database has nothing for. It is not given one thing for the
	// whole of its length: see idle.go.
	return m.wordsIdle()
}

// wordsMove is how a line comes apart and goes back together.
//
// Four of them rather than one, chosen by the line itself, because a change
// that happens the same way every time stops being a change after the third
// chorus. Which one a line gets is worked out from its own text, so a song
// plays the same way twice and a test can watch it do so.
type wordsMove int

const (
	wordsDrifting   wordsMove = iota // in from wherever, on its own line
	wordsRising                      // up from below, a row at a time
	wordsFalling                     // down from above, a row at a time
	wordsBursting                    // in from outside, along the line to the middle
	wordsSpreading                   // out from the middle, as if it were pushed
	wordsWiping                      // left to right, like something being written
	wordsWipingBack                  // right to left
	wordsBlurring                    // barely anywhere: a picture coming into focus
	wordsPopping                     // one piece at a time, and each one bursts
	wordsMoves
)

// wordsMoveFor picks one from the line, from when it is sung, and never the one
// before it.
//
// From the text alone it was the same move every time the words were, and a
// chorus is the same words three times in four minutes: watched on Ocho Macho's
// "Jó Nekem", where one line comes round three times and arrived identically on
// each, which reads as a loop rather than as a chorus. When it is sung is in the
// deal too now, so the same line comes in differently each time it comes round —
// and a record still plays the same way twice, because the sheet says the same
// times on a second listen.
//
// Popping is not among them: it is a way of leaving, and a line that arrived by
// un-bursting would be a film run backwards.
func wordsMoveFor(text string, starts int64, was wordsMove) wordsMove {
	var h uint32 = 2166136261
	for _, r := range text {
		h = (h ^ uint32(r)) * 16777619
	}
	// The moment folded in, and then the whole thing stirred. Folded in without
	// stirring, two lines forty seconds apart came out on the same move as often
	// as not: the top bits of a millisecond count barely move over a record, and
	// a remainder reads the bottom ones. This avalanches, so a second's
	// difference is a different answer.
	x := uint64(h)*0x9e3779b97f4a7c15 + uint64(starts)*0xd6e8feb86659fd93
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 29

	// One of the seven that is not the one just used, rather than one of the
	// eight. Dealt freely, three arrivals out of eight ways collide better than
	// a third of the time — measured on a chorus that comes round three times,
	// where two of the three drew the same way and looked it. A line coming in
	// exactly as the last one did is the one thing that reads as a mistake, and
	// it is also the only thing this deal can promise not to do.
	step := x % uint64(wordsPopping-1)
	return wordsMove((uint64(was) + 1 + step) % uint64(wordsPopping))
}

// wordsState is the picture the words are drawn from, and how far it has
// gathered since the line changed.
type wordsState struct {
	// move is how this line is coming together, and leave how the one before it
	// is going: a line that is simply replaced looks like a slide changing, and
	// a line that leaves the way it came looks like it was pushed.
	move, leave wordsMove

	// was is the line before this one, still on its way out, wasWhere is where
	// its pieces were — a picture that goes off a piece at a time has to know
	// what its pieces were — and went is when it was given notice.
	was      cover.Grain
	wasWhere msg.WordLayout
	went     time.Time

	have           cover.Grain
	text           string
	cellsX, cellsY int

	// asked is the text a picture has been sent for, with the size it was sent
	// for at, so that a text the setter refused is tried again when the screen
	// changes shape rather than being given up on for good.
	asked          string
	askedX, askedY int

	// since is when this line arrived. The dots gather over the moment after
	// it, which is what makes a line change something you see rather than
	// something you notice has happened.
	since time.Time

	// where each word of the line landed, so the one being sung can be told from
	// the ones that have been and the ones still to come.
	where msg.WordLayout

	// low and high are the range the sound's centre of gravity has been moving
	// through lately, which is what the colours are spread over. See
	// wordsColourNow.
	low, high float32

	// ends is when the line on screen gives way to the next.
	ends int64

	// beats says what is on screen is a mark rather than words — the note a
	// lyric sheet puts where a line would be if anyone were singing one.
	//
	// It is set like any other line, and drawn like any other line, with the
	// meter above and below it and the water crossing it. What it does not do is
	// keep a colour: nobody is singing it, so there is no moment to paint it
	// with and it follows the music instead.
	beats bool

	// forTrack is the record the picture was made for, so that nothing of one
	// record is left on the screen of the next, and cast which set of marks is
	// up — empty for the notes the face carries. See wordsGrind.
	forTrack string
	cast     string

	// rides is scratch for the sparks, kept so a frame does not allocate.
	rides []float32

	// roll is where the colour has got to on its way to the place the record's
	// turns have shoved it. See wordsRollEase.
	roll float32

	// forced is when the record's name was last asked for by hand, so that the
	// one thing on this screen that happens on purpose can be seen on purpose.
	forced time.Time

	// showed is when the marks were last asked for by hand, and picked which set
	// was asked for — empty for whatever the deal gives. A set arrives when the record turns over, which is a handful of
	// times in a record and never on demand — so a new one cannot be looked at
	// without a key, and a set that cannot be looked at cannot be judged. See
	// marksWalk.
	picked string
	showed time.Time

	// telling says what is on screen is the record's own name, put up once in
	// the middle of a solo. It arrives the same way every time and it holds
	// still while it is there: everything around it is already moving, and a
	// title card that jumps about with the rest is not a card, it is noise.
	telling bool

	// starts is when the line on screen is sung, on the playback clock. The
	// gathering is timed against it rather than against the moment the picture
	// happened to be built, so the words are complete as the line begins rather
	// than half a second into it.
	starts int64

	// band and head are the room the meter has, in rows under whatever is on
	// and in dots over it, eased towards what the picture actually leaves. See
	// wordsEase.
	band, head float32

	// swellLow and swellHigh are the range of loudness, in decibels, the record
	// has been moving through lately, which is what the size of every movement
	// is read against. See swell.go.
	swellLow, swellHigh float64

	// drive is how far the row is allowed to sway, nought to one, worked out
	// from how hard the low end is hitting; swayWas, swayHit and swayCeil are
	// what that is followed with. See sway.go.
	drive                      float32
	swayWas, swayHit, swayCeil float32
	swayHeard                  bool

	// leanAt is the bar the picture on screen arrived under, which is what its
	// lean is dealt from — rather than the bar playing now. The marks stay up
	// across a change of bar, and a row that snaps to a new set of angles
	// without going anywhere is a row that has been dealt again in place. See
	// wordsLeanSeed.
	leanAt int64
}

// wordPaint is a word's colour: which hue of the palette, and at what strength.
type wordPaint struct {
	hue, level int8
	set        bool
}

const (
	// wordsGather is how long the dots take to come together. A lyric line
	// stands for two or three seconds, so anything slower would leave the
	// picture permanently half made.
	wordsGather = 420 * time.Millisecond

	// wordsScatter is how far a dot starts from where it belongs, in dot rows
	// and columns.
	wordsScatter = 26

	// wordsLeaving is how long the line before this one takes to go. Shorter
	// than the gathering: what is arriving is what the eye should be on, and the
	// two overlap.
	wordsLeaving = 300 * time.Millisecond

	// wordsStagger is the share of the gathering that is spent waiting, for the
	// moves that arrive a row or a column at a time: at nothing they all land
	// together, at one the last dot only starts as the first one finishes.
	wordsStagger = 0.55

	// wordsPopHold is how much of the leaving one piece's own burst takes. The
	// starts are spread over the rest, so the bursts overlap — which is what
	// makes it a run down the row rather than a queue being served.
	wordsPopHold = 0.42

	// wordsPopFlies is how far a dot of a bursting piece gets, as a share of
	// how far it already is from the middle of that piece.
	wordsPopFlies = 2.5

	// wordsAhead is how bright a word that has not been sung yet is drawn, as a
	// share of the palette. Dim enough to read as waiting, bright enough to read
	// at all: the whole line has to be legible before it is sung, or it is a
	// guessing game rather than a lyric.
	wordsAhead = 0.28

	// wordsBand is the fewest rows worth giving the music under the words.
	//
	// A lyric rarely fills a screen — one line of type in the middle of forty
	// rows leaves most of them empty — and empty is the one thing this picture
	// can afford least. What is left under the words goes to the meter, so the
	// screen shows the words and the sound at once, and neither is squeezed:
	// under this many rows there is not enough of a picture to be worth the
	// distraction.
	wordsBand = 6

	// wordsRangeLeast is the narrowest range of sound the colours are spread
	// over, in bands, and wordsRangeClose how fast the range closes back in when
	// the sound stops going that far. Slow: a chorus that opens the range should
	// still be using it a verse later.
	wordsRangeLeast = 2.5
	wordsRangeClose = 0.0015

	// wordsLeanRise is how far the far end of a leaning word stands above its
	// near one, as a share of the height of the type it is set in, and
	// wordsTiltMost the steepest it is ever allowed to get there.
	//
	// The rise is measured against the type rather than fixed as an angle
	// because a word can be anything from three dots wide to a hundred: at one
	// angle for all of them the short words would not move at all and the long
	// ones would stand on end. Against the type, every word leans by the same
	// amount of what is on screen — and the cap keeps a three-letter word from
	// getting there by going vertical.
	wordsLeanRise = 0.35
	wordsTiltMost = 0.16

	// wordsTiltMark is how far a mark slants instead, sideways over its own
	// height — a shade past what a type designer would call italic, because
	// three notes leaning are meant to be noticed and a lower case l is not.
	wordsTiltMark = 0.22

	// wordsWordRide is how far a word rides its own part of the sound. Well under
	// what a note is given: a lyric has to be read while it moves, and a row of
	// seven marks can throw itself about in a way a line of a dozen words cannot
	// without coming apart. Less again because the words move against each other
	// — the same travel reads as twice as much when the word beside it went the
	// other way.
	wordsWordRide = 12

	// wordsBounce is how far a mark rides its own part of the sound, in dots.
	//
	// Three and a half cells at a beat. It was a third of that when a bar was
	// three marks in the middle of the screen and a big movement read as a
	// fairground; a row of seven across the width is a different picture — there
	// the travel is what makes the sound run along the row, and too little of it
	// is a line of type that happens to twitch.
	wordsBounce = 28

	// wordsLit is how bright a dot of the rasterised type has to be to be set.
	// Half: type is drawn white on black and anti-aliased, so this is the edge
	// of the letter — above it the stroke, below it the air around it.
	wordsLit = 128

	// wordsNotes is the shortest bar of marks there is: what a screen too narrow
	// for anything else gets. See wordsMarks for what is usually put up.
	wordsNotes = "♪ ♫ ♪"

	// wordsMark is how much of the height a mark standing in for a line is set
	// in, as against a line of words, which is given all of it.
	wordsMark = 0.34

	// wordsCeiling is the row the hanging picture starts at.
	//
	// It used to be the third, because the track's name and the clock were set
	// over the top of everything else. They are gone, and the two rows they were
	// keeping are the top of the screen — so the meter hangs from the very edge,
	// where the picture starts.
	wordsCeiling = 0

	// wordsReach is how much of its band a column of the meter takes on this
	// screen, as against the 0.72 the other pictures leave. Here the water is
	// thrown from the tips and given the whole terminal to cross, so the room
	// held back over the columns bought nothing — it was a strip of empty screen
	// at the top and the bottom, which is what a picture like this can least
	// afford.
	wordsReach = 0.94

	// wordsRoomEase is how much of the way the meter's room moves towards what
	// the picture leaves it, each frame. At thirty frames a second this is most
	// of the way inside the time a line takes to gather. See wordsEase.
	wordsRoomEase = 0.12

	// wordsSpray is what a drop keeps of its light the moment it leaves the band
	// and enters the lyric, and wordsSpent how sharply what is left falls away
	// with height.
	//
	// Both are set low. What crosses the words has to be visible as movement and
	// invisible as ink: a spark at the same strength as a letter is read as part
	// of the letter, and a screenful of them turns a lyric into a texture. Cubed
	// rather than squared, the fall is quick enough that the air around the type
	// is nearly clear while the band underneath still throws hard.
	wordsSpray = 0.28
	wordsSpent = 3

	// wordsLift is what the water's throw is multiplied by on this screen.
	//
	// Set so that a hard hit carries a drop the height of the terminal: a drop
	// rises by its speed squared over twice the gravity, and the ordinary throw
	// is cut for a picture whose columns are the whole of it. Here they are a
	// band along the foot and everything above them is the lyric, which is
	// exactly what the water is for — it is the one thing on this screen that
	// crosses everything else on it.
	wordsLift = 2
)

// wordsLines draws the words, w cells across and rows deep.
func (m Model) wordsLines(w, rows int) []string {
	g := m.words.have
	if g.DotsX == 0 || g.CellsX != w || g.CellsY != rows {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	freqs, levels := len(m.styles.Words), len(m.styles.Words[0])

	grid := make([]uint8, w*rows)
	paint := make([]int8, w*rows)
	hue := make([]int8, w*rows)
	for i := range paint {
		paint[i] = -1
	}
	for r := range rows {
		for c := range w {
			hue[r*w+c] = int8(min(c*freqs/w, freqs-1))
		}
	}

	// How far along the gathering is. Held at one once it is over, so the
	// steady picture costs no arithmetic it does not need.
	gather := float32(1)
	if since := time.Since(m.words.since); since < wordsGather {
		gather = float32(since) / float32(wordsGather)
	}

	// The line before this one, on its way out: the same arithmetic run
	// backwards, and fading as it goes.
	if since := time.Since(m.words.went); since < wordsLeaving && m.words.was.DotsX == dotsX {
		m.drawLeaving(grid, paint, hue, w, rows, float32(since)/float32(wordsLeaving), levels)
	}

	// What each word is burning at, and in what colour.
	//
	// The whole line, all of it lit, and every word answering its own share of
	// the sound: the first word beats with the bass and the last with the
	// cymbals, so the line moves in time without any of it claiming to know
	// where the singer has got to.
	//
	// It used to claim exactly that — a light travelling along the line, word by
	// word. The trouble is that a lyric sheet says when a line starts and
	// nothing else, so where the singer is inside it is a guess; and a guess
	// that is right most of the time is worse than no guess at all, because the
	// times it is wrong are the times you are looking straight at it.
	paints := make([]wordPaint, m.words.where.Count)
	for i := range paints {
		paints[i] = m.wordsPaint(i, len(paints), freqs, levels)
		if gather < 1 {
			paints[i].level = int8(float32(paints[i].level) * gather)
		}
	}

	bounce := m.wordsRiding(len(paints))
	tilt, middle := m.wordsTilting(len(paints))
	turned := m.wordsTurning(len(paints))

	for y := range dotsY {
		for x := range dotsX {
			if g.Lum[y*dotsX+x] < wordsLit {
				continue
			}

			// Where this dot is on its way from. The offset is worked out from
			// where the dot belongs rather than stored, so a screenful of them
			// costs nothing to remember and comes apart the same way twice.
			at, to := x, y

			// How far along this particular dot is. Some of the moves arrive a
			// row or a column at a time, so each dot has its own share of the
			// gathering rather than all of them having the whole of it.
			if word := m.words.where.WordAt(x, y); word >= 0 {
				if word < len(bounce) {
					to += bounce[word]
				}
				// Turned round on the spot, about its own ink rather than about
				// the space it stands in — mirrored about the middle of its
				// column and the gap beside it, a mark slides half a gap
				// sideways every time it turns, which reads as a shuffle.
				if word < len(turned) && turned[word] {
					at = m.words.where.Lefts[word] + m.words.where.Rights[word] - x
				}
				if word < len(tilt) && tilt[word] != 0 {
					// A shear rather than a turn. At these angles the two look
					// the same, and a shear moves whole columns of a word — or
					// whole rows of a mark — together; a real rotation would
					// open holes between them and the shape would come apart at
					// the seams.
					if m.words.beats {
						at += int(math.Round(float64(middle[word]-y) * float64(tilt[word])))
					} else {
						to += int(math.Round(float64(x-middle[word]) * float64(tilt[word])))
					}
				}
				if to < 0 || to >= dotsY {
					continue
				}
			}

			p := wordsAlong(m.words.move, gather, x, y, dotsX, dotsY)

			if p < 1 {
				dx, dy := wordsFrom(m.words.move, x, y, dotsX, dotsY)
				at += int(dx * (1 - p))
				to += int(dy * (1 - p))
				if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
					continue
				}
			}

			// Every offset above can carry a dot off the screen — the lean
			// shears whole columns sideways, and a mark turning round starts
			// its column somewhere else — so the last word on where a dot lands
			// is here rather than inside each of them. Left to the movements to
			// check for themselves, one of them did not: a turned mark leaning
			// took a dot to a negative column, and the picture went down with
			// an index of minus one.
			if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
				continue
			}

			cell := (to/dotsPerCellY)*w + at/dotsPerCellX
			grid[cell] |= 1 << brailleBit[at%dotsPerCellX][to%dotsPerCellY]

			// The colour comes from the word this dot belongs to, wherever the
			// dot has been thrown to: a letter keeps its word's colour while it
			// is still in the air.
			if word := m.words.where.WordAt(x, y); word >= 0 && word < len(paints) {
				if p := paints[word]; p.level > paint[cell] {
					paint[cell], hue[cell] = p.level, p.hue
				}
			} else if paint[cell] < 0 {
				paint[cell] = int8(min(int(wordsAhead*float32(levels)), levels-1))
			}
		}
	}

	// Whatever the words left over goes to the music, drawn into the same grid
	// rather than into a picture of its own — which is what lets the water off
	// the leash: a drop thrown from the meter can cross the whole screen and
	// pass through the lyric, instead of stopping at the top of a box.
	if tall, head := m.wordsBandNow(w, rows); tall >= wordsBand {
		m.wordsUnder(grid, paint, hue, w, rows, tall, head)
	}

	return m.drawCellsIn(w, rows, grid, paint, hue, m.styles.Words, m.styles.WordsSeq)
}

// wordsUnder draws the meter into the rows the words left over, and its water
// over the whole screen.
func (m Model) wordsUnder(grid []uint8, paint, hue []int8, w, rows, tall, head int) {
	dotsX := w * dotsPerCellX
	dotsY := rows * dotsPerCellY
	levels, freqs := len(m.styles.Words[0]), len(m.styles.Words)

	// The floor the columns stand on and the water is thrown from: the foot of
	// the screen, which is also the foot of the band. The ceiling is where the
	// same picture hangs from, upside down — the lyric sits between the two, and
	// the screen reads as one thing rather than as a strip with a heading over
	// it.
	floor := dotsY - 1
	ceiling := wordsCeiling * dotsPerCellY

	light := func(x, y int, step int8) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if step > paint[cell] {
			paint[cell] = step
			hue[cell] = int8(min(x/dotsPerCellX*freqs/w, freqs-1))
		}
	}

	// The columns, standing on the floor and hanging from the ceiling, as tall
	// as the band allows.
	reach := wordsReach * float32(tall*dotsPerCellY)

	for x := range dotsX {
		if x%stagePitch != 0 {
			continue
		}
		height := int(m.stageLevel(x, dotsX) * reach)
		for y := range height {
			up := float32(y) / float32(max(height-1, 1))
			step := int8(min(int(up*float32(levels)), levels-1))

			light(x, floor-y, step)
			if y < head {
				light(x, ceiling+y, step)
			}
		}
	}

	// The water, which belongs to the whole screen. A drop thrown hard enough
	// rises past the words and falls back through them, which is the one thing
	// on this picture that crosses everything else on it.
	//
	// It fades as it climbs. Inside the band it burns as it would on any of the
	// other screens; above it, where the lyric is, it is spent — by the top of
	// the terminal there is almost nothing left of it. The drops are meant to be
	// seen through the words rather than against them, and a bright speck beside
	// a letter is read as part of the letter.
	band := float32(tall * dotsPerCellY)
	sky := max(float32(dotsY)-band, 1)

	for _, d := range m.stage.drops {
		spent := float32(1)
		if above := d.at - band; above > 0 {
			spent = max(1-above/sky, 0)
			spent = float32(math.Pow(float64(spent), wordsSpent)) * wordsSpray
		}

		step := int8(min(int(d.bright*spent*float32(levels)), levels-1))
		light(d.col, floor-int(d.at), step)
		if head > 0 {
			light(d.col, ceiling+int(d.at), step)
		}
	}
}

// wordsMarks is the row of marks a bar of music gets: as many as go across the
// screen at the size one of them is set at, rather than the three that used to
// sit in the middle of it.
//
// The size is not the question — the marks are set in the same band a line of
// type would be, and that is what decides how tall they are. The question is
// how many of them fit beside each other at that height, and the answer is
// found by asking the setter: add one until it would have to make them smaller.
func wordsMarks(w, h int) string {
	if w <= 0 || h <= 0 {
		return wordsNotes
	}

	fit := int(wordsMark * float64(h))
	line := "♪"
	was := wordsSize([]string{line}, w, fit)
	if was <= 0 {
		return wordsNotes
	}

	// Alternating, so a row of them is a phrase rather than a fence.
	next := []string{"♫", "♪", "♪", "♫", "♪"}
	for n := range wordsMarksMost {
		try := line + " " + next[n%len(next)]
		if wordsSize([]string{try}, w, fit) < was {
			break
		}
		line = try
	}
	return line
}

// wordsMarksMost is where the row stops however wide the screen is. Past this
// they stop being marks somebody put up and start being wallpaper.
const wordsMarksMost = 24

// wordsBeats reports that a line is a mark rather than words: the note a lyric
// sheet puts where a line would be if anyone were singing one.
func wordsBeats(text string) bool {
	var marks int
	for _, r := range text {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			return false
		case !unicode.IsSpace(r):
			marks++
		}
	}
	return marks > 0
}

// wordsEnds is when the line that starts at a given moment gives way.
func (m Model) wordsEnds(starts int64) int64 {
	for i, line := range m.lyrics.lines {
		if line.At == starts && i+1 < len(m.lyrics.lines) {
			return m.lyrics.lines[i+1].At
		}
	}
	return 0
}

// wordsClock is the playhead the big screen reads the words against: the one the
// music is actually at, with nothing added to it.
//
// The window on the player screen reads half a second ahead, because a line lit
// as its first syllable sounds has already been passed by the time the eye gets
// to it. Up here that lead is wrong twice over: the line is the size of the
// screen and takes no finding, and the gathering — which is nearly half a second
// of dots flying into place — is itself the warning that a line is coming. Read
// ahead as well, the finished line stands there for half a second before anybody
// sings it, and the picture appears to be rushing the song.
func (m Model) wordsClock() int64 { return m.elapsed().Milliseconds() }

// wordsAt is which line that clock is on.
func (m Model) wordsAt() int {
	if !m.lyrics.synced {
		return 0
	}

	pos, at := m.wordsClock(), -1
	for i, line := range m.lyrics.lines {
		if line.At > pos {
			break
		}
		at = i
	}
	return at
}

// wordsSilent reports that the song has words but is not singing any right now:
// the bar before the first line, or a gap a lyric sheet marks with an empty one.
//
// It is not the same as having no lyrics at all. A song with none falls back to
// its title, which is worth looking at for three minutes; a song between two
// lines is a song playing, and what belongs on the screen then is the playing.
func (m Model) wordsSilent() bool {
	// What is coming counts as words: the gathering of the next line begins
	// before the singer reaches it. So does a chase, which has no words in it
	// but is very much something to look at.
	lines, _ := m.wordsComing()
	return len(lines) == 0
}

// wordsHeadroom is how far the picture hanging from the ceiling may come down
// before it reaches the words, in dot rows.
func (m Model) wordsHeadroom(rows int) int {
	if len(m.words.where.Tops) == 0 {
		return 0
	}

	high := m.words.where.Tops[0]
	for _, t := range m.words.where.Tops {
		high = min(high, t)
	}

	// A row of clear air under it, as under the words themselves.
	return max(high-wordsCeiling*dotsPerCellY-dotsPerCellY, 0)
}

// wordsRoomNow is the room the meter has this frame, whatever is holding the
// middle of the screen: a line of words, a row of marks, a card, a drawn figure
// or the face this code draws itself.
//
// Every one of them leaves a different amount, and until this was asked in one
// place each of them worked it out for itself at the moment it was drawn. The
// meter stands on the floor and hangs from the ceiling either way, so what
// changed at every change of picture was how far the columns reached — and it
// changed between one frame and the next. A row of marks giving way to a line
// of words moved three rows of meter across the whole width of the screen in a
// single frame, which is the picture cutting rather than the picture moving.
func (m Model) wordsRoomNow(w, rows int) (tall, head int) {
	if p, _, top, ok := m.faceRoom(w, rows); ok && m.faceUp() && m.faceWho() == "" {
		return max((rows*dotsPerCellY-(top+p.h))/dotsPerCellY, 0), max(top-dotsPerCellY, 0)
	}
	if tall, head, ok := m.figureRoom(w, rows); ok {
		return tall, head
	}
	_, tall = m.wordsRoom(rows)
	return tall, m.wordsHeadroom(rows)
}

// wordsEase moves the meter's room towards what the picture on screen leaves
// it, rather than taking it whole.
//
// Over the time a line takes to gather, so that the meter arrives with whatever
// it is making room for: while it is still too tall for the new picture the new
// picture is not there yet — it is halfway through coming together, a scatter of
// dim specks — and by the time there is a solid line of type to touch, the
// columns have pulled back under it.
func (m *Model) wordsEase(w, rows int) {
	tall, head := m.wordsRoomNow(w, rows)
	if m.words.band == 0 && m.words.head == 0 {
		m.words.band, m.words.head = float32(tall), float32(head)
		return
	}

	m.words.band += (float32(tall) - m.words.band) * wordsRoomEaseAt
	m.words.head += (float32(head) - m.words.head) * wordsRoomEaseAt
}

// wordsBandNow is the meter's room as it is drawn this frame: what it has been
// eased to, or what the picture leaves it on the first frame of all, before
// there has been anything to ease from.
func (m Model) wordsBandNow(w, rows int) (tall, head int) {
	if m.words.band == 0 && m.words.head == 0 {
		return m.wordsRoomNow(w, rows)
	}
	return int(m.words.band + 0.5), int(m.words.head + 0.5)
}

// wordsRoom is the band left under the words: where it starts, in rows, and how
// many there are.
func (m Model) wordsRoom(rows int) (from, tall int) {
	if len(m.words.where.Bottoms) == 0 {
		return 0, 0
	}

	low := 0
	for _, b := range m.words.where.Bottoms {
		low = max(low, b)
	}

	// A row of clear air under the type, so the meter never touches the letters.
	from = low/dotsPerCellY + 2
	return from, max(rows-from, 0)
}

// drawLeaving puts the line before this one back on its way out: where each dot
// came in from is where it goes, and it fades as it gets there.
func (m Model) drawLeaving(grid []uint8, paint, hue []int8, w, rows int, gone float32, levels int) {
	g := m.words.was
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	step := int8(min(int((1-gone)*wordsAhead*float32(levels)), levels-1))
	if step < 0 {
		return
	}

	for y := range dotsY {
		for x := range dotsX {
			if g.Lum[y*dotsX+x] < wordsLit {
				continue
			}

			var at, to int
			var burn int8

			if m.words.leave == wordsPopping {
				var ok bool
				at, to, burn, ok = m.wordsPop(x, y, gone, levels)
				if !ok {
					continue
				}
			} else {
				p := 1 - wordsAlong(m.words.leave, gone, x, y, dotsX, dotsY)
				dx, dy := wordsFrom(m.words.leave, x, y, dotsX, dotsY)
				at, to, burn = x+int(dx*(1-p)), y+int(dy*(1-p)), step
			}

			if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
				continue
			}

			// Every offset above can carry a dot off the screen — the lean
			// shears whole columns sideways, and a mark turning round starts
			// its column somewhere else — so the last word on where a dot lands
			// is here rather than inside each of them. Left to the movements to
			// check for themselves, one of them did not: a turned mark leaning
			// took a dot to a negative column, and the picture went down with
			// an index of minus one.
			if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
				continue
			}

			cell := (to/dotsPerCellY)*w + at/dotsPerCellX
			grid[cell] |= 1 << brailleBit[at%dotsPerCellX][to%dotsPerCellY]
			if burn > paint[cell] {
				paint[cell] = burn
			}
		}
	}
}

// wordsPop is where a dot of the picture on its way out has got to, when the
// picture is going off one piece at a time.
//
// A row of marks does not want to be wiped or slid away: it wants to go the way
// a row of anything goes when somebody walks down it — the first one bursts, and
// then the next, and then the next. Each piece waits its turn, holds still while
// it waits, and then flies apart from its own middle and is gone.
func (m Model) wordsPop(x, y int, gone float32, levels int) (int, int, int8, bool) {
	where := m.words.wasWhere
	piece := where.WordAt(x, y)
	if piece < 0 || where.Count == 0 {
		return 0, 0, 0, false
	}

	// Whose turn it is, and how far into that turn we are. The turns are spread
	// over what is left after one burst's own length, so the last of them
	// finishes exactly as the leaving does rather than being cut off.
	start := float32(piece) / float32(max(where.Count-1, 1)) * (1 - wordsPopHold)
	mine := (gone - start) / wordsPopHold
	switch {
	case mine <= 0:
		// Still standing there, waiting to go.
		return x, y, int8(min(int(wordsAhead*float32(levels)), levels-1)), true
	case mine >= 1:
		return 0, 0, 0, false
	}

	// Gone off: out from the middle of its own piece, fast at first, and out of
	// light before it has gone far.
	cx, cy := where.Middle(piece)
	dx, dy := float32(x-cx), float32(y-cy)
	fly := mine * mine * wordsPopFlies

	burn := int8(float32(levels-1) * (1 - mine) * wordsAhead * 2)
	if burn < 0 {
		return 0, 0, 0, false
	}
	return x + int(dx*fly), y + int(dy*fly), burn, true
}

// wordsStep is one dot's share of the gathering, given how far down the queue
// it is: the ones at the front are done before the ones at the back start.
func wordsStep(gather, behind float32) float32 {
	p := (gather - behind*wordsStagger) / (1 - wordsStagger)
	return min(max(p, 0), 1)
}

// wordsAlong is how far along one dot is, which for the moves that arrive a row
// or a column at a time is its own share rather than the whole of it.
func wordsAlong(move wordsMove, gather float32, x, y, dotsX, dotsY int) float32 {
	switch move {
	case wordsWiping:
		return wordsStep(gather, float32(x)/float32(dotsX))
	case wordsWipingBack:
		return wordsStep(gather, 1-float32(x)/float32(dotsX))
	case wordsRising:
		return wordsStep(gather, 1-float32(y)/float32(dotsY))
	case wordsFalling:
		return wordsStep(gather, float32(y)/float32(dotsY))
	}
	return gather
}

// wordsFrom is where a dot comes in from, given the move and where it lands.
func wordsFrom(move wordsMove, x, y, dotsX, dotsY int) (float32, float32) {
	out := func(by float32) (float32, float32) {
		dx, dy := float32(x-dotsX/2), float32(y-dotsY/2)
		d := float32(math.Hypot(float64(dx), float64(dy)))
		if d == 0 {
			return 0, 0
		}
		return dx / d * by, dy / d * by
	}

	switch move {
	case wordsRising:
		// Up from under the line, with enough of a wobble that it is a crowd
		// arriving rather than a blind coming up.
		dx, _ := wordsDrift(x, y)
		return dx * 0.15, float32(dotsY-y) * 0.9

	case wordsFalling:
		dx, _ := wordsDrift(x, y)
		return dx * 0.15, -float32(y) * 0.9

	case wordsBursting:
		// In from outside, along the line out from the middle — the picture
		// implodes onto the words.
		return out(wordsScatter * 2.2)

	case wordsSpreading:
		// The other way about: everything starts in the middle and is pushed
		// out into its place.
		return out(-wordsScatter * 0.9)

	case wordsWiping:
		// From the left, as if the line were being written.
		return -float32(wordsScatter) * 1.4, 0

	case wordsWipingBack:
		return float32(wordsScatter) * 1.4, 0

	case wordsBlurring:
		// Hardly anywhere at all: every dot a hair out of place, so the line
		// arrives the way a lens comes into focus rather than by travelling.
		dx, dy := wordsDrift(x, y)
		return dx * 0.16, dy * 0.16
	}

	return wordsDrift(x, y)
}

// wordsDrift is where a dot comes in from when it comes in on its own line: a
// fixed swirl rather than a random walk, so the same line always gathers the
// same way and a test can watch it do so.
func wordsDrift(x, y int) (float32, float32) {
	h := uint32(x)*2654435761 + uint32(y)*2246822519
	h ^= h >> 13
	h *= 3266489917
	h ^= h >> 16

	// Two numbers from one hash: an angle and a distance.
	angle := float64(h&0xffff) / 0xffff * 2 * math.Pi
	far := float32(h>>16&0xff)/0xff*wordsScatter + 4
	return far * float32(math.Cos(angle)), far * float32(math.Sin(angle))
}

// wordsGrind builds the picture for the line now, if what is held is not it or
// not the size of this screen.
func (m *Model) wordsGrind() tea.Cmd {
	// A picture belongs to the record it was made for.
	//
	// Without this the last line of the record before is still held when the
	// next one's first line arrives — and the arrival hands whatever is held to
	// the leaving animation, so a sentence from the previous song flew out
	// across the first line of this one. Every skip, every time.
	// A status that names no record is not a change of record. The daemon
	// carries no track at all while it loads the next one, and read as a change
	// that threw the picture away and put nothing back — the empty band across
	// the middle, for as long as the load took, which on a run of skips is most
	// of the time.
	//
	// And a bar of marks survives a change of record where a line of type cannot.
	// The line was sung on the record it was set for and has no business on the
	// next one; the marks belong to no record — they are what this screen rests
	// on between the words — so carrying them over is the change of record
	// costing nothing at all rather than costing a gather.
	if m.ps != nil && m.ps.TrackID != "" && m.words.forTrack != m.ps.TrackID {
		m.words.forTrack = m.ps.TrackID
		if !wordsBeats(m.words.text) {
			m.words.have, m.words.was = cover.Grain{}, cover.Grain{}
			m.words.where, m.words.wasWhere = msg.WordLayout{}, msg.WordLayout{}
			m.words.text, m.words.asked = "", ""
		}
	}

	lines, starts := m.wordsComing()
	if len(lines) == 0 {
		return nil
	}
	was := m.words.starts
	m.words.starts, m.words.ends = starts, m.wordsEnds(starts)
	m.words.beats = len(lines) == 1 && wordsBeats(lines[0])
	// What is going up is the record's name when it is the record's name: asked
	// of the lines rather than of the clock, so that the one place which knows
	// the card is on is the one that puts it there.
	name := m.soloName()
	m.words.telling = len(name) > 0 && slices.Equal(lines, name)

	text := strings.Join(lines, "\n")

	// Which marks a bar is dealt is part of what is on screen, so it belongs in
	// the test below. Without it the row went up once and stayed: every bar of
	// marks is the same string of notes, so a later bar dealt the drawings was
	// taken for the picture already held and never asked for. Over a whole
	// wordless record that is one deal rather than one every half minute.
	cast := ""
	if m.words.beats && marksDrawn {
		cast = markCastFor(m.words.forTrack, starts)

		// A set asked for by hand holds the screen for as long as the asking
		// does, and then the deal has it back. Held for good, the key was not a
		// look at a set but a choice of one: the row never changed again for the
		// rest of the session, and the notes — which are the deal's own empty
		// hand, and what a bar of music has looked like here from the start —
		// never came up at all. The choice is remembered so that pressing the
		// key again carries on where it left off. See marksWalk.
		if m.words.picked != "" && m.marksForcing() {
			cast = m.words.picked
		}

		// Silenced, and the row says so: the same company with their fingers in
		// their ears, for as long as it lasts. Last, because it is a state
		// rather than a request — whatever else was up, this is what is true.
		if m.muted() {
			cast = markHush
		}
	}

	if m.words.text == text && m.words.cast == cast && m.words.cellsX == m.width && m.words.cellsY == m.height {
		// The same words over again, a turn or a line later: nothing to draw
		// that is not already held, but it does have to arrive rather than
		// simply be there — a chorus that repeats a line, and a wordless record
		// dealt the same card twice, both come through here.
		//
		// Except the marks. They are not a line that has come round again, they
		// are what the screen rests on between the lines, and they are stamped
		// afresh every time the bar under them moves — at the top of every gap,
		// every half minute of a record with no words, and at every change of
		// record. Made to arrive again each time, the same three notes flew
		// apart and gathered back for no reason anybody watching could see. The
		// worst of them was the change from one record to the next: nine
		// hundred cells in a single frame, against ninety either side of it,
		// for a picture that did not change at all.
		if starts != was && !m.words.beats {
			m.words.since, m.words.leanAt = time.Now(), starts

			// And dealt afresh, which it was not. This is the branch a chorus
			// comes through — the same words, so the picture already held is the
			// picture wanted and nothing is sent for — and the deal lives in
			// wordsAdopt, which is only reached when a picture arrives. So the
			// gathering restarted with the move the line had last time, every
			// time: a refrain that came in identically on each of its returns,
			// while the line after it, whose words were new, came in differently.
			// Read straight off the screen with the move printed on it — 4 for
			// every occurrence of the chorus, 5 for the line that followed.
			m.words.move = wordsMoveFor(text, starts, m.words.move)
		}
		return nil
	}
	// A bar of marks may be drawn rather than set. Which it is comes from the
	// bar, so a record shows both — see markCastFor — and when it is drawn the
	// picture is already dots: there is nothing to send for and nothing to wait
	// for, so it goes up in this frame rather than the next.
	//
	// Before the guard below, not after it. That one remembers what was asked of
	// the setter and refuses to ask twice, and every bar of marks asks for the
	// same string of notes — so once a row of notes had been sent for, the
	// drawings were never reached again for the rest of the record.
	if cast != "" {
		if grain, layout, ok := markPicture(cast, m.width, m.height, starts); ok {
			m.wordsAdopt(grain, layout, text)
			m.words.cast = cast
			m.words.cellsX, m.words.cellsY = m.width, m.height
			m.words.leanAt = starts
			return nil
		}
	}
	m.words.cast = ""

	// A picture that could not be built is not asked for again — the face has
	// no glyph for it, or there is no room to set it — but only at the size it
	// failed at. Without the size in the test, one refusal held for the rest of
	// the record and a window that had since been made wider changed nothing.
	if m.words.asked == text && m.words.askedX == m.width && m.words.askedY == m.height {
		return nil
	}

	m.words.asked, m.words.askedX, m.words.askedY = text, m.width, m.height
	m.words.leanAt = starts
	return wordsCmd(lines, m.width, m.height)
}

// wordsAdopt puts a picture on the screen: the one that was there is given
// notice, and this one is told how to arrive.
//
// Shared by the two ways a picture is made. A line of type is set by the face,
// off this goroutine, and comes back as a message; a bar of drawn marks is
// already dots and is put up in the frame it was asked for. What happens to it
// once it is here is the same either way.
func (m *Model) wordsAdopt(grain cover.Grain, where msg.WordLayout, text string) {
	// What is on screen is given notice, and goes out the way the line before
	// it came in.
	if m.words.have.DotsX == grain.DotsX {
		m.words.was, m.words.wasWhere = m.words.have, m.words.where
		m.words.went, m.words.leave = time.Now(), m.words.move

		// A row of marks goes off the way a row of anything goes when somebody
		// walks down it: one bursts, then the next. See wordsPop.
		//
		// Asked of the text that is leaving rather than of the flag, which by
		// now says what is arriving.
		if wordsBeats(m.words.text) {
			m.words.leave = wordsPopping
		}
	}

	m.words.have, m.words.text = grain, text
	m.words.move = wordsMoveFor(text, m.words.starts, m.words.move)

	if m.words.telling {
		// The one picture that is not dealt its arrival: the record's name comes
		// in from outside every time, and the marks it took the bar from go out
		// the way it came. A ceremony has to be the same twice or it is just
		// another transition.
		m.words.move = wordsBursting
	}
	m.words.where = where

	// Wound back so that the gathering finishes as the line is sung rather than
	// starting then: whatever is left of the wait is taken off what the
	// gathering costs.
	//
	// Except the record's name, which is not sung and so has nothing to be on
	// time for. Its moment is when it starts arriving, not when it has to be
	// there — wound back it would be complete before anybody saw it move, and
	// the card that comes up on the key would be the livelier of the two for no
	// reason anyone could name.
	if m.words.telling {
		m.words.since = time.Now()
		return
	}

	wait := time.Duration(m.words.starts-m.wordsClock()) * time.Millisecond
	m.words.since = time.Now().Add(-max(wordsGather-wait, 0))
}

// wordsFlow keeps the range of sound the colours are spread over.
//
// It is the only thing the line needs following now: where the singer has got
// to inside it is not asked, because a lyric sheet cannot answer it, and every
// word answers a part of the sound instead.
// wordsRollEase is how fast the colour slides to where the join put it.
//
// A step and not a jump, and that is the whole of whether it can be seen. A
// lyric line lives a few seconds and a join comes every twenty to sixty, so the
// line that was up before a step is long gone by the time of the one after it —
// nobody ever sees the same words in two colours. Slid over a second or so, the
// step is movement while it happens, which is a thing to watch rather than two
// states to compare.
const wordsRollEase = 0.02

// wordsRollFlow carries the colour toward where the record's turns have put it,
// the shorter way round.
func (m *Model) wordsRollFlow() {
	want := m.wordsRollAt()
	d := want - m.words.roll
	for d > 0.5 {
		d--
	}
	for d < -0.5 {
		d++
	}
	m.words.roll += d * wordsRollEaseAt
	for m.words.roll >= 1 {
		m.words.roll--
	}
	for m.words.roll < 0 {
		m.words.roll++
	}
}

func (m *Model) wordsFlow(w, rows int) {
	// How hard the low end is hitting, which is how far the row may sway, and
	// how loud the record is against itself, which is how far anything moves at
	// all. Both kept even while there is nothing on screen, so that a bar of
	// marks going up in the middle of a record arrives into a picture that
	// already knows what the record is doing rather than starting from nothing.
	m.swayFlow()
	m.swellFlow()

	if m.words.where.Count == 0 {
		return
	}

	if centre, _ := wordsCentre(m.scope.bands); len(m.scope.bands) > 0 {
		m.wordsFollowCentre(centre)
	}
}

// How a line keeps time has been narrowed twice, and both are worth keeping
// written down because both were dealt from the line and looked like variety.
//
// One line in three used to keep it not at all — a caption sitting perfectly
// still in the middle of a screen where the meter, the water and the marks are
// all answering the music. Beside a line that moves, the one that does not reads
// as a picture that has stopped rather than as a line with a different
// character.
//
// Half of what was left nodded as one block, on the loudest thing playing. That
// is gone too: watched against each other, a line whose words each live on their
// own part of the sound is better than the same line moving as a slab, and a
// slab is what a whole line lifted together is. What is left is the one way, and
// the variety comes from the music rather than from a deal.

// wordsRiding is how far each word of the line is lifted this frame.
//
// Never a dot at a time: the letters of a word move together or not at all, or
// the word comes apart and stops being one. And never as far as a note goes — a
// note is a mark keeping time and a word is something to read.
//
// The record's name rides with the rest of them. It used to be held still, on
// the grounds that a card is a caption and a caption that jumps about is not
// one; but it only ever goes up because somebody sent for it, into a screen
// where the meter, the water and the marks are all answering the music, and the
// one thing standing rigid in the middle of that reads as a picture that has
// stopped rather than as a card.
func (m Model) wordsRiding(count int) []int {
	if count <= 0 {
		return nil
	}

	if m.words.beats {
		// The notes: each on its own part of the sound, and further than a word
		// would go.
		//
		// The sound's, all of it, whether or not the screen is keeping time.
		// Putting the rise on the beat as well was tried and thrown out — a
		// height answering how loud a band is and where the beat is at the same
		// time answers neither, and what is left is a row twitching. The beat
		// has the lean instead: see sway.go.
		//
		// How far they go at all grows with the record: see swell.go.
		// Centred on the line the row was set on, rather than hanging off it.
		//
		// The ride only ever goes one way — a band is loud or it is not — so the
		// whole row rises as the travel does, and at a travel worth watching it
		// had climbed clear of where the picture put it. Half the travel down
		// puts the middle of the movement back on the line: a quiet band rests a
		// little below it, a loud one goes as far above.
		travel := wordsBounce * m.swell()
		out := make([]int, count)
		for i := range out {
			out[i] = int(travel/2) - int(m.wordsBeatRide(i, count)*travel)
		}
		return out
	}

	// The sound's, and only the sound's. The beat used to be multiplied in here
	// as well, and it is the same mistake it was on the marks: a height that
	// answers how loud a band is *and* where the beat is answers neither. The
	// beat is the marks' to keep — a line of a song is there to be read.
	travel := wordsWordRide * m.swell()
	out := make([]int, count)
	for i := range out {
		out[i] = -int(m.wordsBeatRide(i, count) * travel)
	}
	return m.wordsSettleMarks(out)
}

// wordsSettleMarks stops a mark being carried higher than the word it hangs off.
//
// A full stop halfway up the line reads as a mistake however lively it is. And
// it was not even lively: the last piece of a line answers the top of the range,
// which is where the cymbals are, and a hi-hat is going nearly all the time — so
// the mark at the end of a line sat up there rather than bouncing. Below the
// word is left alone, because a comma dipping is what a comma does.
func (m Model) wordsSettleMarks(out []int) []int {
	marks := m.wordsMarksNow()
	for i := range out {
		if i >= len(marks) || !marks[i] {
			continue
		}
		if by, ok := wordsBesideMark(marks, i); ok && by < len(out) {
			// Up is a smaller number, so the lower of the two is the one to
			// keep: the mark may sit with its word or under it, never over it.
			out[i] = max(out[i], out[by])
		}
	}
	return out
}

// wordsMarksNow says, piece by piece, which of what is on screen is punctuation
// rather than a word.
//
// Worked out from the text again rather than carried on the layout: it is a
// handful of short strings once a frame, against another field on the message
// that would have to be kept in step with the picture it describes.
func (m Model) wordsMarksNow() []bool {
	var out []bool
	for _, line := range strings.Split(m.words.text, "\n") {
		for _, piece := range wordsPieces(line) {
			out = append(out, wordsBeats(line[piece.from:piece.to]))
		}
	}
	return out
}

// wordsBesideMark is the word a mark belongs to: the one in front of it where
// there is one, and the one after it for a mark that opens something.
func wordsBesideMark(marks []bool, at int) (int, bool) {
	for i := at - 1; i >= 0; i-- {
		if !marks[i] {
			return i, true
		}
	}
	for i := at + 1; i < len(marks); i++ {
		if !marks[i] {
			return i, true
		}
	}
	return 0, false
}

// wordsTilting is how far each word of the line leans, and the column it leans
// about.
//
// Most lines do not lean at all, and the ones that do lean by a third of their
// own height — enough that the line looks hand-set rather than typeset, little
// enough that nobody has to tip their head. Which lines and which way is the
// line's own business, so a song reads the same twice.
// A bar of marks is sheared the other way about. A word is mostly horizontal
// strokes, so slanting its baseline is what reads as a lean; a note is a stem
// with a head on it, and a vertical shear slides that stem up and down without
// ever tipping it over. What tips a stem is the shear type has always used for
// its italics — the rows moved sideways, further the higher they are.
//
// Which way each is sheared, and so what the middle it turns about means, is
// the marks' business: see wordsSlanting.
func (m Model) wordsTilting(count int) ([]float32, []int) {
	if count <= 0 || m.words.telling {
		return nil, nil
	}

	// The row keeps time by leaning, so when there is a beat to keep the lean is
	// the beat's rather than dealt. See sway.go.
	sway, swaying := m.wordsSwaying(count)

	// Marks lean oftener as well as differently: a solo is the one place there
	// is nothing else on the screen for it to be doing.
	every := uint64(3)
	if m.words.beats {
		every = 2
	}

	seed := m.wordsLeanSeed()
	if !swaying && seed%every != 0 {
		return nil, nil
	}

	tilt, middle := make([]float32, count), make([]int, count)

	// Where each word sits, so it leans about its own middle rather than about
	// the middle of the screen — which would fling the ends of the line about,
	// and how tall the line it belongs to is set, which is what it leans by.
	where := m.words.where
	first, last, tall, mid := make([]int, count), make([]int, count), make([]int, count), make([]int, count)
	for i := range first {
		first[i], last[i] = -1, -1
	}
	for i, at := range where.At {
		if at < 0 || int(at) >= count {
			continue
		}
		x, line := i%max(where.DotsX, 1), i/max(where.DotsX, 1)
		if first[at] < 0 {
			first[at] = x
		}
		last[at] = x
		if line < len(where.Tops) && line < len(where.Bottoms) {
			tall[at] = where.Bottoms[line] - where.Tops[line] + 1
			mid[at] = (where.Tops[line] + where.Bottoms[line]) / 2
		}
	}

	for i := range tilt {
		wide := max(last[i]-first[i]+1, 1)

		// A word turns about the column down its own middle and leans by a
		// share of its height; a mark turns about the row across its middle and
		// leans like an italic, by a slant that does not depend on how wide the
		// shape happens to be.
		lean := min(wordsLeanRise*float32(tall[i])/float32(wide), wordsTiltMost)
		middle[i] = (max(first[i], 0) + max(last[i], 0)) / 2
		if m.words.beats {
			lean, middle[i] = wordsTiltMark, mid[i]
		}

		// Swaying, the lean is the beat's and the deal is not asked. Which way
		// each of them goes is the figure the bar was dealt — together, at each
		// other, every other one the other way, or rolling along the row.
		if swaying {
			tilt[i] = sway[i]
			continue
		}

		// Three ways for a word: this way, that way, or flat.
		h := seed ^ uint64(i+1)*0x9e3779b97f4a7c15
		h ^= h >> 29
		h *= 0xbf58476d1ce4e5b9
		h ^= h >> 32

		switch h % 3 {
		case 0:
			tilt[i] = lean
		case 1:
			tilt[i] = -lean
		}
	}
	return tilt, middle
}

// wordsLeanSeed is what decides whether the thing on screen leans and which way
// each of its parts goes.
//
// A line of words is asked of its own text. The notes are not: they are always
// the same three marks, so asked the same way they would either lean in every
// solo of every record or in none of them. Theirs comes from when the bar is,
// which is the same thing the moves are drawn from — so one solo leans and the
// next does not.
func (m Model) wordsLeanSeed() uint64 {
	var h uint64 = 0xcbf29ce484222325
	for _, r := range m.words.text {
		h = (h ^ uint64(r)) * 0x100000001b3
	}
	if m.words.beats {
		h ^= uint64(m.words.leanAt) * 0x9e3779b97f4a7c15
	}

	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 29
	return h
}

// wordsLeans is the same question for a line of words alone, which is what a
// test can ask without a model.
func wordsLeans(text string) bool {
	m := Model{}
	m.words.text = text
	return m.wordsLeanSeed()%3 == 0
}

// wordsBeatRide is how hard the part of the sound a note answers for is going,
// 0..1 — which is how far it jumps.
func (m Model) wordsBeatRide(word, count int) float32 {
	bands := m.scope.bands
	if len(bands) == 0 || count <= 0 {
		return 0
	}

	from := word * len(bands) / count
	to := max((word+1)*len(bands)/count, from+1)

	var loud float32
	for _, v := range bands[from:min(to, len(bands))] {
		loud = max(loud, v)
	}
	return loud
}

// wordsPaint is what one piece of what is on screen burns at.
//
// A mark and a word want different things. A mark *is* the spectrum: it stands
// for a part of the range and burning by what that part is doing is the whole of
// what it says. A word is there to be read, and a line whose words are lit
// differently from each other is a line that is harder to read for no gain —
// measured over ninety seconds of a record, the brightest and dimmest word of a
// six word line stood two of the palette's six steps apart at the median and
// three in the top tenth, changing all the while, and no reader can turn that
// into anything. Nobody can see "the third word's band is loud".
//
// So a line is lit evenly and breathes with the whole of the music, and what
// each word's own part of the sound is doing shows in how far it rides. One
// question, one movement.
func (m Model) wordsPaint(word, count, freqs, levels int) wordPaint {
	if m.words.beats {
		return m.wordsBeatPaint(word, count, freqs, levels)
	}

	// The hue runs along the line, and where along the palette it starts is
	// where the harmony has carried it. See hue.go.
	//
	// The whole line together, always: what moves is which part of the palette
	// the line is written in, not which word is brighter than which. That was
	// the thing measured and thrown out — a line is read, and only if it is lit
	// like one — and this leaves the brightness alone.
	// The hue runs along the line, and the beat shoves it round.
	//
	// The meter owns up and down: a band is loud or it is not, and that is a
	// height. A line of type is a horizontal thing and this is its own axis —
	// the colours stepping along it, one way, one shove a beat. Every word takes
	// its colour from where it stands plus how far the wheel has turned, so it
	// is one wheel all the way along; words coloured independently would be
	// confetti, however lively.
	//
	// And it comes home. A whole turn is wordsRollBeats of them, and at the end
	// of it every word is on the colour it started with.
	along := (float32(word)+0.5)/float32(count) + m.words.roll
	for along >= 1 {
		along--
	}
	hue := int8(min(int(along*float32(freqs)), freqs-1))

	var loud float32
	for _, v := range m.scope.bands {
		loud = max(loud, v)
	}

	// Half of it from the moment, half from where the record has got to.
	//
	// The bands are normalised against the record's own scale, so the loudest of
	// them reads the same in a hush as in a chorus — which is why a line used to
	// sit at the same colour through a whole record however it built. swell is
	// the one thing that knows the difference, and half the weight is enough
	// that a verse and its chorus are plainly different without the line going
	// grey when a bar happens to be quiet.
	swell := min(max((m.swell()-swellLeast)/(swellMost-swellLeast), 0), 1)
	loud = 0.25*loud + 0.75*swell

	return wordPaint{
		hue:   hue,
		level: int8(min(int((wordsLineLit+(1-wordsLineLit)*loud)*float32(levels)), levels-1)),
		set:   true,
	}
}

// wordsLineLit is how much of the palette a line has when the record is silent,
// the rest coming from how loud it is. High: what moves here is the line's
// height, and a line of type that fades towards nothing between phrases is a
// line somebody was reading.
const wordsLineLit = 0.55

// wordsBeatPaint is what one of the notes burns at: its own share of the
// spectrum, low to high from left to right.
//
// By its place in the row rather than by where it landed on the screen. The
// notes are set small and sit together in the middle, so their columns all fall
// over the same few bands — read that way, three notes would answer the same
// sound three times.
func (m Model) wordsBeatPaint(word, count, freqs, levels int) wordPaint {
	bands := m.scope.bands
	if len(bands) == 0 || count <= 0 {
		hue, level := m.wordsColourNow()
		return wordPaint{hue: hue, level: level, set: true}
	}

	// The share of the range this one answers for, and the loudest thing in it.
	from := word * len(bands) / count
	to := max((word+1)*len(bands)/count, from+1)

	var loud float32
	for _, v := range bands[from:min(to, len(bands))] {
		loud = max(loud, v)
	}

	share := (float32(word) + 0.5) / float32(count)
	return wordPaint{
		hue:   int8(min(int(share*float32(freqs)), freqs-1)),
		level: int8(min(int((0.3+0.7*loud)*float32(levels)), levels-1)),
		set:   true,
	}
}

// wordsColourNow is the colour of the sound in the air: the hue from where its
// weight sits across the spectrum, the strength from how loud it is.
//
// Two things had to be measured to get this to give more than one colour.
//
// The first: the loudest band is not where the sound is. Measured over seventy
// seconds of a record, the loudest of twenty-eight bands was the second one
// thirty-eight per cent of the time and anywhere in the top half only twenty —
// so a hue taken from it sat at one end of the palette almost always. The centre
// of gravity of the whole spectrum moves where the loudest band does not: a
// word sung over a bass line and one sung over a cymbal are genuinely different
// numbers.
//
// The second: it does not move far. Over the same stretch that centre ran from
// about twelve to seventeen of twenty-eight, so mapped straight onto the palette
// it would still have used a third of it. What is drawn is where it sits in the
// range it has been moving through lately, which spends the whole arc whatever
// the material — the same trick the spectrum's own envelope plays with loudness.
func (m Model) wordsColourNow() (hue, level int8) {
	freqs, levels := len(m.styles.Words), len(m.styles.Words[0])

	bands := m.scope.bands
	if len(bands) == 0 {
		return int8(freqs / 2), int8(levels - 1)
	}

	centre, loud := wordsCentre(bands)

	span := max(m.words.high-m.words.low, wordsRangeLeast)
	at := (centre - m.words.low) / span

	hue = int8(min(max(int(at*float32(freqs)), 0), freqs-1))
	level = int8(min(int((0.45+0.55*loud)*float32(levels)), levels-1))
	return hue, level
}

// wordsCentre is where the weight of the spectrum sits, in bands, and how loud
// it is altogether.
func wordsCentre(bands []float32) (centre, loud float32) {
	var sum, weighted float32
	for i, v := range bands {
		sum += v
		weighted += float32(i) * v
		loud = max(loud, v)
	}
	if sum == 0 {
		return float32(len(bands)) / 2, 0
	}
	return weighted / sum, loud
}

// wordsFollowCentre keeps the range the centre of gravity has been moving
// through: quick to open when the sound goes somewhere new, slow to close, so
// the colours spread over what this record is doing rather than over what a
// spectrum could theoretically do.
func (m *Model) wordsFollowCentre(centre float32) {
	if m.words.high <= m.words.low {
		m.words.low, m.words.high = centre-wordsRangeLeast/2, centre+wordsRangeLeast/2
	}

	if centre < m.words.low {
		m.words.low = centre
	} else {
		m.words.low += (centre - m.words.low) * wordsRangeCloseAt
	}
	if centre > m.words.high {
		m.words.high = centre
	} else {
		m.words.high += (centre - m.words.high) * wordsRangeCloseAt
	}
}

// Sparks off the words that are being thrown hardest.
//
// The water already comes off this screen, but it comes off the sound: a column
// throws when the level in it rises, wherever the letters happen to be. This
// throws from the words themselves, and only from the ones going — so the line
// sheds where it is moving rather than where the spectrum is.
//
// It is not a second question. What lifts a word is its own slice of the
// spectrum, and what throws the spark is the same slice: the spark is where the
// lift already is, made visible. The mistake to avoid here is the one the height
// made with the beat — two answers in one direction — and this is one answer
// drawn twice.
const (
	// wordsSparkLeast is how hard a word has to be going before anything comes
	// off it. High: everything is always moving a little, and a line shedding
	// all the time is a line on fire rather than a line being played.
	wordsSparkLeast = 0.62

	// wordsSparkThrow is what the ride past that is worth in speed.
	wordsSparkThrow = 0.9

	// wordsSparkSpray is the share of the chances that come to anything, so what
	// comes off is ragged the way the rest of the water is.
	wordsSparkSpray = 0.10

	// wordsSparkPitch is how far apart the columns that may throw are, in dots.
	wordsSparkPitch = 6
)

func (m *Model) wordsSparks(w, rows int) {
	if !m.stage.on || m.words.have.DotsX == 0 || m.words.where.Count == 0 {
		return
	}
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 {
		return
	}

	// Kept rather than made: a slice a frame is work for the collector, and its
	// pauses fall outside the update where nothing can account for them.
	where := m.words.where
	if len(m.words.rides) != where.Count {
		m.words.rides = make([]float32, where.Count)
	}
	rides := m.words.rides
	for i := range rides {
		rides[i] = m.wordsBeatRide(i, where.Count)
	}

	span, _ := stageSpan(dotsY)
	for x := 0; x < dotsX; x += wordsSparkPitch {
		if len(m.stage.drops) >= stageDrops {
			return
		}

		// Which word owns this column, asked at the row the words stand on.
		word := where.WordAt(x, where.Tops[0])
		if word < 0 || word >= len(rides) {
			continue
		}
		ride := rides[word]
		if ride < wordsSparkLeast || m.stage.roll() > wordsSparkSprayAt {
			continue
		}

		m.stage.drops = append(m.stage.drops, stageDrop{
			col:    x,
			at:     1,
			speed:  paceSpeed((ride - wordsSparkLeast) / (1 - wordsSparkLeast) * wordsSparkThrow * float32(math.Sqrt(float64(span)))),
			bright: ride,
		})
	}
}

// wordsRollSteps is how many turns of the record a whole turn of the colour
// takes. Six: a record has a handful of joins in it, so the colour comes most
// of the way round in one and all the way in two.
const wordsRollSteps = 6

// wordsRoll is how far round the palette the record's turns have shoved the
// colour.
func (m Model) wordsRollAt() float32 {
	// One shove a join rather than one a beat. The beat is already answered
	// twice over on this screen; the record turning over is a thing that
	// happens rarely enough to be worth a step, and often enough that a whole
	// turn comes round inside a record. See joins.go.
	at := m.joinsTurns() % wordsRollSteps
	return float32(at) / wordsRollSteps
}
