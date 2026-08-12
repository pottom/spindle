package awake

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

// The flags SetThreadExecutionState takes. Continuous makes the request stand
// until it is withdrawn rather than counting as a single nudge.
const (
	esContinuous      = 0x80000000
	esSystemRequired  = 0x00000001
	esDisplayRequired = 0x00000002
)

var setThreadExecutionState = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetThreadExecutionState")

// holdMachine asks Windows to keep the display and the machine up.
//
// The state belongs to a thread rather than to the process, so it is set on a
// thread of its own that is locked to its OS thread and does nothing else until
// it is time to let go. That is also what makes this the one platform with
// nothing to clean up after a crash: the state dies with the process, so there
// is no equivalent of caffeinate's -w to get wrong.
func holdMachine() (func(), string, error) {
	done := make(chan struct{})
	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if r, _, err := setThreadExecutionState.Call(esContinuous | esSystemRequired | esDisplayRequired); r == 0 {
			ready <- fmt.Errorf("SetThreadExecutionState: %w", err)
			return
		}
		ready <- nil

		<-done
		setThreadExecutionState.Call(esContinuous) //nolint:errcheck // letting go, and nothing to do if it fails
	}()

	if err := <-ready; err != nil {
		return nil, "", err
	}
	return func() { close(done) }, "display and system", nil
}
