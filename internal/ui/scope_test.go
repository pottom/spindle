package ui

import (
	"math"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func scopeModel(w, h int) Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "playing", Artists: []string{"someone"}, Playing: true, DeviceName: "spindle"}
	m.width, m.height = w, h
	m.resize()
	return m
}

// The trace is on to begin with, and the key puts it away and brings it back.
func TestScopeKeyTogglesTheTrace(t *testing.T) {
	m := scopeModel(100, 40)
	if !m.scopeVisible() {
		t.Fatal("the trace is not on by default")
	}

	// The key cycles: waveform, spectrum, the mirrored one, the ladder, off,
	// and round.
	want := []scopeMode{scopeBars, scopeMirror, scopeLadder, scopeOff, scopeWave}
	var tm tea.Model = m
	for i, mode := range want {
		var cmd tea.Cmd
		tm, cmd = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
		got := tm.(Model)
		if got.scope.modes[tabPlayer] != mode {
			t.Fatalf("press %d put it in mode %d, want %d", i+1, got.scope.modes[tabPlayer], mode)
		}
		if mode != scopeOff && cmd == nil && !got.scope.running {
			t.Errorf("press %d left nothing to draw the frames", i+1)
		}
	}
}

// Turning the trace on must not move anything: it goes into rows that were
// already blank. A visualiser is not worth making the cover jump.
func TestScopeMovesNothing(t *testing.T) {
	for _, size := range [][2]int{{100, 44}, {100, 40}, {120, 50}} {
		off := scopeModel(size[0], size[1])
		off.scope.modes[tabPlayer] = scopeOff
		if !off.scopeAvailable() {
			t.Fatalf("%dx%d: no room for the trace, so there is nothing to test", size[0], size[1])
		}
		before := plain(off.render())

		on := off
		on.scope.modes[tabPlayer] = scopeWave
		after := plain(on.render())

		b := strings.Split(before, "\n")
		a := strings.Split(after, "\n")
		if len(a) != len(b) {
			t.Fatalf("%dx%d: %d rows with the trace, %d without", size[0], size[1], len(a), len(b))
		}
		// Exactly the trace's own rows may differ; everything else has to be
		// character for character what it was.
		first := tabBarHeight + off.scopeTop(off.layout()) + scopeChrome
		for i := range b {
			if i >= first && i < first+scopeRows {
				continue
			}
			if a[i] != b[i] {
				t.Errorf("%dx%d: row %d moved\n  off: %q\n  on:  %q", size[0], size[1], i, b[i], a[i])
			}
		}
		// And the artwork keeps its size, or the cover would be re-rendered.
		if on.layout().artHeight != off.layout().artHeight {
			t.Errorf("%dx%d: the artwork changed size", size[0], size[1])
		}
	}
}

// Where there are not enough blank rows the trace is not offered, and the key
// says so by doing nothing rather than by rearranging the screen.
func TestScopeIsNotOfferedWithoutRoom(t *testing.T) {
	m := scopeModel(80, minHeight+2)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if cmd != nil {
		t.Error("v started the trace on a terminal too short for it")
	}
	if tm.(Model).scopeVisible() {
		t.Error("the trace is visible on a terminal too short for it")
	}
}

// Thirty redraws a second is the whole cost of the feature, so it has to stop
// the moment the trace leaves the screen.
func TestScopeStopsWhenItLeavesTheScreen(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer], m.scope.running = scopeWave, true

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"}) // to the playlists
	if got := tm.(Model); got.scopeVisible() {
		t.Fatal("the trace survived the move to a tab that does not draw it")
	}

	// The tick already in flight has to be the last one.
	tm, cmd := tm.Update(msgScopeTick())
	if cmd != nil {
		t.Error("the trace scheduled another frame after leaving the screen")
	}
	if tm.(Model).scope.running {
		t.Error("the loop still reports itself as running")
	}
}

// The trace is a line, not a scatter: consecutive samples are joined, so a steep
// slope stays continuous instead of breaking into separate dots.
func TestScopeDrawsAContinuousLine(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer] = scopeWave
	m.resize()

	// A step from bottom to top inside two dots: every row between has to be lit.
	m.scope.frame = []float32{-1, -1, 1, 1}
	m.scope.follow(m.scope.frame)
	lines := m.scopeRender(4, scopeRows)
	if len(lines) != scopeRows {
		t.Fatalf("scopeLines = %d rows, want %d", len(lines), scopeRows)
	}
	for i, line := range lines {
		if strings.TrimSpace(plain(line)) == "" {
			t.Errorf("row %d is blank, so the line broke where it rose", i)
		}
	}

	// Silence rests on the centre line, and only there.
	m.scope.frame = []float32{0, 0, 0, 0}
	m.scope.follow(m.scope.frame)
	lit := 0
	for _, line := range m.scopeRender(4, scopeRows) {
		if strings.TrimSpace(plain(line)) != "" {
			lit++
		}
	}
	if lit != 1 {
		t.Errorf("silence lit %d rows, want 1 — the centre line", lit)
	}
}

