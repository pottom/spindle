package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// briefly makes the grace short enough that a test need not sit through it.
func briefly(t *testing.T) {
	t.Helper()
	grace := shutdownGrace
	t.Cleanup(func() { shutdownGrace = grace })
	shutdownGrace = 20 * time.Millisecond
}

// answered returns what leave decided, or fails the test if it decided nothing
// in time. Every case here is meant to be over in milliseconds; a leave that
// blocks is the bug, so waiting for it is not a kindness.
func answered(t *testing.T, run func() error) error {
	t.Helper()
	out := make(chan error, 1)
	go func() { out <- run() }()
	select {
	case err := <-out:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("leave did not come back: the daemon is holding its lock and cannot be stopped")
		return nil
	}
}

// A daemon that is asked to stop and stops. Nothing to say, and nothing to
// wait for.
func TestLeavingWhenTheLoopComesBack(t *testing.T) {
	briefly(t)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	var out strings.Builder

	cancel()
	done <- nil

	if err := answered(t, func() error {
		return leave(ctx, done, make(chan error), newLogger(&out, nil))
	}); err != nil {
		t.Errorf("leaving answered %v, want nothing wrong with it", err)
	}
	if out.Len() != 0 {
		t.Errorf("said %q about an ordinary shutdown", out.String())
	}
}

// And one that is asked to stop and cannot. This is the sign-in still waiting
// for its redirect: go-librespot closes the API on the context and then goes on
// waiting, so the process was left with no API, its lock still held, and no way
// out — which is what "the daemon stopped answering but did not let go within
// 5s" was, and why restart put nothing back.
func TestLeavingWithoutALoopThatWillNotCome(t *testing.T) {
	briefly(t)

	ctx, cancel := context.WithCancel(context.Background())
	var out strings.Builder

	cancel()

	// Never written to: the loop is not coming.
	if err := answered(t, func() error {
		return leave(ctx, make(chan error), make(chan error), newLogger(&out, nil))
	}); err != nil {
		t.Errorf("leaving answered %v, want it to go anyway", err)
	}
	if !strings.Contains(out.String(), "did not finish") {
		t.Errorf("said %q, want it to say what it left behind", out.String())
	}
}

// Nothing about stopping changes what happens when the loop simply fails.
func TestTheLoopsOwnFailureIsReported(t *testing.T) {
	briefly(t)

	done := make(chan error, 1)
	done <- errors.New("no such audio device")

	err := answered(t, func() error {
		return leave(context.Background(), done, make(chan error), newLogger(&strings.Builder{}, nil))
	})
	if err == nil || !strings.Contains(err.Error(), "no such audio device") {
		t.Errorf("leaving answered %v, want the loop's own complaint", err)
	}
}

// A context that ended is how a shutdown reaches the loop, so the loop saying
// so is the shutdown working, not a failure to report.
func TestACancelledLoopIsNotAFailure(t *testing.T) {
	briefly(t)

	done := make(chan error, 1)
	done <- context.Canceled

	if err := answered(t, func() error {
		return leave(context.Background(), done, make(chan error), newLogger(&strings.Builder{}, nil))
	}); err != nil {
		t.Errorf("leaving answered %v, want a cancelled loop taken as a shutdown", err)
	}
}

// The watchdog's verdict is passed on, because the process ending is what lets
// a fresh device take over. See watchdog.go.
func TestTheWatchdogsVerdictIsPassedOn(t *testing.T) {
	briefly(t)

	wedge := make(chan error, 1)
	wedge <- ErrWedged

	err := answered(t, func() error {
		return leave(context.Background(), make(chan error), wedge, newLogger(&strings.Builder{}, nil))
	})
	if !errors.Is(err, ErrWedged) {
		t.Errorf("leaving answered %v, want the watchdog's verdict", err)
	}
}
