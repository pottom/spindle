package player

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"
)

// mockLatency is the artificial round-trip time every mock call pays, so the UI
// is exercised against the same timing an optimistic update has to survive.
const mockLatency = 150 * time.Millisecond

// Mock is an offline backend: a fixed catalogue, real-time progress and no
// network beyond the artwork itself. Its methods are called from tea.Cmd
// goroutines, so all state is mutex-guarded.
type Mock struct {
	mu        sync.Mutex
	now       func() time.Time
	queue     []Track
	queued    []Track // put there by hand, played before the rest
	index     int
	playing   bool
	elapsed   time.Duration // progress into the current track, as of startedAt
	startedAt time.Time
	shuffle   bool
	repeat    string
	volume    int
	deviceID  string
	devices   []Device
}

// NewMock returns a mock player already playing the first track.
func NewMock() *Mock {
	m := &Mock{
		now:      time.Now,
		queue:    slices.Clone(mockCatalogue[:4]),
		playing:  true,
		repeat:   RepeatOff,
		volume:   72,
		deviceID: "mock-macbook",
		devices:  mockDevices(),
	}
	m.startedAt = m.now()
	return m
}

func (m *Mock) State(ctx context.Context) (*State, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.advance()

	t := m.queue[m.index]
	dev := m.activeDevice()
	return &State{
		TrackID:    t.ID,
		Title:      t.Title,
		Artists:    slices.Clone(t.Artists),
		Album:      t.Album,
		CoverURL:   t.CoverURL,
		Progress:   m.elapsed,
		Duration:   t.Duration,
		Playing:    m.playing,
		Shuffle:    m.shuffle,
		Repeat:     m.repeat,
		Volume:     m.volume,
		DeviceID:   dev.ID,
		DeviceName: dev.Name,
	}, nil
}

func (m *Mock) Play(ctx context.Context) error {
	return m.mutate(ctx, func() error {
		if !m.playing {
			m.playing = true
			m.startedAt = m.now()
		}
		return nil
	})
}

func (m *Mock) Pause(ctx context.Context) error {
	return m.mutate(ctx, func() error {
		m.playing = false
		return nil
	})
}

func (m *Mock) Next(ctx context.Context) error {
	return m.mutate(ctx, func() error {
		m.promoteQueued()
		m.seekTo(m.index+1, 0)
		return nil
	})
}

func (m *Mock) Previous(ctx context.Context) error {
	return m.mutate(ctx, func() error {
		m.seekTo(m.index-1, 0)
		return nil
	})
}

func (m *Mock) Seek(ctx context.Context, pos time.Duration) error {
	return m.mutate(ctx, func() error {
		m.seekTo(m.index, pos)
		return nil
	})
}

func (m *Mock) SetVolume(ctx context.Context, pct int) error {
	return m.mutate(ctx, func() error {
		m.volume = min(max(pct, 0), 100)
		return nil
	})
}

func (m *Mock) SetShuffle(ctx context.Context, on bool) error {
	return m.mutate(ctx, func() error {
		m.shuffle = on
		return nil
	})
}

func (m *Mock) SetRepeat(ctx context.Context, mode string) error {
	return m.mutate(ctx, func() error {
		switch mode {
		case RepeatOff, RepeatContext, RepeatTrack:
			m.repeat = mode
			return nil
		default:
			return fmt.Errorf("set repeat: unknown mode %q", mode)
		}
	})
}

func (m *Mock) Devices(ctx context.Context) ([]Device, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.devices), nil
}

func (m *Mock) TransferTo(ctx context.Context, deviceID string) error {
	return m.mutate(ctx, func() error {
		if !slices.ContainsFunc(m.devices, func(d Device) bool { return d.ID == deviceID }) {
			return fmt.Errorf("transfer playback: unknown device %q", deviceID)
		}
		m.deviceID = deviceID
		for i := range m.devices {
			m.devices[i].Active = m.devices[i].ID == deviceID
		}
		return nil
	})
}