// With no backend to ask, the trace rests flat rather than failing: there is
// nothing to explain to anyone.
func TestScopeWithoutASourceRestsFlat(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer] = scopeWave
	m.resize()
	m.scope.frame = nil

	if got := m.scopeRender(20, scopeRows); len(got) != scopeRows {
		t.Fatalf("scopeLines = %d rows, want %d", len(got), scopeRows)
	}
}

func msgScopeTick() tea.Msg { return msg.WaveformReady{} }

// Measured against a live stream, peaks within one track ran from 0.06 to 0.87.
// At a fixed scale that is a flat line for half the track and a clipped one for
// the rest, so the trace follows the recent loudness instead.
func TestScopeFollowsTheLoudness(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer] = scopeWave
	m.resize()

	quiet := []float32{0.06, -0.06, 0.05, -0.05}
	loud := []float32{0.87, -0.87, 0.8, -0.8}

	// A quiet passage still reaches most of the way up.
	for range 40 {
		m.scope.follow(quiet)
	}
	m.scope.frame = quiet
	if got := m.scopeSample(0, 0, 4); got < 0.7 {
		t.Errorf("quiet passage scaled to %.2f, want most of the deflection", got)
	}

	// A sudden hit louder than anything before is drawn at the edge, not past it.
	m.scope.follow(loud)
	m.scope.frame = loud
	if got := m.scopeSample(0, 0, 4); got > 1 || got < 0.9 {
		t.Errorf("loud hit scaled to %.2f, want it pinned at the edge", got)
	}

	// And the gain does not run away in silence, where only noise is left.
	for range 500 {
		m.scope.follow([]float32{0, 0, 0, 0})
	}
	if m.scope.envelope < scopeFloor {
		t.Errorf("envelope fell to %.3f in silence, want it held at %.2f", m.scope.envelope, scopeFloor)
	}
}

// helpHeight has to ask for the help without knowing whether the waveform key
// is offered, because the layout decides that and the layout needs the height.
// That is only safe while the bar is the same height either way.
func TestHelpHeightDoesNotDependOnTheScope(t *testing.T) {
	for _, full := range []bool{false, true} {
		m := scopeModel(100, 44)
		m.help.ShowAll = full

		with := lipgloss.Height(m.help.View(m.keys.forPlayer(true, false, false, false, 120)))
		without := lipgloss.Height(m.help.View(m.keys.forPlayer(false, false, false, false, 120)))
		if with != without {
			t.Errorf("ShowAll=%v: help is %d rows with the waveform key and %d without", full, with, without)
		}
	}
}

// The trace stops whenever it leaves the screen, and nothing else would start
// it again: leaving for a playlist and coming back used to kill it until it was
// switched off and on by hand.
func TestScopeResumesOnItsOwn(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave

	var tm tea.Model = m
	tm, _ = tm.Update(msg.WaveformReady{Samples: []float32{0.5, -0.5}})

	// Away to another tab: the frame in flight is the last one.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	tm, cmd := tm.Update(msg.WaveformReady{})
	if cmd != nil || tm.(Model).scope.running {
		t.Fatal("the trace kept running after leaving the player")
	}

	// Back again, and it picks itself up without being asked.
	tm, cmd = tm.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if cmd == nil {
		t.Fatal("returning to the player did not restart the trace")
	}
	if !tm.(Model).scope.running {
		t.Error("the trace is not running after returning to the player")
	}
}

// A device arriving, or the first state landing, also leaves the trace stopped.
// The one-second tick is the safety net for everything the tab switch misses.
func TestScopeResumesOnTheTick(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave
	m.noDevice = true

	var tm tea.Model = m
	tm, _ = tm.Update(msg.WaveformReady{})
	if tm.(Model).scope.running {
		t.Fatal("the trace ran with no device")
	}

	got := tm.(Model)
	got.noDevice = false
	tm, cmd := tea.Model(got).Update(msg.Tick{})
	if cmd == nil {
		t.Fatal("the tick produced no command")
	}
	if !tm.(Model).scope.running {
		t.Error("the tick did not pick the trace back up")
	}
}

// A held note has to stand still. Without a trigger each frame is cut at an
// arbitrary point in the wave and the picture shimmers; with one it barely
// moves. Measured on a steady tone: 83% of cells changed per frame free-running
// against 23% triggered.
func TestTriggerSteadiesThePicture(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave
	w := m.layout().interior - leftMargin - rightMargin

	// A steady tone sampled at a phase that drifts frame to frame, which is
	// what an unaligned circular buffer hands over.
	frameAt := func(phase float64) []float32 {
		f := make([]float32, 2*player.WaveformWindow)
		for i := range f {
			f[i] = float32(0.7 * math.Sin(float64(i)*0.19+phase))
		}
		return f
	}

	churn := func(triggered bool) float64 {
		var prev []string
		var changed, total float64
		for k := range 40 {
			m.scope.frame = frameAt(float64(k) * 1.7)
			m.scope.follow(m.scope.frame)

			lines := m.scopeLinesFrom(w, scopeRows, 0)
			if triggered {
				lines = m.scopeRender(w, scopeRows)
			}
			if prev != nil {
				for i := range lines {
					a, b := plain(lines[i]), plain(prev[i])
					for c := 0; c < len(a) && c < len(b); c++ {
						if a[c] != b[c] {
							changed++
						}
					}
					total += float64(len(a))
				}
			}
			prev = lines
		}
		return changed / total
	}

	steady, free := churn(true), churn(false)
	if steady >= free*0.6 {
		t.Errorf("trigger changed %.0f%% of cells a frame against %.0f%% free-running, want it far steadier",
			steady*100, free*100)
	}
}

