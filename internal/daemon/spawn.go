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

// ErrSigningIn reports that the daemon came up and is waiting to be signed in
// to Spotify before it can play anything. The link is daemon.Waiting.
var ErrSigningIn = errors.New("the device is waiting to be signed in to Spotify")

// Start launches a detached daemon and waits until it answers, unless one is
// already running — in which case there is nothing to do and nothing to say.
// The path of the log it writes to comes back either way, because that is where
// a failure will have gone.
func Start(ctx context.Context) (logPath string, err error) {
	logPath, started, err := Spawn()
	if err != nil || !started {
		return logPath, err
	}

	if err := WaitReady(ctx); err != nil {
		// A daemon waiting for a person is not a daemon that failed. It has the
		// port, it has the lock, and it will play as soon as the sign-in it is
		// waiting on is finished — so it is reported as itself rather than as
		// twenty seconds of silence. See signin.go.
		if Waiting() != "" {
			return logPath, ErrSigningIn
		}
		return logPath, fmt.Errorf("%w (see %s)", err, logPath)
	}
	return logPath, nil
}

// Spawn launches a detached daemon and comes straight back, saying whether it
// launched one at all — false means one was already answering.
//
// Waiting is separate from launching because the interface must not wait. The
// daemon's API only answers once it has reached Spotify's access point, and
// that is not always quick: after the machine has been asleep the name lookups
// fail for a while and the login retries, which was measured taking the whole
// of the twenty seconds WaitReady allows. All of that happened before Bubble
// Tea took the terminal, so what the listener saw was a shell prompt and
// nothing else — the interface had not started, and nothing said why.
//
// It does not need to wait. The screen draws from whatever the Web API says
// until the device is there, and the local player reconnects on its own, so the
// daemon arriving late costs nothing but the moment it takes to arrive.
func Spawn() (logPath string, started bool, err error) {
	logPath, log, err := openLog()
	if err != nil {
		return "", false, err
	}
	defer log.Close() //nolint:errcheck // the child holds its own handle

	if Running() {
		return logPath, false, nil
	}

	self, err := os.Executable()
	if err != nil {
		return logPath, false, fmt.Errorf("locate spindle: %w", err)
	}

	cmd := exec.Command(self, "daemon", "--foreground")
	cmd.Stdout, cmd.Stderr = log, log
	// Setsid detaches from the controlling terminal, so a Ctrl-C meant for the
	// shell that started it does not reach the music.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return logPath, false, fmt.Errorf("start daemon: %w", err)
	}
	// Nothing waits for it, so let the process table reap it rather than
	// leaving a zombie parented to a shell that has moved on.
	if err := cmd.Process.Release(); err != nil {
		return logPath, true, fmt.Errorf("release daemon: %w", err)
	}
	return logPath, true, nil
}

// Restart stops the daemon and starts it again, which is what a setting the
// device only reads at start-up needs.
func Restart(ctx context.Context) error {
	if err := Stop(ctx); err != nil && !isNoDaemon(err) {
		return err
	}
	_, err := Start(ctx)
	// The device is there and the sign-in is not this caller's to finish: the
	// screen that asked for a restart got one. The daemon's own log, and
	// whoever started it from a terminal, say the rest.
	if errors.Is(err, ErrSigningIn) {
		return nil
	}
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
