package ui

import (
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The figures come from drawings, converted by cmd/spindle-figures. What the
// generator has to hand over is a line drawing — the strokes are the figure —
// and, if the manifest asked for one, a hollow head for a face to go in.
func TestAFigureArrivesAsALineDrawing(t *testing.T) {
	d, ok := figureFor("robot")
	if !ok {
		t.Fatal("the robot was not generated — run go run ./cmd/spindle-figures")
	}
	if d.licence == "" || d.from == "" {
		t.Error("a figure was generated without saying where it came from")
	}

	for _, tall := range []int{62, 100} {
		p, ok := d.at(tall, "idle")
		if !ok {
			t.Fatalf("no idle pose at %d dots", tall)
		}
		if p.tall != tall {
			t.Errorf("asked for %d dots and got %d", tall, p.tall)
		}

		var lit int
		p.draw(false, func(_, _ int) { lit++ })
		share := float64(lit) / float64(p.wide*p.tall)
		t.Logf("%3d dots tall: %dx%d, %d lit (%.0f%%), face box %dx%d at %d,%d",
			tall, p.wide, p.tall, lit, share*100, p.headW, p.headH, p.headX, p.headY)

		if lit == 0 {
			t.Fatal("nothing was drawn")
		}
		// A line drawing, not a silhouette: a filled figure would be most of
		// its own box, and on this screen that is a hole rather than a picture.
		if share > 0.30 {
			t.Errorf("%.0f%% of the figure is lit, want a drawing rather than a mass", share*100)
		}

		// A figure that asked for a face has a hollow head to put one in; one
		// that did not is left exactly as it was drawn, face and all.
		var inside int
		p.draw(false, func(x, y int) {
			if x >= p.headX && x < p.headX+p.headW && y >= p.headY && y < p.headY+p.headH {
				inside++
			}
		})
		switch {
		case p.headW == 0:
			t.Log("      the head was left as it was drawn")
		case inside != 0:
			t.Errorf("%d dots are still inside the head, want it cleared for the face", inside)
		case p.headW < 8 || p.headH < 6:
			t.Errorf("the face box is %dx%d, too small to put a face in", p.headW, p.headH)
		}
	}

	// And the whole walk is there.
	for i := range 8 {
		if _, ok := d.at(100, "walk"+string(rune('0'+i))); !ok {
			t.Errorf("walk frame %d is missing", i)
		}
	}
	// And every drawing the acts call for. See figureActs.
	for _, act := range figureActNames {
		for _, f := range figureActs[act] {
			if _, ok := d.at(100, f.pose); !ok {
				t.Errorf("the %s act wants the %s pose, which is missing", act, f.pose)
			}
		}
	}

	// The nearest size is the one that comes back, whatever is asked for.
	tallest := 0
	for _, size := range d.sizes {
		tallest = max(tallest, size.tall)
	}
	p, _ := d.at(1000, "idle")
	if p.tall != tallest {
		t.Errorf("asked for a figure 1000 dots tall and got %d, want the tallest there is (%d)", p.tall, tallest)
	}
}

// And it draws where it is put.
func TestAFigureDrawsWhereItIsPut(t *testing.T) {
	d, _ := figureFor("robot")
	p, ok := d.at(100, "walk2")
	if !ok {
		t.Fatal("no pose")
	}

	grid := make([][]byte, p.tall)
	for y := range grid {
		grid[y] = []byte(strings.Repeat(".", p.wide))
	}
	p.draw(false, func(x, y int) {
		if x < 0 || y < 0 || x >= p.wide || y >= p.tall {
			t.Fatalf("a dot at %d,%d is outside the %dx%d figure", x, y, p.wide, p.tall)
		}
		grid[y][x] = '#'
	})

	var rows int
	for _, row := range grid {
		if strings.ContainsRune(string(row), '#') {
			rows++
		}
	}
	t.Logf("the walking robot covers %d of its %d rows", rows, p.tall)
	if rows < p.tall/2 {
		t.Errorf("the figure covers %d of %d rows, want it filling its own box", rows, p.tall)
	}
}

// A drawn figure takes the slot the same way the geometry does: he walks on,
// the meters stand above and below him, and his face is in his head.
func TestADrawnFigureTakesTheSlot(t *testing.T) {
	const w, rows = 160, 46

	m := scopeModel(w, rows)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.5
	}
	m.scope.bands = bands

	// A bar that was dealt a drawn figure rather than the geometry.
	var found bool
	for bar := range int64(200) {
		m.words.starts = bar * 7_000
		if m.faceWho() != "" {
			found = true
			break
		}
	}
	if !found {
		t.Skip("no bar in two hundred was dealt a drawn figure")
	}
	t.Logf("bar %d was dealt %q", m.words.starts, m.faceWho())

	// And he is sent for, which is the only way he comes on: the bar decides
	// which of them it is, the key decides whether anybody is there at all.
	m.face.picked, m.face.shown = m.faceWho(), time.Now().Add(-faceShows/2)

	m.setProgress(time.Duration(m.words.starts)*time.Millisecond + faceEnters + m.faceStay()/2)

	art := m.figureLines(w, rows)
	if art == nil {
		t.Fatal("the figure drew nothing")
	}
	if len(art) != rows {
		t.Fatalf("the figure drew %d rows, want %d", len(art), rows)
	}
	for i, line := range art {
		if got := len([]rune(ansiOff(line))); got != w {
			t.Errorf("row %d is %d cells wide, want %d", i, got, w)
		}
	}

	// He is somewhere in the middle of it, with the meter above and below.
	var top, bottom, middle int
	for i, line := range art {
		if strings.TrimSpace(ansiOff(line)) == "" {
			continue
		}
		switch {
		case i < rows/4:
			top++
		case i > 3*rows/4:
			bottom++
		default:
			middle++
		}
	}
	t.Logf("%d rows lit at the top, %d in the middle, %d at the foot", top, middle, bottom)
	if top == 0 || bottom == 0 {
		t.Error("the meters are not standing above and below him")
	}
	if middle == 0 {
		t.Error("nothing is drawn where he is")
	}

	// And the pose follows what he is doing.
	if got := m.figurePose(); got == "" {
		t.Error("he is in no pose at all")
	} else {
		t.Logf("he is drawn as %q", got)
	}
}

