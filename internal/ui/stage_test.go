package ui

import (
	"math"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// stageModel is a player screen with the big picture up and something to draw.
func stageModel(w, h int) Model {
	m := scopeModel(w, h)
	m.stage.on, m.stage.mode = true, stageOpens

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.6
	}
	m.scope.bands = bands
	return m
}

// The key gives the whole terminal to the visualiser, and the next key takes it
// back: it is a screen you watch rather than work on.
func TestStageTakesTheScreenAndGivesItBack(t *testing.T) {
	m := scopeModel(100, 40)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	got := tm.(Model)
	if !got.stage.on {
		t.Fatal("f did not open the big screen")
	}
	if cmd == nil && !got.scope.running {
		t.Error("nothing was left to fetch the frames it draws")
	}
	if !got.scopeVisible() {
		t.Error("the frames stop while the big screen is up")
	}

	// A key this screen has no use for does nothing at all. It used to be the
	// way out — every key was — and a room only has to lean on the keyboard
	// once to lose the picture it was watching.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	if held := tm.(Model); !held.stage.on {
		t.Error("a key the big screen has no use for put it away")
	}

	// esc comes back, and takes nothing with it.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	back := tm.(Model)
	if back.stage.on {
		t.Error("esc left the big screen up")
	}
	if back.tab != tabPlayer {
		t.Errorf("the key that closed the screen also changed tab, to %d", back.tab)
	}
}

// And q and f are the other two ways out: q means "back" everywhere else, and f
// is the key that put the screen up in the first place. A hand that wants the
// working screen back has three answers and does not have to remember which —
// reaching for the key that opened it is what happens before anybody remembers
// there is an esc.
func TestThreeKeysLeaveTheBigScreen(t *testing.T) {
	for _, out := range []tea.KeyPressMsg{
		{Code: 'q', Text: "q"},
		{Code: 'f', Text: "f"},
		{Code: tea.KeyEscape},
	} {
		var tm tea.Model = scopeModel(100, 40)
		tm, _ = tm.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
		if !tm.(Model).stage.on {
			t.Fatal("f did not open the big screen")
		}

		tm, _ = tm.Update(out)
		if got := tm.(Model); got.stage.on {
			t.Errorf("%s left the big screen up", out)
		}
	}
}

// The music keeps its keys: stopping it or turning it down is not a reason to
// lose the picture.
func TestStageKeepsTheTransport(t *testing.T) {
	m := stageModel(100, 40)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !tm.(Model).stage.on {
		t.Error("the space bar closed the big screen instead of pausing")
	}
	if tm.(Model).ps.Playing {
		t.Error("the space bar did not reach the transport")
	}
}

// It fills the terminal exactly: every row, every column, no more and no less.
func TestStageFillsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 40}, {200, 60}} {
		m := stageModel(size[0], size[1])

		art := m.stageArt(size[0], size[1])
		if len(art) != size[1] {
			t.Fatalf("%dx%d: drew %d rows", size[0], size[1], len(art))
		}
		for i, line := range art {
			if got := len([]rune(ansiOff(line))); got != size[0] {
				t.Errorf("%dx%d: row %d is %d cells wide", size[0], size[1], i, got)
			}
		}
	}
}

// The picture is a reflection: what stands above the middle stands below it,
// which is what fills the frame rather than leaving the top half empty.
func TestStageMirrorsAboutTheMiddle(t *testing.T) {
	m := stageModel(100, 40)
	art := m.stageArt(100, 40)

	var lit []int
	for r, line := range art {
		if strings.TrimSpace(ansiOff(line)) != "" {
			lit = append(lit, r)
		}
	}
	if len(lit) < 2 {
		t.Fatalf("the picture lit %d rows, want it filling the frame", len(lit))
	}

	// The first and last lit rows have to be the same distance from the middle.
	first, last := lit[0], lit[len(lit)-1]
	above, below := 40/2-first, last-(40/2-1)
	if above-below > 1 || below-above > 1 {
		t.Errorf("the picture reaches %d rows up and %d down, want a reflection", above, below)
	}
}

