package main

import (
	"fmt"
	"os"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
)

// runQuality reports or sets the audio quality. Without an argument it prints
// the current setting, so the answer to "what am I listening to" does not
// require opening a JSON file.
func runQuality(args []string) error {
	if len(args) == 0 {
		return reportQuality()
	}
	if len(args) > 1 {
		return fmt.Errorf("spindle quality takes one argument, got %d", len(args))
	}

	q, err := daemon.ParseQuality(args[0])
	if err != nil {
		return err
	}
	if err := auth.SaveQuality(string(q)); err != nil {
		return err
	}

	fmt.Printf("spindle: quality set to %s (%d kbps)\n", q, q.Bitrate())
	// The bitrate is chosen when the session opens, so a running daemon keeps
	// streaming at the old one until it is restarted.
	if daemon.Running() {
		fmt.Println("spindle: restart the daemon to apply it — spindle daemon stop, then spindle")
	}
	return nil
}

func reportQuality() error {
	name, err := auth.Quality()
	if err != nil {
		return err
	}
	q, err := daemon.ParseQuality(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spindle: %v; using %s\n", err, daemon.DefaultQuality)
		q = daemon.DefaultQuality
	}

	fmt.Printf("quality:  %s (%d kbps)\n", q, q.Bitrate())
	fmt.Printf("settings: %s\n", auth.SettingsPath())
	fmt.Println("set it:   spindle quality low | middle | high")
	return nil
}