// He does things rather than holding a sign. A still picture swapped in for a
// second is a figure showing you something; two or three in a row is somebody
// doing something, which is the reason he walked on.
func TestHisActsAreRunsOfDrawings(t *testing.T) {
	d, ok := figureFor("robot")
	if !ok {
		t.Fatal("no robot")
	}

	// Every act is made of poses the figure actually has, and every one of them
	// is more than a single held picture.
	for _, name := range figureActNames {
		act, ok := figureActs[name]
		if !ok {
			t.Fatalf("%q is dealt but has no drawings", name)
		}
		var seen int
		was := ""
		for _, f := range act {
			if _, ok := d.at(100, f.pose); !ok {
				t.Errorf("%q wants the pose %q, which the robot has not got", name, f.pose)
			}
			if f.pose != was {
				seen++
			}
			was = f.pose
		}
		t.Logf("%-6s %d frames, %d changes, %s", name, len(act), seen, figureActLong(name))
		if seen < 2 {
			t.Errorf("%q never changes drawing, which is a sign being held up", name)
		}
		if long := figureActLong(name); long < 300*time.Millisecond || long > 2*time.Second {
			t.Errorf("%q runs %s, want it long enough to read and short enough to be a turn", name, long)
		}
	}

	// And which one he does is the bar's business, so a record plays the same
	// way twice and a long visit is not the same trick over and over.
	seen := map[string]bool{}
	for turn := range 6 {
		seen[figureActFor("robot", 7_000, turn)] = true
	}
	t.Logf("over six turns he does %d different acts", len(seen))
	if len(seen) < 3 {
		t.Error("he does the same act every time in one visit")
	}
	once := figureActFor("robot", 12_345, 0)
	if again := figureActFor("robot", 12_345, 0); again != once {
		t.Errorf("one turn was dealt %q and then %q", once, again)
	}
}

