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
// Carried in the binary rather than fetched. It is 900k of a 30 megabyte
// program, and the moment it exists for is the moment the network is not
// answering — a logo downloaded on demand would be missing exactly when it was
// wanted.

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
// The player screen with nothing playing anywhere, which is where spindle spends
// the second or two after it is started and every second of a longer wait. Not
// while the device picker is open over it: somebody choosing a device is past
// looking at a logo.
func (m Model) splashWanted() bool {
	return m.covers != nil && m.tab == tabPlayer && m.noDevice && !m.devices.open
}

// splashRoom is the box the logo is given: everything the panel has, less the
// rows its words and its device list want.
func (m Model) splashRoom() (w, rows int) {
	l := m.layout()
	return l.interior - leftMargin - rightMargin,
		max(l.bodyHeight-1-splashKeeps, 0)
}

// splashRows is what was rendered, ready to put in the panel.
func (m Model) splashRows() []string {
	if m.splash.cells == "" {
		return nil
	}
	return strings.Split(m.splash.cells, "\n")
}
