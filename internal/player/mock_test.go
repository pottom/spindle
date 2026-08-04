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
	tick(m.queue[0].Duration + 10*time.Second)
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
	tick(m.queue[0].Duration + 10*time.Second)
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
	if m.elapsed != m.queue[0].Duration {
		t.Errorf("elapsed = %v, want %v", m.elapsed, m.queue[0].Duration)
	}
}

func TestMockSeekToWrapsIndex(t *testing.T) {
	m, _ := newTestMock()
	last := len(m.queue) - 1

	m.seekTo(-1, 0)
	if m.index != last {
		t.Errorf("index = %d, want %d", m.index, last)
	}

	m.seekTo(len(m.queue), 0)
	if m.index != 0 {
		t.Errorf("index = %d, want 0", m.index)
	}
}

// Reordering a track that came from the rotation has to lift it out: leaving it
// there would play it twice, once where it was moved to and once where it was.
func TestMockReorderTakesTracksOutOfTheRotation(t *testing.T) {
	m, _ := newTestMock()
	ctx := t.Context()

	before, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if len(before.Upcoming) < 3 {
		t.Fatalf("the mock has %d tracks coming, want at least 3", len(before.Upcoming))
	}
	first, second := before.Upcoming[0].ID, before.Upcoming[1].ID

	if err := m.Reorder(ctx, []string{second, first}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	after, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if after.Upcoming[0].ID != second || after.Upcoming[1].ID != first {
		t.Errorf("upcoming starts %s, %s — want them swapped", after.Upcoming[0].ID, after.Upcoming[1].ID)
	}

	seen := map[string]int{}
	for _, t := range after.Upcoming {
		seen[t.ID]++
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("%s is waiting %d times, want once", id, n)
		}
	}
	if len(after.Upcoming) != len(before.Upcoming) {
		t.Errorf("%d tracks coming after the move, want the %d there were", len(after.Upcoming), len(before.Upcoming))
	}
}