// A figure drawn from the side is turned around when he goes the other way, and
// one drawn front on never is.
func TestASideOnFigureTurnsAround(t *testing.T) {
	var sided, front []string
	for name, d := range figures {
		if d.faces == 0 {
			front = append(front, name)
			continue
		}
		sided = append(sided, name)
	}
	sort.Strings(sided)
	sort.Strings(front)
	t.Logf("drawn from the side: %v; drawn front on: %v", sided, front)
	if len(sided) == 0 {
		t.Fatal("no figure is drawn from the side, so nothing is ever turned")
	}

	// Turning is a mirror, dot for dot.
	for _, name := range sided {
		d := figures[name]
		p, ok := d.at(100, "idle")
		if !ok {
			t.Fatalf("%s has no idle pose", name)
		}

		as, over := map[[2]int]bool{}, map[[2]int]bool{}
		p.draw(false, func(x, y int) { as[[2]int{x, y}] = true })
		p.draw(true, func(x, y int) { over[[2]int{x, y}] = true })

		if len(as) != len(over) {
			t.Errorf("%s draws %d dots as drawn and %d turned", name, len(as), len(over))
		}
		for at := range as {
			if !over[[2]int{p.wide - 1 - at[0], at[1]}] {
				t.Errorf("%s has a dot at %v that is not there when he is turned", name, at)
				break
			}
		}

		// And she is worth turning: a drawing the same both ways is one that
		// was never drawn from the side.
		var same int
		for at := range as {
			if over[at] {
				same++
			}
		}
		t.Logf("%s: %d dots, %d of them in the same place either way", name, len(as), same)
		if same > len(as)*3/4 {
			t.Errorf("%s is %d%% the same turned over, which is not a figure with a side to it",
				name, same*100/max(len(as), 1))
		}
	}

	// And the figure the whole screen is drawn from is turned to face the way
	// he is going, whichever way that is.
	d := figures[sided[0]]
	m := scopeModel(160, 46)
	if got := m.figureTurned(d); got == m.figureTurned(figures[front[0]]) && got {
		t.Error("a figure drawn front on was turned around with the rest")
	}
	if m.figureTurned(figures[front[0]]) {
		t.Error("a figure drawn front on must never be turned")
	}
}

// Nobody is asked for a pose they were never drawn in. The drawings come from
// different hands and carry different poses, so what a figure is dealt has to
// be dealt out of what that figure can do.
func TestEachFigureIsOnlyAskedForWhatItCanDo(t *testing.T) {
	for name := range figures {
		can := figureActsFor(name)
		t.Logf("%-9s %d acts: %v", name, len(can), can)
		if len(can) < 3 {
			t.Errorf("%s can do %d acts, which is not enough for a visit to be worth watching", name, len(can))
		}

		for _, act := range can {
			for _, f := range figureActs[act] {
				for _, size := range figures[name].sizes {
					if _, ok := size.poses[f.pose]; !ok {
						t.Errorf("%s is dealt %q at %d dots, which wants the pose %q it has not got",
							name, act, size.tall, f.pose)
					}
				}
			}
		}

		// And what is dealt is only ever one of them.
		for turn := range 40 {
			got := figureActFor(name, int64(turn)*7_000, turn)
			if !slices.Contains(can, got) {
				t.Errorf("%s was dealt %q, which is not one of the %d it can do", name, got, len(can))
			}
		}
	}

	if got := figureActFor("nobody at all", 7_000, 0); got != "" {
		t.Errorf("a figure that does not exist was dealt %q", got)
	}
}

// And the drawing follows the act while it runs.
func TestTheDrawingFollowsTheAct(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// Standing in the middle of a visit, where an act is done.
	for bar := range int64(60) {
		if at := bar * 7_000; true {
			m.words.starts = at
			break
		}
	}
	m.setProgress(time.Duration(m.words.starts)*time.Millisecond + faceEnters + m.faceStay()/2)

	m.face.act, m.face.actAt = "cheer", time.Now()
	if got := m.figurePose(); got != figureActs["cheer"][0].pose {
		t.Errorf("at the top of the act he is drawn as %q, want %q", got, figureActs["cheer"][0].pose)
	}

	// Past the end of it he is back to himself: standing about, or walking on
	// to the next place he stops at, but not still cheering.
	m.face.actAt = time.Now().Add(-figureActLong("cheer") - time.Second)
	got := m.figurePose()
	t.Logf("after the act he is drawn as %q", got)
	for _, f := range figureActs["cheer"] {
		if got == f.pose && f.pose != "idle" {
			t.Errorf("after the act he is still drawn as %q", got)
		}
	}

	// And an act nobody has heard of does not stop him being drawn.
	m.face.act = "nonesuch"
	if got := m.figurePose(); got == "" {
		t.Error("an unknown act left him with no drawing at all")
	}
}

