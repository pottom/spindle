package ui

import (
	"math"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// The stage: the whole terminal given over to the music.
//
// Everything else spindle draws is a working screen, and a visualiser on one of
// those is a strip a few rows deep — enough to glance at, never enough to
// watch. This is the other thing: the cover, the lists and the help are put
// away, and what is left is the spectrum at whatever size the window is, which
// on an ordinary terminal is a couple of hundred columns of dots by a hundred
// and sixty rows of them.
//
// It is drawn about the middle line, upwards and mirrored downwards, because a
// spectrum standing on the floor of the screen leaves the top half empty and a
// mirrored one fills the frame. The drops thrown off the peaks are what makes
// it move like water rather than like a meter: a band that jumps throws what it
// gained into the air, and it falls back.

const (
	// stageReach is how much of the half-height a band at the top of the scale
	// takes, leaving somewhere for the drops to go.
	stageReach = 0.72

	// stageTight is the half-height below which the picture is not mirrored at
	// all, and stands on the floor instead.
	//
	// A reflection costs half the height, which is a fair price on a screen and
	// an impossible one in the strip under the artwork: mirrored there, a
	// column has seven dot rows to rise through and the water has two, and what
	// comes out is two dotted lines rather than a picture. Standing on the
	// floor it has all sixteen. The choice is the canvas's, not the viewer's —
	// it is the same picture either way, drawn at the size there is.
	stageTight = 16

	// stageGap is how many dot rows are left clear along the middle, so the two
	// halves read as a reflection rather than as one shape.
	stageGap = 1

	// stagePitch is one line every this many dot columns. A spectrum drawn as a
	// solid mass is a silhouette; the gaps are what make it read as a comb of
	// fine lines, which is the whole look of the thing.
	stagePitch = 3

	// stageThrow is how hard a band's jump throws the drops off it, per root of
	// the half-height it is drawn in.
	//
	// The root is what makes the same picture work at any size: a drop rises by
	// its speed squared over the gravity, so a speed set by the root of the
	// canvas arcs to the same fraction of it whether that is eighty dot rows on
	// a full screen or eight in the strip under the artwork. A fixed speed
	// would send every drop clean off the small one.
	stageThrow = 0.65

	// stageGravity is what pulls them back, in dot rows a frame. Together with
	// the throw it sets the arc: about a second in the air for a hard hit.
	stageGravity = 0.22

	// stageJump is how much a band has to rise in a frame before it throws
	// anything at all. Below this it is the music breathing, not hitting.
	stageJump = 0.02

	// stageSpray is the share of the columns that throw when they jump. All of
	// them at once would be a curtain going up rather than water coming off,
	// but most of them is what makes the air busy enough to watch.
	stageSpray = 0.7

	// stageDim is what a drop keeps of its light each frame.
	//
	// Water thrown up is brightest as it leaves, and it has to stay lit long
	// enough to be watched all the way back down: what makes the picture read
	// as water rather than as sparks is the falling, not the throwing.
	stageDim = 0.99

	// stageDrops is the most that can be in the air. A wide screen on a busy
	// track keeps a few hundred; the cap is what stops a stuck display from
	// filling memory.
	stageDrops = 1024
)

// stageDrop is a drop thrown off a peak: which column it left, how far it is
// from the middle, how fast it is going and how brightly it was thrown.
type stageDrop struct {
	col       int
	at, speed float32
	bright    float32
}

// stageState is what the big screen carries between frames.
//
// Which picture it shows is not one of them: the big screen and the strip under
// the artwork are the same choice at two sizes, so they share it. Pressing the
// key on either moves both.
type stageState struct {
	// on is whether the screen is up. It is not a tab: it is something you fall
	// into and leave with the next key, the way turning the lights down is.
	on bool

	drops []stageDrop

	// was is how high every column stood last frame, which is what a jump is
	// measured against.
	was []float32

	seed uint32
}

// stageKey answers while the big screen is up, which it mostly does by leaving.
//
// Any key at all: this is a screen you watch rather than work on, so the way
// out is whatever your hand does next. The transport and the volume are the
// exception — stopping the music or turning it down without losing the picture
// is what anybody would expect of those keys, so they are left to the handlers
// they have everywhere else.
func (m *Model) stageKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.stage.on {
		return nil, false
	}

	switch {
	case key.Matches(k, m.keys.Gag):
		// Whatever fills a solo, without waiting for one. Which of them is a
		// coin: the point of the key is to see them, and seeing the same one
		// every time would defeat it.
		m.pullFace()
		return nil, true

	case key.Matches(k, m.keys.Scope):
		// The one key that changes the picture rather than putting it away, and
		// it changes the same setting the strip uses — with the difference that
		// there is no "off" up here: a blank full screen is not a picture,
		// it is a mistake.
		m.scope.modes[m.tab] = m.scopeMode().next()
		if m.scopeMode() == scopeOff {
			m.scope.modes[m.tab] = scopeOff.next()
		}
		m.stage.drops = nil
		return tea.Batch(m.startScope(), m.savePrefs()), true

	case key.Matches(k, m.keys.PlayPause),
		key.Matches(k, m.keys.VolUp), key.Matches(k, m.keys.VolDown),
		key.Matches(k, m.keys.Mute),
		key.Matches(k, m.keys.Next), key.Matches(k, m.keys.Prev):
		return nil, false
	}

	m.stage.on = false
	m.stage.drops = nil
	m.stage.was = nil
	return nil, true
}

