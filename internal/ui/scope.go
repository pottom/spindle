package ui

import (
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
)

const (
	// scopeRows is how tall the trace is. Four rows of braille is sixteen dots,
	// which is enough for the shape of a waveform to read without the trace
	// becoming the loudest thing on the screen.
	scopeRows = 4

	// scopeChrome is the blank row kept above the trace, so it never butts
	// against the artwork.
	scopeChrome = 1

	// brailleBase is U+2800, where the eight-dot cells begin.
	brailleBase = 0x2800

	// A braille cell is two dots wide and four tall.
	dotsPerCellX = 2
	dotsPerCellY = 4
)

// brailleBit maps a dot's position inside a cell to its bit. The order is not
// sequential: the eight-dot cells were added to Unicode after the six-dot ones,
// so dots seven and eight sit above the rest.
var brailleBit = [dotsPerCellX][dotsPerCellY]uint8{
	{0, 1, 2, 6},
	{3, 4, 5, 7},
}

// scopeMode is what the visualiser is showing. The key cycles through them.
type scopeMode int

const (
	scopeOff scopeMode = iota
	scopeWave
	// scopeBars is the spectrum with the peak markers. A plain version without
	// them was built alongside it and dropped: the markers are what give the
	// meter its character, and without them it is only a row of blocks.
	scopeBars
	// scopeMirror is the big screen's picture, in the strip: the spectrum about
	// a middle line with the water thrown off it. See stage.go.
	scopeMirror
	// scopeLadder is the same spectrum as a stack of lamps. See ladder.go.
	scopeLadder
	// scopeWords is the line being sung, in dots. The whole screen only.
	// See words.go.
	scopeWords
	scopeModes // how many there are, for the cycle
)

// next is the mode the key moves to.
func (s scopeMode) next() scopeMode { return (s + 1) % scopeModes }

// wants reports whether a mode needs the waveform rather than the bands.
func (s scopeMode) wave() bool   { return s == scopeWave }
func (s scopeMode) bars() bool   { return s == scopeBars }
func (s scopeMode) mirror() bool { return s == scopeMirror }
func (s scopeMode) ladder() bool { return s == scopeLadder }
func (s scopeMode) words() bool  { return s == scopeWords }

// big reports whether a mode is only worth drawing on the whole screen. In the
// strip the key steps over it, the way the big screen steps over "off".
func (s scopeMode) big() bool { return s.words() }

// spectrum reports whether a mode is drawn from the bands, whichever way it
// draws them.
// The words take the bands too, though only to breathe with them.
func (s scopeMode) spectrum() bool {
	return s.bars() || s.mirror() || s.ladder() || s.words()
}

// scopeState is the visualiser the player screen is drawing, and what it is
// drawing.
type scopeState struct {
	// modes is what the key cycles, one per tab. Each screen keeps its own: the
	// player is watched and the queue is read, and someone who wants the trace
	// on one of them does not necessarily want it on the other. Each costs a
	// redraw every 33ms, which is not something to spend without being asked.
	modes [tabCount]scopeMode

	// running guards the tick loop, so two callers cannot start it twice and
	// make the trace move at double speed.
	running bool

	// frame is the latest waveform, one value per horizontal dot, in -1..1. It
	// is resampled to whatever width the screen turns out to be.
	frame []float32

	// bands is the latest spectrum, lowest first, each 0..1, and peaks the
	// highest each has reached lately — the marker that falls back slowly is
	// what gives a meter its hi-fi character.
	bands []float32
	peaks []float32

	// beat is where the beats of what is playing are, as the daemon last
	// reported them, and beatAt when that report arrived — the two together are
	// what any of this can be drawn against, because the report ages between
	// frames and a beat that is not extrapolated is a beat a frame late.
	beat   player.Beat
	beatAt time.Time

	// trail is what the last few frames drew, newest last. A cathode ray tube
	// leaves a glow behind the beam; without it a terminal trace looks redrawn
	// thirty times a second, because it is. The trigger is what makes this work
	// rather than smear: the older frames sit almost on top of the newest.
	trail [][]uint8

	// sparks is what the trace has thrown off and not yet taken back, and crest
	// is how far each column swung last frame — which is what a throw is
	// measured against. See spark.go.
	sparks []spark

	// wasLoud is what the envelope stood at last frame, which is how a hit is
	// told from the wave simply moving. See spark.go.
	wasLoud float32
	seed    uint32

	// envelope follows the recent loudness so the trace can be scaled to it.
	// Measured against a live stream, peaks ran from 0.06 to 0.87 within one
	// track: at a fixed scale the quiet passages are a flat line and the loud
	// ones clip. It rises at once and falls slowly, as a meter does.
	envelope float32
}

