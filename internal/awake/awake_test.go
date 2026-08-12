package awake

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
)

// The screen turning on drives Keep without knowing whether it is already held —
// every keypress on the picture could plausibly ask again — so asking twice must
// take one hold, not two, and letting go once must be enough to release it.
func TestAskingTwiceTakesOneHold(t *testing.T) {
	held, released := 0, 0
	swap(t, func() (func(), string, error) {
		held++
		return func() { released++ }, "display and idle", nil
	})

	for range 3 {
		if err := Keep(); err != nil {
			t.Fatalf("Keep: %v", err)
		}
	}
	if held != 1 {
		t.Errorf("took %d holds for three asks, want 1", held)
	}
	if !Held() {
		t.Error("not held after asking")
	}
	if What() != "display and idle" {
		t.Errorf("holding %q, want the name the machine gave", What())
	}

	Drop()
	Drop()
	if released != 1 {
		t.Errorf("released %d times, want 1", released)
	}
	if Held() {
		t.Error("still held after dropping")
	}
	if What() != "" {
		t.Errorf("still says it is holding %q after dropping", What())
	}
}

// A machine that cannot be kept awake — no systemd, no caffeinate — must leave
// nothing held, so that a later ask tries again rather than believing itself
// covered by a hold that never happened.
func TestAMachineThatRefusesIsNotHeld(t *testing.T) {
	swap(t, func() (func(), string, error) { return nil, "", errors.New("no such thing here") })

	if err := Keep(); err == nil {
		t.Fatal("Keep said nothing went wrong on a machine that refused")
	}
	if Held() {
		t.Error("held, after the hold failed")
	}
}

// And the picture goes down again after a refusal without anything to undo.
func TestDroppingWhatWasNeverHeld(t *testing.T) {
	swap(t, func() (func(), string, error) { return nil, "", errors.New("no") })
	Drop()
}

func swap(t *testing.T, with func() (func(), string, error)) {
	t.Helper()
	was := hold
	hold = with
	t.Cleanup(func() {
		Drop()
		hold = was
	})
}

// The tests above swap the machine out, which leaves the one thing that can
// really go wrong untested: whether the hold this machine actually takes works,
// and whether letting go really lets go. Skipped by default because it starts a
// process and briefly stops the machine sleeping.
//
//	SPINDLE_LIVE=1 go test ./internal/awake/
func TestLiveHoldOnThisMachine(t *testing.T) {
	if os.Getenv("SPINDLE_LIVE") == "" {
		t.Skip("set SPINDLE_LIVE to hold this machine awake for a moment")
	}
	if err := Keep(); err != nil {
		t.Fatalf("Keep: %v", err)
	}
	if !holding(t) {
		t.Error("nothing is holding the machine awake")
	}
	Drop()
	if holding(t) {
		t.Error("something is still holding the machine awake after dropping")
	}
}

// holding asks the system whether anything of ours is holding it, by the same
// route a person would: the process list.
func holding(t *testing.T) bool {
	t.Helper()
	name := map[string]string{"darwin": "caffeinate", "linux": "systemd-inhibit"}[runtime.GOOS]
	if name == "" {
		t.Skipf("no way to check this on %s", runtime.GOOS)
	}
	out, _ := exec.Command("pgrep", "-f", name+".*"+strconv.Itoa(os.Getpid())).Output()
	return len(bytes.TrimSpace(out)) > 0
}
