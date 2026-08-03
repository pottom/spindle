package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pottom/spindle/internal/daemon"
)

// runDaemon runs the Connect device in the foreground until interrupted. The
// TUI starts one of these detached when it cannot find a running daemon; it is
// also perfectly reasonable to run it by hand, which is what the log is for.
func runDaemon(port int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := daemon.Run(ctx, daemon.Options{Port: port, Log: os.Stderr})
	if errors.Is(err, daemon.ErrAlreadyRunning) {
		// Nothing to complain about: the device the caller wanted exists.
		fmt.Fprintln(os.Stderr, "spindle: a daemon is already running")
		return nil
	}
	return err
}
