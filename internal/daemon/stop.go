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

	"github.com/pottom/spindle/internal/xdg"
)

// stopTimeout is how long a daemon is given to shut down politely before the
// caller is told it did not.
const stopTimeout = 5 * time.Second

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

	return waitGone(ctx)
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
