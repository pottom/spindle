package player

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/auth"
)

// liveBackend builds a backend against the real API using the stored grant.
func liveBackend(t *testing.T) *Spotify {
	t.Helper()
	if os.Getenv("SPINDLE_LIVE") == "" {
		t.Skip("set SPINDLE_LIVE to measure against the real API")
	}

	session, err := auth.NewSession(context.Background(), io.Discard)
	if err != nil {
		t.Fatalf("authorise: %v", err)
	}
	return NewSpotify(session.Client(context.Background()))
}

func summarise(name string, samples []time.Duration) string {
	if len(samples) == 0 {
		return name + ": no samples"
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	var total time.Duration
	for _, d := range samples {
		total += d
	}
	return fmt.Sprintf("%-22s n=%2d  min %5v  median %5v  max %5v",
		name,
		len(samples),
		samples[0].Round(time.Millisecond),
		samples[len(samples)/2].Round(time.Millisecond),
		samples[len(samples)-1].Round(time.Millisecond),
	)
}

// TestLiveControlLatency times the control endpoints. Every call here writes
// back the value it just read, so nothing audible changes.
func TestLiveControlLatency(t *testing.T) {
	s := liveBackend(t)
	ctx := context.Background()

	before, err := s.State(ctx)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	t.Logf("measuring against %q on %s", before.Title, before.DeviceName)

	const rounds = 5
	calls := map[string]func() error{
		"State (read)":   func() error { _, err := s.State(ctx); return err },
		"Devices (read)": func() error { _, err := s.Devices(ctx); return err },
		"SetVolume":      func() error { return s.SetVolume(ctx, before.Volume) },
		"SetShuffle":     func() error { return s.SetShuffle(ctx, before.Shuffle) },
		"SetRepeat":      func() error { return s.SetRepeat(ctx, before.Repeat) },
	}

	names := make([]string, 0, len(calls))
	for name := range calls {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		samples := make([]time.Duration, 0, rounds)
		for range rounds {
			start := time.Now()
			if err := calls[name](); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			samples = append(samples, time.Since(start))
			time.Sleep(300 * time.Millisecond)
		}
		t.Log(summarise(name, samples))
	}
}

// TestLivePropagation is the question the optimistic window exists to answer:
// once a command returns, how long until a poll agrees with it?
func TestLivePropagation(t *testing.T) {
	s := liveBackend(t)
	ctx := context.Background()

	before, err := s.State(ctx)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !before.Playing {
		t.Skip("start playback before measuring propagation")
	}

	// Pause and wait for the API to agree, then resume and do the same. The
	// music stops for about a second.
	pauseDelay := waitFor(t, s, "pause", func() error { return s.Pause(ctx) },
		func(st *State) bool { return !st.Playing })
	playDelay := waitFor(t, s, "resume", func() error { return s.Play(ctx) },
		func(st *State) bool { return st.Playing })

	t.Logf("pause  visible to the API after %v", pauseDelay.Round(time.Millisecond))
	t.Logf("resume visible to the API after %v", playDelay.Round(time.Millisecond))

	// One skip: this is the number the 400 ms confirming fetch was guessed at.
	was := before.TrackID
	skipDelay := waitFor(t, s, "skip", func() error { return s.Next(ctx) },
		func(st *State) bool { return st.TrackID != was })
	t.Logf("track change visible after %v  (the confirming fetch waits %v)",
		skipDelay.Round(time.Millisecond), 400*time.Millisecond)
}

// waitFor issues a command and polls until the API reflects it.
func waitFor(t *testing.T, s *Spotify, name string, do func() error, done func(*State) bool) time.Duration {
	t.Helper()
	ctx := context.Background()

	start := time.Now()
	if err := do(); err != nil {
		t.Fatalf("%s: %v", name, err)
	}

	deadline := start.Add(15 * time.Second)
	for time.Now().Before(deadline) {
		st, err := s.State(ctx)
		if err == nil && done(st) {
			return time.Since(start)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s never showed up in the API within 15s", name)
	return 0
}

// TestLivePollingCadence runs the shipping cadence for ten minutes and counts
// what Spotify makes of it. This is the M3 done-criterion.
func TestLivePollingCadence(t *testing.T) {
	s := liveBackend(t)
	if os.Getenv("SPINDLE_LIVE_LONG") == "" {
		t.Skip("set SPINDLE_LIVE_LONG for the ten minute soak")
	}

	const (
		interval = 5 * time.Second
		duration = 10 * time.Minute
	)

	var calls, throttles, failures int
	var samples []time.Duration

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		start := time.Now()
		_, err := s.State(context.Background())
		calls++

		var limited *RateLimitedError
		switch {
		case errors.As(err, &limited):
			throttles++
			t.Logf("429 after %d calls, asked to wait %v", calls, limited.RetryAfter)
			time.Sleep(limited.RetryAfter)
		case err != nil && !errors.Is(err, ErrNoActiveDevice):
			failures++
			t.Logf("call %d failed: %v", calls, err)
		default:
			samples = append(samples, time.Since(start))
		}

		time.Sleep(interval - min(time.Since(start), interval))
	}

	t.Logf("%d calls over %v at one per %v", calls, duration, interval)
	t.Log(summarise("State under soak", samples))
	t.Logf("429s: %d, other failures: %d", throttles, failures)

	if throttles > 0 {
		t.Errorf("the shipping cadence was rate limited %d times", throttles)
	}
}
