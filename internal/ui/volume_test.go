package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

func volumeModel(t *testing.T, w, rows int) Model {
	t.Helper()
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = w, rows
	m.stage.on = true
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: 3 * time.Minute, Playing: true, Volume: 60}
	return m
}

// The lamps come up on a change and take themselves away.
//
// The big screen has no furniture on it, so a reading that stayed would be the
// first thing to break that — and how loud the room is only matters while
// somebody is changing it.
func TestTheVolumeSaysItselfAndThenLeaves(t *testing.T) {
	m := volumeModel(t, 80, 30)

	m.volumeFlow(m.height)
	if m.volumeShowing() {
		t.Error("the lamps were up before anything had been changed")
	}

	m.setVolume(75)
	m.volumeFlow(m.height)
	if !m.volumeShowing() {
		t.Fatal("changing the volume did not put the lamps up")
	}
	if got, want := m.volumeLit(), 75/volumeLampIs; got != want {
		t.Errorf("seventy five lit %d lamps, want %d", got, want)
	}

	// Once it settles they let go of the wall rather than fading out.
	m.volume.at = time.Now().Add(-volumeShows - time.Millisecond)
	m.volumeFlow(m.height)
	if m.volumeShowing() {
		t.Error("the lamps stayed up after the volume had settled")
	}
	if len(m.volume.falling) == 0 {
		t.Fatal("the lamps vanished instead of falling off")
	}

	// And they go: nothing is left in the air for ever.
	for range 600 {
		m.volumeFlow(m.height)
	}
	if n := len(m.volume.falling); n != 0 {
		t.Errorf("%d lamps were still in the air after twenty seconds", n)
	}
}

// The lamps never touch the ring the record's progress runs on.
//
// The outermost dots are the head and its tail, and a second reading beside them
// is read as part of them — the head would look like it had grown a tail it does
// not have.
func TestTheVolumeKeepsClearOfTheProgress(t *testing.T) {
	for _, size := range [][2]int{{80, 30}, {200, 53}, {40, 20}} {
		w, rows := size[0], size[1]
		m := volumeModel(t, w, rows)
		m.volumeFlow(rows) // adopt the level it started at
		m.setVolume(100)
		m.volumeFlow(rows)

		dotsX, dotsY := w*dotsPerCellX, rows*dotsPerCellY
		grid := make([]uint8, w*rows)
		paint, hue := make([]int8, w*rows), make([]int8, w*rows)
		for i := range paint {
			paint[i] = -1
		}
		m.volumeDraw(w, rows, grid, paint, hue, 6, 64)

		// Nothing in the first cell of any row, which is where the ring runs.
		var lit, wall int
		for r := range rows {
			for c := range w {
				if grid[r*w+c] == 0 {
					continue
				}
				lit++
				if c == 0 || c == w-1 {
					wall++
				}
			}
		}
		if lit == 0 {
			t.Errorf("%dx%d: a full column drew nothing", w, rows)
		}
		if wall > 0 {
			t.Errorf("%dx%d: %d cells of the volume landed on the progress ring", w, rows, wall)
		}

		// The whole column is on the screen: a meter whose top has run off is
		// not a meter, and one standing on the bottom row is in the water.
		top, foot := dotsY, 0
		for r := range rows {
			for c := range w {
				if grid[r*w+c] != 0 {
					top = min(top, r*dotsPerCellY)
					foot = max(foot, r)
				}
			}
		}
		if top <= 0 {
			t.Errorf("%dx%d: the column reached the top of the screen", w, rows)
		}
		if foot >= rows-1 {
			t.Errorf("%dx%d: the column stood on the very bottom row", w, rows)
		}
		_ = dotsX
	}
}

// Silence is not a level, so the lamps do not answer it.
//
// Muting is already said by the one with his fingers in his ears, and coming
// back out of it is a restoring rather than a choosing. The lamps arriving on
// the way back would be the picture announcing something nobody asked for.
func TestTheVolumeSaysNothingAboutSilence(t *testing.T) {
	m := volumeModel(t, 80, 30)
	m.volumeFlow(m.height) // adopt the level it started at

	m.toggleMute()
	m.volumeFlow(m.height)
	if m.volumeShowing() {
		t.Error("muting put the lamps up")
	}

	m.toggleMute()
	m.volumeFlow(m.height)
	if m.volumeShowing() {
		t.Error("coming back from mute put the lamps up")
	}
	if got, want := m.ps.Volume, 60; got != want {
		t.Fatalf("came back to %d rather than %d", got, want)
	}

	// An ordinary change still does.
	m.setVolume(80)
	m.volumeFlow(m.height)
	if !m.volumeShowing() {
		t.Error("a change of level after all that said nothing")
	}
}
