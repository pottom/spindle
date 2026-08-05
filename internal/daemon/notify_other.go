//go:build !darwin

package daemon

import (
	"context"
	"os/exec"
	"time"
)

// notify posts a desktop notification through notify-send, which is what a
// freedesktop session offers and what is already installed wherever one is
// running. Anywhere else this is a no-op: a player that fails to notify is
// still a player, and there is nothing sensible to fall back to.
func notify(title, body string) {
	if title == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	_ = exec.CommandContext(ctx, "notify-send", "--app-name=spindle", title, body).Run()
}

// notifyTimeout keeps a wedged helper from piling up goroutines.
const notifyTimeout = 5 * time.Second
