package ui

import (
	"slices"

	"github.com/pottom/spindle/internal/player"
)

// devicePane is the device list. It serves two screens: the picker opened with
// `d`, and the whole of the no-device screen — which is the same list with
// nothing to overlay it on. Keeping one implementation means the two cannot
// drift apart.
type devicePane struct {
	items  []player.Device
	cursor listState
	open   bool // the picker is showing over the player
}

// has reports whether a device is in the list Spotify last named.
func (d devicePane) has(id string) bool {
	return slices.ContainsFunc(d.items, func(dev player.Device) bool { return dev.ID == id })
}

// selected returns the device under the cursor.
func (p devicePane) selected() *player.Device {
	if p.cursor.cursor < 0 || p.cursor.cursor >= len(p.items) {
		return nil
	}
	return &p.items[p.cursor.cursor]
}

// adopt takes a fresh listing, keeping the cursor on the same device where it
// can — a list that reorders under the cursor is how you transfer to the wrong
// speaker.
func (p *devicePane) adopt(devices []player.Device) {
	var was string
	if sel := p.selected(); sel != nil {
		was = sel.ID
	}

	p.items = devices
	for i, d := range devices {
		if d.ID == was {
			p.cursor.cursor = i
			return
		}
	}

	// Nothing to hold on to: start on whichever device is already active.
	p.cursor.reset()
	for i, d := range devices {
		if d.Active {
			p.cursor.cursor = i
			return
		}
	}
}

// rows renders the list. A row is the active marker, the name, and the kind of
// thing it is.
func (m Model) deviceRows(w int, showCursor bool) []string {
	if len(m.devices.items) == 0 {
		return []string{m.styles.Empty.Render("No devices reported.")}
	}

	rows := make([]string, 0, len(m.devices.items))
	for i, d := range m.devices.items {
		rows = append(rows, m.deviceRow(d, w, showCursor && i == m.devices.cursor.cursor))
	}
	return rows
}

func (m Model) deviceRow(d player.Device, w int, selected bool) string {
	gutter := "  "
	if selected {
		gutter = m.styles.Cursor.Render(rowCursor) + " "
	}

	// The dot says which device Spotify still considers active; the cursor says
	// which one you are about to choose. They are different questions.
	mark := "  "
	name := m.styles.RowPrimary
	if d.Active {
		mark = m.styles.DeviceOn.Render(deviceDot) + " "
		name = m.styles.RowSelected
	} else if selected {
		name = m.styles.RowSelected
	}

	body := max(w-4, 0)
	return gutter + mark + spread(name.Render(d.Name), m.styles.RowSecondary.Render(d.Type), body)
}
