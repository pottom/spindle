package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofrs/flock"

	"github.com/pottom/spindle/internal/xdg"
)

// stopTimeout is how long a daemon is given to shut down politely before the
// caller is told it did not.
const stopTimeout = 5 * time.Second

// shutdownGrace is how long a daemon gives its own playback loop to come back
// after it has been asked to stop, before ending the process without it. See
// leave.
//
// It has to be comfortably less than stopTimeout, because whoever asked is
// waiting on that: a daemon that takes longer to leave than the caller will
// wait is a daemon that cannot be restarted, which is the whole of the bug this
// is here for. A healthy one goes in milliseconds and never touches this.
//
// A variable so that a test need not spend two seconds watching it.
var shutdownGrace = 2 * time.Second

// ErrNoDaemon reports that there was nothing to stop.
var ErrNoDaemon = errors.New("no daemon is running")

func pidPath() (string, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

// writePID records which process holds the daemon, so it can be asked to stop
// later. The lock alone cannot say: flock leaves no trace of who holds it.
func writePID() error {
	path, err := pidPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600)
}

func removePID() {
	if path, err := pidPath(); err == nil {
		_ = os.Remove(path)
	}
}

// Stop asks the running daemon to shut down and waits for it to stop answering.
func Stop(ctx context.Context) error {
	path, err := pidPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoDaemon
	}
	if err != nil {
		return fmt.Errorf("read daemon pid: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("read daemon pid: %q is not a pid", data)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return ErrNoDaemon
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Already gone, and the file outlived it.
		removePID()
		return ErrNoDaemon
	}

	if err := waitGone(ctx); err != nil {
		return err
	}
	return waitFree(ctx)
}

// waitFree blocks until the lock a daemon holds can be taken.
//
// The api is the first thing to go and the lock is the last: between them the
// process is still letting go of the audio device and the session, and a daemon
// started in that window is refused by the one on its way out. Which is exactly
// what restart did — stop, start, and nothing running, with "a daemon is
// already running" as the only clue.
func waitFree(ctx context.Context) error {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "daemon.lock")

	ctx, cancel := context.WithTimeout(ctx, stopTimeout)
	defer cancel()

	ticker := time.NewTicker(readyPoll)
	defer ticker.Stop()

	for {
		lock := flock.New(path)
		held, err := lock.TryLock()
		if held {
			// Straight back, so that whoever is starting next can have it.
			_ = lock.Unlock()
			return nil
		}
		if err != nil {
			return fmt.Errorf("wait for the daemon to let go: %w", err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("the daemon stopped answering but did not let go within %s", stopTimeout)
		case <-ticker.C:
		}
	}
}

// waitGone blocks until nothing answers on the API any more.
func waitGone(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, stopTimeout)
	defer cancel()

	ticker := time.NewTicker(readyPoll)
	defer ticker.Stop()

	for {
		if !alive(ctx) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("the daemon did not stop within %s", stopTimeout)
		case <-ticker.C:
		}
	}
}
