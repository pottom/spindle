package ui

import (
	"math"
	"strings"
	"time"

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

	// stageEdgeSnap is how far the head may be from the playhead before it
	// stops keeping up and starts travelling.
	//
	// A third of a second. Ordinary playback moves the head by a frame at a
	// time and the poll that corrects the clock moves it by a few tens of
	// milliseconds; the smallest seek anybody can ask for is five seconds. There
	// is nothing in between to get wrong.
	stageEdgeSnap = 350 * time.Millisecond

	// stageEdgeRun is how much of the distance to the playhead is left after a
	// frame of travelling. Tuned at 30 fps and converted for the rate actually
	// drawn at — see pace.go.
	//
	// Eight tenths, which crosses a five second seek in about a quarter of a
	// second: long enough to be a movement and short enough that holding the key
	// does not leave the head a second behind your finger.
	stageEdgeRun = 0.80

	// stageEdgeRunTail is how many times its own length the tail grows to while
	// the head is travelling, at the moment the run begins.
	//
	// A comet is a comet because of what it drags. Running with the same short
	// tail it keeps at rest, the head reads as a dot that has been moved rather
	// than one that went — and which way it went is exactly what has to be
	// legible.
	stageEdgeRunTail = 4.0

	// stageEdgeBeatTail is how much of the comet's tail is left between two
	// beats. It draws itself out on the beat and pulls back in after it, which
	// is a movement rather than a flash — a flashing dot on a screen is an
	// alarm, and this is a metronome.
	stageEdgeBeatTail = 0.45

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

	// loose says the picture keeps time with the record rather than only
	// answering how loud it is. On unless the key has turned it off, because
	// the two ways cannot be judged against each other unless both can be seen
	// on the same record. See beatKeeping.
	loose bool

	drops []stageDrop

	// was is how high every column stood last frame, which is what a jump is
	// measured against.
	was []float32

	// edgeAt is where the head running round the edge is showing, and edgeFor
	// the record that is. It follows the playhead rather than being it, so that
	// a jump is something you watch happen. See stageEdgeFlow.
	edgeAt  time.Duration
	edgeFor string

	seed uint32
}

// stageEdgeFlow walks the head round the edge toward where the record actually
// is, so a seek is a run rather than a jump.
//
// The big screen has no progress bar and no clock — the head on the edge is the
// only thing on it that says where the record has got to, so it is also the only
// thing that can say you have moved. Made to travel, it says how far as well:
// five seconds is a nudge, holding the key is a sprint round the picture.
//
// It follows the playhead rather than the key, which means it answers a seek
// from the command line, from a phone, or from Spotify itself, and not only from
// this keyboard. Ordinary playback never triggers it because ordinary playback
// never moves faster than stageEdgeSnap in a frame.
func (m *Model) stageEdgeFlow() {
	if m.ps == nil {
		return
	}
	now := m.elapsed()

	// A different record starts where it starts. Nothing is being seeked and a
	// head sprinting round the edge would say something untrue.
	if m.stage.edgeFor != m.ps.TrackID {
		m.stage.edgeFor, m.stage.edgeAt = m.ps.TrackID, now
		return
	}

	off := now - m.stage.edgeAt
	if off < 0 {
		off = -off
	}
	if off < stageEdgeSnap {
		m.stage.edgeAt = now
		return
	}
	m.stage.edgeAt += time.Duration(float64(now-m.stage.edgeAt) * (1 - float64(paceKeep(stageEdgeRun))))
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

	case key.Matches(k, m.keys.Loose):
		// The two ways of drawing, side by side on the one record: keeping time
		// with it, or only answering how loud it is. Nothing else on this
		// screen can be judged against its alternative, and this one has to be.
		m.stage.loose = !m.stage.loose
		return nil, true

	case key.Matches(k, m.keys.Tell):
		// Says what is playing, there and then. Like the picture key, it does
		// something up here rather than putting the screen away.
		m.words.forced = time.Now()
		return nil, true

	case key.Matches(k, m.keys.Marks):
		// Walks the sets of marks a press at a time, and round to the deal
		// again. See marksWalk.
		m.marksWalk()
		return nil, true

	case key.Matches(k, m.keys.Spill):
		// The hatch: the line on screen lets go now, and comes back dealt a
		// fresh way in. Not in the help — see keys.Spill.
		m.wordsLetGo()
		return nil, true

	case key.Matches(k, m.keys.Face):
		// Puts a face up, and walks its expressions a press at a time.
		m.faceShow()
		return nil, true

	case key.Matches(k, m.keys.PlayPause),
		key.Matches(k, m.keys.VolUp), key.Matches(k, m.keys.VolDown),
		key.Matches(k, m.keys.Mute),
		key.Matches(k, m.keys.Next), key.Matches(k, m.keys.Prev),
		key.Matches(k, m.keys.SeekFwd), key.Matches(k, m.keys.SeekBack),
		key.Matches(k, m.keys.Shuffle), key.Matches(k, m.keys.Repeat):
		// The transport, which belongs to whatever is playing rather than to
		// whichever screen is up. Everything not named here closes the picture,
		// which is what makes it a picture rather than a screen with a way out:
		// any key gets you back. Seeking was missing from the list, so the two
		// keys nearest to hand did the one thing nobody wanted.
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
	if rows < 6 || m.width < 20 {
		return ""
	}

	// Not on the state, though. The picture is drawn from the sound, and the
	// sound keeps coming while the daemon is being restarted or the device is
	// being taken back — moments when there is no track to speak of. Waiting
	// for one turned the big screen black just as something interesting was
	// happening, which is the opposite of what it is for. What needs a track is
	// the progress round the edge, and that asks for itself.

	// Nothing but the picture. What was here — the track's name, the clock, the
	// progress bar along the foot — is what the player screen is for; up here it
	// was a caption on something that does not need one, and it cost three rows
	// of the only screen that wants all of them.
	//
	// Where the record has got to is still on screen, drawn into the very edge
	// of it. See stageEdge.
	return strings.Join(m.stagePicture(m.width, rows), "\n")
}

