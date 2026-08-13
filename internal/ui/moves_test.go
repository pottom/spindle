package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// The baked set is what the manifest asked for: every move there, every frame
// drawn, and every one of them standing in the cell it was cut from.
func TestTheDanceIsBakedWhole(t *testing.T) {
	set, ok := moveSetFor("break")
	if !ok {
		t.Fatal("the break set was not baked")
	}
	if len(set.sizes) == 0 {
		t.Fatal("the set came out with no sizes")
	}

	for _, size := range set.sizes {
		if size.wide <= 0 || size.tall <= 0 {
			t.Errorf("a size came out %dx%d", size.wide, size.tall)
		}
		for _, name := range set.names() {
			d, ok := size.moves[name]
			if !ok {
				t.Errorf("%q is missing at %d dots", name, size.tall)
				continue
			}
			if len(d.frames) == 0 {
				t.Errorf("%q has no frames at %d dots", name, size.tall)
				continue
			}
			for i, f := range d.frames {
				switch {
				case f.wide <= 0 || f.tall <= 0:
					t.Errorf("%q frame %d came out %dx%d", name, i, f.wide, f.tall)
				case f.x < 0 || f.x+f.wide > size.wide:
					t.Errorf("%q frame %d stands from %d to %d, outside a cell %d wide",
						name, i, f.x, f.x+f.wide, size.wide)
				case f.tall > size.tall*2:
					t.Errorf("%q frame %d is %d dots tall where the standing pose is %d — the scale slipped",
						name, i, f.tall, size.tall)
				}
			}
		}
	}
}

// The standing pose is the height the set was baked to. It is the one frame
// whose size is not a matter of what he is doing, and the whole sheet is scaled
// by whatever it took, so if this is wrong everything is.
func TestTheStandingPoseIsTheHeightItWasBakedTo(t *testing.T) {
	set, _ := moveSetFor("break")
	for _, size := range set.sizes {
		for _, name := range set.names() {
			d := size.moves[name]
			if len(d.frames) == 0 {
				continue
			}
			// To the dot, give or take one: an outline resampled loses the
			// faintest row of its own edge, and the baker corrects for that by
			// measuring rather than by fudging — but the correction lands on
			// whichever side of the boundary the resampler leaves it.
			if got := d.frames[0].tall; got < size.tall-1 || got > size.tall {
				t.Errorf("%q begins %d dots tall at a size baked to %d", name, got, size.tall)
			}
		}
	}
}

// A move lasts as long as it is asked to: the entry and the exit are what they
// are, and the rounds are where the time goes.
func TestAMoveLastsAsLongAsItIsAskedTo(t *testing.T) {
	set, _ := moveSetFor("break")
	size, d, ok := set.at(90, "sixstep")
	if !ok {
		t.Fatal("the sixstep is not in the set")
	}
	_ = size

	loop := d.span(d.loopFrom, d.loopTo)
	if loop < 4 {
		t.Fatalf("the loop is %d frames, which is not a loop", loop)
	}
	if one, two := d.steps(1), d.steps(2); two-one != loop {
		t.Errorf("a second round added %d frames, want the loop's %d", two-one, loop)
	}

	// It runs from the first frame to the last and then stops, and every step
	// along the way has a drawing.
	for step := range d.steps(3) {
		if _, going := d.frameAt(step, 3); !going {
			t.Fatalf("the move stopped at step %d of %d", step, d.steps(3))
		}
	}
	if _, going := d.frameAt(d.steps(3), 3); going {
		t.Error("the move was still going after its last frame")
	}
}

// The loop is a loop: gone round twice, it comes back to the frame it started
// on rather than walking off the end of the sheet.
func TestTheLoopComesRound(t *testing.T) {
	set, _ := moveSetFor("break")
	_, d, _ := set.at(90, "backspin")

	in := d.span(d.inFrom, d.inTo)
	loop := d.span(d.loopFrom, d.loopTo)

	first, _ := d.frameAt(in, 2)
	round, _ := d.frameAt(in+loop, 2)
	if first != round {
		t.Error("a round of the loop did not come back to the frame it began on")
	}
}

// A move with nothing to go into is all loop, which is what the bounce is: he
// does it standing, and he is doing it whenever nothing else is going on.
func TestTheBounceIsAllLoop(t *testing.T) {
	set, _ := moveSetFor("break")
	_, d, ok := set.at(90, "bounce")
	if !ok {
		t.Fatal("the bounce is not in the set")
	}
	if got := d.span(d.inFrom, d.inTo) + d.span(d.outFrom, d.outTo); got != 0 {
		t.Errorf("the bounce has %d frames of entering and leaving, want none", got)
	}
	if d.span(d.loopFrom, d.loopTo) < 12 {
		t.Errorf("the bounce loops over %d frames, want the whole sheet", d.span(d.loopFrom, d.loopTo))
	}
}