// Every figure in the box is whole: it has the drawings the acts call for, at
// every size, and it says where it came from.
func TestEveryFigureIsWhole(t *testing.T) {
	if len(figures) < 2 {
		t.Fatal("there is only one figure — run go run ./cmd/spindle-figures")
	}

	names := make([]string, 0, len(figures))
	for name := range figures {
		names = append(names, name)
	}
	sort.Strings(names)
	t.Logf("%d figures: %v", len(names), names)

	for _, name := range names {
		d := figures[name]
		if d.from == "" || d.licence == "" {
			t.Errorf("%s does not say where it came from", name)
		}
		if len(d.sizes) < 2 {
			t.Errorf("%s comes in %d sizes, want one for a narrow terminal and one for a wide", name, len(d.sizes))
		}

		for _, size := range d.sizes {
			for _, act := range figureActsFor(name) {
				for _, f := range figureActs[act] {
					p, ok := size.poses[f.pose]
					if !ok {
						t.Errorf("%s at %d dots has no %q, which the %s act wants", name, size.tall, f.pose, act)
						continue
					}
					var lit int
					p.draw(false, func(_, _ int) { lit++ })
					if lit == 0 {
						t.Errorf("%s at %d dots draws nothing for %q", name, size.tall, f.pose)
					}
				}
			}
			for i := range 8 {
				if _, ok := size.poses["walk"+string(rune('0'+i))]; !ok {
					t.Errorf("%s at %d dots is missing walk frame %d", name, size.tall, i)
				}
			}
		}
	}

	// And they take turns: a record does not get the same one every time.
	m := scopeModel(160, 46)
	m.words.beats = true
	seen := map[string]int{}
	for bar := range int64(120) {
		m.words.starts = bar * 7_000
		seen[m.faceWho()]++
	}
	t.Logf("over a hundred and twenty bars: %v (the empty one is the geometry)", seen)
	if len(seen) < len(names)+1 {
		t.Errorf("only %d of the %d figures and the geometry ever came on", len(seen), len(names))
	}
	if seen[""] == 0 {
		t.Error("the one this code draws itself never came on")
	}
}

// He does not only walk on and off. He can gather out of specks and come apart
// into them, which is the one thing this screen can do that a cartoon cannot:
// every one of him is a dot, and a dot can be sent anywhere — or handed to the
// water.
func TestHeComesAndGoesTwoWays(t *testing.T) {
	d, _ := figureFor("robot")
	p, ok := d.at(100, "idle")
	if !ok {
		t.Fatal("no pose")
	}

	// At rest every way leaves him exactly where he was drawn.
	for way := range figureWays {
		var moved, lost int
		p.draw(false, func(x, y int) {
			nx, ny, burn, on := figureWarp(way, 1, x, y, p.wide, p.tall, 184)
			if !on {
				lost++
			} else if nx != x || ny != y || burn != 1 {
				moved++
			}
		})
		if moved != 0 || lost != 0 {
			t.Errorf("way %d moved %d dots and dropped %d with the movement over", way, moved, lost)
		}
	}

	// And part way through, every way but walking has moved him.
	for way := range figureWays {
		if figureSliding(way) {
			continue
		}
		var moved, lost, all int
		p.draw(false, func(x, y int) {
			all++
			nx, ny, _, on := figureWarp(way, 0.4, x, y, p.wide, p.tall, 184)
			switch {
			case !on:
				lost++
			case nx != x || ny != y:
				moved++
			}
		})
		t.Logf("way %d part way: %d of %d dots moved, %d gone", way, moved, all, lost)
		if moved+lost < all/5 {
			t.Errorf("way %d moved %d and dropped %d of %d dots, want it doing something", way, moved, lost, all)
		}
	}

	// A dot wanders the same way twice, so a record comes apart the same twice.
	once := figureSpeck(11, 7)
	if again := figureSpeck(11, 7); again != once {
		t.Errorf("a dot was given %d and then %d", once, again)
	}
	if figureSpeck(11, 7) == figureSpeck(12, 7) {
		t.Error("two dots were given the same number, so they will wander together")
	}

	// He always walks on. How he leaves is dealt: on his feet, or by coming
	// apart. What was thrown out was spinning, dropping in from over the top,
	// rising through the floor, bursting apart — those moved him about the
	// screen, which is what a sprite does — and gathering out of specks, which
	// however it was worked out read as a fade, and a fade is what happens to a
	// picture rather than to somebody.
	m := scopeModel(160, 46)
	m.words.beats = true
	out := map[figureWay]int{}
	for bar := range int64(60) {
		m.words.starts = bar * 7_000
		if m.figureComesBy() != figureWalks {
			t.Errorf("bar %d comes on some way other than on his feet", m.words.starts)
		}
		out[m.figureGoesBy()]++
	}
	t.Logf("over sixty visits he leaves: %v", out)

	for _, way := range []figureWay{figureWalks, figureCrumbles} {
		if out[way] == 0 {
			t.Errorf("he never left by way %d", way)
		}
	}
}

