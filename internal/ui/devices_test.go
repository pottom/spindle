package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func devices(active string, names ...string) []player.Device {
	out := make([]player.Device, 0, len(names))
	for _, n := range names {
		out = append(out, player.Device{ID: n, Name: n, Type: "computer", Active: n == active})
	}
	return out
}

// Devices come back from Spotify in whatever order it feels like. A cursor that
// stays on an index rather than on a device is how you transfer to the wrong
// speaker.
func TestDeviceCursorFollowsTheDevice(t *testing.T) {
	var p devicePane
	p.adopt(devices("", "laptop", "phone", "speaker"))
	p.cursor.move(2, 3) // on "speaker"

	p.adopt(devices("", "phone", "speaker", "laptop"))

	if sel := p.selected(); sel == nil || sel.ID != "speaker" {
		t.Errorf("cursor landed on %v, want speaker", sel)
	}
}

// With nothing to hold on to, start where playback already is.
func TestDeviceCursorStartsOnTheActiveOne(t *testing.T) {
	var p devicePane
	p.adopt(devices("phone", "laptop", "phone", "speaker"))

	if sel := p.selected(); sel == nil || sel.ID != "phone" {
		t.Errorf("cursor started on %v, want the active device", sel)
	}
}

func TestDeviceCursorSurvivesADeviceVanishing(t *testing.T) {
	var p devicePane
	p.adopt(devices("", "laptop", "phone", "speaker"))
	p.cursor.move(2, 3)

	p.adopt(devices("", "laptop"))

	if sel := p.selected(); sel == nil || sel.ID != "laptop" {
		t.Errorf("cursor landed on %v, want laptop", sel)
	}
}

func TestDeviceSelectionOnEmptyList(t *testing.T) {
	var p devicePane
	p.adopt(nil)
	if sel := p.selected(); sel != nil {
		t.Errorf("selected = %v, want nil", sel)
	}
}

// Opening and closing the picker must not disturb what is playing.
func TestPickerTogglesWithD(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Title: "something"}
	m.devices.adopt(devices("laptop", "laptop", "phone"))

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !tm.(Model).devices.open {
		t.Fatal("d did not open the picker")
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if tm.(Model).devices.open {
		t.Error("esc did not close the picker")
	}
}

// The no-device screen is the same list with nothing behind it, so esc has
// nothing to close and must not make it disappear.
func TestNoDeviceScreenCannotBeClosed(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	var tm tea.Model = m
	tm, _ = tm.Update(msg.NoActiveDevice{})
	tm, _ = tm.Update(msg.DevicesFetched{Devices: devices("", "laptop", "phone")})

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !tm.(Model).noDevice {
		t.Error("esc dismissed a screen with nothing behind it")
	}

	// The arrows still have to work: this screen is where you choose.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if sel := tm.(Model).devices.selected(); sel == nil || sel.ID != "phone" {
		t.Errorf("cursor on %v, want phone", sel)
	}
}

// The no-device screen offers q in its help bar, and has nothing behind it to
// go back to: swallowing the key left no way out of the program at all.
func TestQuitWorksOnTheNoDeviceScreen(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.noDevice = true
	m.width, m.height = 100, 30
	m.resize()

	for _, k := range []tea.KeyPressMsg{
		{Code: 'q', Text: "q"},
		{Code: 'c', Mod: tea.ModCtrl},
	} {
		if _, cmd := m.deviceKey(k); cmd {
			t.Errorf("%v was swallowed by the device screen", k)
		}
	}

	// And it reaches the quit case for real.
	var tm tea.Model = m
	if _, cmd := tm.Update(tea.KeyPressMsg{Code: 'q', Text: "q"}); cmd == nil {
		t.Error("q on the no-device screen quits nothing")
	}
}

// The picker and the queue cannot both have the keyboard: the help bar and the
// arrow keys drive the device list, so the device list is what must be drawn.
func TestTheDevicePickerIsDrawnOverEveryTab(t *testing.T) {
	for _, tab := range []tabID{tabPlayer, tabQueue, tabLibrary} {
		m := New(player.NewMock(), nil, defaultTestCell)
		m.ps = &player.State{TrackID: "a", Title: "a", Playing: true}
		m.tab = tab
		m.devices.open = true
		m.devices.items = []player.Device{{ID: "d1", Name: "kitchen speaker"}}
		m.width, m.height = 120, 36
		m.resize()

		if got := plain(m.render()); !strings.Contains(got, "kitchen speaker") {
			t.Errorf("tab %d: the picker has the keyboard but is not on screen:\n%s", tab, got)
		}
	}
}

