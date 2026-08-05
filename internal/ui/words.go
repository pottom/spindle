package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"time"
	"unicode"

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

	size := wordsSize(lines, w, h)
	if size < 4 {
		return nil, msg.WordLayout{}, false
	}

	face, err := opentype.NewFace(wordsFont, &opentype.FaceOptions{
		Size: float64(size), DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, msg.WordLayout{}, false
	}
	defer face.Close() //nolint:errcheck // nothing to do about a face that will not close

	img := image.NewGray(image.Rect(0, 0, w, h))
	d := &font.Drawer{Dst: img, Src: image.NewUniform(color.Gray{Y: 255}), Face: face}

	metrics := face.Metrics()
	lead := int(float64(size) * wordsLead)
	block := lead * (len(lines) - 1)
	top := (h-block)/2 + metrics.Ascent.Round()/2

	// Where every word lands is worked out here rather than afterwards: this is
	// the only place the widths are known, because a proportional face gives no
	// formula for how wide a word is, only a measurement.
	layout := msg.WordLayout{DotsX: w, At: make([]int16, w*len(lines))}
	for i := range layout.At {
		layout.At[i] = -1
	}

	for i, line := range lines {
		width := d.MeasureString(line).Round()
		left := (w - width) / 2
		baseline := top + i*lead

		d.Dot = fixed.P(left, baseline)
		d.DrawString(line)

		// The line of type covers from the top of its tallest letter to the
		// bottom of its lowest, with a little either side so that a dot on the
		// edge of a stroke still counts as part of the word it belongs to.
		layout.Tops = append(layout.Tops, max(baseline-metrics.Ascent.Round(), 0))
		layout.Bottoms = append(layout.Bottoms, min(baseline+metrics.Descent.Round(), h-1))

		at := left
		for _, word := range strings.Fields(line) {
			wide := d.MeasureString(word).Round()
			for x := at; x < at+wide && x < w; x++ {
				if x >= 0 {
					layout.At[i*w+x] = int16(layout.Count)
				}
			}
			at += wide + d.MeasureString(" ").Round()
			layout.Count++
		}
	}
	return img, layout, true
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
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	for n := 1; n <= wordsMostLines; n++ {
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
			if int(float64(w)*(1-wordsMargin))/max(longest, 1) >= wordsReadable || n == wordsMostLines {
				return lines
			}
		}
	}
	return wordsSplit(line, wordsMostLines)
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
	if m.lyrics.synced && m.ps != nil && m.lyrics.forTrack == m.ps.TrackID {
		at, clock := m.lyricsAt(), m.lyricsClock()

		// The next line, if it is close enough that its gathering has begun.
		if next := at + 1; next < len(m.lyrics.lines) {
			if m.lyrics.lines[next].At-clock <= int64(wordsGather/time.Millisecond) {
				at = next
			}
		}

		if at >= 0 && at < len(m.lyrics.lines) {
			if line := strings.TrimSpace(m.lyrics.lines[at].Words); line != "" {
				// A bar with no words for it. Nothing is set: what goes up there
				// is drawn — a face, or the chase — so there is no type to ask
				// for and the picture is put together as it is shown.
				if wordsBeats(line) {
					return nil, m.lyrics.lines[at].At
				}
				return wordsWrap(line, m.width*dotsPerCellX, m.height*dotsPerCellY),
					m.lyrics.lines[at].At
			}
		}

		// A song with words that is not singing any: nothing to set, and the
		// music has the screen until the singer comes back.
		return nil, 0
	}

	if m.ps == nil || m.ps.Title == "" {
		return nil, 0
	}

	// The record, then — but only at the top of it. A title is worth reading
	// once, as the record starts, the way a sleeve is worth a look as it goes
	// on the turntable; held up for the whole of a song with no words in the
	// database it is a caption on a picture nobody is looking at any more. After
	// its few seconds the screen goes to the music.
	if m.elapsed() > wordsTitle {
		return nil, 0
	}

	lines := []string{m.ps.Title}
	if len(m.ps.Artists) > 0 {
		lines = append(lines, strings.Join(m.ps.Artists, ", "))
	}
	return lines, 0
}