// He is drawn under everything else on the screen — under the water and the
// sparks, never mind the type. A figure at full strength is a cut-out laid over
// the music; one drawn beneath it is in the room with it, and the eye goes to
// him because he moves rather than because he is the brightest thing there.
func TestHeIsFainterThanTheSparks(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// Loud, so the meter and its water are burning as brightly as they can.
	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 1
	}
	m.scope.bands = bands
	m.scope.envelope = 1

	for bar := range int64(60) {
		if at := bar * 7_000; faceWhoFor(at) != "" {
			m.words.starts = at
			break
		}
	}
	m.setProgress(time.Duration(m.words.starts)*time.Millisecond + faceEnters + m.faceStay()/2)

	// What the meter is given against what he is given, out of the same palette.
	levels := len(m.styles.Words[0])
	meter := m.wordsBeatPaint(0, int(faceParts_), len(m.styles.Words), levels).level
	his := int8(float32(meter) * figureBurn)
	t.Logf("standing still he burns at %d of %d, where the meter beside him burns at %d", his, levels-1, meter)

	if figureBurn >= 1 {
		t.Error("he is drawn at full strength, want him under the rest of the picture")
	}
	if his >= meter && meter > 0 {
		t.Errorf("he burns at %d and the meter at %d, want him the fainter of the two", his, meter)
	}

	// And fainter still while he is coming or going.
	if figureFaint >= figureBurn {
		t.Errorf("he arrives at %.2f and stands at %.2f, want the arrival the fainter", figureFaint, figureBurn)
	}
}

// He comes apart a speck at a time. Not evenly — the top goes first, the way a
// wall does, and each dot on its own moment, because a shape that comes apart
// in rows is a shape being wiped.
func TestHeComesApartOutOfSpecks(t *testing.T) {
	d, _ := figureFor("robot")
	p, _ := d.at(100, "idle")

	there := func(at float64) int {
		var n int
		p.draw(false, func(x, y int) {
			if _, _, _, on := figureWarp(figureCrumbles, at, x, y, p.wide, p.tall, 184); on {
				n++
			}
		})
		return n
	}

	early, half, whole := there(0.1), there(0.5), there(1)
	t.Logf("%d dots a tenth of the way through coming apart, %d half way, %d standing", early, half, whole)

	if early >= half || half >= whole {
		t.Errorf("it went %d → %d → %d, want him coming apart", early, half, whole)
	}
	if early > whole/3 {
		t.Errorf("%d of %d dots are still up a tenth of the way through", early, whole)
	}

	// Walking on and off moves nothing: he is off the side of the screen, and a
	// figure being walked on is not a figure being drawn on.
	var moved int
	p.draw(false, func(x, y int) {
		if nx, ny, burn, on := figureWarp(figureWalks, 0.3, x, y, p.wide, p.tall, 184); !on || nx != x || ny != y || burn != 1 {
			moved++
		}
	})
	if moved != 0 {
		t.Errorf("walking on moved %d dots, want him whole and on his feet", moved)
	}
}

// And when he comes apart, the pieces are not drawn by this code at all: they
// are thrown into the water the meter throws, and from then on they arc, fall
// and fade on the same physics — which is what makes it read as sparks rather
// than as a figure being dimmed.
func TestComingApartHandsHimToTheWater(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.6
	}
	m.scope.bands = bands

	// A visit that ends by coming apart.
	var found bool
	for bar := range int64(400) {
		m.words.starts = bar * 7_000
		if m.faceWho() != "" && m.figureGoesBy() == figureCrumbles {
			found = true
			break
		}
	}
	if !found {
		t.Skip("no crumbling exit in four hundred bars")
	}

	base := time.Duration(m.words.starts) * time.Millisecond
	var most int
	for step := range 80 {
		m.setProgress(base + faceEnters + time.Duration(float64(step)/79*float64(m.faceStay())))
		m.faceFlow()
		m.figureSpray(m.width, m.height)
		most = max(most, len(m.stage.drops))
	}
	t.Logf("coming apart threw %d drops into the water", most)

	if most == 0 {
		t.Error("he came apart and the water got nothing")
	}
	if most > stageDrops {
		t.Errorf("he threw %d drops, past the %d the water holds", most, stageDrops)
	}

	// And a visit that walks off quietly throws nothing.
	m.stage.drops = nil
	for bar := range int64(400) {
		m.words.starts = bar * 7_000
		if m.faceWho() != "" && m.figureGoesBy() == figureWalks {
			break
		}
	}
	base = time.Duration(m.words.starts) * time.Millisecond
	for step := range 80 {
		m.setProgress(base + faceEnters + time.Duration(float64(step)/79*float64(m.faceStay())))
		m.faceFlow()
		m.figureSpray(m.width, m.height)
	}
	if len(m.stage.drops) != 0 {
		t.Errorf("walking off threw %d drops, want him leaving on his feet", len(m.stage.drops))
	}
}