// The glow is what stops the trace looking redrawn thirty times a second. It
// has to leave more lit than the current frame alone, and it has to be drawn
// behind the beam rather than as part of it.
func TestGlowTrailsTheBeam(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave
	w := 40

	wave := func(phase float64) []float32 {
		f := make([]float32, 2*player.WaveformWindow)
		for i := range f {
			f[i] = float32(0.8 * math.Sin(float64(i)*0.19+phase))
		}
		return f
	}

	m.scope.frame = wave(0)
	m.scope.follow(m.scope.frame)
	bare := lit(m.scopeLinesFrom(w, scopeRows, 0))

	// A few frames at shifting phases, each remembered.
	for k := range scopeTrail {
		m.scope.frame = wave(float64(k) * 0.6)
		m.scope.follow(m.scope.frame)
		grid, _ := m.scopeGrid(w, scopeRows, 0)
		m.scope.remember(grid)
	}
	withGlow := lit(m.scopeLinesFrom(w, scopeRows, 0))

	if withGlow <= bare {
		t.Errorf("glow lit %d cells against %d without it, want more", withGlow, bare)
	}

	// A resize leaves grids of the wrong size behind; they have to be ignored
	// rather than drawn at the wrong offset.
	if got := m.scopeLinesFrom(w+7, scopeRows, 0); len(got) != scopeRows {
		t.Errorf("scopeLinesFrom at a new width = %d rows, want %d", len(got), scopeRows)
	}
}

func lit(lines []string) int {
	n := 0
	for _, line := range lines {
		for _, r := range plain(line) {
			if r != ' ' {
				n++
			}
		}
	}
	return n
}

// The screen always shows the same slice of time, whatever width it is. Reading
// one sample per dot instead zooms in on a wide terminal and out on a narrow
// one, and the trace appears to speed up or slow down with nothing but the
// window size — which is how it once quietly lost a third of its pace.
func TestScopeSpansTheSameTimeAtAnyWidth(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave

	// A tone at a fixed rate: the number of times it crosses zero across the
	// screen is how much time the screen is showing.
	f := make([]float32, 2*player.WaveformWindow)
	for i := range f {
		f[i] = float32(math.Sin(float64(i) * 0.19))
	}
	m.scope.frame = f
	m.scope.follow(f)

	crossings := func(w int) int {
		n, prev := 0, float32(0)
		for x := range w * dotsPerCellX {
			v := m.scopeSample(0, x, w*dotsPerCellX)
			if prev < 0 && v >= 0 {
				n++
			}
			prev = v
		}
		return n
	}

	narrow, wide := crossings(40), crossings(95)
	if narrow != wide {
		t.Errorf("a 40-cell screen shows %d cycles and a 95-cell one %d, want the same span of time", narrow, wide)
	}
}

// The extremes of the swing are drawn back from the accent and the middle sits
// in it, so the line has a lit core instead of one flat colour top to bottom.
func TestScopeShadesTowardsItsCore(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave

	// A swing wide enough to reach every row.
	f := make([]float32, 2*player.WaveformWindow)
	for i := range f {
		f[i] = float32(math.Sin(float64(i) * 0.19))
	}
	m.scope.frame = f
	m.scope.follow(f)

	lines := m.scopeLinesFrom(40, scopeRows, 0)
	if len(lines) != scopeRows {
		t.Fatalf("scopeLinesFrom = %d rows, want %d", len(lines), scopeRows)
	}

	// At one amplitude the inner rows have to come out brighter than the edges,
	// which is visible as a different escape sequence for the same dots.
	inner, outer := colourUsed(lines[1]), colourUsed(lines[0])
	if inner == "" || outer == "" {
		t.Fatalf("rows carry no colour: inner %q outer %q", inner, outer)
	}
	if inner == outer {
		t.Error("the edges and the core are the same colour, so the trace is flat")
	}
}

// colourUsed is the first colour escape in a rendered row.
func colourUsed(line string) string {
	start := strings.Index(line, "\x1b[")
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start:], "m")
	if end < 0 {
		return ""
	}
	return line[start : start+end+1]
}

