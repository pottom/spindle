package player

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// liveLocal points a Local at the daemon running on this machine and reads its
// state once, so that it is not idle and the daemon path is the one taken.
func liveLocal(t *testing.T) *Local {
	t.Helper()
	web := liveBackend(t) // skips unless SPINDLE_LIVE is set

	l := NewLocal(web, "http://127.0.0.1:"+daemonPortForTest(), &http.Client{Timeout: 5 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l.refresh(ctx); err != nil {
		t.Skipf("no daemon to talk to: %v", err)
	}
	if l.idle() {
		t.Skip("the daemon is not holding anything, so this would test the Web API instead")
	}
	return l
}

func daemonPortForTest() string {
	if p := os.Getenv("SPINDLE_DAEMON_PORT"); p != "" {
		return p
	}
	return "3678"
}

// The two keys that were asked about. Both set a flag on the daemon and both are
// drawn from what the daemon says afterwards, so the round trip is the thing
// worth testing: not that the request was sent, but that asking again gives back
// what was asked for.
//
// Repeat is the one with room to go wrong, because spindle keeps one mode where
// the daemon keeps two booleans, and every mode has to clear the half it is not.
func TestLiveRepeatSurvivesTheRoundTrip(t *testing.T) {
	l := liveLocal(t)
	ctx := context.Background()

	was := stateNow(t, l)
	t.Cleanup(func() {
		_ = l.SetRepeat(context.Background(), was.Repeat)
		_ = l.SetShuffle(context.Background(), was.Shuffle)
	})

	// Every mode, and each one reached from every other, since the bug this
	// guards against is a half left set from the mode before.
	for _, from := range []string{RepeatOff, RepeatContext, RepeatTrack} {
		for _, to := range []string{RepeatOff, RepeatContext, RepeatTrack} {
			if err := l.SetRepeat(ctx, from); err != nil {
				t.Fatalf("set repeat %s: %v", from, err)
			}
			if err := l.SetRepeat(ctx, to); err != nil {
				t.Fatalf("set repeat %s after %s: %v", to, from, err)
			}
			if got := stateNow(t, l).Repeat; got != to {
				t.Errorf("%s then %s: the daemon reports %q", from, to, got)
			}
		}
	}
}

func TestLiveShuffleSurvivesTheRoundTrip(t *testing.T) {
	l := liveLocal(t)
	ctx := context.Background()

	was := stateNow(t, l)
	t.Cleanup(func() { _ = l.SetShuffle(context.Background(), was.Shuffle) })

	for _, want := range []bool{true, false, true, false} {
		if err := l.SetShuffle(ctx, want); err != nil {
			t.Fatalf("set shuffle %v: %v", want, err)
		}
		if got := stateNow(t, l).Shuffle; got != want {
			t.Errorf("asked for shuffle %v, the daemon reports %v", want, got)
		}
	}
}

// stateNow reads the daemon again rather than trusting the snapshot, which is
// the whole point: the interface draws these two from the daemon's answer, so a
// setting that does not come back is a setting that flickers on and reverts.
func stateNow(t *testing.T, l *Local) *State {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l.refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	st, err := l.State(ctx)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	return st
}