// Queue is whatever follows the current track, wrapping round the end.
func (m *Mock) Queue(ctx context.Context) (Queue, error) {
	if err := m.delay(ctx); err != nil {
		return Queue{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.advance()

	current := m.queue[m.index]
	out := Queue{
		Current:  &current,
		Upcoming: make([]Track, 0, len(m.queued)+len(m.queue)-1),
	}
	for _, t := range m.queued {
		t.Queued = true
		out.Upcoming = append(out.Upcoming, t)
	}
	for i := 1; i < len(m.queue); i++ {
		out.Upcoming = append(out.Upcoming, m.queue[m.wrap(m.index+i)])
	}
	return out, nil
}

// PlayFrom jumps to an upcoming track, taking the hand-queued ones that sit in
// front of it along the way.
func (m *Mock) PlayFrom(ctx context.Context, trackID string) error {
	return m.mutate(ctx, func() error {
		for i, t := range m.queued {
			if t.ID != trackID {
				continue
			}
			m.queued = m.queued[i:]
			m.promoteQueued()
			m.seekTo(m.index+1, 0)
			return nil
		}
		for i := range m.queue {
			if m.queue[i].ID == trackID {
				m.seekTo(i, 0)
				return nil
			}
		}
		return fmt.Errorf("play from queue: no track %q", trackID)
	})
}

// AddToQueue appends a track from the catalogue.
func (m *Mock) AddToQueue(ctx context.Context, trackID string) error {
	return m.mutate(ctx, func() error {
		t, ok := findMockTrack(trackID)
		if !ok {
			return fmt.Errorf("add to queue: no track %q", trackID)
		}
		m.queued = append(m.queued, t)
		return nil
	})
}

// SetQueue replaces the hand-queued tracks, so the mock can exercise the same
// edits the daemon allows.
func (m *Mock) SetQueue(ctx context.Context, trackIDs []string) error {
	return m.mutate(ctx, func() error {
		next := make([]Track, 0, len(trackIDs))
		for _, id := range trackIDs {
			t, ok := findMockTrack(id)
			if !ok {
				return fmt.Errorf("set queue: no track %q", id)
			}
			next = append(next, t)
		}
		m.queued = next
		return nil
	})
}

// Reorder puts the named tracks at the head of what is coming, in the order
// given, taking any that came from the catalogue rotation out of it. That is
// the same trade the daemon makes: the queue is the only part of the list whose
// order can be set, so a track has to be lifted into it to be moved.
func (m *Mock) Reorder(ctx context.Context, trackIDs []string) error {
	return m.mutate(ctx, func() error {
		// Matched slot by slot rather than through a map of ids: the same track
		// can legitimately be waiting twice, and a map would collapse the two
		// into one and leave a slot that nothing ever fills.
		found := make([]Track, len(trackIDs))
		left := len(trackIDs)
		take := func(t Track) bool {
			for i, id := range trackIDs {
				if id == t.ID && found[i].ID == "" {
					found[i], left = t, left-1
					return true
				}
			}
			return false
		}

		var spare []Track
		for _, t := range m.queued {
			if !take(t) {
				spare = append(spare, t)
			}
		}

		var passed []Track
		taken := make(map[int]bool)
		for step := 1; left > 0 && step < len(m.queue); step++ {
			at := m.wrap(m.index + step)
			t := m.queue[at]
			taken[at] = true
			if !take(t) {
				passed = append(passed, t)
			}
		}
		if left > 0 {
			return fmt.Errorf("reorder: %d of the tracks are not waiting", left)
		}

		// A track cannot be waiting twice: the three lists are drawn from the
		// queue and from the rotation, and the same song can be in both.
		m.queued = m.queued[:0]
		seen := make(map[string]bool, len(found))
		for _, t := range append(append(found, spare...), passed...) {
			if seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			m.queued = append(m.queued, t)
		}

		// Everything lifted out leaves the rotation, which is what stops it
		// being heard a second time when the rotation comes round again. What
		// is playing is never taken, so it is still there to point at.
		current := m.queue[m.index]
		kept := make([]Track, 0, len(m.queue))
		for i, t := range m.queue {
			if !taken[i] {
				kept = append(kept, t)
			}
		}
		m.queue = kept
		m.index = slices.IndexFunc(kept, func(t Track) bool { return t.ID == current.ID })
		return nil
	})
}

// Drop removes an upcoming track, from the queue or from the catalogue rotation.
func (m *Mock) Drop(ctx context.Context, trackID string) error {
	return m.mutate(ctx, func() error {
		for i, t := range m.queued {
			if t.ID == trackID {
				m.queued = slices.Delete(m.queued, i, i+1)
				return nil
			}
		}
		for i, t := range m.queue {
			if t.ID == trackID && i != m.index {
				m.queue = slices.Delete(m.queue, i, i+1)
				if i < m.index {
					m.index--
				}
				return nil
			}
		}
		return fmt.Errorf("drop: no track %q waiting", trackID)
	})
}

// promoteQueued moves the first hand-queued track in front of the context, so
// that it is what plays next. Callers must hold m.mu.
func (m *Mock) promoteQueued() {
	if len(m.queued) == 0 {
		return
	}
	at := m.wrap(m.index + 1)
	m.queue = slices.Insert(m.queue, at, m.queued[0])
	m.queued = m.queued[1:]
}

// findMockTrack looks a track up in the catalogue.
func findMockTrack(id string) (Track, bool) {
	for _, t := range mockCatalogue {
		if t.ID == id {
			return t, true
		}
	}
	return Track{}, false
}

// Search matches the query against the title, the artists and the album. An
// empty query returns nothing rather than the whole catalogue.
func (m *Mock) Search(ctx context.Context, query string) ([]Track, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, nil
	}

	var hits []Track
	for _, t := range mockCatalogue {
		haystack := strings.ToLower(t.Title + " " + strings.Join(t.Artists, " ") + " " + t.Album)
		if strings.Contains(haystack, q) {
			hits = append(hits, t)
		}
	}
	return hits, nil
}

func (m *Mock) Playlists(ctx context.Context) ([]Playlist, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}

	out := make([]Playlist, 0, len(mockPlaylists))
	for _, def := range mockPlaylists {
		p := def.Playlist
		p.Tracks = len(def.trackIDs)
		for _, t := range tracksByID(def.trackIDs) {
			p.Duration += t.Duration
		}
		out = append(out, p)
	}
	return out, nil
}

