package main

import (
	"fmt"
	"time"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
)

// runCrossfade reports or sets how long one track overlaps the next.
func runCrossfade(args []string) error {
	if len(args) == 0 {
		return reportCrossfade()
	}
	if len(args) > 1 {
		return fmt.Errorf("spindle crossfade takes one argument, got %d", len(args))
	}

	d, err := daemon.ParseCrossfade(args[0])
	if err != nil {
		return err
	}
	if err := auth.SaveCrossfade(args[0]); err != nil {
		return err
	}

	fmt.Printf("spindle: crossfade set to %s\n", describeCrossfade(d))
	// The overlap is fixed when the audio pipeline is built, so a running
	// daemon keeps its own until it is restarted.
	if daemon.Running() {
		fmt.Println("spindle: restart the daemon to apply it — spindle daemon stop, then spindle")
	}
	return nil
}

func reportCrossfade() error {
	value, err := auth.Crossfade()
	if err != nil {
		return err
	}
	d, err := daemon.ParseCrossfade(value)
	if err != nil {
		return err
	}

	fmt.Printf("spindle: crossfade is %s\n", describeCrossfade(d))
	fmt.Printf("spindle: set it with spindle crossfade <seconds>, up to %s, or off\n", daemon.MaxCrossfade)
	return nil
}

func describeCrossfade(d time.Duration) string {
	if d <= 0 {
		return "off — tracks run into each other as they were recorded"
	}
	return d.String()
}

// configuredCrossfade reads the overlap from the settings file.
func configuredCrossfade() (time.Duration, error) {
	value, err := auth.Crossfade()
	if err != nil {
		return 0, err
	}
	return daemon.ParseCrossfade(value)
}
