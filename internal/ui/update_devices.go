package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// deviceKey handles the picker and the no-device screen, which are the same list
// wearing different hats. It returns handled=false for anything neither owns.
func (m *Model) deviceKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	// The no-device screen is a picker that cannot be closed: there is nothing
	// behind it to go back to.
	active := m.devices.open || (m.tab == tabPlayer && m.noDevice)
	if !active {
		if m.pressed(k, m.keys.Devices) && !m.search.typing {
			m.devices.open = true
			return tea.Batch(fetchDevicesCmd(m.player), m.syncCover()), true
		}
		return nil, false
	}

	if m.listKey(k, &m.devices.cursor, len(m.devices.items), true) {
		return nil, true
	}

	switch {
	case m.pressed(k, m.keys.Enter):
		return m.transfer(), true

	case m.pressed(k, m.keys.Refresh):
		return tea.Batch(fetchDevicesCmd(m.player), fetchStateCmd(m.player)), true

	case m.pressed(k, m.keys.Devices), m.pressed(k, m.keys.Back):
		if !m.devices.open {
			return nil, true // nothing to close on the no-device screen
		}
		m.devices.open = false
		return m.syncCover(), true
	}

	// Anything else belongs to whoever handles it next. The transport keys stop
	// at their own guard, which already refuses to act with no device; claiming
	// them here swallowed quit and help as well, and left the no-device screen
	// with no way out of it at all.
	return nil, false
}

// transfer moves playback and closes the picker at once. The confirming fetch is
// delayed for the same reason a skip's is: Spotify takes a moment to agree.
func (m *Model) transfer() tea.Cmd {
	sel := m.devices.selected()
	if sel == nil {
		return nil
	}

	id, p := sel.ID, m.player
	m.devices.open = false
	m.hold()

	// The music carries on doing what it was doing, in the new place. Choosing
	// a device is not the same act as pressing play — Spotify's own clients
	// pass the state through, and so does spotify-player — and a device picked
	// while nothing is on must not start whatever was last loaded.
	playing := m.ps != nil && m.ps.Playing

	return tea.Batch(
		controlCmd("move playback", func(ctx context.Context) error {
			return p.TransferTo(ctx, id, playing)
		}),
		refetchCmd(confirmFirst),
		fetchDevicesCmd(m.player),
		m.syncCover(),
	)
}