// A band that jumps throws water into the air, and what goes up comes down: the
// air empties again once the music stops jumping.
func TestStageThrowsWaterAndTakesItBack(t *testing.T) {
	m := stageModel(100, 40)

	quiet := make([]float32, 28)
	for i := range quiet {
		quiet[i] = 0.2
	}
	m.scope.bands = quiet
	m.stageFlow(100, 40)

	// A hit across the bottom of the range.
	hit := make([]float32, 28)
	copy(hit, quiet)
	for i := range 6 {
		hit[i] = 1
	}
	m.scope.bands = hit
	m.stageFlow(100, 40)

	if len(m.stage.drops) == 0 {
		t.Fatal("a hit threw nothing into the air")
	}
	t.Logf("the hit threw %d drops", len(m.stage.drops))

	// Left alone, every one of them comes back.
	m.scope.bands = quiet
	for range 400 {
		m.stageFlow(100, 40)
	}
	if len(m.stage.drops) != 0 {
		t.Errorf("%d drops are still in the air after the music stopped jumping", len(m.stage.drops))
	}
}

// The water is the one thing on this screen that keeping time does not touch.
//
// Held to the beat the drops left in ranks, which reads as a curtain rather
// than as spray; loose, they come off wherever the music moved. Both were seen
// on the same record and the loose one is the picture. So the same music has to
// throw the same water whether the screen is keeping time or not.
func TestTheWaterIgnoresTheBeat(t *testing.T) {
	water := func(keeping bool) []stageDrop {
		m := stageModel(100, 40)
		if keeping {
			m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
			m.scope.beatAt = time.Now()
			if !m.beatKeeping() {
				t.Fatal("a beat was found and the screen did not keep time")
			}
		}

		bands := make([]float32, 28)
		// Long enough that every phase of the beat is passed through several
		// times: a gate anywhere in the throw would show up as a gap.
		for f := range 120 {
			for i := range bands {
				bands[i] = float32(0.5 + 0.45*math.Sin(float64(f)/3+float64(i)))
			}
			m.scope.bands = bands
			if keeping {
				m.scope.beat.Since = time.Duration(f%15) * 33 * time.Millisecond
				m.scope.beatAt = time.Now()
			}
			m.stageFlow(100, 40)
		}
		return m.stage.drops
	}

	loose, kept := water(false), water(true)
	t.Logf("%d drops in the air answering the loudness, %d keeping time", len(loose), len(kept))

	if len(loose) == 0 {
		t.Fatal("the music threw no water at all")
	}
	if len(loose) != len(kept) {
		t.Fatalf("%d drops keeping time against %d answering the loudness, want the beat left out of it", len(kept), len(loose))
	}
	for i, d := range loose {
		if kept[i] != d {
			t.Fatalf("drop %d is %+v keeping time and %+v answering the loudness, want the beat left out of it", i, kept[i], d)
		}
	}
}

// The key that cycles the strip cycles the big screen too, and it is the one
// key up there that does not put the picture away. It walks this screen's own
// picture, from the one it opened in.
func TestStageCyclesItsPictures(t *testing.T) {
	m := stageModel(100, 40)
	m.scope.frame = make([]float32, 2*256)
	for i := range m.scope.frame {
		m.scope.frame[i] = 0.5
	}

	var tm tea.Model = m
	// Round from the picture it opens in, and never "off": a blank full screen
	// is not one of the pictures.
	for i, want := range []scopeMode{scopeWave, scopeBars, scopeMirror, scopeLadder, scopeWords} {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
		got := tm.(Model)
		if !got.stage.on {
			t.Fatalf("press %d put the big screen away instead of changing the picture", i+1)
		}
		if got.scopeMode() != want {
			t.Fatalf("press %d showed picture %d, want %d", i+1, got.scopeMode(), want)
		}
	}
}

