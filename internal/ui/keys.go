package ui

import "charm.land/bubbles/v2/key"

// keyMap is the full key set of the player screen. Bindings that form a pair
// (next/previous, seek, volume) carry the combined help text on the first of the
// pair; the sibling is never listed, so it needs no help text of its own.
type keyMap struct {
	PlayPause key.Binding
	Next      key.Binding
	Prev      key.Binding
	SeekFwd   key.Binding
	SeekBack  key.Binding
	VolUp     key.Binding
	VolDown   key.Binding
	Shuffle   key.Binding
	Repeat    key.Binding
	Help      key.Binding
	Quit      key.Binding
	QuitAll   key.Binding

	// Navigation, shared by the browsing tabs.
	NextTab key.Binding
	PrevTab key.Binding
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Devices key.Binding
	Refresh key.Binding
}

func newKeyMap() keyMap {
	k := keyMap{
		PlayPause: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "play / pause"),
		),
		Next: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n / p", "next / previous"),
		),
		Prev: key.NewBinding(key.WithKeys("p")),
		SeekFwd: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("← / →", "seek ∓5s"),
		),
		SeekBack: key.NewBinding(key.WithKeys("left")),
		VolUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑ / ↓", "volume ±5"),
		),
		VolDown: key.NewBinding(key.WithKeys("down")),
		Shuffle: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "shuffle"),
		),
		Repeat: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "cycle repeat"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		QuitAll: key.NewBinding(
			key.WithKeys("Q"),
			key.WithHelp("Q", "quit and stop playback"),
		),

		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		PrevTab: key.NewBinding(key.WithKeys("shift+tab")),
		Up:      key.NewBinding(key.WithKeys("up", "ctrl+p")),
		Down: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↑↓", "select"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Devices: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "devices"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}

	return k
}

// tabKeys is the help for one screen. Each tab works differently, so each gets
// its own bar rather than a lowest common denominator.
type tabKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (t tabKeys) ShortHelp() []key.Binding  { return t.short }
func (t tabKeys) FullHelp() [][]key.Binding { return t.full }

// hint builds a display-only binding. The short bar needs terser wording than
// the expanded table to survive a 64-column terminal.
func hint(keys, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, desc))
}

// forNoDevice is the help while nothing is playing: the transport keys have
// nothing to act on, so offering them would be a lie.
func (k keyMap) forNoDevice() tabKeys {
	return tabKeys{
		short: []key.Binding{
			hint("↑↓", "select"),
			hint("enter", "play here"),
			hint("r", "refresh"),
			hint("q", "quit"),
		},
		full: [][]key.Binding{
			{k.Down, k.Enter, k.Refresh},
			{k.NextTab, k.Quit},
		},
	}
}

// forDevices is the help while the picker is open over the player.
func (k keyMap) forDevices() tabKeys {
	return tabKeys{
		short: []key.Binding{
			hint("↑↓", "select"),
			hint("enter", "play here"),
			hint("r", "refresh"),
			hint("esc", "close"),
		},
		full: [][]key.Binding{
			{k.Down, k.Enter, k.Refresh, k.Back},
		},
	}
}

func (k keyMap) forTab(t tabID) tabKeys {
	switch t {
	case tabPlaylists:
		return tabKeys{
			short: []key.Binding{
				hint("↑↓", "select"),
				hint("enter", "play"),
				hint("esc", "back"),
				hint("tab", "switch"),
				hint("?", "help"),
			},
			full: [][]key.Binding{
				{k.Down, k.Enter, k.Back, k.NextTab},
				{k.PlayPause, k.Next, k.Help, k.Quit},
			},
		}

	case tabSearch:
		return tabKeys{
			short: []key.Binding{
				hint("type", "to search"),
				hint("↑↓", "select"),
				hint("enter", "play"),
				hint("tab", "switch"),
			},
			full: [][]key.Binding{
				{k.Down, k.Enter, k.Back, k.NextTab},
				{hint("ctrl+c", "quit"), k.Help},
			},
		}

	default:
		return tabKeys{
			short: []key.Binding{
				hint("space", "play/pause"),
				hint("n/p", "track"),
				hint("←→", "seek"),
				hint("d", "devices"),
				hint("?", "help"),
			},
			full: [][]key.Binding{
				{k.PlayPause, k.Next, k.SeekFwd, k.VolUp},
				{k.Shuffle, k.Repeat, k.Devices, k.NextTab},
				{k.Quit, k.QuitAll},
			},
		}
	}
}