// The bar is dealt between the marks and the dancer, and dealt the same way
// twice: a record that puts him up at two minutes puts him up at two minutes
// every time it is played.
func TestTheDancerIsDealtTheSameWayTwice(t *testing.T) {
	const record = "4AUqttoxyPsm7wchxAuk3G"

	var his int
	for bar := range 90 {
		at := int64(bar) * wordsSpell.Milliseconds()
		if danceCastFor(record, at) != danceCastFor(record, at) {
			t.Fatalf("the bar at %d was dealt differently twice in a row", at)
		}
		if danceCastFor(record, at) {
			his++
		}
	}

	// About one in three, which is what the deal is for: often enough to be
	// seen on a wordless record, rare enough to still be an event.
	if his < 15 || his > 45 {
		t.Errorf("the dancer took %d of 90 bars, want about a third", his)
	}

	// And a different record dances differently.
	same := 0
	for bar := range 90 {
		at := int64(bar) * wordsSpell.Milliseconds()
		if danceCastFor(record, at) == danceCastFor("70t03HmXqfKmqEzWNvCXVv", at) {
			same++
		}
	}
	if same == 90 {
		t.Error("two records were given the same bars, so the deal is not the record's")
	}
}

// He dances to the record: the frames step off the beat, so the same move takes
// a bar whether the record is slow or fast, and he holds still where there is no
// beat to keep.
func TestTheDanceStepsOnTheBeat(t *testing.T) {
	m := stageModel(120, 44)
	m.stage.mode = scopeWords
	m.words.dancing, m.dance.move, m.dance.rounds = true, "bounce", 4
	m.dance.since = time.Now().Add(-time.Second)

	if !m.danceUp() {
		t.Fatal("the dancer is not up")
	}

	// With no beat to keep, he stands where he is rather than dancing to a
	// clock of his own.
	m.stage.loose = false
	if got := m.danceAt(); got != 0 {
		t.Errorf("with the beat off he had gone %.1f keyframes, want none", got)
	}

	// And with one, a second of a 120 bpm record is half a bar: half of the
	// sixteen keyframes a turn is written in.
	m.stage.loose = true
	m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
	m.scope.beatAt = time.Now()
	slow := m.danceAt()
	if slow < 6 || slow > 10 {
		t.Errorf("a second of a 120 bpm record took him %.1f keyframes, want about eight", slow)
	}

	// Twice the tempo, twice as far through the move in the same second: he is
	// dancing to the record rather than beside it.
	m.scope.beat = player.Beat{Period: 250 * time.Millisecond}
	if fast := m.danceAt(); fast < 2*slow-2 || fast > 2*slow+2 {
		t.Errorf("at twice the tempo he had gone %.1f keyframes, want about %.1f", fast, 2*slow)
	}
}

// The loud moves belong to the loud passages. Not a rule — a lean — so this
// asks for the lean rather than for any particular deal.
func TestTheBigMovesWantALoudPassage(t *testing.T) {
	big := map[string]bool{"backspin": true, "headstand": true, "sixstep": true}
	if len(danceNames()) < 2 {
		t.Skip("only one move is written as numbers, so there is nothing to lean between")
	}

	// The swell is read off the record's own loudness against the range it has
	// been moving through, so a hush and a chorus are set by moving the reading
	// rather than by writing the answer down. See swell.go.
	count := func(loud float64, drive float32) int {
		var n int
		for bar := range 200 {
			m := stageModel(120, 44)
			m.words.drive = drive
			m.words.swellLow, m.words.swellHigh = -30, -10
			m.scope.beat = player.Beat{Period: 500 * time.Millisecond, Loud: loud}
			m.danceDeal("a record", int64(bar)*1000, 0)
			if big[m.dance.move] {
				n++
			}
		}
		return n
	}

	quiet, loud := count(-30, 0), count(-10, 1)
	t.Logf("of 200 bars, the big moves came up %d times in a hush and %d at full tilt", quiet, loud)
	if loud <= quiet {
		t.Errorf("the big moves came up %d times in a hush and %d at full tilt, want more when the record is giving", quiet, loud)
	}
}

// He arrives once and then dances. The bar he stands in is stamped afresh every
// time it comes round, and a picture adopted every frame would fly apart and
// gather back sixty times a second — which is what happened to the marks once,
// and is written up in wordsComing.
func TestTheDancerArrivesOnce(t *testing.T) {
	m := stageModel(120, 44)
	m.stage.mode = scopeWords
	m.lyrics.synced = true
	m.words.forTrack = m.ps.TrackID

	// A bar that is his, found rather than assumed.
	spell := -1
	for at := range 200 {
		if danceCastFor(m.ps.TrackID, int64(at)*wordsSpell.Milliseconds()) {
			spell = at
			break
		}
	}
	if spell < 0 {
		t.Skip("no bar in two hundred fell to the dancer")
	}
	m.setProgress(time.Duration(spell)*wordsSpell + time.Second)

	if cmd := m.wordsGrind(); cmd != nil {
		t.Error("the dancer sent for a picture instead of drawing himself")
	}
	if !m.danceUp() {
		t.Fatal("the bar was his and he is not up")
	}
	came := m.words.since

	// Every frame after that leaves him where he is: same picture, same
	// arrival, no second gathering.
	for range 30 {
		m.wordsGrind()
	}
	if !m.words.since.Equal(came) {
		t.Error("he arrived again while he was already dancing")
	}
	if !m.danceUp() {
		t.Error("he was taken down while the bar was still his")
	}
}