// When he walks on while a bar of marks is up, he walks into them. What he
// reaches comes apart and goes into the water — the same water the meter
// throws — and what he has knocked over stays knocked over while he wanders.
//
// It is the one place two of this screen's machines touch each other, and the
// only reason he shares a frame with anything.
func TestHeWalksThroughTheMarks(t *testing.T) {
	const w, rows = 100, 30
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	m := scopeModel(w, rows)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.6
	}
	m.scope.bands = bands

	// A visit that comes on from the side, with a row of marks up.
	var found bool
	for bar := range int64(400) {
		m.words.starts = bar * 7_000
		if m.faceWho() != "" && m.figureComesBy() == figureWalks {
			found = true
			break
		}
	}
	if !found {
		t.Skip("no walking visit in four hundred bars")
	}

	line := wordsMarks(dotsX, dotsY)
	img, layout, ok := wordsImage([]string{line}, dotsX, dotsY)
	if !ok {
		t.Fatal("the row would not draw")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.where, m.words.beats, m.words.text = layout, true, line
	m.face.sweptLow, m.face.sweptHigh = figureUnswept, -figureUnswept

	if !m.figureSweeps() {
		t.Fatal("he walked on over a row of marks and did not walk into it")
	}

	whole := func() int {
		low, high, from := m.figureSwept()
		var n int
		for piece := range layout.Count {
			cx, _ := layout.Middle(piece)
			if figureBroken(low, high, from, cx, dotsX/layout.Count) < 1 {
				n++
			}
		}
		return n
	}

	base := time.Duration(m.words.starts) * time.Millisecond
	was, drops := layout.Count, 0
	for step := range 60 {
		m.setProgress(base + faceEnters + time.Duration(float64(step)/59*float64(m.faceStay())))
		m.faceFlow()
		m.figureSweep(w, rows)

		if now := whole(); now > was {
			t.Errorf("step %d: %d marks are standing where %d were, want them staying down", step, now, was)
		} else {
			was = now
		}
		drops = max(drops, len(m.stage.drops))
	}
	t.Logf("he left %d of %d marks standing and threw %d drops into the water", was, layout.Count, drops)

	if was == layout.Count {
		t.Error("he walked the whole way and knocked nothing over")
	}
	if drops == 0 {
		t.Error("what he knocked over never reached the water")
	}

	// And a bar with no marks up is a bar with nothing to walk into.
	m.words.beats = false
	if m.figureSweeps() {
		t.Error("he walked into a row of marks that was not there")
	}
}

// He finishes the row. Every mark he walked on to walk through is down by the
// time he leaves — not most of them, and not all but the one on the end.
//
// It used to be all but the ends. How far past a mark he had got was measured
// against the nearer end of the stretch he had walked, so a mark at either end
// of that stretch was never past him by much, however far he went; and on the
// visits where he turned back the way he came, the whole far side of the row
// was never walked at all.
func TestHeLeavesNoMarkStanding(t *testing.T) {
	const w, rows = 160, 46
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	line := wordsMarks(dotsX, dotsY)
	img, layout, ok := wordsImage([]string{line}, dotsX, dotsY)
	if !ok {
		t.Fatal("the row would not draw")
	}
	grain := cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	span := dotsX / max(layout.Count, 1)

	stood, visits := map[int]int{}, 0
	for bar := range int64(150) {
		m := scopeModel(w, rows)
		m.stage.on = true
		m.scope.modes[tabPlayer] = scopeWords
		m.ps.Duration = 90 * time.Minute
		m.words.starts = bar * 7_000
		if m.faceWho() == "" {
			continue
		}
		m.words.have, m.words.where = grain, layout
		m.words.beats, m.words.text = true, line
		m.face.sweptLow, m.face.sweptHigh = figureUnswept, -figureUnswept
		visits++

		base := time.Duration(m.words.starts) * time.Millisecond
		for step := range 120 {
			m.setProgress(base + faceEnters + time.Duration(float64(step)/119*float64(m.faceStay())))
			m.figureSweep(w, rows)
		}

		low, high, from := m.figureSwept()
		for piece := range layout.Count {
			cx, _ := layout.Middle(piece)
			if figureBroken(low, high, from, cx, span) < 1 {
				stood[piece]++
			}
		}
	}

	t.Logf("over %d visits through a row of %d marks, what was left standing: %v", visits, layout.Count, stood)
	if visits == 0 {
		t.Fatal("no visit in a hundred and fifty bars walked through the marks")
	}
	if len(stood) > 0 {
		t.Errorf("marks were left standing after he had gone: %v", stood)
	}
}

