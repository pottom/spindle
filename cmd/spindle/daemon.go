package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
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

		// "spindle daemon" starts one, and so does "spindle daemon start" —
		// which is what everybody types, and what used to be answered with an
		// error and no daemon. A command that refuses the obvious word for what
		// it does is a command that looks broken.
		case "start":

		case "stop":
			return stopDaemon()
		case "restart":
			if err := stopDaemon(); err != nil {
				return err
			}
			return detachDaemon()
		case "status":
			return reportDaemon()

		default:
			return fmt.Errorf("unknown option %q for spindle daemon: want start, stop, restart or status", arg)
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

	notify, err := auth.Notify()
	if err != nil {
		return err
	}

	err = daemon.Run(ctx, daemon.Options{
		Log:       os.Stderr,
		Quality:   quality,
		Crossfade: crossfade,
		Notify:    notify,
	})
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

// reportDaemon says whether one is running, for a script or for somebody who
// has lost track of it. The answer leaves through the exit code as well, and
// through the same one the other commands use for it: nothing went wrong, the
// answer is simply no.
func reportDaemon() error {
	if !daemon.Running() {
		fmt.Println("spindle: no daemon is running")
		os.Exit(exitNoDaemon)
	}
	fmt.Println("spindle: a daemon is running")
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

// detachDaemon starts one in the background and says where it went. The work
// is in the daemon package, because the interface starts one too: a setting the
// device only reads at start-up wants a way to restart it from the screen it
// was changed on.
func detachDaemon() error {
	if daemon.Running() {
		fmt.Println("spindle: a daemon is already running")
		return nil
	}

	logPath, err := daemon.Start(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("spindle: daemon started, logging to %s\n", logPath)
	return nil
}
