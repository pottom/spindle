package ui

import (
	"bytes"
	"image"
	_ "image/png"
	"strings"
	"sync"

	"github.com/pottom/spindle/internal/logo"
)

// What there is to look at while there is nothing to look at.
//
// Starting spindle has a wait in it that cannot be argued away: the interface
// draws in about a quarter of a second, and the playback device takes another
// second to answer — longer after the machine has been asleep, because the name
// lookups fail for a while and the login retries. Measured, that can be the
// whole twenty seconds the daemon is given.
//
// So the screen that says "no device yet" gets the picture, as large as it will
// go, and it goes away by itself the moment there is a device. No timer and no
// key: the wait is the thing it is for, and the end of the wait is the end of
// it.
//
// Carried in the binary rather than fetched. It is under two megabytes of a
// thirty megabyte program, and the moment it exists for is the moment the
// network is not answering — a logo downloaded on demand would be missing
// exactly when it was wanted.
//
// It has a transparent background, so what is round it is whatever the terminal
// is. The kitty renderer hands the alpha to the terminal and it composites;
// the halfblock renderer has to do it itself, because a cell can only be one
// colour — see Halfblock.behind, and the background the terminal is asked for
// when the interface starts.

// splashSlot is which picture on the screen this is, kept apart from the three
// the artwork uses so that a renderer holding images in the terminal does not
// have them replacing each other. See cover.Renderer.
const splashSlot = 3

// splashLeast is the smallest box worth drawing the logo in, in cells. Below
// this it is a smudge, and the words underneath say more.
const (
	splashLeastW = 24
	splashLeastH = 8

	// splashKeeps is how many rows the panel's own words and list keep for
	// themselves. The logo gets the rest, which is the whole point of it: it is
	// as large as the screen allows.
	splashKeeps = 9

	// splashAboutShare is how much of the help screen the picture heading it
	// takes, as a percentage of the body.
	//
	// Half. The keys are a page that scrolls under it, so this is a choice
	// about how the screen looks rather than a fight over rows: half leaves
	// most of a group of keys showing under the picture, which says the page
	// goes on without having to say so.
	splashAboutShare = 50
)

var (
	splashOnce sync.Once
	splashImg  image.Image
)

// splashPicture decodes the logo, once.
func splashPicture() image.Image {
	splashOnce.Do(func() {
		if img, _, err := image.Decode(bytes.NewReader(logo.PNG)); err == nil {
			splashImg = img
		}
	})
	return splashImg
}

// splashState is the rendered logo and the box it was rendered for.
type splashState struct {
	cells   string
	forW    int
	forH    int
	renders uint64
}

// splashFlow renders the logo when the wait is on and the box it goes in has
// changed, and throws it away when the wait ends.
//
// In the update rather than in the view, and rendered on a change rather than
// every frame, for the same reason: the kitty backend writes the whole picture
// into the terminal. View is a pure function and may not do that, and doing it
// sixty times a second for a still image would be sixty times the bytes for no
// difference at all.
func (m *Model) splashFlow() {
	w, rows := m.splashRoom()
	if !m.splashWanted() || w < splashLeastW || rows < splashLeastH {
		if m.splash.cells != "" {
			// A picture the kitty backend has put in the terminal is not
			// undrawn by the screen being redrawn without it. Whatever comes
			// next has to send its own, which is what every other picture here
			// already does.
			m.splash = splashState{renders: m.splash.renders}
		}
		return
	}
	if m.splash.cells != "" && m.splash.forW == w && m.splash.forH == rows {
		return
	}
	img := splashPicture()
	if img == nil {
		return
	}
	m.splash.renders++
	cells, err := m.covers.Renderer().Render(img, w, rows, m.splash.renders, splashSlot)
	if err != nil {
		return
	}
	m.splash.cells, m.splash.forW, m.splash.forH = cells, w, rows
}

// splashWanted is whether the wait it belongs to is on.
//
// Two screens, and the first of them is the one that matters. Before any state
// has arrived the player is a single line reading "Connecting…", and that is
// where spindle spends the second after it is started — or the twenty, after the
// machine has been asleep. The other is the player with nothing playing
// anywhere, which is the same wait gone long.
//
// It was on the second only, which is why it never showed: on a machine whose
// daemon answers, the state arrives with a device in it and the first screen is
// the only one anybody sees.
//
// Not while the device picker is open over it: a logo above a list of speakers
// is a logo in somebody's way. What it is over the picker is what it is on the
// help screen — see splashAbout, which is where a picture of the program
// belongs.
func (m Model) splashWanted() bool {
	if m.covers == nil || m.devices.open {
		return false
	}
	if m.tab == tabHelp {
		return true
	}
	return m.tab == tabPlayer && (m.ps == nil || m.noDevice)
}

// splashRoom is the box the logo is given: everything the screen has, less what
// the words under it want.
//
// The connecting screen has one line under it and the device screen has a list,
// so they are not given the same room — the picture is as large as what is left
// allows, and on the connecting screen almost nothing is left over.
func (m Model) splashRoom() (w, rows int) {
	l := m.layout()
	w = l.interior - leftMargin - rightMargin
	if m.tab == tabHelp {
		// Half the page. The keys scroll under it now rather than sharing the
		// rows with it, so the picture no longer has to be whatever was left
		// over — and what it was left over was nothing at all on any terminal
		// under about eighty rows.
		return w, max(l.bodyHeight*splashAboutShare/100, 0)
	}
	keeps := splashKeeps
	if m.ps == nil {
		keeps = 2
	}
	return w, max(l.bodyHeight-1-keeps, 0)
}

// splashRows is what was rendered, ready to put in the panel.
func (m Model) splashRows() []string {
	if m.splash.cells == "" {
		return nil
	}
	return strings.Split(m.splash.cells, "\n")
}