func (m *Mock) PlaylistTracks(ctx context.Context, playlistID string) ([]Track, error) {
	if err := m.delay(ctx); err != nil {
		return nil, err
	}

	def := playlistByID(playlistID)
	if def == nil {
		return nil, fmt.Errorf("playlist tracks: unknown playlist %q", playlistID)
	}
	return tracksByID(def.trackIDs), nil
}

// PlayTrack queues the whole catalogue from the chosen track, so next and
// previous still lead somewhere.
func (m *Mock) PlayTrack(ctx context.Context, trackID string) error {
	return m.mutate(ctx, func() error {
		i := slices.IndexFunc(mockCatalogue, func(t Track) bool { return t.ID == trackID })
		if i < 0 {
			return fmt.Errorf("play track: unknown track %q", trackID)
		}
		m.setQueue(slices.Clone(mockCatalogue), i)
		return nil
	})
}

func (m *Mock) PlayPlaylist(ctx context.Context, playlistID string, offset int) error {
	return m.mutate(ctx, func() error {
		def := playlistByID(playlistID)
		if def == nil {
			return fmt.Errorf("play playlist: unknown playlist %q", playlistID)
		}
		m.setQueue(tracksByID(def.trackIDs), offset)
		return nil
	})
}

// setQueue replaces what is playing. Callers must hold m.mu.
func (m *Mock) setQueue(tracks []Track, index int) {
	if len(tracks) == 0 {
		return
	}
	m.queue = tracks
	m.index = min(max(index, 0), len(tracks)-1)
	m.elapsed = 0
	m.startedAt = m.now()
	m.playing = true
}

// mutate pays the artificial latency, then runs fn with the lock held and the
// track position brought up to date.
func (m *Mock) mutate(ctx context.Context, fn func() error) error {
	if err := m.delay(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.advance()
	return fn()
}

func (m *Mock) delay(ctx context.Context) error {
	t := time.NewTimer(mockLatency)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// advance folds the time passed since startedAt into elapsed and rolls over to
// the following tracks if that carried past the end. Callers must hold m.mu.
func (m *Mock) advance() {
	if !m.playing {
		return
	}
	now := m.now()
	m.elapsed += now.Sub(m.startedAt)
	m.startedAt = now

	for {
		d := m.queue[m.index].Duration
		if d <= 0 || m.elapsed < d {
			return
		}
		m.elapsed -= d
		if m.repeat != RepeatTrack {
			m.promoteQueued()
			m.index = m.wrap(m.index + 1)
		}
	}
}

// seekTo jumps to a track and position, clamping both. Callers must hold m.mu.
func (m *Mock) seekTo(index int, pos time.Duration) {
	m.index = m.wrap(index)
	m.elapsed = min(max(pos, 0), m.queue[m.index].Duration)
	m.startedAt = m.now()
}

// activeDevice returns the device playback currently sits on. Callers must hold m.mu.
func (m *Mock) activeDevice() Device {
	for _, d := range m.devices {
		if d.ID == m.deviceID {
			return d
		}
	}
	return Device{}
}

func (m *Mock) wrap(i int) int {
	n := len(m.queue)
	return ((i % n) + n) % n
}

func playlistByID(id string) *mockPlaylistDef {
	for i := range mockPlaylists {
		if mockPlaylists[i].ID == id {
			return &mockPlaylists[i]
		}
	}
	return nil
}

func tracksByID(ids []string) []Track {
	out := make([]Track, 0, len(ids))
	for _, id := range ids {
		if i := slices.IndexFunc(mockCatalogue, func(t Track) bool { return t.ID == id }); i >= 0 {
			out = append(out, mockCatalogue[i])
		}
	}
	return out
}

// Waveform is a stand-in trace, since the mock plays no sound. It exists so the
// player screen can be worked on without a device: what it draws is the shape
// of a waveform, not a recording of one.
func (m *Mock) Waveform() []float32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.advance()

	if !m.playing {
		return nil
	}

	t := m.elapsed.Seconds()
	beat := math.Exp(-math.Mod(t*2, 1) * 6)

	// Twice the window, as a real device sends: the surplus is trigger slack.
	out := make([]float32, 2*WaveformWindow)
	for i := range out {
		x := float64(i) / float64(len(out))
		out[i] = float32(
			math.Sin(x*34+t*5.0)*(0.40+beat*0.44) +
				math.Sin(x*81-t*8.2)*0.19 +
				math.Sin(x*193+t*17.0)*0.085*(0.4+beat),
		)
	}
	return out
}
