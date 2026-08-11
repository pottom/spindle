package ui

import (
	"math"
	"testing"
	"time"
)

// The same second of music does the same thing at either rate.
//
// Everything on this screen was tuned per frame at thirty a second. Asked for
// frames twice as often, every one of those constants ran twice as fast — the
// water fell at double speed and the row settled before it had leant. This is
// the assertion that the conversion puts it back: a second's worth of frames at
// sixty leaves things where a second's worth at thirty does.
func TestASecondIsASecondAtEitherRate(t *testing.T) {
	defer SetFrameRate(int(time.Second / paceTuned))

	// An easing: how far something has travelled towards its target after a
	// second of being eased at each rate.
	eased := func(fps int, k float32) float64 {
		SetFrameRate(fps)
		at, step := 0.0, float64(paceEase(k))
		for range fps {
			at += (1 - at) * step
		}
		return at
	}
	// What is left of something that keeps a share of itself each frame.
	kept := func(fps int, d float32) float64 {
		SetFrameRate(fps)
		at, step := 1.0, float64(paceKeep(d))
		for range fps {
			at *= step
		}
		return at
	}
	// How high a bead thrown at a speed rises under a pull, and how long it
	// spends in the air — the two things an arc is.
	arc := func(fps int, v, g float32) (float64, float64) {
		SetFrameRate(fps)
		at, speed, pull := 0.0, float64(paceSpeed(v)), float64(paceFall(g))
		frames := 0
		high := 0.0
		for at >= 0 && frames < fps*10 {
			speed -= pull
			at += speed
			high = math.Max(high, at)
			frames++
		}
		return high, float64(frames) / float64(fps)
	}

	for _, c := range []struct {
		what string
		at30 float64
		at60 float64
		by   float64
	}{
		{"the face easing", eased(30, faceEase), eased(60, faceEase), 0.005},
		{"the colour's range", eased(30, wordsRangeClose), eased(60, wordsRangeClose), 0.005},
		{"a spark's light", kept(30, sparkDim), kept(60, sparkDim), 0.005},
		{"the sway falling back", kept(30, swayFall), kept(60, swayFall), 0.005},
	} {
		if math.Abs(c.at30-c.at60) > c.by {
			t.Errorf("%s: %.4f after a second at thirty, %.4f at sixty", c.what, c.at30, c.at60)
		}
	}

	high30, air30 := arc(30, 3.5, sparkGravity)
	high60, air60 := arc(60, 3.5, sparkGravity)
	if math.Abs(high30-high60) > high30*0.05 {
		t.Errorf("a bead rose %.2f dots at thirty and %.2f at sixty", high30, high60)
	}
	if math.Abs(air30-air60) > 0.05 {
		t.Errorf("a bead was in the air %.3fs at thirty and %.3fs at sixty", air30, air60)
	}
}

// A rate that is not offered is not taken, and the frame time follows the rate.
func TestTheRateIsWhatItIsAskedFor(t *testing.T) {
	defer SetFrameRate(int(time.Second / paceTuned))

	for _, c := range []struct {
		asked int
		want  time.Duration
	}{
		{30, time.Second / 30},
		{60, time.Second / 60},
		{0, time.Second / paceLeast},
		{1000, time.Second / paceMost},
	} {
		SetFrameRate(c.asked)
		if scopeInterval != c.want {
			t.Errorf("asked for %d a second and the frame came out %s, not %s", c.asked, scopeInterval, c.want)
		}
	}

	// And at the rate everything was tuned at, nothing is converted at all.
	SetFrameRate(int(time.Second / paceTuned))
	if paceEase(faceEase) != faceEase || paceKeep(sparkDim) != sparkDim {
		t.Error("at the rate they were tuned at the constants were changed anyway")
	}
}
