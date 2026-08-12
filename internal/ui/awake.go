package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/awake"
)

// keepAwake turns the machine's sleep off while the picture is up, and back on
// when it comes down.
//
// It is a command rather than something Update does directly because it starts
// a process and kills one, and Update is not the place for that.
//
// Tied to the picture rather than to spindle running. Somebody left on the queue
// screen has walked away from a list, and a list is not worth stopping a machine
// from sleeping over; somebody who put the picture up is watching it from across
// a room, and every minute they do not touch the keyboard is the point rather
// than a sign they have gone.
//
// A machine that will not be held — no systemd, a caffeinate that would not
// start — is left alone. The picture is still worth having on a screen that
// eventually dims, and nothing here is worth an error in front of it. The state
// is on the debug screen for anyone who wonders.
func keepAwake(on bool) tea.Cmd {
	return func() tea.Msg {
		if on {
			_ = awake.Keep()
		} else {
			awake.Drop()
		}
		return nil
	}
}