// stagePicture draws whichever of the three the big screen is set to, at the
// size of the whole terminal.
func (m Model) stagePicture(w, rows int) []string {
	var art []string
	switch {
	// Silenced, and the screen has one thing to say. The row of marks is the
	// only picture here with anybody in it to put their fingers in their ears,
	// and a room with nothing coming out of it is worth more than whichever
	// picture was up — the water and the meters answer the sound, and there
	// isn't any. The picture you chose comes back the moment it does.
	case m.muted():
		if art = m.wordsLines(w, rows); art == nil {
			art = m.wordsIdleArt(w, rows)
		}
	case m.scopeMode().wave():
		art = m.scopeLinesFrom(w, rows, m.scopeTrigger(w*dotsPerCellX))
	case m.scopeMode().bars():
		art = m.barsLines(w, rows)
	case m.scopeMode().ladder():
		art = m.ladderLines(w, rows)
	case m.scopeMode().words():
		// Between the lines of a song — an opening bar, a solo, the gap a
		// lyric sheet marks with nothing — there are no words to set. The
		// picture does not go blank and does not hold the last line up either:
		// it gives the whole screen to the music until the singer comes back.
		if art = m.faceLines(w, rows); art != nil {
			break
		}
		if m.wordsSilent() {
			art = m.wordsIdleArt(w, rows)
		} else if art = m.wordsLines(w, rows); art == nil {
			art = m.wordsIdleArt(w, rows)
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

	return m.drawCellsIn(w, rows, grid, paint, hue, m.styles.Bars, m.styles.BarsSeq)
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
		d.speed -= stageGravityAt
		d.at += d.speed
		d.bright *= stageDimAt

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
	// The water is the one thing here that is not put on the beat. Held to it,
	// the drops leave in ranks and the spray reads as a curtain going up; left
	// to the rise in each column, they come off where the music actually moved,
	// which is ragged, and ragged is what water looks like. Judged both ways on
	// the same record with the key, and the loose one wins by a distance.
	for x := 0; x < dotsX; x += stagePitch {
		now := m.stageLevel(x, dotsX)
		jump := now - m.stage.was[x]
		m.stage.was[x] = now

		if jump < stageJump || len(m.stage.drops) >= stageDrops {
			continue
		}
		// Only some of the columns throw, so the spray is ragged the way water
		// is rather than a curtain going up.
		if m.stage.roll() > stageSprayAt {
			continue
		}

		m.stage.drops = append(m.stage.drops, stageDrop{
			col:    x,
			at:     now * reach,
			speed:  paceSpeed(jump * throw),
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

const (
	// stageEdgeHead is how many dots at the leading end are drawn at the same
	// strength, so where it has got to can be found at a glance, and
	// stageEdgeHeat how bright that is as a share of the palette.
	//
	// Well short of the top of it: this is the one thing on the big screen that
	// is not the music, and it has no business being the brightest thing there.
	// It is for glancing at, not for reading.
	stageEdgeHead = 12
	stageEdgeHeat = 0.6

	// stageEdgeTail is how much of the way round the screen the fading part
	// behind the head is worth. A share rather than a count, so the same comet
	// goes round a small terminal and a large one.
	stageEdgeTail = 0.05
)

// stageEdge draws the record's progress into the outermost dots of the screen:
// something small going round the border clockwise from the top left corner,
// arriving back at it as the track ends.
//
// Nothing else is left of the furniture on the big screen, and this costs
// nothing — the border of a terminal is a row and a column that no picture ever
// puts anything in. Where the song is is a fact the eye can have for free.
//
// Only the head and a short tail behind it are drawn. Keeping the whole path
// lit would say the same thing — how far round it has got — and would say it by
// ruling a bright line down the side of the picture, which is a frame around
// the very thing the big screen exists to show.
func (m Model) stageEdge(w, rows int, grid []uint8, paint []int8, levels int) {
	if m.ps == nil || m.ps.Duration <= 0 {
		return
	}

	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
	round := 2 * (dotsX + dotsY)

	// The round closes where the next record starts, not where this one runs
	// out. With a crossfade set those are different moments: the last seconds
	// of the track are already the next one coming up underneath it, and a
	// progress still climbing through them would be counting down to something
	// that has begun.
	ends := m.ps.Duration
	if fade := m.settings.crossfade; fade > 0 && fade < ends {
		ends -= fade
	}

	// Where the head is showing, which is where the record is except while it
	// is travelling to a place somebody has just seeked to. See stageEdgeFlow.
	//
	// A record the head has not been walked for yet is drawn at the playhead
	// rather than at nought: the first frame after a change of record has to be
	// right on its own, without waiting for anything to have run.
	at := m.elapsed()
	if m.stage.edgeFor == m.ps.TrackID {
		at = m.stage.edgeAt
	}
	gone := float64(at) / float64(ends)
	along := int(min(max(gone, 0), 1) * float64(round))

	tail := max(int(stageEdgeTail*float64(round)), stageEdgeHead+1)
	heat := max(int(stageEdgeHeat*float64(levels-1)), 1)

	// How far it still has to run, as a share of the way round: the tail is
	// drawn out by it, and drawn on the side it came from — which is behind the
	// head going forward and in front of it going back, the way a comet's tail
	// is always away from where it is heading.
	behind := 1
	if left := float64(m.elapsed()-at) / float64(ends); left != 0 {
		reach := min(math.Abs(left)/stageEdgeTail, 1)
		tail = max(int(float64(tail)*(1+(stageEdgeRunTail-1)*reach)), stageEdgeHead+1)
		if left < 0 {
			behind = -1
		}
	}

	// The head keeps time when the picture does.
	//
	// Which mode the screen is in is worth knowing and not worth a caption:
	// this is the one thing already on screen whose whole job is to be read
	// without being looked at, so it carries the answer as well. Lit steadily,
	// the picture is answering the loudness; beating, it knows where the beat
	// is — and if it is beating in the wrong place, that is worth seeing too.
	if keeping := m.beatKeeping(); keeping {
		heat = max(int((stageEdgeHeat+(1-stageEdgeHeat)*float64(m.beatPulse()))*float64(levels-1)), 1)
		tail = max(int(float64(tail)*(stageEdgeBeatTail+(1-stageEdgeBeatTail)*float64(m.beatPulse()))), stageEdgeHead+1)
	}

	for back := 1; back <= tail; back++ {
		// How far back down the tail this dot is, and so how much of it is
		// left: the head's own strength, then falling away to nothing behind it.
		i := along - back*behind
		if i < 0 || i >= round {
			continue
		}

		step := int8(heat)
		if back > stageEdgeHead {
			left := 1 - float64(back-stageEdgeHead)/float64(tail-stageEdgeHead)
			step = int8(left * float64(heat))
			if step <= 0 {
				continue // faded out: leaving it dark is the point
			}
		}

		x, y := stageRound(i, dotsX, dotsY)
		cell := (y/dotsPerCellY)*w + x/dotsPerCellX
		grid[cell] |= 1 << brailleBit[x%dotsPerCellX][y%dotsPerCellY]
		if step > paint[cell] {
			paint[cell] = step
		}
	}
}

// stageRound is the nth dot of the way round the edge, clockwise from the top
// left corner.
func stageRound(at, dotsX, dotsY int) (x, y int) {
	switch {
	case at < dotsX:
		return at, 0
	case at < dotsX+dotsY:
		return dotsX - 1, at - dotsX
	case at < 2*dotsX+dotsY:
		return dotsX - 1 - (at - dotsX - dotsY), dotsY - 1
	default:
		return 0, dotsY - 1 - (at - 2*dotsX - dotsY)
	}
}
