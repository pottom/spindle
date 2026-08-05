package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/ui/msg"
)

// stageModel is a player screen with the big picture up and something to draw.
func stageModel(w, h int) Model {
	m := scopeModel(w, h)
	m.stage.on = true

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

	// Anything at all comes back, including a key that means something else
	// everywhere: this is the way out and it has to be the first thing tried.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	back := tm.(Model)
	if back.stage.on {
		t.Error("a key press left the big screen up")
	}
	if back.tab != tabPlayer {
		t.Errorf("the key that closed the screen also changed tab, to %d", back.tab)
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

// The same key that cycles the strip cycles the big screen, and it is the one
// key up there that does not put the picture away.
func TestStageCyclesItsPictures(t *testing.T) {
	m := stageModel(100, 40)
	m.scope.frame = make([]float32, 2*256)
	for i := range m.scope.frame {
		m.scope.frame[i] = 0.5
	}

	var tm tea.Model = m
	// The same three the strip cycles, and never "off": a blank full screen is
	// not one of the pictures.
	for i, want := range []scopeMode{scopeBars, scopeMirror, scopeLadder, scopeGrain, scopeWords, scopeWave} {
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
			m.scope.modes[m.tab] = mode
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

// The strip and the big screen are one choice: what is showing under the
// artwork is what fills the screen when it is asked to.
func TestStageAndStripAgree(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer] = scopeBars

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if got := tm.(Model); got.scopeMode() != scopeBars {
		t.Errorf("the big screen opened showing %d, want the strip's %d", got.scopeMode(), scopeBars)
	}

	// And changing it up there is the same change down here.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	after := tm.(Model)
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if got := tm.(Model); got.scopeMode() != after.scopeMode() {
		t.Errorf("the strip came back as %d after the big screen was left at %d", got.scopeMode(), after.scopeMode())
	}
}

// Paused, the analyser keeps handing over the last thing it heard, because a
// stopped output feeds it nothing new. Drawn as it arrives, the picture freezes
// mid-beat and sits there looking crashed; it has to sink to where silence
// would have left it instead.
func TestPausedPictureSettles(t *testing.T) {
	m := stageModel(100, 40)
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

	// And the picture that draws them is empty rather than held.
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
// instead: a line from the top left corner, clockwise, as far as the track has
// played.
func TestTheEdgeIsTheProgress(t *testing.T) {
	const w, rows = 60, 20

	m := scopeModel(100, 44)
	m.width, m.height = w, rows
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeBars

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

	start, half, end := ink(0), ink(50*time.Second), ink(100*time.Second)
	t.Logf("the edge draws %d cells at the start, %d halfway, %d at the end", start, half, end)

	if !(start < half && half < end) {
		t.Errorf("the edge went %d → %d → %d, want it filling as the record plays", start, half, end)
	}

	// It goes round: at the halfway mark the top and the right are drawn and
	// the left is not.
	m.setProgress(50 * time.Second)
	lines := m.stagePicture(w, rows)
	if strings.TrimSpace(ansiOff(lines[0])) == "" {
		t.Error("halfway through, the top edge is empty")
	}

	// The left edge is the last quarter of the way round, so at the halfway
	// mark it is still to come. Counted in runes: a braille cell is three bytes
	// and slicing a string through the middle of one proves nothing.
	var left []rune
	for _, line := range lines[1 : len(lines)-1] {
		left = append(left, []rune(ansiOff(line))[0])
	}
	if strings.TrimSpace(string(left)) != "" {
		t.Errorf("halfway through, the left edge already reads %q", string(left))
	}
}

// And the screen really is nothing but the picture.
func TestTheBigScreenIsAllPicture(t *testing.T) {
	m := scopeModel(100, 40)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeBars
	m.ps.Title, m.ps.Artists = "A Title Nobody Wants Up There", []string{"An Artist"}

	drawn := m.stageView()
	if strings.Contains(ansiOff(drawn), m.ps.Title) {
		t.Error("the track's name is still on the big screen")
	}
	if strings.Contains(ansiOff(drawn), "/") {
		t.Error("the clock is still on the big screen")
	}
}