// Choosing a device moves the music without changing what it is doing: it is a
// list of speakers, not a play button. Spotify's own clients pass the state
// through, and so does spotify-player.
func TestChoosingADeviceKeepsTheState(t *testing.T) {
	for _, playing := range []bool{true, false} {
		p := &recordingTransfer{Player: player.NewMock()}
		m := New(p, nil, defaultTestCell)
		m.devices.items = devices("", "spindle")
		m.devices.open = true
		m.ps = &player.State{TrackID: "t1", Title: "one", Playing: playing}

		cmd := m.transfer()
		if cmd == nil {
			t.Fatal("choosing a device did nothing")
		}
		runControls(cmd)

		if p.asked != playing {
			t.Errorf("with playing = %v the transfer asked for %v", playing, p.asked)
		}
	}
}

// recordingTransfer remembers what the transfer asked the music to do.
type recordingTransfer struct {
	player.Player
	asked bool
}

func (r *recordingTransfer) TransferTo(ctx context.Context, id string, playing bool) error {
	r.asked = playing
	return r.Player.TransferTo(ctx, id, playing)
}

// ownDaemon is a backend with a playback device of its own, the way the local
// daemon is.
type ownDaemon struct {
	player.Player
	id       string
	moved    string
	askedFor bool
}

func (o *ownDaemon) OwnDevice() string { return o.id }

func (o *ownDaemon) TransferTo(ctx context.Context, id string, playing bool) error {
	o.moved, o.askedFor = id, playing
	return o.Player.TransferTo(ctx, id, playing)
}

// Nobody should have to pick their own machine out of a list of speakers: the
// daemon is spindle's own, spindle started it, and the device list is the only
// reason it is on screen.
func TestSpindleTakesTheDeviceItStarted(t *testing.T) {
	p := &ownDaemon{Player: player.NewMock(), id: "own"}
	m := New(p, nil, defaultTestCell)
	m.noDevice = true
	m.devices.adopt([]player.Device{{ID: "own", Name: "spindle"}, {ID: "phone", Name: "phone"}})

	cmd := m.takeOwnDevice()
	if cmd == nil {
		t.Fatal("the device spindle started was left for somebody to pick by hand")
	}
	runControls(cmd)

	if p.moved != "own" {
		t.Errorf("moved playback to %q, want spindle's own device", p.moved)
	}
	if p.askedFor {
		t.Error("taking the device asked for playback, want it to stay as it was")
	}

	// Only once: a device deliberately left for another speaker stays left.
	if cmd := m.takeOwnDevice(); cmd != nil {
		t.Error("the device was claimed a second time")
	}
}

// Music coming out of a phone stays there. A transfer carries the state across,
// so taking a playing session would move it here — which is why this only
// happens with nothing playing anywhere.
func TestNothingIsTakenWhileSomethingPlays(t *testing.T) {
	p := &ownDaemon{Player: player.NewMock(), id: "own"}
	m := New(p, nil, defaultTestCell)
	m.noDevice = false
	m.devices.adopt([]player.Device{{ID: "own", Name: "spindle"}})

	if cmd := m.takeOwnDevice(); cmd != nil {
		t.Error("playback was taken from a device that had it")
	}
}

// The daemon takes a few seconds to register with Spotify, and until it does
// there is nothing to take.
func TestNothingIsTakenBeforeTheDaemonAppears(t *testing.T) {
	p := &ownDaemon{Player: player.NewMock(), id: ""}
	m := New(p, nil, defaultTestCell)
	m.noDevice = true
	m.devices.adopt([]player.Device{{ID: "phone", Name: "phone"}})

	if cmd := m.takeOwnDevice(); cmd != nil {
		t.Error("something was taken before the daemon had registered")
	}
	if m.tookOwnDevice {
		t.Error("the one chance was spent on a device that was not there")
	}
}
