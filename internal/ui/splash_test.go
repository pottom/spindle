package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

func splashModel(t *testing.T, w, rows int) Model {
	t.Helper()
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.width, m.height = w, rows
	m.tab = tabPlayer
	m.noDevice = true
	return m
}

// The picture is up while the device is being waited for, and gone when it is
// not.
//
// The wait is real and cannot be argued away — the interface draws in a quarter
// of a second and the device answers a second later, or twenty after the machine
// has been asleep — so the screen that says "no device yet" is given the logo
// rather than left blank above a list.
func TestTheLogoFillsTheWait(t *testing.T) {
	m := splashModel(t, 120, 40)
	m.splashFlow()
	if len(m.splashRows()) == 0 {
		t.Fatal("nothing was drawn while the device was being waited for")
	}
	w, rows := m.splashRoom()
	if got := len(m.splashRows()); got > rows+1 {
		t.Errorf("the logo came out %d rows in a box of %d", got, rows)
	}
	for _, line := range m.splashRows() {
		if n := lipgloss.Width(line); n > w {
			t.Errorf("a row of the logo is %d cells wide in a box of %d", n, w)
		}
	}

	// The device arrives with a record on it: the wait is over and the picture
	// goes, properly rather than being left in the terminal for whatever comes
	// next to draw over.
	m.noDevice = false
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: time.Minute}
	m.splashFlow()
	if len(m.splashRows()) != 0 {
		t.Error("the logo stayed after the device arrived")
	}

	// A logo above a list of speakers is a logo in somebody's way.
	m.ps, m.noDevice, m.devices.open = nil, true, true
	m.splashFlow()
	if len(m.splashRows()) != 0 {
		t.Error("the logo was up over the device picker")
	}

	// The help screen has it whatever is playing: that is where a picture of
	// the program belongs, and it is the answer to "what is this". It wants a
	// tall terminal — the keys alone lay out in 36 rows.
	m.devices.open = false
	m.tab = tabHelp
	m.height = 80
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: time.Minute}
	m.splashFlow()
	if len(m.splashRows()) == 0 {
		t.Error("the help screen had no picture of the program on it")
	}
}

// A terminal with no room for it gets the words instead.
func TestTheLogoKnowsWhenThereIsNoRoom(t *testing.T) {
	m := splashModel(t, 30, 12)
	m.splashFlow()
	if len(m.splashRows()) != 0 {
		t.Error("the logo was drawn into a box too small to read it in")
	}
	// The panel still says what it has to say.
	panel := strings.Join(m.noDevicePanel(m.layout(), 10), "\n")
	if !strings.Contains(panel, "No active playback device") {
		t.Error("the panel lost its words")
	}
}

// The picture is kept up to date by the ordinary tick.
//
// It hung off a frame of the visualiser first, and that branch leaves at once
// when the trace is not on screen — which is the very case the picture exists
// for, so it never ran once. A test that only called splashFlow would not have
// caught it; this one goes through Update.
func TestTheLogoIsDrawnByTheTick(t *testing.T) {
	m := splashModel(t, 120, 40)
	if len(m.splashRows()) != 0 {
		t.Fatal("something was rendered before a single tick")
	}
	out, _ := m.Update(msg.Tick{})
	got, ok := out.(Model)
	if !ok {
		t.Fatal("the update did not hand back a model")
	}
	if len(got.splashRows()) == 0 {
		t.Error("a tick on the screen the logo is for drew nothing")
	}
}

// The picture is on screen in the second the program starts, not a second later.
//
// It is brought up to date after every message rather than on the tick, and this
// is why: the tick fires once a second and the wait is about a second long, so
// the first tick that could have drawn it arrived after the state it was waiting
// for. This test does what a real start does — a size, and then nothing — and
// looks at what View actually put out.
func TestTheLogoIsUpBeforeTheFirstTick(t *testing.T) {
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.tab = tabPlayer

	out, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got, ok := out.(Model)
	if !ok {
		t.Fatal("the update did not hand back a model")
	}
	if got.ps != nil {
		t.Fatal("this test wants the moment before any state has arrived")
	}
	if n := len(got.splashRows()); n == 0 {
		t.Fatal("the size arrived and nothing was rendered")
	}

	// And it is in what the screen actually shows, not only in the model.
	screen := fmt.Sprint(got.View())
	first := got.splashRows()[len(got.splashRows())/2]
	if strings.TrimSpace(first) == "" {
		t.Fatal("the middle row of the logo is blank")
	}
	if !strings.Contains(screen, strings.TrimSpace(first)) {
		t.Error("the logo was rendered but never reached the screen")
	}
	if !strings.Contains(screen, "Connecting") {
		t.Error("the screen stopped saying what it is waiting for")
	}
}

// The logo is a mark on the waiting screen, not the waiting screen.
//
// It used to take everything left over, which measured 88 to 95 per cent of the
// body at four terminal sizes. That was right for the old logo — a wide banner —
// and wrong for a square one: given the whole height it stops being a large logo
// and becomes a wall, with the line saying what is being waited for reading as a
// caption on a poster.
func TestTheLogoDoesNotTakeTheWholeWaitingScreen(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {120, 40}, {160, 50}, {200, 60}, {300, 90}} {
		m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
		m.width, m.height = size[0], size[1]
		m.tab, m.ps = tabPlayer, nil

		l := m.layout()
		_, rows := m.splashRoom()

		if rows > splashTallest {
			t.Errorf("%dx%d: the logo is given %d rows, past the ceiling of %d",
				size[0], size[1], rows, splashTallest)
		}
		if share := 100 * rows / max(l.bodyHeight, 1); share > 50 {
			t.Errorf("%dx%d: the logo is given %d%% of the body", size[0], size[1], share)
		}
		// And it is still worth drawing: a mark, not a smudge.
		if rows < splashLeastH && l.bodyHeight > 2*splashLeastH {
			t.Errorf("%dx%d: the logo is down to %d rows on a body of %d",
				size[0], size[1], rows, l.bodyHeight)
		}
	}
}

// The help screen keeps its own half, which is a separate decision.
func TestTheAboutPageIsNotCappedWithTheWaitingScreen(t *testing.T) {
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.width, m.height = 200, 60
	m.tab = tabHelp

	_, rows := m.splashRoom()
	if want := m.layout().bodyHeight * splashAboutShare / 100; rows != want {
		t.Errorf("the About picture is given %d rows, want its own %d", rows, want)
	}
	if rows <= splashTallest {
		t.Errorf("the About picture was capped with the waiting screen: %d rows", rows)
	}
}
