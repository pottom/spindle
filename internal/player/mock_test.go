package player

import (
	"testing"
	"time"
)

// newTestMock returns a mock driven by a clock the test moves by hand.
func newTestMock() (*Mock, func(time.Duration)) {
	m := NewMock()
	clock := time.Now()
	m.now = func() time.Time { return clock }
	m.startedAt = clock
	return m, func(d time.Duration) { clock = clock.Add(d) }
}

func TestMockAdvanceRollsOverTracks(t *testing.T) {
	m, tick := newTestMock()
	tick(mockTracks[0].duration + 10*time.Second)
	m.advance()

	if m.index != 1 {
		t.Errorf("index = %d, want 1", m.index)
	}
	if m.elapsed != 10*time.Second {
		t.Errorf("elapsed = %v, want %v", m.elapsed, 10*time.Second)
	}
}

func TestMockAdvanceRepeatsOneTrack(t *testing.T) {
	m, tick := newTestMock()
	m.repeat = RepeatTrack
	tick(mockTracks[0].duration + 10*time.Second)
	m.advance()

	if m.index != 0 {
		t.Errorf("index = %d, want 0 under repeat track", m.index)
	}
	if m.elapsed != 10*time.Second {
		t.Errorf("elapsed = %v, want %v", m.elapsed, 10*time.Second)
	}
}

func TestMockAdvanceIsFrozenWhilePaused(t *testing.T) {
	m, tick := newTestMock()
	tick(30 * time.Second)
	m.advance()
	m.playing = false

	tick(time.Hour)
	m.advance()

	if m.elapsed != 30*time.Second {
		t.Errorf("elapsed = %v, want %v", m.elapsed, 30*time.Second)
	}
}

func TestMockSeekToClamps(t *testing.T) {
	m, _ := newTestMock()

	m.seekTo(0, -time.Minute)
	if m.elapsed != 0 {
		t.Errorf("elapsed = %v, want 0", m.elapsed)
	}

	m.seekTo(0, time.Hour)
	if m.elapsed != mockTracks[0].duration {
		t.Errorf("elapsed = %v, want %v", m.elapsed, mockTracks[0].duration)
	}
}

func TestMockSeekToWrapsIndex(t *testing.T) {
	m, _ := newTestMock()
	last := len(mockTracks) - 1

	m.seekTo(-1, 0)
	if m.index != last {
		t.Errorf("index = %d, want %d", m.index, last)
	}

	m.seekTo(len(mockTracks), 0)
	if m.index != 0 {
		t.Errorf("index = %d, want 0", m.index)
	}
}