// Every one of the three fills the terminal, whichever is showing and whether
// or not anything has arrived to draw yet.
func TestStageFillsTheScreenInEveryPicture(t *testing.T) {
	for mode := scopeWave; mode < scopeModes; mode++ {
		for _, ready := range []bool{false, true} {
			m := stageModel(100, 40)
			m.stage.mode = mode
			if ready {
				m.scope.frame = make([]float32, 2*256)
				m.scope.follow(m.scope.frame)
			} else {
				m.scope.bands, m.scope.frame = nil, nil
			}

			art := m.stagePicture(100, 40)
			if len(art) != 40 {
				t.Fatalf("picture %d (ready=%v) drew %d rows, want 40", mode, ready, len(art))
			}
			for i, line := range art {
				if got := len([]rune(ansiOff(line))); got != 100 {
					t.Errorf("picture %d (ready=%v): row %d is %d cells wide", mode, ready, i, got)
				}
			}
		}
	}
}

// The same picture, drawn at the size there is: given the whole screen it is
// mirrored about the middle, and in the strip under the artwork — where a
// reflection would cost the half of the height that makes it a picture at all —
// it stands on the floor.
func TestStageStandsOnTheFloorWhenItIsShallow(t *testing.T) {
	m := stageModel(100, 44)
	m.scope.modes[tabPlayer] = scopeMirror
	w := m.scopeWidth(m.layout())

	strip := m.stageArt(w, scopeRows)
	if len(strip) != scopeRows {
		t.Fatalf("the strip drew %d rows, want %d", len(strip), scopeRows)
	}
	if strings.TrimSpace(ansiOff(strip[scopeRows-1])) == "" {
		t.Error("the bottom row of the strip is empty, so the picture is not standing on it")
	}

	// And on a screen there is room, so it is mirrored: the top row and the
	// bottom row are both drawn.
	full := m.stageArt(100, 40)
	if strings.TrimSpace(ansiOff(full[20])) == "" {
		t.Error("nothing was drawn along the middle of the big picture")
	}
}

// The strip and the big screen are two choices, not one.
//
// They shared a setting once. The room's screen then came up in whatever was
// running in the corner of a working screen, and whatever was pressed for the
// room was still there on the working screens afterwards — one key, two
// meanings, and neither of them what was wanted.
func TestStageKeepsItsOwnPicture(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer] = scopeBars

	// It opens in its own picture, whatever the strip is set to.
	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if got := tm.(Model); got.scopeMode() != stageOpens {
		t.Errorf("the big screen opened showing %d, want %d", got.scopeMode(), stageOpens)
	}

	// Changed up there, and left in something else.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	after := tm.(Model)
	if after.scopeMode() == stageOpens {
		t.Fatal("the key did not change the big screen's picture, so there is nothing to leave it in")
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	back := tm.(Model)
	if back.stage.on {
		t.Fatal("the big screen is still up")
	}
	if back.scopeMode() != scopeBars {
		t.Errorf("the strip came back as %d after the big screen was left at %d, want the %d it was set to",
			back.scopeMode(), after.scopeMode(), scopeBars)
	}

	// And opening it again starts where it always starts, not where it was left.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if got := tm.(Model); got.scopeMode() != stageOpens {
		t.Errorf("the big screen came back showing %d, want %d — it is not a setting", got.scopeMode(), stageOpens)
	}
}

// Each working screen keeps its own picture, and the big screen does not touch
// any of them.
func TestTheStripStaysPerTab(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer], m.scope.modes[tabQueue] = scopeBars, scopeWave

	var tm tea.Model = m
	for _, k := range []string{"f", "v", "v"} {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: rune(k[0]), Text: k})
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	got := tm.(Model)
	if got.scope.modes[tabPlayer] != scopeBars || got.scope.modes[tabQueue] != scopeWave {
		t.Errorf("the screens came back as %v, want the bars on the player and the wave on the queue", got.scope.modes)
	}
}

