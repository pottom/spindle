package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/pottom/spindle/internal/ui/cover"
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
func wordsImage(lines []string, w, h int) (*image.Gray, bool) {
	if wordsFont == nil || w <= 0 || h <= 0 || len(lines) == 0 {
		return nil, false
	}
	if !wordsDrawable(lines) {
		return nil, false
	}

	size := wordsSize(lines, w, h)
	if size < 4 {
		return nil, false
	}

	face, err := opentype.NewFace(wordsFont, &opentype.FaceOptions{
		Size: float64(size), DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, false
	}
	defer face.Close() //nolint:errcheck // nothing to do about a face that will not close

	img := image.NewGray(image.Rect(0, 0, w, h))
	d := &font.Drawer{Dst: img, Src: image.NewUniform(color.Gray{Y: 255}), Face: face}

	metrics := face.Metrics()
	lead := int(float64(size) * wordsLead)
	block := lead * (len(lines) - 1)
	top := (h-block)/2 + metrics.Ascent.Round()/2

	for i, line := range lines {
		width := d.MeasureString(line).Round()
		d.Dot = fixed.P((w-width)/2, top+i*lead)
		d.DrawString(line)
	}
	return img, true
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

// wordsState is the picture the words are drawn from, and how far it has
// gathered since the line changed.
type wordsState struct {
	have           cover.Grain
	text           string
	cellsX, cellsY int
	asked          string

	// since is when this line arrived. The dots gather over the moment after
	// it, which is what makes a line change something you see rather than
	// something you notice has happened.
	since time.Time
}

const (
	// wordsGather is how long the dots take to come together. A lyric line
	// stands for two or three seconds, so anything slower would leave the
	// picture permanently half made.
	wordsGather = 420 * time.Millisecond

	// wordsScatter is how far a dot starts from where it belongs, in dot rows
	// and columns.
	wordsScatter = 26
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
	step := int8(min(int(gather*float32(levels)), levels-1))

	for y := range dotsY {
		for x := range dotsX {
			if g.Lum[y*dotsX+x] < grainLit {
				continue
			}

			// Where this dot is on its way from. The offset is worked out from
			// where the dot belongs rather than stored, so a screenful of them
			// costs nothing to remember and comes apart the same way twice.
			at, to := x, y
			if gather < 1 {
				dx, dy := wordsDrift(x, y)
				at += int(dx * (1 - gather))
				to += int(dy * (1 - gather))
				if at < 0 || to < 0 || at >= dotsX || to >= dotsY {
					continue
				}
			}

			cell := (to/dotsPerCellY)*w + at/dotsPerCellX
			grid[cell] |= 1 << brailleBit[at%dotsPerCellX][to%dotsPerCellY]
			if step > paint[cell] {
				paint[cell] = step
			}
		}
	}

	return m.drawCells(w, rows, grid, paint, hue)
}

// wordsDrift is where a dot comes in from, from where it lands: a fixed swirl
// rather than a random walk, so the same line always gathers the same way and a
// test can watch it do so.
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
