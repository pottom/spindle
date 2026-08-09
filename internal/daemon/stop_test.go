package daemon

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"

	"github.com/pottom/spindle/internal/xdg"
)

// lockForTest is the lock a daemon would hold, in a config directory of this
// test's own.
func lockForTest(t *testing.T) *flock.Flock {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir, err := xdg.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	lock := flock.New(filepath.Join(dir, "daemon.lock"))
	held, err := lock.TryLock()
	if err != nil || !held {
		t.Fatalf("could not take the lock to hold it: %v", err)
	}
	return lock
}

// Stopping does not come back until the daemon has let go of its lock.
//
// The api is the first thing a daemon takes down and the lock is the last thing
// it releases, so a stop that waits only for the api to go quiet comes back
// while the process is still on its way out — and the daemon started straight
// afterwards is refused by it. That is restart leaving nothing running: stop,
// start, and "a daemon is already running" as the only clue.
func TestStoppingWaitsForTheLockToBeLetGo(t *testing.T) {
	lock := lockForTest(t)
	defer lock.Unlock() //nolint:errcheck // the test is ending

	waited := make(chan error, 1)
	go func() { waited <- waitFree(context.Background()) }()

	select {
	case err := <-waited:
		t.Fatalf("the wait came back while the lock was still held: %v", err)
	case <-time.After(600 * time.Millisecond):
		t.Log("still waiting, which is the point")
	}
}

// And it comes back as soon as it is free.
func TestStoppingComesBackWhenTheLockIsFree(t *testing.T) {
	lock := lockForTest(t)

	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = lock.Unlock()
	}()

	at := time.Now()
	if err := waitFree(context.Background()); err != nil {
		t.Fatalf("the wait gave up on a lock that was let go: %v", err)
	}
	t.Logf("the wait came back %s after it was let go", time.Since(at)-300*time.Millisecond)
}