// Paused, the analyser keeps handing over the last thing it heard, because a
// stopped output feeds it nothing new. Drawn as it arrives, the picture freezes
// mid-beat and sits there looking crashed; it has to sink to where silence
// would have left it instead.
func TestPausedPictureSettles(t *testing.T) {
	m := stageModel(100, 40)
	// The water, because it is the picture that keeps something of its own
	// between frames: the drops already thrown have to land as well.
	m.stage.mode = scopeMirror
	m.scope.frame = make([]float32, 2*256)
	for i := range m.scope.frame {
		m.scope.frame[i] = 0.9
	}
	m.scope.follow(m.scope.frame)
	m.scope.adoptBands(m.scope.bands)

	loud := m.scope.bands[0]
	if loud < 0.5 {
		t.Fatalf("the spectrum starts at %.2f, so there is nothing to watch fall", loud)
	}

	m.ps.Playing = false
	var tm tea.Model = m

	// A second of frames, all carrying the same stale measurement.
	for range 30 {
		tm, _ = tm.Update(msg.WaveformReady{
			Bands:   []float32{0.9, 0.9, 0.9},
			Samples: []float32{0.9, -0.9, 0.9, -0.9},
		})
	}

	got := tm.(Model)
	for i, v := range got.scope.bands {
		if v > 0.05 {
			t.Errorf("band %d still reads %.2f a second after the music stopped", i, v)
		}
	}
	for i, v := range got.scope.frame {
		if v > 0.05 || v < -0.05 {
			t.Errorf("the trace still swings to %.2f at sample %d", v, i)
			break
		}
	}

	// And the picture that draws them is empty rather than held — once what was
	// already in the air has landed. The measurements settle in the second
	// above; the drops thrown off them are still falling for a fifth of one
	// after that, which is the arc doing what it is meant to and not the
	// picture being held. Measured: the last of them is down by frame 36.
	for range 10 {
		tm, _ = tm.Update(msg.WaveformReady{
			Bands:   []float32{0.9, 0.9, 0.9},
			Samples: []float32{0.9, -0.9, 0.9, -0.9},
		})
	}
	got = tm.(Model)
	if len(got.stage.drops) > 0 {
		t.Errorf("%d drops are still in the air", len(got.stage.drops))
	}
	for r, line := range got.stageArt(100, 40) {
		if strings.TrimSpace(ansiOff(line)) != "" {
			t.Errorf("row %d of the paused picture still has %q in it", r, strings.TrimSpace(ansiOff(line)))
			break
		}
	}
}

// And it picks up again the moment the music does.
func TestPlayingPictureIsNotSettled(t *testing.T) {
	m := stageModel(100, 40)
	m.ps.Playing = true

	var tm tea.Model = m
	for range 10 {
		tm, _ = tm.Update(msg.WaveformReady{Bands: []float32{0.9, 0.9, 0.9}})
	}
	if got := tm.(Model).scope.bands[0]; got < 0.5 {
		t.Errorf("the spectrum reads %.2f while the music plays, want what arrived", got)
	}
}