// Two terminals side by side on a laptop is about eighty columns each, and the
// artwork has to survive that: it is most of why the screen looks the way it
// does. Below that there is a point where the words need the room more, and
// the picture is what goes.
func TestArtworkSurvivesAHalfScreen(t *testing.T) {
	for w := compactBelow; w <= 100; w += 2 {
		m := scopeModel(w, 45)
		l := m.layout()
		if !l.hasArt() {
			t.Errorf("%d columns: no artwork, want it kept", w)
			continue
		}
		if l.infoWidth < minInfoCols {
			t.Errorf("%d columns: %d for the words, want at least %d", w, l.infoWidth, minInfoCols)
		}
	}

	// And nothing is drawn wider than the terminal at any of them.
	for w := minWidth; w <= 200; w += 3 {
		m := scopeModel(w, 45)
		for i, row := range strings.Split(plain(m.render()), "\n") {
			if got := len([]rune(row)); got > w {
				t.Fatalf("%d columns: row %d is %d wide", w, i, got)
			}
		}
	}
}

// Below the artwork's floor everything still works: the text and the lists take
// the whole width, which is the point of dropping the picture.
func TestCompactDropsOnlyTheArtwork(t *testing.T) {
	m := scopeModel(compactBelow-4, 30)
	l := m.layout()
	if l.hasArt() {
		t.Fatal("the artwork survived below its floor")
	}
	if l.infoWidth != l.interior-leftMargin-rightMargin {
		t.Errorf("info column is %d, want the whole width (%d)", l.infoWidth, l.interior-leftMargin-rightMargin)
	}

	out := plain(m.render())
	if !strings.Contains(out, "playing") {
		t.Errorf("the track is not named:\n%s", out)
	}
	if rows := strings.Split(out, "\n"); len(rows) != m.height {
		t.Errorf("render() = %d rows, want %d", len(rows), m.height)
	}
}

// The waveform lives in the rows the artwork leaves. A cover free to grow until
// it fills the body took those rows away, and the trace vanished on exactly the
// terminals with the most space to draw it — so the cover is held to two thirds
// of the height.
func TestWaveformSurvivesAGrowingCover(t *testing.T) {
	for _, size := range [][2]int{
		{80, 30}, {100, 30}, {100, 44}, {132, 40}, {160, 45}, {200, 50}, {200, 60},
	} {
		m := scopeModel(size[0], size[1])
		if !m.scopeAvailable() {
			l := m.layout()
			t.Errorf("%dx%d: no room for the waveform (body %d, art %dx%d, room %d)",
				size[0], size[1], l.bodyHeight, l.artWidth, l.artHeight, m.scopeRoom(l))
		}
	}
}

// The waveform is on when spindle starts. A feature nobody knows to ask for may
// as well not exist, and this one is most of what makes the screen feel alive.
func TestWaveformIsOnByDefault(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	if m.scope.modes[tabPlayer] != scopeWave {
		t.Errorf("the visualiser starts in mode %d, want the waveform", m.scope.modes[tabPlayer])
	}

	// Nothing has been drawn yet, so it must not claim to be running either —
	// the tick loop starts when there is something to draw on.
	if m.scope.running {
		t.Error("the tick loop is running before anything is on screen")
	}
}

// Bass on the left, treble on the right, as every analyser reads.
func TestBarsPutBassOnTheLeft(t *testing.T) {
	m := scopeModel(120, 44)
	w := m.layout().interior - leftMargin - rightMargin

	bands := make([]float32, 28)
	bands[1] = 1 // a low band, loud
	m.scope.adoptBands(bands)

	lines := m.barsLines(w, scopeRows)
	if len(lines) != scopeRows {
		t.Fatalf("barsLines = %d rows, want %d", len(lines), scopeRows)
	}

	// The lit columns have to sit in the left tenth of the row.
	at := strings.IndexFunc(plain(lines[0]), func(r rune) bool { return r != ' ' })
	if at < 0 {
		t.Fatal("nothing was drawn for a loud low band")
	}
	if at > w/10 {
		t.Errorf("the lowest band was drawn at column %d of %d, want it on the left", at, w)
	}
}

// The marker hangs where a hit reached and falls back slowly, which is what
// gives a meter its character. It never sits below the bar it fell from.
func TestPeakMarkersFallBack(t *testing.T) {
	m := scopeModel(120, 44)

	loud := make([]float32, 28)
	for i := range loud {
		loud[i] = 0.9
	}
	m.scope.adoptBands(loud)

	quiet := make([]float32, 28)
	m.scope.adoptBands(quiet)
	if m.scope.peaks[0] < 0.5 {
		t.Errorf("the marker fell to %.2f in one frame, want it to hang", m.scope.peaks[0])
	}

	for range 200 {
		m.scope.adoptBands(quiet)
	}
	if m.scope.peaks[0] != 0 {
		t.Errorf("the marker is still at %.2f after two hundred frames", m.scope.peaks[0])
	}
}