const (
	// scopeDeflection is how much of the half-height a passage at the current
	// envelope fills, leaving room for something louder to still read as louder.
	scopeDeflection = 0.86

	// scopeRelease is the fraction of the envelope kept each frame while the
	// music is quieter than it was — about a second and a half to fall by half.
	scopeRelease = 0.985

	// scopeFloor stops the gain running away in silence, where the only thing
	// left to amplify is the noise.
	scopeFloor = 0.05

	// scopeTrail is how many frames the glow lasts. One frame behind the beam
	// reads as afterglow; more than that smears, because a thirtieth of a
	// second is long enough for the wave to have moved right across the screen.
	scopeTrail = 2

	// scopeTriggerFollow is how fast the trigger's low pass follows the wave.
	// At the rate the tapped frames arrive it is a corner around three hundred
	// hertz: below the note being played, above the beat.
	scopeTriggerFollow = 0.2

	// scopeSettle is what a paused picture keeps of itself each frame. Fast
	// enough to be over in about a second, slow enough that the eye reads it as
	// the music settling rather than as the screen being cleared.
	scopeSettle = 0.88

	// scopeDwell is how much of a cell's brightness comes from how long the beam
	// spent in it rather than from how loud that moment was.
	//
	// This is what a cathode ray tube does and the reason its traces look the
	// way they do: the phosphor is lit by a beam moving at a constant speed, so
	// where the trace is flat the beam dwells and the line is bright, and where
	// it climbs steeply the same beam is spread over ten times the distance and
	// the line fades. Loudness keeps a share of the say so that a hit still
	// flares rather than going dark for being fast.
	scopeDwell = 0.6
)

// settle sinks everything the visualisers draw towards nothing, a frame at a
// time, for when the music has stopped.
//
// The envelope is deliberately left where it is: the trace is drawn against it,
// so bringing both down together would leave the picture exactly as it was.
func (s *scopeState) settle() {
	for i := range s.sparks {
		s.sparks[i].bright *= scopeSettleAt
	}
	for i := range s.bands {
		s.bands[i] *= scopeSettleAt
	}
	for i := range s.peaks {
		s.peaks[i] *= scopeSettleAt
	}
	for i := range s.frame {
		s.frame[i] *= scopeSettleAt
	}
}

// follow updates the loudness envelope from a new frame.
func (s *scopeState) follow(frame []float32) {
	var peak float32
	for _, v := range frame {
		if v < 0 {
			v = -v
		}
		peak = max(peak, v)
	}

	s.envelope = max(peak, s.envelope*scopeReleaseAt)
	s.envelope = max(s.envelope, scopeFloor)
}

// scopeAvailable reports whether the trace can be drawn without moving anything.
//
// It goes into the blank rows the player screen already has below the artwork,
// never into rows taken from it: shrinking the body would move the cover and
// the text every time the key was pressed, and a visualiser is not worth making
// the rest of the screen jump.
func (m Model) scopeAvailable() bool {
	if m.noDevice || m.ps == nil {
		return false
	}
	l := m.layout()
	if m.tab == tabPlayer {
		return m.scopeRoom(l) >= scopeRows+scopeChrome
	}
	// On a list screen the room is beside the artwork rather than beneath it:
	// the detail panel has never filled that column, and the list below is not
	// moved by anything put there.
	return m.listTab() && queueScopeWidth(l) > 0 && l.artHeight >= scopeRows
}