// Nothing is left on the big screen but the picture — no name, no clock, no bar
// along the foot. Where the record has got to is drawn into the very edge of it
// instead: something going round the border clockwise from the top left corner.
// Only the head of it and a short tail behind, fading out — the way it came is
// not kept lit, because a lit border is a frame around the picture.
func TestTheEdgeIsTheProgress(t *testing.T) {
	const w, rows = 60, 20

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.stage.on = true
	m.stage.mode = scopeBars

	bands := make([]float32, 28)
	m.scope.bands = bands // silent, so the only ink is the edge
	m.ps.Duration = 100 * time.Second

	ink := func(at time.Duration) int {
		m.setProgress(at)
		var n int
		for _, line := range m.stagePicture(w, rows) {
			for _, r := range ansiOff(line) {
				if r != ' ' {
					n++
				}
			}
		}
		return n
	}

	quarter, half, most := ink(25*time.Second), ink(50*time.Second), ink(75*time.Second)
	t.Logf("the edge draws %d cells a quarter in, %d halfway, %d three quarters in", quarter, half, most)

	for _, n := range []int{quarter, half, most} {
		if n == 0 {
			t.Error("the edge drew nothing while the record was playing")
		}
		// A border of this screen is well over a hundred cells. Whatever is
		// drawn has to be a great deal less than one.
		if n > w/2 {
			t.Errorf("the edge drew %d cells, want a head and a tail rather than a border", n)
		}
	}

	// A quarter of the way in the head is along the top, and the foot has not
	// been reached.
	m.setProgress(25 * time.Second)
	lines := m.stagePicture(w, rows)
	if strings.TrimSpace(ansiOff(lines[0])) == "" {
		t.Error("a quarter of the way in, the top edge is empty")
	}
	if strings.TrimSpace(ansiOff(lines[rows-1])) != "" {
		t.Error("a quarter of the way in, the foot is already lit")
	}

	// Three quarters in it is along the foot, and the top it left behind is
	// dark again.
	m.setProgress(75 * time.Second)
	lines = m.stagePicture(w, rows)
	if strings.TrimSpace(ansiOff(lines[rows-1])) == "" {
		t.Error("three quarters in, the foot is empty")
	}
	if got := strings.TrimSpace(ansiOff(lines[0])); got != "" {
		t.Errorf("three quarters in, the top edge is still lit: %q", got)
	}
}

// The round closes where the next record starts. With a crossfade set that is
// not the end of the track: the last seconds are the two of them together, and
// a progress still climbing through them would be counting down to something
// that has already begun.
func TestTheEdgeClosesWhereTheNextRecordStarts(t *testing.T) {
	const w, rows = 60, 20

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.stage.on = true
	m.stage.mode = scopeBars
	m.scope.bands = make([]float32, 28)
	m.ps.Duration = 100 * time.Second

	// The corner it starts from and comes back to, in runes: a braille cell is
	// three bytes and slicing a string through the middle of one proves nothing.
	corner := func() rune {
		return []rune(ansiOff(m.stagePicture(w, rows)[0]))[0]
	}

	m.setProgress(90 * time.Second)
	if got := corner(); got != ' ' {
		t.Errorf("ten seconds from the end and gapless, the corner already reads %q", got)
	}

	m.settings.crossfade = 10 * time.Second
	if got := corner(); got == ' ' {
		t.Error("ten seconds from the end with a ten second crossfade, the round has not closed")
	}
}

// And the screen really is nothing but the picture.
func TestTheBigScreenIsAllPicture(t *testing.T) {
	m := scopeModel(100, 40)
	m.stage.on = true
	m.stage.mode = scopeBars
	m.ps.Title, m.ps.Artists = "A Title Nobody Wants Up There", []string{"An Artist"}

	drawn := m.stageView()
	if strings.Contains(ansiOff(drawn), m.ps.Title) {
		t.Error("the track's name is still on the big screen")
	}
	if strings.Contains(ansiOff(drawn), "/") {
		t.Error("the clock is still on the big screen")
	}
}