// stageView draws the whole screen: the picture, with the track and the clock
// set into the top of it and the progress along the foot.
func (m Model) stageView() string {
	rows := m.height
	if rows < 6 || m.width < 20 || m.ps == nil {
		return ""
	}

	art := m.stagePicture(m.width, rows)

	clock := m.styles.Time.Render(formatDuration(m.elapsed()) + " / " + formatDuration(m.ps.Duration))
	art[0] = spread(m.styles.Title.Render(m.ps.Title), clock, m.width)
	art[1] = fit(m.styles.Artist.Render(strings.Join(m.ps.Artists, ", ")), m.width)
	art[rows-1] = m.progressLine(m.width)

	return strings.Join(art, "\n")
}

// stagePicture draws whichever of the three the big screen is set to, at the
// size of the whole terminal.
func (m Model) stagePicture(w, rows int) []string {
	var art []string
	switch {
	case m.scopeMode().wave():
		art = m.scopeLinesFrom(w, rows, m.scopeTrigger(w*dotsPerCellX))
	case m.scopeMode().bars():
		art = m.barsLines(w, rows)
	case m.scopeMode().ladder():
		art = m.ladderLines(w, rows)
	case m.scopeMode().grain():
		art = m.grainLines(w, rows)
	case m.scopeMode().words():
		// Between the lines of a song — an opening bar, a solo, the gap a
		// lyric sheet marks with nothing — there are no words to set. The
		// picture does not go blank and does not hold the last line up either:
		// it gives the whole screen to the music until the singer comes back.
		if m.wordsSilent() {
			art = m.stageArt(w, rows)
		} else if art = m.wordsLines(w, rows); art == nil {
			art = m.stageArt(w, rows)
		}
	default:
		art = m.stageArt(w, rows)
	}

	// Whatever it drew, the screen is that many rows: a picture waiting for its
	// first frame draws nothing, and nothing still has to fill the terminal.
	for len(art) < rows {
		art = append(art, strings.Repeat(" ", w))
	}
	return art[:rows]
}