// wordsMove is how a line comes apart and goes back together.
//
// Four of them rather than one, chosen by the line itself, because a change
// that happens the same way every time stops being a change after the third
// chorus. Which one a line gets is worked out from its own text, so a song
// plays the same way twice and a test can watch it do so.
type wordsMove int

const (
	wordsDrifting wordsMove = iota // in from wherever, on its own line
	wordsRising                    // up from below, a row at a time
	wordsBursting                  // in from outside, along the line to the middle
	wordsWiping                    // left to right, like something being written
	wordsMoves
)

// wordsMoveFor picks one from the line.
func wordsMoveFor(text string) wordsMove {
	var h uint32 = 2166136261
	for _, r := range text {
		h = (h ^ uint32(r)) * 16777619
	}
	return wordsMove(h % uint32(wordsMoves))
}

// wordsState is the picture the words are drawn from, and how far it has
// gathered since the line changed.
type wordsState struct {
	// move is how this line is coming together.
	move wordsMove

	have           cover.Grain
	text           string
	cellsX, cellsY int
	asked          string

	// since is when this line arrived. The dots gather over the moment after
	// it, which is what makes a line change something you see rather than
	// something you notice has happened.
	since time.Time

	// where each word of the line landed, so the one being sung can be told from
	// the ones that have been and the ones still to come.
	where msg.WordLayout

	// beats says the line is not words at all — the note a lyric sheet puts in
	// place of a line it has none for. Those follow the music rather than being
	// painted once and kept: there is no singer to follow, so the sound itself
	// is what the colour has to answer to.
	beats bool

	// low and high are the range the sound's centre of gravity has been moving
	// through lately, which is what the colours are spread over. See
	// wordsColourNow.
	low, high float32

	// ends is when the line on screen gives way to the next.
	ends int64

	// starts is when the line on screen is sung, on the playback clock. The
	// gathering is timed against it rather than against the moment the picture
	// happened to be built, so the words are complete as the line begins rather
	// than half a second into it.
	starts int64

	// sung is how many words of the line have been reached, and paint is what
	// each of them was sung in.
	//
	// This is the picture's whole idea. A word is coloured by the sound that was
	// in the air as it went by — the hue from where the loudest of the spectrum
	// sat, the strength from how loud it was — and then it keeps that colour for
	// the rest of the line. By the end of a line the words are a record of how
	// they sounded: a growled low word stays dark and red, a word sung over a
	// cymbal crash stays bright at the other end of the scale. Nothing else here
	// remembers anything; this is the one picture that does.
	sung  int
	paint []wordPaint
}