// The marks he has not reached yet go on riding the music.
//
// He walks into the row; he does not replace it. Drawn still, a whole row of
// them went from riding the sound to standing to attention on the frame he
// arrived — which read as the picture breaking rather than as somebody
// arriving.
func TestTheMarksGoOnRidingWhileHeWalksIn(t *testing.T) {
	const w, rows = 160, 46
	dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY

	m := scopeModel(w, rows)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute

	var found bool
	for bar := range int64(400) {
		m.words.starts = bar * 7_000
		if m.faceWho() != "" && m.figureComesBy() == figureWalks {
			found = true
			break
		}
	}
	if !found {
		t.Skip("no walking visit in four hundred bars")
	}

	line := wordsMarks(dotsX, dotsY)
	img, layout, ok := wordsImage([]string{line}, dotsX, dotsY)
	if !ok {
		t.Fatal("the row would not draw")
	}
	m.words.have = cover.Grind(grayToImage(img), w, rows, dotsPerCellX, dotsPerCellY)
	m.words.where, m.words.beats, m.words.text = layout, true, line
	m.face.sweptLow, m.face.sweptHigh = figureUnswept, -figureUnswept
	m.setProgress(time.Duration(m.words.starts)*time.Millisecond + faceEnters)

	// The top of the row as he draws it, with the music loud and with it silent.
	top := func(loud float32) int {
		bands := make([]float32, 28)
		for i := range bands {
			bands[i] = loud
		}
		m.scope.bands = bands

		high := 1 << 30
		m.figureThrough(w, rows, func(_, y, _ int, _ float32) { high = min(high, y) })
		return high
	}

	still, riding := top(0), top(0.9)
	t.Logf("the row he is walking into stands at %d silent and %d loud", still, riding)
	if riding >= still {
		t.Errorf("the row is at %d loud and %d silent, want the music lifting it", riding, still)
	}
}

// A figure that gets about by hopping has to be lifted off the floor, because
// the drawing will not do it: a rabbit in mid air over the same patch of ground
// is a rabbit standing on its toes.
func TestAHopperLeavesTheGround(t *testing.T) {
	bunny, ok := figureFor("bunny")
	if !ok {
		t.Fatal("no bunny — run go run ./cmd/spindle-figures")
	}
	robot, _ := figureFor("robot")

	t.Logf("the bunny rises %d dots as it goes, the robot %d", bunny.bob, robot.bob)
	if bunny.bob <= robot.bob {
		t.Errorf("the bunny rises %d and the robot %d, want the hopper the higher", bunny.bob, robot.bob)
	}

	// And it is on the screen: the top of him moves as he goes.
	//
	// With the spectrum silenced, because the meter hangs from the ceiling and
	// the top row of a loud picture is the meter rather than the figure. What is
	// being measured here is him.
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.scope.bands = nil
	m.words.beats, m.words.text = true, wordsNotes

	// Long enough to hold every bar looked at: the clock the visit is timed
	// against stops at the end of the track, and a visit dealt past the end of
	// one never comes on at all.
	m.ps.Duration = 90 * time.Minute

	for bar := range int64(600) {
		m.words.starts = bar * 7_000
		if m.faceWho() == "bunny" && m.figureComesBy() == figureWalks {
			break
		}
	}
	if m.faceWho() != "bunny" {
		t.Skip("no bunny visit in six hundred bars")
	}

	// And it happens as he crosses: over the middle of the visit, where he is
	// all the way on and on the move, the rabbit leaves the floor and the robot
	// hardly does.
	base := time.Duration(m.words.starts) * time.Millisecond
	hop, walk, moved := 0, 0, false
	for step := range 60 {
		gone := 0.25 + 0.5*float64(step)/59
		m.setProgress(base + faceEnters + time.Duration(gone*float64(m.faceStay())))
		if _, moving := m.faceGoing(); !moving {
			continue
		}
		moved = true
		hop, walk = max(hop, m.figureLift(bunny)), max(walk, m.figureLift(robot))
	}
	t.Logf("crossing the screen the rabbit rose %d dots, the robot %d", hop, walk)
	if !moved {
		t.Fatal("he never moved, so there was nothing to measure")
	}
	if hop < bunny.bob/2 {
		t.Errorf("the rabbit rose %d dots of its %d, want it off the ground", hop, bunny.bob)
	}
	if hop <= walk {
		t.Errorf("the rabbit rose %d and the robot %d, want the hopper the higher", hop, walk)
	}

	// Standing still he is on the floor: something bobbing on the spot is being
	// animated at rather than getting anywhere.
	for step := range 60 {
		m.setProgress(base + faceEnters + time.Duration(float64(step)/59*float64(m.faceStay())))
		if _, moving := m.faceGoing(); !moving && m.figureLift(bunny) != 0 {
			t.Errorf("standing still he is %d dots off the ground", m.figureLift(bunny))
			break
		}
	}
}