// Markers fall by a share of where they are, not by a fixed step. With a fixed
// step every marker descends at one rate, so markers that started together stay
// together — and drew a straight line clean across the screen, over bands that
// were silent.
func TestPeakMarkersDoNotFallInStep(t *testing.T) {
	m := scopeModel(120, 44)

	// Two bands that reached different heights.
	start := make([]float32, 28)
	start[0], start[1] = 1, 0.4
	m.scope.adoptBands(start)

	quiet := make([]float32, 28)
	for range 8 {
		m.scope.adoptBands(quiet)
	}

	high, low := m.scope.peaks[0], m.scope.peaks[1]
	if high <= low {
		t.Fatalf("markers at %.2f and %.2f, want the higher one still higher", high, low)
	}
	// The gap has to have closed, which a fixed step never does.
	if gap := high - low; gap >= 0.6 {
		t.Errorf("the markers are still %.2f apart, want them falling at their own rates", gap)
	}
}

// The glow belongs to the waveform. In the spectrum no samples arrive, so the
// trail holds a flat centre line — and it burned straight across the bars, over
// bands that were silent and past the last one entirely.
func TestBarsCarryNoWaveformTrail(t *testing.T) {
	m := scopeModel(120, 44)
	m.scope.modes[tabPlayer] = scopeBars
	w := m.layout().interior - leftMargin - rightMargin

	// A trail as it would be left by silence: the flat centre line.
	m.scope.frame = make([]float32, 2*player.WaveformWindow)
	m.scope.follow(m.scope.frame)
	for range scopeTrail {
		grid, _ := m.scopeGrid(w, scopeRows, 0)
		m.scope.remember(grid)
	}

	bands := make([]float32, 28)
	bands[2] = 0.9
	m.scope.adoptBands(bands)

	rows := m.barsLines(w, scopeRows)
	if len(rows) != scopeRows {
		t.Fatalf("barsLines = %d rows, want %d", len(rows), scopeRows)
	}

	// The far right has no band with any level, so it has to be empty.
	for i, row := range rows {
		tail := []rune(plain(row))
		if len(tail) < 10 {
			continue
		}
		if strings.TrimSpace(string(tail[len(tail)-10:])) != "" {
			t.Errorf("row %d has something drawn past the last band: %q", i, string(tail[len(tail)-10:]))
		}
	}
}

// A meter whose columns are not the same size reads as a meter that is wrong.
// What will not divide is left as a margin rather than given to some bands.
func TestBarsAreAllTheSameWidth(t *testing.T) {
	m := scopeModel(100, 40)
	m.scope.modes[tabPlayer] = scopeBars
	m.resize()

	bands := 28
	m.scope.bands = make([]float32, bands)
	m.scope.peaks = make([]float32, bands)
	for i := range m.scope.bands {
		m.scope.bands[i] = 1
	}

	// A width that divides evenly and one that does not.
	for _, w := range []int{bands * 3, bands*3 + 17} {
		widths := map[int]int{}
		for _, line := range m.barsLines(w, scopeRows) {
			run := 0
			for _, r := range plain(line) {
				if r == ' ' {
					if run > 0 {
						widths[run]++
					}
					run = 0
					continue
				}
				run++
			}
			if run > 0 {
				widths[run]++
			}
		}
		if len(widths) != 1 {
			t.Errorf("at width %d the bars come out in %d different widths: %v", w, len(widths), widths)
		}
	}
}

// The spectrum is the full width of the screen it is given. It used to leave
// whatever a whole number of equal bars would not divide as a margin, which at
// a wide terminal was a hand's width of nothing at each end.
func TestBarsFillTheWidth(t *testing.T) {
	for _, w := range []int{40, 77, 100, 133, 160, 199, 240} {
		bands := make([]float32, 28)
		for i := range bands {
			bands[i] = 1
		}

		m := scopeModel(w+leftMargin+rightMargin+2, 44)
		m.scope.adoptBands(bands)

		row := plain(m.barsLines(w, scopeRows)[0])
		first := strings.IndexFunc(row, func(r rune) bool { return r != ' ' })
		last := strings.LastIndexFunc(row, func(r rune) bool { return r != ' ' })
		if first < 0 {
			t.Fatalf("w = %d: nothing was drawn for a full spectrum", w)
		}

		// A little slack at each end: the bars carry a blank column between
		// them, the last one keeps its own, and a margin under barsSlack is
		// left rather than drawing fewer bars for it.
		if first > barsSlack || last < w-barsSlack {
			t.Errorf("w = %d: the bars run from %d to %d, want them to reach both ends", w, first, last)
		}
	}
}

// Whatever the width, the bars fill it: either there is one for every band and
// what is left over is a cell or two, or there are fewer and wider ones and
// there is nothing left over at all.
func TestBarsFitLeavesLittleOver(t *testing.T) {
	for _, bands := range []int{16, 28, 64} {
		for w := 8; w <= 400; w++ {
			pitch, count := barsFit(w, bands)
			if count < 1 || count > bands {
				t.Fatalf("w = %d: %d bars, want between 1 and %d", w, count, bands)
			}
			if over := w - pitch*count; over < 0 || over > barsSlack {
				t.Errorf("w = %d over %d bands: %d bars of %d cells leave %d over", w, bands, count, pitch, over)
			}
		}
	}
}

