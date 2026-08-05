package main

import (
	"fmt"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
)

// runNotify turns the announcements on or off, and says which they are.
func runNotify(args []string) error {
	if len(args) == 0 {
		return reportNotify()
	}
	if len(args) > 1 {
		return fmt.Errorf("spindle notify takes one argument, got %d", len(args))
	}

	var on bool
	switch args[0] {
	case "on":
		on = true
	case "off":
	default:
		return fmt.Errorf("unknown setting %q: want on or off", args[0])
	}

	if err := auth.SaveNotify(on); err != nil {
		return err
	}
	fmt.Printf("spindle: track notifications %s\n", stateWord(on))

	// The announcer is started with the daemon and watches its event stream,
	// so a running one keeps doing whatever it was doing until it is restarted.
	if daemon.Running() {
		fmt.Println("spindle: restart the daemon to apply it — spindle daemon stop, then spindle")
	}
	return nil
}

func reportNotify() error {
	on, err := auth.Notify()
	if err != nil {
		return err
	}
	fmt.Printf("spindle: track notifications are %s\n", stateWord(on))
	fmt.Println("spindle: set it with spindle notify on, or spindle notify off")
	return nil
}

func stateWord(on bool) string {
	if on {
		return "on"
	}
	return "off"
}
