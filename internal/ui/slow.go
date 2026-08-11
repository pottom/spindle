package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pottom/spindle/internal/xdg"
)

// Where the frames go when they go missing.
//
// Watched on the running interface for weeks: every so often the picture stops
// for long enough to see, then carries on as if nothing happened. Nothing about
// it is reproducible on demand and nothing in the code says which part of a
// frame took the time, so it has been argued about instead of measured — which
// is the one way of working this project has already learned not to trust.
//
// So a frame times itself. What is written down is only the frames that went
// over, and what is written with them is enough to tell the three candidates
// apart: the work of deciding what to draw, the work of drawing it, and the time
// that went somewhere else entirely — a garbage collection, the terminal not
// reading, the machine busy with something that is not us.
//
// It costs two clock reads a frame and a comparison. The file is only opened
// when something is worth writing to it.

const (
	// slowBudget is how long a frame has: whatever the interface set out to
	// draw at. Anything past it has already cost a frame.
	slowBudget = scopeInterval

	// slowGap is how far apart two frames have to land before the gap is worth
	// writing down. A frame and a half: a frame that merely ran late is not a
	// frame that went missing.
	//
	// Both are taken from the rate rather than written down beside it. Left as
	// the numbers that suited thirty a second, a run at sixty would have called
	// every frame on time whatever it did.
	slowGap = scopeInterval * 3 / 2

	slowFile = "frames.jsonl"
)

// slowState is what one frame's timing needs to remember. It is package state
// rather than model state because View has a value receiver and cannot write
// back, and because a dropped frame is a property of the process rather than of
// any one model.
var slowState struct {
	mu sync.Mutex

	last    time.Time // when the last frame's update began
	began   time.Time // when this frame's update began
	update  time.Duration
	pending bool          // a frame is open and has not been closed off yet
	asked   time.Duration // how long the daemon took to answer for it

	frames int
	missed int
	worst  time.Duration

	// The last frame's parts, and the rate they are arriving at. Kept for the
	// bar on ctrl+shift+d: the file says what went wrong an hour ago, and the
	// bar has to say what is going on now.
	lastGap, lastUpdate, lastRender time.Duration
	fps                             float64

	off bool // set once the file cannot be written, so it is not tried again
}

// slowRead is the timing as a reader wants it: one lock, one copy, nothing that
// can change underneath whoever is drawing it.
type slowRead struct {
	frames, missed                    int
	worst, gap, update, render, asked time.Duration
	fps                               float64
}

// slowNow hands out that copy.
func slowNow() slowRead {
	slowState.mu.Lock()
	defer slowState.mu.Unlock()

	return slowRead{
		frames: slowState.frames,
		missed: slowState.missed,
		worst:  slowState.worst,
		gap:    slowState.lastGap,
		update: slowState.lastUpdate,
		render: slowState.lastRender,
		asked:  slowState.asked,
		fps:    slowState.fps,
	}
}

// slowResume says the frame loop has just been started again after being off,
// so the next frame is not measured against the last one before it stopped.
func slowResume() {
	slowState.mu.Lock()
	defer slowState.mu.Unlock()

	slowState.last = time.Time{}
}

// slowFrameBegan is called as a frame's update starts.
func slowFrameBegan() {
	slowState.mu.Lock()
	defer slowState.mu.Unlock()

	slowState.began, slowState.pending = time.Now(), true
}

// slowAsked records how long the daemon took to answer the request this frame
// was waiting on. The picture is driven by asking the daemon for a frame and
// waiting, so a slow answer is a late frame however fast this program is.
func slowAsked(d time.Duration) {
	slowState.mu.Lock()
	defer slowState.mu.Unlock()

	slowState.asked = d
}

// slowUpdateDone is called when the update is over and the render is about to
// begin.
func slowUpdateDone() {
	slowState.mu.Lock()
	defer slowState.mu.Unlock()

	slowState.update = time.Since(slowState.began)
}

// slowRenderDone closes the frame off, and writes it down if it went over.
//
// The three numbers are kept apart on purpose. gap is from the start of the last
// frame to the start of this one, which is what a watcher actually sees; update
// and render are the parts of it this program is responsible for. A gap that is
// long while update and render are short is time that went somewhere else, and
// no amount of looking at this code will find it.
func slowRenderDone(m Model, render time.Duration) {
	slowState.mu.Lock()
	defer slowState.mu.Unlock()

	// Only a frame is closed off. View runs for every message the interface
	// takes — a keypress, a list arriving — and those are not frames; timed as
	// if they were, the first one before any frame at all reported a gap since
	// the zero time, which is where the nonsense in the first reading came from.
	if !slowState.pending {
		return
	}
	slowState.pending = false

	began, update := slowState.began, slowState.update
	slowState.frames++

	// The first frame of a stretch has nothing to be measured against: either
	// nothing has been drawn yet, or the picture has been off screen and the
	// loop with it. Measured before this was here — five and a half seconds
	// spent on the library tab, filed as the worst frame of an eleven thousand
	// frame run, and the rate it was averaged into with it.
	if slowState.last.IsZero() {
		slowState.last = began
		slowState.lastUpdate, slowState.lastRender = update, render
		return
	}

	gap := began.Sub(slowState.last)
	slowState.last = began

	// Every frame, not only the late ones: a rate is only a rate if the frames
	// that arrived on time are counted too. Eased, because a single gap is
	// noise and what anybody reading it wants is the rate over the last second
	// or so. Anything past a second is the interface having been left alone,
	// not a rate.
	slowState.lastGap, slowState.lastUpdate, slowState.lastRender = gap, update, render
	if gap > 0 && gap < time.Second {
		now := float64(time.Second) / float64(gap)
		if slowState.fps == 0 {
			slowState.fps = now
		} else {
			slowState.fps += (now - slowState.fps) * 0.1
		}
	}

	if slowState.off {
		return
	}
	if gap < slowGap && update+render < slowBudget {
		return
	}

	slowState.missed++
	if gap > slowState.worst {
		slowState.worst = gap
	}

	line := struct {
		At     string `json:"at"`
		GapMs  int64  `json:"gap_ms"`
		Update int64  `json:"update_us"`
		Render int64  `json:"render_us"`
		Elsew  int64  `json:"elsewhere_us"`
		Asked  int64  `json:"daemon_us"`
		Screen string `json:"screen"`
		Mode   int    `json:"mode"`
		Wide   int    `json:"wide"`
		High   int    `json:"high"`
		Words  bool   `json:"words_up"`
		Figure bool   `json:"figure_up"`
		Frames int    `json:"frames"`
		Missed int    `json:"missed"`
	}{
		At:     began.Format(time.RFC3339Nano),
		GapMs:  gap.Milliseconds(),
		Update: update.Microseconds(),
		Render: render.Microseconds(),
		Elsew:  (gap - update - render).Microseconds(),
		Asked:  slowState.asked.Microseconds(),
		Screen: debugScreen(m),
		Mode:   int(m.scopeMode()),
		Wide:   m.width,
		High:   m.height,
		Words:  m.words.have.DotsX > 0,
		Figure: m.faceUp(),
		Frames: slowState.frames,
		Missed: slowState.missed,
	}

	raw, err := json.Marshal(line)
	if err != nil {
		return
	}
	dir, err := xdg.StateDir()
	if err != nil {
		slowState.off = true
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, slowFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slowState.off = true
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}