// A spectrum drawn in a narrow panel keeps its bands: they are what it is, and
// a couple of unused cells at the edge is not worth half of them.
func TestNarrowBarsKeepTheirBands(t *testing.T) {
	const bands = 64
	if _, count := barsFit(68, bands); count != bands {
		t.Errorf("at 68 cells the spectrum came out in %d bars, want one for each of the %d bands", count, bands)
	}
}

// A tube's beam is bright where it dwells and faint where it races: the same
// beam drawing a steep edge is spread over ten times the distance, so it lights
// each dot for a tenth as long. Drawn without that, a waveform is a plot; drawn
// with it, it is a trace.
func TestBeamDwellsOnTheFlatAndFadesOnTheEdge(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave
	w := m.layout().interior - leftMargin - rightMargin

	dwellOf := func(step float64) float32 {
		f := make([]float32, 2*player.WaveformWindow)
		for i := range f {
			f[i] = float32(0.8 * math.Sin(float64(i)*step))
		}
		m.scope.frame = f
		m.scope.follow(f)

		_, _, dwell := m.scopeBeam(w, scopeRows, 0)
		var sum float32
		for _, v := range dwell {
			sum += v
		}
		return sum / float32(len(dwell))
	}

	// A slow wave is nearly flat across a cell; a fast one climbs through it.
	slow, fast := dwellOf(0.02), dwellOf(0.9)
	t.Logf("dwell: slow %.2f, fast %.2f", slow, fast)

	if slow <= fast {
		t.Errorf("the beam dwells %.2f on a slow wave and %.2f on a fast one, want the slow one brighter", slow, fast)
	}
}

// The trigger listens through a low pass, the way the coupling switch on a
// scope does: a zero crossing found on the hiss riding on a note is somewhere
// else every frame, and the picture shimmers for it.
func TestTriggerIgnoresTheHissOnTheNote(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave
	w := m.layout().interior - leftMargin - rightMargin
	dots := w * dotsPerCellX

	note := func(noise float64) []float32 {
		f := make([]float32, 2*player.WaveformWindow)
		for i := range f {
			v := 0.7 * math.Sin(float64(i)*0.05)
			if i%2 == 0 {
				v += noise
			} else {
				v -= noise
			}
			f[i] = float32(v)
		}
		return f
	}

	m.scope.frame = note(0)
	m.scope.follow(m.scope.frame)
	clean := m.scopeTrigger(dots)

	m.scope.frame = note(0.25)
	m.scope.follow(m.scope.frame)
	hissy := m.scopeTrigger(dots)

	t.Logf("trigger: clean at %d, with hiss at %d", clean, hissy)

	// Within a few samples is the same place on the wave; the noise is not
	// allowed to pick a different crossing altogether.
	if hissy-clean > 6 || clean-hissy > 6 {
		t.Errorf("the trigger moved from %d to %d when hiss was added, want it on the same crossing", clean, hissy)
	}
}

// The trace throws sparks: a hit sheds beads off its crests, which arc away
// from the line and are pulled back to it.
func TestTraceThrowsSparks(t *testing.T) {
	m := scopeModel(100, 44)
	m.scope.modes[tabPlayer] = scopeWave
	w := m.scopeWidth(m.layout())

	wave := func(amp float64) []float32 {
		f := make([]float32, 2*player.WaveformWindow)
		for i := range f {
			f[i] = float32(amp * math.Sin(float64(i)*0.22))
		}
		return f
	}

	// A quiet passage, then something twice as loud: the trace is scaled to the
	// recent loudness, so what marks the hit is the level underneath it.
	for range 30 {
		m.scope.frame = wave(0.3)
		m.scope.follow(m.scope.frame)
		m.throwSparks(w, scopeRows)
	}
	m.scope.frame = wave(0.95)
	m.scope.follow(m.scope.frame)
	m.throwSparks(w, scopeRows)

	if len(m.scope.sparks) == 0 {
		t.Fatal("the hit threw nothing off the trace")
	}
	t.Logf("the hit threw %d sparks", len(m.scope.sparks))

	// They are drawn with the trace rather than in a layer of their own.
	off := m.scopeRender(w, scopeRows)
	m.scope.sparks = nil
	bare := m.scopeRender(w, scopeRows)
	if strings.Join(off, "\n") == strings.Join(bare, "\n") {
		t.Error("the sparks changed nothing on screen")
	}

	// And left alone, every one of them comes back to the line.
	m.scope.frame = wave(0.95)
	for range 300 {
		m.scope.follow(m.scope.frame)
		m.throwSparks(w, scopeRows)
	}
	m.scope.frame = make([]float32, 2*player.WaveformWindow)
	for range 300 {
		m.scope.follow(m.scope.frame)
		m.throwSparks(w, scopeRows)
	}
	if len(m.scope.sparks) != 0 {
		t.Errorf("%d sparks are still in the air over a silent trace", len(m.scope.sparks))
	}
}