// stageArt draws the mirrored spectrum with its drops, w cells by rows rows.
func (m Model) stageArt(w, rows int) []string {
	if len(m.styles.Bars) == 0 || w <= 0 || rows <= 0 {
		return make([]string, max(rows, 0))
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	freqs, levels := len(m.styles.Bars), len(m.styles.Bars[0])
	middle := dotsY / 2

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

	light := func(x, y int, step int8) {
		if x < 0 || y < 0 || x >= dotsX || y >= dotsY {
			return
		}
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if step > paint[cell] {
			paint[cell] = step
		}
	}

	span, mirrored := stageSpan(dotsY)

	// place puts a dot y rows up the picture, and its reflection where there is
	// room for one.
	place := func(x, y int, step int8) {
		if !mirrored {
			light(x, dotsY-1-y, step)
			return
		}
		light(x, middle-stageGap-y, step)
		light(x, middle+stageGap+y, step)
	}

	// The columns.
	reach := stageReach * span
	for x := range dotsX {
		if x%stagePitch != 0 {
			continue
		}
		height := int(m.stageLevel(x, dotsX) * reach)
		for y := range height {
			// Brightest at the tip, the way the bars are drawn: a short column
			// still burns at its own top.
			up := float32(y) / float32(max(height-1, 1))
			place(x, y, int8(min(int(up*float32(levels)), levels-1)))
		}
	}

	// The water.
	for _, d := range m.stage.drops {
		place(d.col, int(d.at), int8(min(int(d.bright*float32(levels)), levels-1)))
	}

	return m.drawCells(w, rows, grid, paint, hue, m.styles.Bars)
}

// stageSpan is how many dot rows a column has to rise through, and whether the
// picture is mirrored about the middle. See stageTight.
func stageSpan(dotsY int) (span float32, mirrored bool) {
	if half := dotsY/2 - stageGap; half >= stageTight {
		return float32(half), true
	}
	return float32(dotsY), false
}

// stageLevel is how loud the column at x is, 0..1.
//
// There are more columns than there are bands, so what falls between two of
// them is interpolated rather than repeated: twenty-eight blocks on a screen
// this size is a bar chart, and what this screen is for is the shape of the
// sound rather than its readings.
func (m Model) stageLevel(x, dotsX int) float32 {
	bands := m.scope.bands
	if len(bands) == 0 {
		return 0
	}
	if len(bands) == 1 {
		return bands[0]
	}

	at := float32(x) / float32(max(dotsX-1, 1)) * float32(len(bands)-1)
	i := min(int(at), len(bands)-2)
	f := at - float32(i)

	// Smoothed rather than straight, so the joins between bands do not read as
	// corners along the top of the picture.
	f = f * f * (3 - 2*f)
	return bands[i]*(1-f) + bands[i+1]*f
}

// stageFlow advances the water by a frame: what is in the air moves on, and a
// band that jumped throws more into it.
//
// It is a step of a simulation rather than a drawing, so it happens in the
// update loop and leaves the drawing a pure function of what it left behind.
func (m *Model) stageFlow(w, rows int) { m.stageFlowIn(w, rows, stageThrows{}) }

// stageThrows is how far the water goes, for the pictures that do not keep it
// inside their own columns.
//
// On the words screen the meter is a band along the foot and the rest of the
// terminal is the lyric, so a drop leaves the tips of a band a few rows tall and
// is given the whole screen to cross. Left to work itself out from the picture
// it is drawn in, it would either be born halfway up the screen or never leave
// the band.
type stageThrows struct {
	// span is how far a drop may travel before it is forgotten, reach how high
	// the columns it leaves from stand, and lift what the throw is multiplied by
	// to carry it that far. All in dot rows; zero means work it out from the
	// picture, which is what every other screen wants.
	span, reach, lift float32
}

func (m *Model) stageFlowIn(w, rows int, t stageThrows) {
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 {
		return
	}

	span := t.span
	if span <= 0 {
		span, _ = stageSpan(dotsY)
	}

	kept := m.stage.drops[:0]
	for _, d := range m.stage.drops {
		d.speed -= stageGravity
		d.at += d.speed
		d.bright *= stageDim

		// It is gone when it falls back into the middle, leaves the frame, or
		// has nothing left to see.
		if d.at > 0 && d.at < span && d.bright > 0.05 && d.col < dotsX {
			kept = append(kept, d)
		}
	}
	m.stage.drops = kept

	if len(m.stage.was) != dotsX {
		m.stage.was = make([]float32, dotsX)
	}

	reach := t.reach
	if reach <= 0 {
		reach = stageReach * span
	}
	throw := stageThrow * float32(math.Sqrt(float64(span)))
	if t.lift > 0 {
		throw *= t.lift
	}
	for x := 0; x < dotsX; x += stagePitch {
		now := m.stageLevel(x, dotsX)
		jump := now - m.stage.was[x]
		m.stage.was[x] = now

		if jump < stageJump || len(m.stage.drops) >= stageDrops {
			continue
		}
		// Only some of the columns throw, so the spray is ragged the way water
		// is rather than a curtain going up on every beat.
		if m.stage.roll() > stageSpray {
			continue
		}

		m.stage.drops = append(m.stage.drops, stageDrop{
			col:    x,
			at:     now * reach,
			speed:  jump * throw,
			bright: min(now+jump, 1),
		})
	}
}

// roll is a random number in 0..1 from a generator the model carries, so the
// same music draws the same water twice: a picture nobody can reproduce is a
// picture nobody can debug.
func (s *stageState) roll() float32 {
	if s.seed == 0 {
		s.seed = 0x9e3779b9
	}
	// Xorshift: a few instructions, and nothing here needs better.
	s.seed ^= s.seed << 13
	s.seed ^= s.seed >> 17
	s.seed ^= s.seed << 5
	return float32(s.seed>>8) / float32(1<<24)
}