// wordPaint is a word's colour: which hue of the palette, at what strength, and
// whether it has been sung yet at all.
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

	// wordsStagger is the share of the gathering that is spent waiting, for the
	// moves that arrive a row or a column at a time: at nothing they all land
	// together, at one the last dot only starts as the first one finishes.
	wordsStagger = 0.55

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

	// wordsRate is how fast a line is taken to be sung, in characters a second,
	// and wordsLeast the shortest a line is ever given.
	//
	// A lyric sheet says when a line starts and nothing else, so how far into it
	// the singer has got is a guess whatever we do. Spreading the words evenly
	// over the gap to the next line — which is the obvious guess, and what this
	// did — is a bad one: measured over three hundred lines of six records, the
	// rate that assumption implies runs from two and a half characters a second
	// to twenty-two, a fivefold spread inside a single song, and a line at a
	// plausible singing rate fills only about half of its gap. Half the time the
	// highlight was therefore pointing at a word the singer had left.
	//
	// Guessing the length from the line itself is the better guess: it is wrong
	// in the safe direction. Finishing early leaves the last word lit while it
	// is still being sung; finishing late points at a word already gone.
	wordsRate  = 12
	wordsLeast = 800 * time.Millisecond

	// wordsEdge is how many words the light reaches past the one being sung.
	//
	// A hard spotlight on a single word claims a precision the timing does not
	// have — a lyric sheet says when a line starts and nothing else — and worse,
	// it gives the eye nothing to hold: one word in a line of ten looks the same
	// wherever it is. A front of three or four words fading away ahead of it
	// reads as a place in the line rather than as a claim about a word, and it
	// is what a hand does when it follows a line of print.
	wordsEdge = 3

	// wordsBehind is what a word that has been sung keeps of its strength. Well
	// under the front, so where the light has got to is the brightest thing on
	// the screen and the rest of the line is plainly either side of it.
	wordsBehind = 0.5

	// wordsRangeLeast is the narrowest range of sound the colours are spread
	// over, in bands, and wordsRangeClose how fast the range closes back in when
	// the sound stops going that far. Slow: a chorus that opens the range should
	// still be using it a verse later.
	wordsRangeLeast = 2.5
	wordsRangeClose = 0.0015

	// wordsTitle is how long the record's name is worth at the top of it.
	wordsTitle = 5 * time.Second

	// wordsCeiling is the row the hanging picture starts at: under the track's
	// name and the clock, which are set over the top of everything else.
	wordsCeiling = 2

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

	// What each word is burning at, and in what colour. Three states: what has
	// been sung keeps the colour it was sung in, what is being sung now burns at
	// the top of the scale in the colour of this moment, and what is still to
	// come waits, dim.
	nowHue, nowLevel := m.wordsColourNow()
	paints := make([]wordPaint, m.words.where.Count)
	for i := range paints {
		waiting := int8(min(int(wordsAhead*float32(levels)), levels-1))

		switch ahead := i - m.words.sung; {
		case ahead < 0:
			// Behind the light: what it was sung in, kept, and dimmed so that
			// the front of the line is the brightest thing on the screen.
			p := wordPaint{hue: nowHue, level: waiting}
			if i < len(m.words.paint) && m.words.paint[i].set {
				p = m.words.paint[i]
			}
			paints[i] = wordPaint{hue: p.hue, level: int8(float32(p.level) * wordsBehind), set: true}

		case ahead <= wordsEdge:
			// The front: the word the singer is taken to be on, and the few
			// after it fading away — a place in the line rather than a claim
			// about a word.
			fade := 1 - float32(ahead)/float32(wordsEdge+1)
			level := max(int8(float32(nowLevel)*fade), waiting)
			paints[i] = wordPaint{hue: nowHue, level: level, set: true}

		default:
			paints[i] = wordPaint{level: waiting}
		}
		if gather < 1 {
			paints[i].level = int8(float32(paints[i].level) * gather)
		}
	}

	for y := range dotsY {
		for x := range dotsX {
			if g.Lum[y*dotsX+x] < grainLit {
				continue
			}

			// Where this dot is on its way from. The offset is worked out from
			// where the dot belongs rather than stored, so a screenful of them
			// costs nothing to remember and comes apart the same way twice.
			at, to := x, y

			// How far along this particular dot is. Some of the moves arrive a
			// row or a column at a time, so each dot has its own share of the
			// gathering rather than all of them having the whole of it.
			p := gather
			switch m.words.move {
			case wordsWiping:
				p = wordsStep(gather, float32(x)/float32(dotsX))
			case wordsRising:
				p = wordsStep(gather, 1-float32(y)/float32(dotsY))
			}

			if p < 1 {
				dx, dy := wordsFrom(m.words.move, x, y, dotsX, dotsY)
				at += int(dx * (1 - p))
				to += int(dy * (1 - p))
				if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
					continue
				}
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
	if _, tall := m.wordsRoom(rows); tall >= wordsBand {
		m.wordsUnder(grid, paint, hue, w, rows, tall)
	}

	return m.drawCells(w, rows, grid, paint, hue, m.styles.Words)
}

// wordsUnder draws the meter into the rows the words left over, and its water
// over the whole screen.
func (m Model) wordsUnder(grid []uint8, paint, hue []int8, w, rows, tall int) {
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
	reach := stageReach * float32(tall*dotsPerCellY)
	head := m.wordsHeadroom(rows)

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

// wordsStep is one dot's share of the gathering, given how far down the queue
// it is: the ones at the front are done before the ones at the back start.
func wordsStep(gather, behind float32) float32 {
	p := (gather - behind*wordsStagger) / (1 - wordsStagger)
	return min(max(p, 0), 1)
}

// wordsFrom is where a dot comes in from, given the move and where it lands.
func wordsFrom(move wordsMove, x, y, dotsX, dotsY int) (float32, float32) {
	switch move {
	case wordsRising:
		// Up from under the line, with enough of a wobble that it is a crowd
		// arriving rather than a blind coming up.
		dx, _ := wordsDrift(x, y)
		return dx * 0.15, float32(dotsY-y) * 0.9

	case wordsBursting:
		// In from outside, along the line out from the middle — the picture
		// implodes onto the words.
		dx, dy := float32(x-dotsX/2), float32(y-dotsY/2)
		d := float32(math.Hypot(float64(dx), float64(dy)))
		if d == 0 {
			return 0, 0
		}
		return dx / d * wordsScatter * 2.2, dy / d * wordsScatter * 2.2

	case wordsWiping:
		// From the left, as if the line were being written.
		return -float32(wordsScatter) * 1.4, 0
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
	lines, starts := m.wordsComing()
	if len(lines) == 0 {
		return nil
	}
	m.words.starts, m.words.ends = starts, m.wordsEnds(starts)

	text := strings.Join(lines, "\n")
	if m.words.text == text && m.words.cellsX == m.width && m.words.cellsY == m.height {
		return nil
	}
	if m.words.asked == text {
		return nil
	}

	m.words.asked = text
	return wordsCmd(lines, m.width, m.height)
}

// wordsFlow follows the singer along the line, and paints each word with the
// sound that was in the air as it went by.
func (m *Model) wordsFlow(w, rows int) {

	if m.words.where.Count == 0 {
		return
	}
	if len(m.words.paint) != m.words.where.Count {
		m.words.paint = make([]wordPaint, m.words.where.Count)
	}

	if centre, _ := wordsCentre(m.scope.bands); len(m.scope.bands) > 0 {
		m.wordsFollowCentre(centre)
	}

	m.words.sung = m.wordsSung()

	// The word being sung takes the colour of this moment, over and over, so
	// what it keeps is how it sounded as it ended rather than how the line
	// happened to start.
	if at := m.words.sung; at >= 0 && at < len(m.words.paint) {
		hue, level := m.wordsColourNow()
		m.words.paint[at] = wordPaint{hue: hue, level: level, set: true}
	}
}

// wordsSung is how many words of the line have been reached.
func (m Model) wordsSung() int {
	if !m.lyrics.synced || m.ps == nil || m.lyrics.forTrack != m.ps.TrackID {
		return m.words.where.Count // a title is not sung: it is all there at once
	}

	at := m.lyricsAt()
	if at < 0 || at >= len(m.lyrics.lines) {
		return 0
	}

	line := m.lyrics.lines[at].Words

	// How long the line is taken to last: what it would take to sing it, rather
	// than the whole gap before the next one. See wordsRate.
	slot := lyricsDefaultLine
	if at+1 < len(m.lyrics.lines) {
		slot = time.Duration(m.lyrics.lines[at+1].At-m.lyrics.lines[at].At) * time.Millisecond
	}
	sung := time.Duration(float64(len([]rune(line))) / wordsRate * float64(time.Second))
	sung = min(max(sung, wordsLeast), slot)
	if sung <= 0 {
		return 0
	}

	along := float64(m.lyricsClock()-m.lyrics.lines[at].At) / float64(sung/time.Millisecond)
	along = min(max(along, 0), 1)

	// Which word that lands on: the line's own words, so a line broken over
	// three display lines still counts as one sentence.
	return min(int(along*float64(len(strings.Fields(line)))), m.words.where.Count-1)
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
		m.words.low += (centre - m.words.low) * wordsRangeClose
	}
	if centre > m.words.high {
		m.words.high = centre
	} else {
		m.words.high += (centre - m.words.high) * wordsRangeClose
	}
}