// listTab reports whether the current screen is one that draws the trace.
//
// The lists share a composition, but not this: the library is read while
// deciding what to hear next, and something moving beside the words is exactly
// the wrong thing there. The queue is the list you watch, and keeps it.
func (m Model) listTab() bool {
	return m.tab == tabQueue
}

// scopeMode is the visualiser the current tab is set to.
func (m Model) scopeMode() scopeMode { return m.scope.modes[m.tab] }

// scopeWidth is how many cells the trace has on the current tab.
func (m Model) scopeWidth(l layout) int {
	if m.tab == tabQueue {
		return queueScopeWidth(l)
	}
	return l.interior - leftMargin - rightMargin
}

// scopeRender draws whichever visualiser is on, across w cells.
func (m Model) scopeRender(w int) []string {
	switch {
	case m.scopeMode().wave():
		return m.scopeLines(w)
	case m.scopeMode().mirror():
		return m.stageArt(w, scopeRows)
	case m.scopeMode().ladder():
		return m.ladderLines(w, scopeRows)
	default:
		return m.barsLines(w, scopeRows)
	}
}

// scopeRoom is how many blank rows sit below the artwork, which is what the
// trace has to fit into. scopeTop is the first of them.
func (m Model) scopeRoom(l layout) int {
	// Without a picture the body has no blank rows: the text and the list have
	// already taken them, and they are the reason the picture was dropped.
	if !l.hasArt() {
		return 0
	}
	return max(l.bodyHeight-m.scopeTop(l), 0)
}

func (m Model) scopeTop(l layout) int {
	block := l.artHeight
	return max((l.bodyHeight-block)/2, 0) + block
}

// scopeVisible reports whether anything drawn from the measurements is on
// screen right now — the strip under the artwork, or the whole terminal.
func (m Model) scopeVisible() bool {
	return m.stage.on || (m.scopeMode() != scopeOff && m.scopeAvailable())
}

// frameMode is what the next frame should be fetched for. The big screen is
// drawn from the bands whatever the strip underneath it was set to.
func (m Model) frameMode() scopeMode {
	// The big screen always draws something, so it always needs something: with
	// the strip switched off it shows the mirrored picture, which is drawn from
	// the bands.
	if m.stage.on && m.scopeMode() == scopeOff {
		return scopeBars
	}
	return m.scopeMode()
}

// scopeLines renders the trace across w cells.
//
// The waveform is drawn as a line rather than a scatter of points: consecutive
// samples are joined vertically, so a steep slope stays continuous instead of
// breaking into dots. Without that a loud passage looks like static.
//
// Each cell is coloured by how loud that moment is, not by where it sits, so
// the trace flares on a hit and recedes between them.
func (m Model) scopeLines(w int) []string {
	return m.scopeLinesFrom(w, scopeRows, m.scopeTrigger(w*dotsPerCellX))
}

// scopeLinesFrom draws the frame beginning at a given sample. Where that sample
// is decided is scopeTrigger's business; this only draws.
func (m Model) scopeLinesFrom(w, rows, start int) []string {
	if w <= 0 || rows <= 0 || len(m.styles.Bars) == 0 {
		return nil
	}
	grid, loud, dwell := m.scopeBeam(w, rows, start)
	return m.scopeDraw(w, rows, grid, loud, dwell)
}

// scopeGrid plots the frame: which braille dots the trace lights, and how far
// the wave swings under each cell.
func (m Model) scopeGrid(w, rows, start int) ([]uint8, []float32) {
	grid, loud, _ := m.scopeBeam(w, rows, start)
	return grid, loud
}

