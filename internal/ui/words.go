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
	if m.lyrics.synced && m.ps != nil && m.lyrics.forTrack == m.ps.TrackID {
		if at := m.lyricsAt(); at >= 0 && at < len(m.lyrics.lines) {
			if line := strings.TrimSpace(m.lyrics.lines[at].Words); line != "" {
				return wordsWrap(line, m.width*dotsPerCellX, m.height*dotsPerCellY)
			}
		}
	}

	if m.ps == nil || m.ps.Title == "" {
		return nil
	}

	// The record, then: its name over the artist's, which is the order a sleeve
	// puts them in and the order somebody asks about them.
	lines := []string{m.ps.Title}
	if len(m.ps.Artists) > 0 {
		lines = append(lines, strings.Join(m.ps.Artists, ", "))
	}
	return lines
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

	// wordsSungLift is how much of its own strength a word keeps once it has
	// been sung. Below the word being sung, above the ones still to come, so the
	// line reads as three states at a glance.
	wordsSungLift = 0.75
)

// wordsLines draws the words, w cells across and rows deep.
func (m Model) wordsLines(w, rows int) []string {
	g := m.words.have
	if g.DotsX == 0 || g.CellsX != w || g.CellsY != rows {
		return nil
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	freqs, levels := len(m.styles.Bars), len(m.styles.Bars[0])

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
		switch {
		case i < m.words.sung && i < len(m.words.paint) && m.words.paint[i].set:
			p := m.words.paint[i]
			paints[i] = wordPaint{hue: p.hue, level: int8(float32(p.level) * wordsSungLift), set: true}
		case i == m.words.sung:
			paints[i] = wordPaint{hue: nowHue, level: nowLevel, set: true}
		default:
			paints[i] = wordPaint{level: int8(min(int(wordsAhead*float32(levels)), levels-1))}
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

	return m.drawCells(w, rows, grid, paint, hue)
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
	lines := m.wordsNow()
	if len(lines) == 0 {
		return nil
	}

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
	reached := m.lyricsSweep(at, line)

	// The sweep is a place in the line; the picture counts in words.
	var words int
	for i, r := range []rune(line) {
		if i >= reached {
			break
		}
		if unicode.IsSpace(r) {
			words++
		}
	}
	return min(words, m.words.where.Count-1)
}

// wordsColourNow is the colour of the sound in the air: the hue from where the
// loudest of the spectrum sits, the strength from how loud it is.
//
// This is what a word is painted with as it goes by. It is the same mapping the
// spectrum itself uses — low on one side of the palette's arc, high on the other
// — so a word sung over a bass note and the bass note itself are the same
// colour on two different pictures.
func (m Model) wordsColourNow() (hue, level int8) {
	freqs, levels := len(m.styles.Bars), len(m.styles.Bars[0])

	bands := m.scope.bands
	if len(bands) == 0 {
		return int8(freqs / 2), int8(levels - 1)
	}

	loudest, at := bands[0], 0
	for i, v := range bands {
		if v > loudest {
			loudest, at = v, i
		}
	}

	hue = int8(min(at*freqs/len(bands), freqs-1))
	level = int8(min(int((0.45+0.55*loudest)*float32(levels)), levels-1))
	return hue, level
}
