package ui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// deviceKey handles the picker and the no-device screen, which are the same list
// wearing different hats. It returns handled=false for anything neither owns.
func (m *Model) deviceKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	// The no-device screen is a picker that cannot be closed: there is nothing
	// behind it to go back to.
	active := m.devices.open || (m.tab == tabPlayer && m.noDevice)
	if !active {
		if key.Matches(k, m.keys.Devices) && m.tab != tabSearch {
			m.devices.open = true
			return tea.Batch(fetchDevicesCmd(m.player), m.syncCover()), true
		}
		return nil, false
	}

	switch {
	case key.Matches(k, m.keys.Down), key.Matches(k, m.keys.Up):
		delta := 1
		if key.Matches(k, m.keys.Up) {
			delta = -1
		}
		m.devices.cursor.move(delta, len(m.devices.items))
		return nil, true

	case key.Matches(k, m.keys.Enter):
		return m.transfer(), true

	case key.Matches(k, m.keys.Refresh):
		return tea.Batch(fetchDevicesCmd(m.player), fetchStateCmd(m.player)), true

	case key.Matches(k, m.keys.Devices), key.Matches(k, m.keys.Back):
		if !m.devices.open {
			return nil, true // nothing to close on the no-device screen
		}
		m.devices.open = false
		return m.syncCover(), true
	}

	// Keep the transport keys away from a screen where they have nothing to act
	// on, but let quit and tab switching through.
	return nil, m.noDevice
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

	return tea.Batch(
		controlCmd("move playback", func(ctx context.Context) error {
			return p.TransferTo(ctx, id)
		}),
		refetchCmd(trackChangeDelay),
		fetchDevicesCmd(m.player),
		m.syncCover(),
	)
}