// Nothing the pictures carry between frames grows without a bound.
//
// Everything here is a simulation: water thrown off the meter, sparks off the
// trace, a trail of old frames for the phosphor. Each of them keeps what is
// still alive and drops the rest, and each has a ceiling on how much can be
// alive at once — a stuck display, a silent hour or a record that never stops
// hitting must not turn any of them into a list that only ever gets longer.
//
// Driven here rather than reasoned about: a few minutes of playback at thirty
// frames a second, through the loudest music the model can be given, with the
// record changing under it. Measured over half an hour it settles at the same
// numbers; what is kept here is the shortest run that reaches them.
func TestThePicturesDoNotGrow(t *testing.T) {
	m := scopeModel(160, 48)
	m.stage.on = true

	bands := make([]float32, 28)
	samples := make([]float32, 2*512)

	const frames = 5_000
	for f := range frames {
		// Loud and moving, so that every throw and every trigger fires.
		phase := float64(f) / 7
		for i := range bands {
			bands[i] = float32(0.5 + 0.49*math.Sin(phase+float64(i)))
		}
		for i := range samples {
			samples[i] = float32(math.Sin(phase*40 + float64(i)/9))
		}

		m.scope.frame = samples
		m.scope.follow(samples)
		m.scope.adoptBands(bands)
		m.rememberScope()
		m.throwSparks(m.width, m.height)
		m.stageFlow(m.width, m.height)
		m.stageFlowIn(m.width, m.height, stageThrows{
			span: float32(m.height * dotsPerCellY), reach: 40, lift: wordsLift,
		})

		if f%1000 == 999 {
			m.ps.TrackID = "another"
			m.words.forced = time.Now()
		}
	}

	t.Logf("after %d frames: %d drops (room for %d), %d sparks (room for %d), %d frames of trail (room for %d)",
		frames, len(m.stage.drops), cap(m.stage.drops),
		len(m.scope.sparks), cap(m.scope.sparks), len(m.scope.trail), cap(m.scope.trail))

	if got := len(m.stage.drops); got > stageDrops {
		t.Errorf("%d drops are in the air, want no more than %d", got, stageDrops)
	}
	if got := len(m.scope.sparks); got > sparkMost {
		t.Errorf("%d sparks are in the air, want no more than %d", got, sparkMost)
	}
	if got := len(m.scope.trail); got > scopeTrail {
		t.Errorf("the trail holds %d frames, want no more than %d", got, scopeTrail)
	}

	// The room they have taken is bounded too: a slice that is reused by
	// keeping what is alive can still creep if it is regrown every pass.
	for _, c := range []struct {
		what string
		got  int
		most int
	}{
		{"drops", cap(m.stage.drops), 4 * stageDrops},
		{"sparks", cap(m.scope.sparks), 4 * sparkMost},
		{"trail", cap(m.scope.trail), 4 * scopeTrail},
	} {
		if c.got > c.most {
			t.Errorf("the %s have grown room for %d, want no more than %d", c.what, c.got, c.most)
		}
	}
}

// The phase runs from nought on the beat to one just before the next, and it is
// worked out from the clock rather than read off the last report — a report is
// a frame old by the time anything draws to it.
func TestTheBeatPhaseRunsWithTheClock(t *testing.T) {
	m := scopeModel(100, 40)

	if _, ok := m.beatPhase(); ok {
		t.Error("a phase was reported before any beat was found")
	}

	// A beat every half second, the last one heard a moment ago.
	m.scope.beat = player.Beat{Period: 500 * time.Millisecond, Since: 100 * time.Millisecond}
	m.scope.beatAt = time.Now()

	phase, ok := m.beatPhase()
	if !ok {
		t.Fatal("no phase from a beat that was just reported")
	}
	t.Logf("a fifth of the way through a beat: phase %.2f", phase)
	if phase < 0.15 || phase > 0.25 {
		t.Errorf("phase %.2f, want about a fifth", phase)
	}

	// Half a period later it is round the other side, and it wraps rather than
	// running past one.
	m.scope.beatAt = time.Now().Add(-300 * time.Millisecond)
	if phase, _ := m.beatPhase(); phase < 0.75 || phase > 0.85 {
		t.Errorf("phase %.2f four fifths of the way through, want about four fifths", phase)
	}
	m.scope.beatAt = time.Now().Add(-450 * time.Millisecond)
	if phase, _ := m.beatPhase(); phase > 0.2 {
		t.Errorf("phase %.2f just past a beat, want it back near nought", phase)
	}

	// And a report that has stopped arriving stops being kept to.
	m.scope.beatAt = time.Now().Add(-beatLost - time.Second)
	if _, ok := m.beatPhase(); ok {
		t.Error("the phase was still reported from a beat nobody has confirmed for seconds")
	}
}

