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

	// short is the one-line help bar of SCREENS.md 4.1. It needs terser wording
	// than the expanded table to survive a 64-column terminal, so it carries its
	// own display bindings over the same keys.
	short []key.Binding
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
	}

	k.short = []key.Binding{
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "play/pause")),
		key.NewBinding(key.WithKeys("n", "p"), key.WithHelp("n/p", "track")),
		key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→", "seek")),
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
	return k
}

func (k keyMap) ShortHelp() []key.Binding {
	return k.short
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PlayPause, k.Next, k.SeekFwd, k.VolUp},
		{k.Shuffle, k.Repeat, k.Help, k.Quit},
	}
}
