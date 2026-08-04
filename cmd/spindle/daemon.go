package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
	"github.com/pottom/spindle/internal/xdg"
)

// foregroundFlag is what a detached daemon is re-executed with. It is also what
// anyone supervising spindle themselves — systemd, launchd, a terminal — wants,
// since those expect a process that stays put.
const foregroundFlag = "--foreground"

// runDaemon starts the Connect device. Without --foreground it detaches and
// returns straight away, which is what "daemon" ought to mean.
func runDaemon(args []string) error {
	foreground := false
	for _, arg := range args {
		switch arg {
		case foregroundFlag, "-f":
			foreground = true
		case "stop":
			return stopDaemon()
		default:
			return fmt.Errorf("unknown option %q for spindle daemon", arg)
		}
	}

	if !foreground {
		return detachDaemon()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	quality, err := configuredQuality()
	if err != nil {
		return err
	}

	crossfade, err := configuredCrossfade()
	if err != nil {
		return err
	}

	err = daemon.Run(ctx, daemon.Options{Log: os.Stderr, Quality: quality, Crossfade: crossfade})
	if errors.Is(err, daemon.ErrAlreadyRunning) {
		// Nothing to complain about: the device the caller wanted exists.
		fmt.Fprintln(os.Stderr, "spindle: a daemon is already running")
		return nil
	}
	return err
}

// stopDaemon asks a running daemon to leave, and says so when there is none:
// the user asked for silence, and silence is what they got either way.
func stopDaemon() error {
	err := daemon.Stop(context.Background())
	if errors.Is(err, daemon.ErrNoDaemon) {
		fmt.Println("spindle: no daemon is running")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Println("spindle: daemon stopped")
	return nil
}

// configuredQuality reads the audio quality from the settings file.
func configuredQuality() (daemon.Quality, error) {
	name, err := auth.Quality()
	if err != nil {
		return "", err
	}
	return daemon.ParseQuality(name)
}

// detachDaemon re-executes spindle in the background, in its own process group
// so that closing the terminal or interrupting the parent leaves it playing.
func detachDaemon() error {
	if daemon.Running() {
		fmt.Println("spindle: a daemon is already running")
		return nil
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate spindle: %w", err)
	}

	logPath, log, err := daemonLog()
	if err != nil {
		return err
	}
	defer log.Close() //nolint:errcheck // the child holds its own handle

	cmd := exec.Command(self, "daemon", foregroundFlag)
	cmd.Stdout, cmd.Stderr = log, log
	// Setsid detaches from the controlling terminal, so a Ctrl-C meant for the
	// shell that started it does not reach the music.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	// Nothing waits for it, so let the process table reap it rather than
	// leaving a zombie parented to a shell that has moved on.
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release daemon: %w", err)
	}

	if err := daemon.WaitReady(context.Background()); err != nil {
		return fmt.Errorf("%w (see %s)", err, logPath)
	}

	fmt.Printf("spindle: daemon started, logging to %s\n", logPath)
	return nil
}

// daemonLog opens the log a detached daemon writes to. Without a terminal to
// print at, this is the only place its complaints can go.
func daemonLog() (string, *os.File, error) {
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
