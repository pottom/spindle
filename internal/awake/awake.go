// Package awake asks the machine not to go to sleep, and lets it again.
//
// A picture on a screen in a room is watched rather than typed at, and an hour
// without a keypress is exactly what every system reads as "gone away". It dims,
// then goes dark, and the room loses a wall. Nothing about the picture is wrong
// at that point, so nothing about the picture can fix it — the machine has to be
// told.
//
// What is asked for, everywhere, is both halves: the display must stay lit, and
// the machine must not idle-sleep. On macOS playing audio already holds off the
// second, but never the first, so a Mac showing the picture still goes dark.
//
// Nothing here needs cgo. That is deliberate rather than incidental: the party
// screen has to cross-compile, and a package that pulled in C would decide that
// question for everybody.
package awake

import "sync"

var (
	mu   sync.Mutex
	stop func()
	what string

	// hold is a variable so the state around it can be tested without a machine
	// that has a screen to keep awake.
	hold = holdMachine
)

// Keep asks the machine to stay awake. Calling it while it is already held does
// nothing and is not an error, so it can be driven from a screen turning on
// without that screen having to remember whether it did so already.
func Keep() error {
	mu.Lock()
	defer mu.Unlock()
	if stop != nil {
		return nil
	}
	s, held, err := hold()
	if err != nil {
		return err
	}
	stop, what = s, held
	return nil
}

// Drop lets the machine sleep again. Safe when nothing is held.
func Drop() {
	mu.Lock()
	defer mu.Unlock()
	if stop == nil {
		return
	}
	stop()
	stop, what = nil, ""
}

// Held reports whether the machine is being kept awake.
func Held() bool {
	mu.Lock()
	defer mu.Unlock()
	return stop != nil
}

// What names the parts being held, and is the reason this is not simply a bool.
//
// On Linux the two halves are taken by different things and either can be
// missing, so "held" can mean the machine will not suspend while the screen goes
// dark anyway. Somebody watching that happen needs to be able to find out which,
// and the alternative — reporting a hold and letting them work it out — is the
// kind of silence this project keeps having to go back and undo.
func What() string {
	mu.Lock()
	defer mu.Unlock()
	return what
}