// scopeBeam plots the frame and reports how the beam travelled: which dots it
// lit, how far the wave swings under each cell, and how long it dwelt there.
//
// Dwell is the number of columns a cell was crossed by against the number of
// dots that took: a flat stretch is one dot per column and lights brightly, a
// steep edge is a dozen and fades. It is what makes a tube's trace read as a
// beam rather than as a plot.
func (m Model) scopeBeam(w, rows, start int) ([]uint8, []float32, []float32) {
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	grid := make([]uint8, w*rows)
	loud := make([]float32, w)
	dwell := make([]float32, w)

	steps := make([]float32, w) // columns crossing each cell
	spent := make([]float32, w) // dots they took

	prev := -1
	for x := range dotsX {
		y := m.scopeSample(start, x, dotsX)
		at := x / dotsPerCellX
		if a := abs32(y); a > loud[at] {
			loud[at] = a
		}

		// Map -1..1 onto the rows, leaving a dot of headroom at each edge so a
		// clipping passage does not look like a flat line against the border.
		dy := int(math.Round((0.5 - float64(y)*0.5*scopeDeflection) * float64(dotsY-1)))
		dy = min(max(dy, 0), dotsY-1)

		from := dy
		if prev >= 0 {
			from = prev
		}
		for yy := min(from, dy); yy <= max(from, dy); yy++ {
			cell := (yy/dotsPerCellY)*w + at
			grid[cell] |= 1 << brailleBit[x%dotsPerCellX][yy%dotsPerCellY]
			spent[at]++
		}
		steps[at]++
		prev = dy
	}

	for i := range dwell {
		if spent[i] > 0 {
			dwell[i] = steps[i] / spent[i]
		}
	}
	return grid, loud, dwell
}

// scopeDraw turns a dot grid into rows, with the older frames glowing behind it.
//
// A cell the beam is on now is coloured by how loud that moment is; a cell only
// the afterglow reaches is drawn at the quiet end of the palette, so the glow
// reads as behind the trace rather than as part of it.
func (m Model) scopeDraw(w, rows int, grid []uint8, loud, dwell []float32) []string {
	freqs := len(m.styles.Bars)
	if freqs == 0 {
		return make([]string, rows)
	}
	levels := len(m.styles.Bars[0])

	// What the trace has thrown off is drawn with it rather than after it: a
	// bead sharing a cell with the beam has to add its light to it, not sit in
	// a second layer over the top.
	sparks, sparkPaint := m.sparkGrid(w, rows, levels)

	lines := make([]string, rows)
	for r := range rows {
		var sb strings.Builder

		var run strings.Builder
		var style lipgloss.Style
		lit := false
		flush := func() {
			if run.Len() > 0 {
				sb.WriteString(style.Render(run.String()))
				run.Reset()
			}
		}

		for c := range w {
			at := r*w + c
			bits := grid[at]

			// How long the beam dwelt here and how far the wave swings decide
			// the strength together; the glow behind it is drawn at the bottom
			// of the scale, so it reads as behind rather than as part of the
			// trace.
			bright := scopeDwell*float64(dwell[c]) +
				(1-scopeDwell)*math.Sqrt(float64(min(loud[c], 1)))
			level := min(int(bright*float64(levels)), levels-1)

			if s := sparks[at]; s != 0 {
				bits |= s
				level = max(level, int(sparkPaint[at]))
			}
			for age, old := range m.scope.trail {
				if len(old) != len(grid) || old[at] == 0 {
					continue
				}
				if glow := len(m.scope.trail) - age - 1; bits == 0 {
					bits = old[at]
					level = min(glow, levels-1)
				} else {
					bits |= old[at]
				}
			}

			if bits == 0 {
				flush()
				lit = false
				sb.WriteByte(' ')
				continue
			}

			want := m.styles.Bars[min(c*freqs/w, freqs-1)][level]
			if !lit || want.GetForeground() != style.GetForeground() {
				flush()
				style, lit = want, true
			}
			run.WriteRune(rune(brailleBase + int(bits)))
		}
		flush()
		lines[r] = fit(sb.String(), w)
	}
	return lines
}

