package ui

import "testing"

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
			if got := d.frames[0].tall; got != size.tall {
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