// Everything that moves moves on the beat, and the key puts it back the way it
// was.
//
// The two ways of drawing are the whole question — keeping time with a record
// or answering how loud it is — and the only way to answer it is to see both on
// the same record.
func TestTheScreenKeepsTimeAndCanBeToldNotTo(t *testing.T) {
	m := scopeModel(120, 44)
	m.stage.on = true
	m.stage.mode = scopeWords
	m.words.beats, m.words.text = true, wordsNotes

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.8
	}
	m.scope.bands = bands

	// With no beat found, nothing keeps time and the marks ride the loudness,
	// which is what this always did.
	if m.beatKeeping() {
		t.Error("the screen kept time before any beat was found")
	}
	loud := m.wordsRiding(4)
	if _, swaying := m.wordsSway(); swaying {
		t.Error("the row swayed before any beat was found")
	}

	// A beat every half second. What it moves is the lean, not the height: the
	// two were put on one dimension first and it read as twitching, because a
	// mark cannot answer how loud its band is and where the beat is with the
	// same movement. See sway.go.
	m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
	m.scope.beatAt = time.Now()
	if !m.beatKeeping() {
		t.Fatal("a beat was found and the screen did not keep time")
	}

	// A low end that is actually being struck, because the row sways to what is
	// being played rather than to what was found — a beat nobody is hitting
	// moves nothing. See sway.go.
	kick(&m, 4, 0.30)
	for i := range m.scope.bands {
		m.scope.bands[i] = 0.8
	}

	m.scope.beat.Since = 0 // on the beat
	m.scope.beatAt = time.Now()
	on := m.wordsRiding(4)
	onLean, swaying := m.wordsSway()
	if !swaying {
		t.Fatal("a beat was found and the row did not sway")
	}

	m.scope.beat.Since = 250 * time.Millisecond // half way between two
	m.scope.beatAt = time.Now()
	between := m.wordsRiding(4)
	betweenLean, _ := m.wordsSway()

	t.Logf("the marks stand at %v on the beat and %v between two, against %v on loudness alone", on, between, loud)
	t.Logf("and the row leans %+.3f on the beat against %+.3f between two", onLean, betweenLean)

	// The height is the sound's, and the beat does not touch it.
	for i := range on {
		if on[i] != loud[i] || between[i] != loud[i] {
			t.Errorf("mark %d stands at %d on the beat and %d between two, want the %d the sound alone put it at",
				i, on[i], between[i], loud[i])
		}
	}

	// The lean is the beat's, and it is all the way over on it.
	if abs32(onLean) <= abs32(betweenLean) {
		t.Errorf("the row leans %+.3f on the beat and %+.3f between two, want it furthest over on the beat", onLean, betweenLean)
	}
	if abs32(onLean) < wordsSwayMost*0.9 {
		t.Errorf("on the beat the row leans %+.3f, want most of the %.2f it has to give", onLean, wordsSwayMost)
	}

	// And the key hands the old picture back: no sway at all, and the height it
	// always had.
	m.stage.loose = false
	if m.beatKeeping() {
		t.Error("the key was turned off and the screen kept time anyway")
	}
	if _, swaying := m.wordsSway(); swaying {
		t.Error("the key was turned off and the row swayed anyway")
	}
	if off := m.wordsRiding(4); off[0] != loud[0] {
		t.Errorf("with the key off the marks stand at %d, want the %d they stood at before", off[0], loud[0])
	}
}

// In the band beside the cover, a meter stands on the floor and a wave hangs
// from the middle.
//
// Drawn at a fixed four rows in a band of thirteen, the bars floated in the
// middle with their feet on nothing — a row of bars stands on a floor, and a
// floor that lines up with nothing reads as a mistake. The trace is drawn at the
// height it is given now, so the bars reach the foot of the band, which is the
// foot of the picture beside them.
//
// And the wave is unmoved by that, which is the other half of it: it is drawn
// about the middle of whatever it is given, and the middle of that band is the
// row the playhead is on.
func TestTheTraceFillsTheBandItIsGiven(t *testing.T) {
	m := scopeModel(200, 44)
	m.tab = tabQueue
	m.scope.bands = make([]float32, 28)
	for i := range m.scope.bands {
		m.scope.bands[i] = 1
	}

	l := m.layout()
	band := m.listBandRows(l)
	if band < 8 {
		t.Skip("the band is too short at this size to say anything")
	}

	m.scope.modes[tabQueue] = scopeBars
	rows := m.place(m.traceBlock(), queueScopeWidth(l), band)
	if len(rows) != band {
		t.Fatalf("the trace came out %d rows in a band of %d", len(rows), band)
	}
	if strings.TrimSpace(plain(rows[band-1])) == "" {
		t.Error("the bars do not reach the foot of the band")
	}
	if strings.TrimSpace(plain(rows[0])) == "" {
		t.Error("the bars at full tilt do not reach the top of the band")
	}

	// Under the artwork on the player it is still a strip, and deliberately so:
	// a glance rather than the loudest thing on that screen.
	if strip := m.place(m.traceBlock(), m.scopeWidth(l), scopeRows); len(strip) != scopeRows {
		t.Errorf("the strip under the artwork is %d rows, want %d", len(strip), scopeRows)
	}
}