// One visit in a few he crosses the screen backwards. Nobody drew a moonwalk:
// it is the walk played the other way round while the ground goes on going the
// way it was going, which is what the thing actually is.
func TestSometimesHeMoonwalks(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	var moon int
	for bar := range int64(120) {
		m.words.starts = bar * 7_000
		if m.figureMoonwalks() {
			moon++
		}
	}
	t.Logf("%d of a hundred and twenty visits cross backwards", moon)
	if moon == 0 || moon > 40 {
		t.Errorf("%d of a hundred and twenty moonwalk, want it rare and not never", moon)
	}

	// The frames run the other way, and he faces the other way with them.
	walking, backwards := "", ""
	for bar := range int64(600) {
		m.words.starts = bar * 7_000
		if m.faceWho() == "" {
			continue
		}
		m.setProgress(time.Duration(m.words.starts)*time.Millisecond + faceEnters + time.Duration(faceWalkIn/2*float64(m.faceStay())))
		if _, moving := m.faceGoing(); !moving {
			continue
		}
		if m.figureMoonwalks() && backwards == "" {
			backwards = m.figurePose()
		}
		if !m.figureMoonwalks() && walking == "" {
			walking = m.figurePose()
		}
		if walking != "" && backwards != "" {
			break
		}
	}
	t.Logf("walking he is drawn as %q, moonwalking as %q", walking, backwards)
	if walking == "" || backwards == "" {
		t.Skip("did not find one of each")
	}
	if walking == backwards {
		t.Error("the moonwalk is drawn exactly like the walk")
	}
}

// Whoever came on is who is on, for the whole of the visit.
//
// The key's own window and the length of a visit are two different clocks, and
// the visit is the longer of them. Asked afresh once the key's window had
// closed, the screen handed back whoever the bar would have dealt — so a drawn
// figure walked off and the one this code draws itself waved his way in from the
// side to finish somebody else's visit. Nobody had asked for him, and from the
// room it read as a second figure arriving.
func TestOneVisitIsOneFigure(t *testing.T) {
	m := scopeModel(160, 46)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 6 * time.Minute
	m.words.beats, m.words.text = true, wordsNotes

	// A bar the deal would give to the one this code draws itself, and a drawn
	// figure sent for over the top of it — which is the pair that used to swap.
	var bar int64 = -1
	for at := range int64(400) {
		if faceWhoFor(at*7_000) == "" {
			bar = at * 7_000
			break
		}
	}
	if bar < 0 {
		t.Skip("no bar in four hundred goes to the geometry")
	}
	m.words.starts = bar
	m.setProgress(time.Duration(bar) * time.Millisecond)

	m.faceShow()
	who := m.faceWho()
	if who == "" {
		t.Skip("the key walked to the geometry as well, so there is nothing to tell apart")
	}
	m.faceFlow()
	t.Logf("the bar would have dealt the geometry; the key brought %q, and the visit runs %s against the key's %s",
		who, m.faceStayFor(bar), faceShows)

	// The key's window closes while the visit is still running.
	m.face.shown = time.Now().Add(-faceShows - time.Second)
	if !m.faceUp() {
		t.Fatal("the visit ended with the key's window, so there is nothing to test")
	}
	if got := m.faceWho(); got != who {
		t.Errorf("the key's window closed and the figure changed from %q to %q mid-visit", who, got)
	}

	// And once the visit is over, the screen is nobody's again.
	m.face.came = time.Now().Add(-m.faceStayFor(bar) - time.Second)
	if m.faceUp() {
		t.Error("the visit ran past its own stay")
	}
}
