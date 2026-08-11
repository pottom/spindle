package ui

import (
	"math"
	"time"
)

// How fast the picture is drawn, and what that does to everything tuned by eye
// at a different rate.
//
// Every effect on this screen was swept, watched and settled at thirty frames a
// second: what a spark keeps of its light each frame, how far a drop falls each
// frame, how quickly the range the colours are spread over follows the record.
// All of them are written as "per frame", which is only a rate if the frames
// come at a rate — and the moment the frame time became something a listener can
// set, every one of them meant something different on every machine.
//
// Measured before this file existed, by asking for frames at 16ms instead of
// 33ms and watching: the water fell at twice the speed, the row settled twice as
// fast, and the whole screen read as a record played too quickly. It was not
// imagination — it was fifteen constants running at double rate.
//
// So they stay written as they were tuned, and they are converted here for the
// rate actually being drawn at. Four kinds, each with its own arithmetic:
//
//	an easing, or a chance, per frame     k → 1 - (1-k)^r
//	what is kept of something each frame  d → d^r
//	how far something moves in a frame    v → v·r
//	how much that speed changes           a → a·r²
//
// where r is this frame as a share of the frame they were tuned at. The last two
// together are what keeps an arc the same: a bead thrown at v·r under a pull of
// a·r² rises exactly as high as it did and takes exactly as long to come down.
const (
	// paceTuned is the frame every constant in this package was settled against.
	// It is history rather than a setting, and nothing should change it: what
	// changes is scopeInterval, and this is what it is measured against.
	//
	// Thirty a second. It was written as 33ms for years, which is a thirtieth
	// rounded down and one per cent short of one — near enough for a sleep and
	// not near enough to convert against, since it would leave a run at the rate
	// everything was tuned at converting all of it by a hundredth.
	paceTuned = time.Second / 30

	// paceLeast and paceMost are the rates that will be honoured. Below the
	// first a trace is a slideshow, and above the second the terminal is being
	// asked for more frames than a screen refreshes at.
	paceLeast = 15
	paceMost  = 120
)

// scopeInterval is the picture's frame time, which is the one thing here that
// is not a constant: see SetFrameRate.
var scopeInterval = paceTuned

// paceShare is this frame as a share of the one everything was tuned at.
var paceShare = 1.0

// The tuned constants, converted for the rate being drawn at. They are gathered
// here rather than left beside what they belong to because they have to be
// worked out together, once, when the rate is set — and because a list of
// everything on this screen that answers the frame rather than the clock is
// worth being able to read in one place.
var (
	faceEaseAt        float32 // face.go
	faceLiftEaseAt    float32
	joinWatchAt       float32 // joins.go
	swellCloseAt      float64 // swell.go
	wordsRoomEaseAt   float32 // words.go
	wordsRangeCloseAt float32
	wordsRollEaseAt   float32
	wordsSparkSprayAt float32
	scopeReleaseAt    float32 // scope.go
	scopeSettleAt     float32
	scopeTrailAt      int
	sparkDimAt        float32 // spark.go
	sparkGravityAt    float32
	sparkSprayAt      float32
	stageDimAt        float32 // stage.go
	stageGravityAt    float32
	stageSprayAt      float32
	swayFallAt        float32 // sway.go
	swaySettleAt      float32
)

func init() { paceReckon() }

// SetFrameRate fixes how often the picture is drawn. Called once, before the
// interface starts, and never again: a rate that changed under a running screen
// would leave what is already in the air moving at the old one.
//
// It is a listener's setting rather than ours because it is their terminal that
// has to keep up. Sixty is what this was measured at and what it is set to; a
// terminal that cannot draw the whole screen that often will show it in the
// bar's own first field, and thirty is there for it.
func SetFrameRate(fps int) {
	fps = min(max(fps, paceLeast), paceMost)
	scopeInterval = time.Second / time.Duration(fps)
	paceShare = float64(scopeInterval) / float64(paceTuned)
	paceReckon()
}

// paceReckon works the conversions out for the rate now set.
func paceReckon() {
	faceEaseAt = paceEase(faceEase)
	faceLiftEaseAt = paceEase(faceLiftEase)
	joinWatchAt = paceEase(joinWatch)
	swellCloseAt = float64(paceEase(swellClose))
	wordsRoomEaseAt = paceEase(wordsRoomEase)
	wordsRangeCloseAt = paceEase(wordsRangeClose)
	wordsRollEaseAt = paceEase(wordsRollEase)
	wordsSparkSprayAt = paceEase(wordsSparkSpray)
	scopeReleaseAt = paceKeep(scopeRelease)
	scopeSettleAt = paceKeep(scopeSettle)
	scopeTrailAt = paceFrames(scopeTrail)
	sparkDimAt = paceKeep(sparkDim)
	sparkGravityAt = paceFall(sparkGravity)
	sparkSprayAt = paceEase(sparkSpray)
	stageDimAt = paceKeep(stageDim)
	stageGravityAt = paceFall(stageGravity)
	stageSprayAt = paceEase(stageSpray)
	swayFallAt = paceKeep(swayFall)
	swaySettleAt = paceKeep(swaySettle)
}

// paceEase converts a share taken each frame — an easing towards something, or
// a chance of something happening. Half the frame, and each one has to do half
// as much for the same to have happened by the same moment.
func paceEase(k float32) float32 {
	// At the rate they were tuned at, nothing is converted: the arithmetic comes
	// back to the same number, but not to the same bits, and a constant that
	// drifts in the last place for nobody's benefit is worth not touching.
	if paceShare == 1 || k <= 0 || k >= 1 {
		return k
	}
	return float32(1 - math.Pow(float64(1-k), paceShare))
}

// paceKeep converts what is kept of something each frame, which is the same
// arithmetic read from the other end.
func paceKeep(d float32) float32 {
	if paceShare == 1 || d <= 0 || d >= 1 {
		return d
	}
	return float32(math.Pow(float64(d), paceShare))
}

// paceSpeed converts how far something moves in a frame. Applied where a thing
// is thrown rather than every frame it is in the air: a speed set once is a
// speed converted once.
func paceSpeed(v float32) float32 { return v * float32(paceShare) }

// paceFall converts how much a speed changes each frame. Squared, because it is
// a speed per frame per frame, and it is what keeps an arc the shape it was.
func paceFall(a float32) float32 { return a * float32(paceShare*paceShare) }

// paceFrames converts a number of frames — a trail, a hold, anything counted in
// them rather than in seconds.
func paceFrames(n int) int {
	return max(int(math.Round(float64(n)/paceShare)), 1)
}
