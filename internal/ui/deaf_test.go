package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/pottom/spindle/internal/player"
)

// A device that has lost touch with Spotify goes on playing and goes on
// answering, so nothing on the screen looks wrong — it is simply out of Connect's
// reach until it is started again. It said so in the daemon's log and nowhere a
// listener would ever look.
func TestADeafDeviceSaysSoOnScreen(t *testing.T) {
	m := playerModel()
	m.width, m.height = 120, 40
	m.resize()

	if strings.Contains(ansi.Strip(m.render()), "lost touch") {
		t.Fatal("a healthy device was reported as deaf")
	}

	m.ps.Deaf = []string{"the dealer's messages"}
	screen := ansi.Strip(m.render())
	if !strings.Contains(screen, "lost touch") {
		t.Errorf("a device that has gone deaf says nothing:\n%s", screen)
	}
	if !strings.Contains(screen, "restart") {
		t.Error("the line does not say the one thing that mends it")
	}
}

// News about this moment outranks it: a skipped track and a throttle are things
// that just happened, and a deaf device will still be deaf a second later.
func TestWhatJustHappenedOutranksIt(t *testing.T) {
	m := playerModel()
	m.width, m.height = 120, 40
	m.resize()
	m.ps.Deaf = []string{"the accesspoint"}
	m.skipped, m.skippedAt = "t9", time.Now()

	if got, _, _ := m.notice(); !strings.Contains(got, "would not play") {
		t.Errorf("the notice says %q, want the track that was skipped", got)
	}
}

// And nothing that is not our own daemon can say it at all.
func TestOtherBackendsAreNeverDeaf(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "one", Playing: true}
	if m.deviceDeaf() {
		t.Error("a backend that never reports this was called deaf")
	}
}