// The comet keeps time when the picture does, which is how anybody can tell
// which of the two ways it is drawing without being told.
func TestTheCometSaysWhichWayItIsDrawing(t *testing.T) {
	const w, rows = 100, 40

	m := scopeModel(w, rows)
	m.stage.on = true
	m.ps.Duration = 3 * time.Minute
	m.setProgress(time.Minute)

	lit := func() int {
		grid := make([]uint8, w*rows)
		paint := make([]int8, w*rows)
		for i := range paint {
			paint[i] = -1
		}
		m.stageEdge(w, rows, grid, paint, 8)

		var n int
		for _, cell := range grid {
			if cell != 0 {
				n++
			}
		}
		return n
	}

	// Answering the loudness, the comet is the same length whenever it is
	// looked at.
	steady := lit()
	if steady == 0 {
		t.Fatal("the comet drew nothing")
	}

	// Keeping time, it draws itself out on the beat and pulls back in between.
	m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
	m.scope.beatAt = time.Now()
	if !m.beatKeeping() {
		t.Fatal("a beat was found and the screen did not keep time")
	}

	m.scope.beat.Since = 0
	m.scope.beatAt = time.Now()
	on := lit()

	m.scope.beat.Since = 250 * time.Millisecond
	m.scope.beatAt = time.Now()
	between := lit()

	t.Logf("the comet is %d cells on the beat, %d between two, %d answering the loudness",
		on, between, steady)

	if on <= between {
		t.Errorf("the comet is %d cells on the beat and %d between two, want it drawing out on it", on, between)
	}
	if between >= steady {
		t.Errorf("between two beats the comet is %d cells against %d answering the loudness, want it shorter", between, steady)
	}
}

// A seek is a run round the edge, not a jump to the far side.
//
// The big screen has no clock and no progress bar. The head on the edge is the
// only thing on it that says where the record is, so it is the only thing that
// can say you have moved it — and it can only say how far by taking time over
// it. Moved instantly it would be indistinguishable from the picture being
// redrawn.
func TestSeekingRunsTheHeadRoundTheEdge(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 60, 20
	m.stage.on = true
	m.ps = &player.State{TrackID: "one", Duration: 4 * time.Minute, Playing: true}
	m.setProgress(2 * time.Minute)

	// Settled where the record is.
	m.stageEdgeFlow()
	if off := m.elapsed() - m.stage.edgeAt; off > time.Millisecond || off < -time.Millisecond {
		t.Fatalf("the head sat at %s with the record at %s", m.stage.edgeAt, m.elapsed())
	}

	// Seek forward, and watch it walk rather than arrive.
	was := m.elapsed()
	m.setProgress(was + 30*time.Second)

	var frames int
	for m.stage.edgeAt < m.elapsed()-stageEdgeSnap {
		before := m.stage.edgeAt
		m.stageEdgeFlow()
		if m.stage.edgeAt <= before {
			t.Fatalf("the head stopped at %s, short of %s", m.stage.edgeAt, m.elapsed())
		}
		if frames++; frames > 200 {
			t.Fatal("the head never got there")
		}
	}
	t.Logf("a thirty second seek took %d frames to walk", frames)
	if frames < 3 {
		t.Errorf("the head crossed thirty seconds in %d frames, which is a jump", frames)
	}

	// And back the other way.
	m.stageEdgeFlow()
	m.setProgress(m.elapsed() - 30*time.Second)
	before := m.stage.edgeAt
	m.stageEdgeFlow()
	if m.stage.edgeAt >= before {
		t.Errorf("seeking back walked the head forward, from %s to %s", before, m.stage.edgeAt)
	}
}

// A record that has just changed starts where it starts.
//
// Without this the head would sprint the whole way round on every skip, saying
// that somebody had seeked to the top of a track they had simply arrived at.
func TestANewRecordDoesNotRunTheHead(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 60, 20
	m.stage.on = true
	m.ps = &player.State{TrackID: "one", Duration: 4 * time.Minute, Playing: true}
	m.setProgress(3 * time.Minute)
	m.stageEdgeFlow()

	m.ps = &player.State{TrackID: "two", Duration: 3 * time.Minute, Playing: true}
	m.setProgress(0)
	m.stageEdgeFlow()

	if m.stage.edgeAt > time.Millisecond {
		t.Errorf("the new record's head started at %s rather than at the top", m.stage.edgeAt)
	}
}
