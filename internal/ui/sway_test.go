package ui

import (
	"math"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// swayModel is a big screen with a bar of marks up over a record with a beat.
func swayModel(t *testing.T) Model {
	t.Helper()

	m := scopeModel(120, 44)
	m.stage.on = true
	m.stage.mode = scopeWords
	m.words.beats, m.words.text = true, wordsNotes
	m.scope.bands = make([]float32, 28)

	m.scope.beat = player.Beat{Period: 500 * time.Millisecond}
	m.scope.beatAt = time.Now()
	if !m.beatKeeping() {
		t.Fatal("a beat was given and the screen did not keep time")
	}
	return m
}

// kick puts a run of frames through the model: a low end that hits every beat,
// or one that has gone quiet, and reports where the sway ends up.
func kick(m *Model, seconds float64, hit float32) float32 {
	const fps = 30
	frames := int(seconds * fps)
	beat := int(0.5 * fps) // the half-second beat swayModel keeps

	for f := range frames {
		var low float32 = 0.7
		if f%beat < 2 {
			low += hit // the strike, over two frames, as a kick reads on the meter
		}
		for i := range m.scope.bands {
			m.scope.bands[i] = low
		}
		m.swayFlow()
	}
	return m.words.drive
}

// The row sways to what is being played, not to what was found.
//
// A beat is worked out over twelve seconds of listening, so it goes on being
// reported through a passage where nobody is striking it — the drums drop out
// for a few bars and the row goes on swaying to a beat that is not there. What
// tells the difference is how hard the low end jumps: measured over ninety
// seconds of a record, the biggest rise in a second ran 0.30 to 0.37 where the
// kick was playing and 0.09 to 0.12 where it was not, while the level of the
// bass said 0.6 to 0.8 throughout and told nobody anything.
func TestTheRowSwaysToWhatIsBeingPlayed(t *testing.T) {
	m := swayModel(t)

	playing := kick(&m, 8, 0.30)
	t.Logf("with the kick playing the row sways at %.2f of its travel", playing)
	if playing < 0.7 {
		t.Errorf("the kick was playing and the row swayed at %.2f, want most of its travel", playing)
	}

	// The same record, the same beat still reported, and the low end no longer
	// being struck.
	quiet := kick(&m, 8, 0.09)
	t.Logf("with the low end down to a third of that it sways at %.2f", quiet)
	if quiet > playing*0.6 {
		t.Errorf("the kick stopped and the row still sways at %.2f against %.2f, want it easing off", quiet, playing)
	}

	// And it comes back rather than staying down.
	back := kick(&m, 8, 0.30)
	t.Logf("and when the kick comes back, %.2f", back)
	if back < playing*0.8 {
		t.Errorf("the kick came back and the row sways at %.2f against the %.2f it had, want it back", back, playing)
	}
}

// A record with no low end at all is not a record with a very quiet beat: the
// scale the hitting is measured against has a floor under it, or a hush would
// be measured against itself and called a kick.
func TestAHushIsNotABeat(t *testing.T) {
	m := swayModel(t)

	hush := kick(&m, 10, 0.01)
	t.Logf("with the low end never rising by more than a hundredth, the row sways at %.2f", hush)
	if hush > 0.3 {
		t.Errorf("a record with no low end swayed at %.2f, want it nearly still", hush)
	}
}

// The sway is a pendulum, not a hit: furthest over exactly on the beat, upright
// halfway between two, and moving fastest where it passes through. Nothing here
// jumps — the picture that keeps time must not be the thing that breaks the
// movement.
func TestTheSwayIsAPendulum(t *testing.T) {
	m := swayModel(t)
	kick(&m, 6, 0.30) // give it something to sway to

	at := func(phase float64) float32 {
		m.scope.beat.Since = time.Duration(phase * float64(m.scope.beat.Period))
		m.scope.beatAt = time.Now()
		lean, ok := m.wordsSway()
		if !ok {
			t.Fatal("the row is not swaying")
		}
		return lean
	}

	on, quarter, half := at(0), at(0.25), at(0.5)
	t.Logf("the row leans %+.3f on the beat, %+.3f a quarter in, %+.3f halfway", on, quarter, half)

	if math.Abs(float64(quarter)) > math.Abs(float64(on)) {
		t.Errorf("a quarter of the way through the beat it leans %+.3f against %+.3f on it, want the beat furthest over", quarter, on)
	}
	if math.Abs(float64(half)) > 0.02 {
		t.Errorf("halfway between two beats it leans %+.3f, want it upright", half)
	}

	// Frame to frame it moves smoothly: the biggest step over a whole beat is a
	// small share of the travel.
	var worst float64
	was := at(0)
	for f := 1; f < 30; f++ {
		now := at(float64(f) / 30)
		worst = math.Max(worst, math.Abs(float64(now-was)))
		was = now
	}
	t.Logf("over a beat at thirty frames a second the lean never moved more than %.4f in one frame", worst)
	if worst > wordsSwayMost/4 {
		t.Errorf("the lean moved %.4f in one frame against a travel of %.2f, want it swinging rather than snapping", worst, wordsSwayMost)
	}
}

// The same beat, and not the same way twice running: which way each mark leans
// is dealt from the bar. One way is right some of the time and wrong the rest,
// which is the argument for dealing it — and whichever is dealt, every mark in
// the row is leaning. A lone thing moving on a still row was tried and thrown
// out.
func TestTheRowLeansEveryWay(t *testing.T) {
	m := swayModel(t)
	kick(&m, 6, 0.30)

	seen := map[swayFigure]int{}
	for starts := int64(0); starts < 4000; starts += 10 {
		seen[swayFor(starts)]++
	}
	t.Logf("over 400 bars the four figures came up %v", seen)
	for fig := swayTogether; fig < swayFigures; fig++ {
		if seen[fig] == 0 {
			t.Errorf("figure %d was never dealt", fig)
		}
	}

	// What each of them does to a row of eight, on the beat.
	const count = 8
	names := []string{"together", "facing", "alternating", "trailing"}
	for fig := swayTogether; fig < swayFigures; fig++ {
		var bar int64 = -1
		for starts := int64(0); starts < 10_000; starts += 10 {
			if swayFor(starts) == fig {
				bar = starts
				break
			}
		}
		if bar < 0 {
			t.Fatalf("no bar was dealt figure %d", fig)
		}

		m.words.starts = bar
		m.setProgress(time.Duration(bar) * time.Millisecond)
		m.scope.beat.Since = 0
		m.scope.beatAt = time.Now()

		lean, ok := m.wordsSwaying(count)
		if !ok {
			t.Fatalf("%s: the row was not swaying", names[fig])
		}
		t.Logf("%-11s %v", names[fig], lean)

		// Nobody stands upright while the rest lean: whatever the figure, the
		// row is one body doing something.
		for i, at := range lean {
			if at > -0.01 && at < 0.01 {
				// The middle of a row leaning at itself is allowed to be
				// upright — it is the hinge.
				if fig == swayFacing && i*2 >= count-2 && i*2 <= count {
					continue
				}
				t.Errorf("%s: mark %d stands at %+.3f while the row leans", names[fig], i, at)
			}
		}
	}
}

// The picture grows with the record.
//
// Nothing in the spectrum can say whether a record is building — the daemon
// scales the bands to their own recent loudness, so a band reads the same in a
// hush as in a chorus, and measured over ninety seconds the mean of them moved
// between 0.50 and 0.60 while the quietest stretch sat at the top of the range.
// What can say it is the scale itself, in decibels, which the daemon now hands
// out beside the bands.
func TestThePictureGrowsWithTheRecord(t *testing.T) {
	m := scopeModel(120, 44)
	m.scope.bands = make([]float32, 28)

	// A record that has been between two loudnesses lately.
	at := func(db float64) float32 {
		m.scope.beat.Loud = db
		m.swellFlow()
		return m.swell()
	}

	// Open the range over the two ends, the way a record does over a minute.
	for range 100 {
		at(-30)
		at(-12)
	}

	quiet, loud, middle := at(-30), at(-12), at(-21)
	t.Logf("over a record that has run from -30 to -12 dB, a word rides %.1f dots where it is quietest, %.1f in the middle and %.1f where it is loudest",
		quiet*wordsWordRide, middle*wordsWordRide, loud*wordsWordRide)

	if quiet >= middle || middle >= loud {
		t.Errorf("the picture stands at %.2f quiet, %.2f middling and %.2f loud, want it growing with the record", quiet, middle, loud)
	}
	if loud/quiet < 2 {
		t.Errorf("the loudest passage moves %.2f and the quietest %.2f, want the difference worth seeing", loud, quiet)
	}
	if loud <= 1 {
		t.Errorf("at its loudest the record throws the picture %.2f of its travel, want a build to go further than its resting size", loud)
	}

	// And with nothing heard yet, everything is at its resting size rather than
	// at nothing.
	fresh := scopeModel(120, 44)
	if got := fresh.swell(); got != 1 {
		t.Errorf("before anything has played the picture moves %.2f, want its whole travel", got)
	}
}
