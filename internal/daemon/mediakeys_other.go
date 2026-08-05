//go:build !darwin

package daemon

import "context"

// The media keys are answered on macOS only. Linux has a protocol of its own
// for this — MPRIS, over the session bus — which is a different piece of work
// and not one to fake with an event tap.
func watchMediaKeys(_ context.Context, _ string, logf func(string, ...any)) {
	logf("the media keys are not answered on this system")
}
