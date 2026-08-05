package ui

import (
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

	// stageGap is how many dot rows are left clear along the middle, so the two
	// halves read as a reflection rather than as one shape.
	stageGap = 1

	// stagePitch is one line every this many dot columns. A spectrum drawn as a
	// solid mass is a silhouette; the gaps are what make it read as a comb of
	// fine lines, which is the whole look of the thing.
	stagePitch = 3

	// stageThrow is how much of a band's jump becomes speed for the drops it
	// throws off, in dot rows a frame. A hard hit is a jump of about half the
	// scale, and that has to arc rather than leave the screen: at this and the
	// gravity below it goes about ten rows up and comes back inside a second.
	stageThrow = 5

	// stageGravity is what pulls them back, in dot rows a frame. Together with
	// the throw it sets the arc: about a second in the air for a hard hit.
	stageGravity = 0.22

	// stageJump is how much a band has to rise in a frame before it throws
	// anything at all. Below this it is the music breathing, not hitting.
	stageJump = 0.04

	// stageSpray is the share of the columns that throw when they jump. All of
	// them would put up a solid curtain; some of them is water.
	stageSpray = 0.35

	// stageDim is what a drop keeps of its light each frame. Water thrown up is
	// brightest as it leaves.
	stageDim = 0.97

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

// stageMode is which picture the big screen is showing. The same key cycles
// them as cycles the strip on the player, because it is the same question asked
// of a bigger canvas.
type stageMode int

const (
	// stageMirror is the spectrum about the middle line, with the water. It is
	// first because it is the one built for this size.
	stageMirror stageMode = iota
	stageWave
	stageBars
	stageModes
)

func (s stageMode) next() stageMode { return (s + 1) % stageModes }

// stageState is what the big screen carries between frames.
type stageState struct {
	// on is whether the screen is up. It is not a tab: it is something you fall
	// into and leave with the next key, the way turning the lights down is.
	on bool

	mode  stageMode
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
	case key.Matches(k, m.keys.Scope):
		// The one key that changes the picture rather than putting it away: the
		// three of them are the same three the strip offers, at this size.
		m.stage.mode = m.stage.mode.next()
		m.stage.drops = nil
		return m.startScope(), true

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
	switch m.stage.mode {
	case stageWave:
		art = m.scopeLinesFrom(w, rows, m.scopeTrigger(w*dotsPerCellX))
	case stageBars:
		art = m.barsLines(w, rows)
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

	// The columns, and their reflections.
	reach := stageReach * float32(middle-stageGap)
	for x := range dotsX {
		if x%stagePitch != 0 {
			continue
		}
		height := int(m.stageLevel(x, dotsX) * reach)
		for y := range height {
			// Brightest at the tip, the way the bars are drawn: a short column
			// still burns at its own top.
			up := float32(y) / float32(max(height-1, 1))
			step := int8(min(int(up*float32(levels)), levels-1))

			light(x, middle-stageGap-y, step)
			light(x, middle+stageGap+y, step)
		}
	}

	// The water, and its reflection.
	for _, d := range m.stage.drops {
		step := int8(min(int(d.bright*float32(levels)), levels-1))
		y := int(d.at)
		light(d.col, middle-stageGap-y, step)
		light(d.col, middle+stageGap+y, step)
	}

	return m.drawCells(w, rows, grid, paint, hue)
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
func (m *Model) stageFlow() {
	dotsX, dotsY := m.width*dotsPerCellX, m.height*dotsPerCellY
	if dotsX <= 0 || dotsY <= 0 {
		return
	}

	kept := m.stage.drops[:0]
	for _, d := range m.stage.drops {
		d.speed -= stageGravity
		d.at += d.speed
		d.bright *= stageDim

		// It is gone when it falls back into the middle, leaves the frame, or
		// has nothing left to see.
		if d.at > 0 && d.at < float32(dotsY/2) && d.bright > 0.05 && d.col < dotsX {
			kept = append(kept, d)
		}
	}
	m.stage.drops = kept

	if len(m.stage.was) != dotsX {
		m.stage.was = make([]float32, dotsX)
	}

	reach := stageReach * float32(dotsY/2-stageGap)
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
			speed:  jump * stageThrow,
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