// remember keeps a frame's dots so the next few can glow behind the beam.
func (s *scopeState) remember(grid []uint8) {
	s.trail = append(s.trail, grid)
	if len(s.trail) > scopeTrailAt {
		s.trail = s.trail[len(s.trail)-scopeTrailAt:]
	}
}

// scopeTrigger finds where in the frame to start drawing, so that a held note
// stands still on screen instead of shimmering.
//
// It looks for a rising zero crossing, the way an oscilloscope does, and
// requires the wave to have dipped below a small negative threshold first so
// that noise around zero cannot trigger it. Failing to find one is not a
// failure: the trace free-runs from the start of the frame, which is what a
// scope does with no trigger.
func (m Model) scopeTrigger(dots int) int {
	frame := m.scope.frame
	slack := len(frame) - dots
	if slack <= 0 || m.scope.envelope <= 0 {
		return 0
	}

	// A tenth of the recent loudness: high enough to ignore the noise floor,
	// low enough that a quiet passage still triggers.
	threshold := m.scope.envelope * 0.1

	// The trigger listens through a low pass, which is what the coupling switch
	// on a scope's trigger does and for the same reason: a crossing found on the
	// hiss riding on a note is a different crossing every frame, and the picture
	// shimmers. Following the shape of the wave instead holds it still —
	// measured on recorded music, the share of the trace that stays put from one
	// frame to the next goes from 0.68 to 0.70, and what moves is what the music
	// did rather than what the noise did.
	var lowpass float32

	armed := false
	for i := range slack {
		lowpass += (frame[i] - lowpass) * scopeTriggerFollow
		v := lowpass
		if v < -threshold {
			armed = true
			continue
		}
		if armed && v >= 0 {
			return i
		}
	}
	return 0
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// scopeSample reads one sample of the frame, scaled to the recent loudness.
// With no frame at all the trace rests on the centre line, which is what
// silence looks like.
func (m Model) scopeSample(start, x, dots int) float32 {
	if len(m.scope.frame) == 0 || dots <= 0 || m.scope.envelope <= 0 {
		return 0
	}

	// A whole window is spread across the width, however wide that is, so the
	// screen always shows the same slice of time — about a thirtieth of a
	// second. Reading one sample per dot instead would zoom in on a wide
	// terminal and out on a narrow one, and the trace would appear to slow down
	// or speed up with nothing but the window size.
	window := min(player.WaveformWindow, len(m.scope.frame)-start)
	i := start + x*window/dots

	v := m.scope.frame[min(i, len(m.scope.frame)-1)]

	// Scaled to the recent loudness, then clamped: a sudden hit louder than
	// anything before it is drawn at the edge rather than off it.
	return min(max(v/m.scope.envelope, -1), 1)
}

// drawScope writes the trace into the blank rows directly under the artwork,
// leaving everything above exactly where it was.
//
// It hugs the cover rather than sitting at the foot of the screen: the trace
// belongs to the record being played, and a band of empty rows between the two
// would read as an unrelated meter.
func (m Model) drawScope(body []string, l layout) []string {
	at := m.scopeTop(l) + scopeChrome
	if at+scopeRows > len(body) {
		return body
	}

	lines := m.outline(m.scopeRender(m.scopeWidth(l)), m.scopeWidth(l), "trace")
	for i, line := range lines {
		body[at+i] = m.pad(line, l)
	}
	return body
}

// rememberScope keeps this frame's dots so the next few draw a glow behind the
// beam. It is done here rather than while rendering because View has to stay a
// pure function of the model, and a trail is state.
func (m *Model) rememberScope() {
	w := m.scopeWidth(m.layout())
	if m.stage.on {
		w = m.width
	}
	if w <= 0 {
		return
	}
	rows := scopeRows
	if m.stage.on {
		w, rows = m.width, m.height
	}
	grid, _ := m.scopeGrid(w, rows, m.scopeTrigger(w*dotsPerCellX))
	m.scope.remember(grid)
}

// beatLost is how long a beat report is trusted after it arrives.
//
// The daemon sends one with every frame, so a gap this long means the frames
// have stopped or the record has none — either way, whatever was keeping time
// stops keeping it rather than keeping it to a beat that is no longer there.
const beatLost = 2 * time.Second

// beatPhase is where the music is between one beat and the next: nought on the
// beat, rising to one just before the following one.
//
// Extrapolated from the last report rather than taken from it, because the
// report ages between frames and a beat drawn a frame late is a beat drawn
// late. False when there is no beat to keep — the first seconds of every
// record, a recording that has none, and any time the frames stop.
func (m Model) beatPhase() (float32, bool) {
	if !m.scope.beat.Found() || m.scope.beatAt.IsZero() {
		return 0, false
	}

	since := time.Since(m.scope.beatAt)
	if since > beatLost {
		return 0, false
	}

	next := m.scope.beat.Next(since)
	return float32(1 - float64(next)/float64(m.scope.beat.Period)), true
}

// beatKeeping reports that the screen is keeping time with the record rather
// than merely answering it.
//
// Both halves have to hold: there has to be a beat to keep, and the key has to
// have left it on. Everything that moves asks this before it moves, so the two
// ways of drawing can be put side by side on the one record and told apart —
// which is the only way to know whether keeping time is better, rather than
// only different.
func (m Model) beatKeeping() bool {
	if !m.stage.loose {
		return false
	}
	_, ok := m.beatPhase()
	return ok
}

// beatPulse is the shape a beat moves things in: full on the beat and falling
// away before the next one, so what rides it lands rather than sways.
//
// Sharper than a sine on purpose. A beat is an event, not an oscillation: the
// ear hears the strike and the decay, and a picture that spends as long rising
// to the beat as falling from it reads as a swell in the music instead of a
// hit on it.
func (m Model) beatPulse() float32 {
	phase, ok := m.beatPhase()
	if !ok {
		return 0
	}

	// A short rise into the beat, so nothing jumps from nothing, and a long
	// fall out of it.
	if phase > 1-beatRise {
		return (phase - (1 - beatRise)) / beatRise
	}
	fall := 1 - phase/(1-beatRise)
	return fall * fall
}

// beatsIn is how many beats have gone by since something that began a given
// time ago, and whether the record has a beat to count at all.
//
// Counted on the beat grid rather than by dividing the time: what wants the
// number is a picture that moves on the beat, and beats that fall a fifth of a
// period from where the arithmetic puts them are beats in the wrong place. The
// last beat reported anchors the grid, and the count is how many of its
// multiples fall between then and now.
func (m Model) beatsIn(age time.Duration) (int, bool) {
	if !m.beatKeeping() {
		return 0, false
	}
	period := m.scope.beat.Period
	if period <= 0 || age < 0 {
		return 0, false
	}

	// How long ago the last beat the daemon reported was, brought up to date.
	last := time.Since(m.scope.beatAt) + m.scope.beat.Since

	// The grid, counted forwards from that beat: how many have landed by now,
	// less how many had landed when the stretch began. Floored rather than
	// truncated, because the stretch usually began before that beat and a
	// truncated negative would count one beat too few.
	return int(floorDiv(int64(last), int64(period)) - floorDiv(int64(last-age), int64(period))), true
}

// floorDiv divides towards minus infinity, which is what counting a grid
// backwards through zero needs.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

const (
	// beatRise is how much of a beat is spent coming up to it. Small: the rise
	// is there so the strike is not a discontinuity, not so it can be watched.
	beatRise = 0.12
)
