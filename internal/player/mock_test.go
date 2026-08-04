package player

import (
	"slices"
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

// A track can be waiting in the queue and still be sitting in the rotation it
// was copied out of. Reordering must not leave both, or it plays twice.
func TestMockReorderDropsTheSecondCopy(t *testing.T) {
	m, _ := newTestMock()
	ctx := t.Context()

	before, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	twice := before.Upcoming[1].ID
	if err := m.AddToQueue(ctx, twice); err != nil {
		t.Fatalf("AddToQueue: %v", err)
	}

	// Reorder over the run that holds both copies.
	list, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	ids := make([]string, 0, len(list.Upcoming))
	for _, track := range list.Upcoming {
		ids = append(ids, track.ID)
	}
	if err := m.Reorder(ctx, ids); err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	after, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	seen := 0
	for _, track := range after.Upcoming {
		if track.ID == twice {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("%s is waiting %d times after the reorder, want once", twice, seen)
	}
}

// The mock is what --mock and most of the tests run against, so it has to page
// the way the real backends do rather than hand the whole list over at once.
// Its page is small enough that its own short catalogue reaches a second one.
func TestMockPagesAPlaylist(t *testing.T) {
	m, _ := newTestMock()
	whole, err := m.PlaylistTracksPage(t.Context(), "p4", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !whole.More {
		t.Fatalf("p4 fits in one page of %d, so it cannot exercise paging", mockPageLimit)
	}

	cases := []struct {
		name   string
		offset int
		want   int
		more   bool
	}{
		{"the first page", 0, mockPageLimit, true},
		{"the rest of it", mockPageLimit, 9 - mockPageLimit, false},
		{"exactly past the end", 9, 0, false},
		{"far past the end", 500, 0, false},
		{"an offset that went negative", -5, mockPageLimit, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page, err := m.PlaylistTracksPage(t.Context(), "p4", c.offset)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != c.want {
				t.Errorf("%d tracks at offset %d, want %d", len(page.Items), c.offset, c.want)
			}
			if page.More != c.more {
				t.Errorf("More = %v at offset %d, want %v", page.More, c.offset, c.more)
			}
		})
	}
}

// A caller that walks the pages until there are no more must end up with every
// match, each of them once. A page boundary that repeated or skipped an item
// would show up here and nowhere else.
func TestMockSearchPagesCoverEveryMatch(t *testing.T) {
	m, _ := newTestMock()

	var walked []string
	for offset := 0; ; {
		page, err := m.SearchPage(t.Context(), "queen", offset)
		if err != nil {
			t.Fatal(err)
		}
		for _, track := range page.Items {
			walked = append(walked, track.ID)
		}
		if !page.More {
			break
		}
		offset = page.Next
	}

	var want []string
	for _, track := range mockCatalogue {
		if slices.Contains(track.Artists, "Queen") {
			want = append(want, track.ID)
		}
	}
	if len(want) <= mockPageLimit {
		t.Fatalf("%d matches fit in one page, so the walk proves nothing", len(want))
	}
	if !slices.Equal(walked, want) {
		t.Errorf("walking the pages gave %v, want %v", walked, want)
	}
}

// Playing one track is not a reason to throw away a queue somebody spent a
// minute filling: what was waiting still follows it.
func TestPlayNowKeepsTheQueue(t *testing.T) {
	m, _ := newTestMock()
	ctx := t.Context()

	before, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	wanted := before.Upcoming[len(before.Upcoming)-1].ID
	if err := m.AddToQueue(ctx, wanted); err != nil {
		t.Fatalf("AddToQueue: %v", err)
	}

	// Something else entirely, played now.
	other := mockCatalogue[0].ID
	if other == wanted {
		other = mockCatalogue[1].ID
	}
	if err := m.PlayNow(ctx, other); err != nil {
		t.Fatalf("PlayNow: %v", err)
	}

	st, err := m.State(ctx)
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if st.TrackID != other {
		t.Errorf("playing %q, want the track asked for", st.TrackID)
	}

	after, err := m.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	var kept bool
	for _, track := range after.Upcoming {
		if track.ID == wanted && track.Queued {
			kept = true
		}
	}
	if !kept {
		t.Error("the hand-added track was thrown away by playing something else")
	}
}
