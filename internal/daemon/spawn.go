package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pottom/spindle/internal/xdg"
)

// Starting a daemon is spindle starting itself, with one argument and no
// terminal.
//
// It lives here rather than in the command line because the interface needs it
// too: a setting that cannot be heard until the device restarts is a setting
// worth being able to restart the device from, and making the screen shell out
// to the command line for that would be the same code written twice.

// Start launches a detached daemon and waits until it answers, unless one is
// already running — in which case there is nothing to do and nothing to say.
// The path of the log it writes to comes back either way, because that is where
// a failure will have gone.
func Start(ctx context.Context) (logPath string, err error) {
	logPath, log, err := openLog()
	if err != nil {
		return "", err
	}
	defer log.Close() //nolint:errcheck // the child holds its own handle

	if Running() {
		return logPath, nil
	}

	self, err := os.Executable()
	if err != nil {
		return logPath, fmt.Errorf("locate spindle: %w", err)
	}

	cmd := exec.Command(self, "daemon", "--foreground")
	cmd.Stdout, cmd.Stderr = log, log
	// Setsid detaches from the controlling terminal, so a Ctrl-C meant for the
	// shell that started it does not reach the music.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return logPath, fmt.Errorf("start daemon: %w", err)
	}
	// Nothing waits for it, so let the process table reap it rather than
	// leaving a zombie parented to a shell that has moved on.
	if err := cmd.Process.Release(); err != nil {
		return logPath, fmt.Errorf("release daemon: %w", err)
	}

	if err := WaitReady(ctx); err != nil {
		return logPath, fmt.Errorf("%w (see %s)", err, logPath)
	}
	return logPath, nil
}

// Restart stops the daemon and starts it again, which is what a setting the
// device only reads at start-up needs.
func Restart(ctx context.Context) error {
	if err := Stop(ctx); err != nil && !isNoDaemon(err) {
		return err
	}
	_, err := Start(ctx)
	return err
}

func isNoDaemon(err error) bool { return errors.Is(err, ErrNoDaemon) }

// openLog opens the file a detached daemon writes to. Without a terminal to
// print at, this is the only place its complaints can go.
func openLog() (string, *os.File, error) {
	dir, err := xdg.StateDir()
	if err != nil {
		return "", nil, err
	}

	path := filepath.Join(dir, "daemon.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return "", nil, fmt.Errorf("open daemon log: %w", err)
	}
	return path, file, nil
}
