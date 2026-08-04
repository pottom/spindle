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
	// GoTab is the digits: one per screen, in the order they are drawn.
	GoTab   key.Binding
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Devices key.Binding
	Refresh key.Binding
	Scope   key.Binding
	Lyrics  key.Binding
	Peek    key.Binding
	Mute    key.Binding

	// Queue editing. Only the tracks put there by hand can be moved or dropped,
	// so these do nothing on the rest of the list.
	Enqueue key.Binding
	// PlayOne plays the track under the cursor and nothing else, where enter
	// would have played the list it belongs to.
	PlayOne key.Binding
	// EnqueueTyped is the same action on the search tab, where every printable
	// key belongs to the query and cannot mean anything else.
	EnqueueTyped key.Binding
	Drop         key.Binding
	MoveUp       key.Binding
	MoveDn       key.Binding
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
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(key.WithKeys("shift+tab")),
		GoTab: key.NewBinding(
			key.WithKeys("1", "2", "3", "4"),
			key.WithHelp("1–4", "go to tab"),
		),
		Up: key.NewBinding(key.WithKeys("up", "ctrl+p")),
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
		Scope: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "waveform"),
		),
		Lyrics: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "lyrics"),
		),
		Peek: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "up next"),
		),
		Mute: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "mute"),
		),

		Enqueue: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add to queue"),
		),
		PlayOne: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "play only this"),
		),
		EnqueueTyped: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "add to queue"),
		),
		Drop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "remove from queue"),
		),
		MoveUp: key.NewBinding(key.WithKeys("k")),
		MoveDn: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j / k", "move down / up"),
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

// forReadOnlyQueue is the help for a queue that can be walked but not edited.
func (k keyMap) forReadOnlyQueue() tabKeys {
	return tabKeys{
		short: []key.Binding{
			hint("↑↓", "select"),
			hint("enter", "play"),
			hint("tab", "switch"),
			hint("?", "help"),
		},
		full: [][]key.Binding{
			{k.Down, k.Enter, k.NextTab, k.GoTab},
			{k.PlayPause, k.Next, k.Help, k.Quit},
		},
	}
}

// forPlayer is the player screen's help. The waveform key is only listed where
// there is room to draw one: advertising a key that does nothing is worse than
// a shorter bar.
func (k keyMap) forPlayer(scope, lyrics, peek bool) tabKeys {
	short := []key.Binding{
		hint("space", "play/pause"),
		hint("n/p", "track"),
		hint("←→", "seek"),
	}
	switch {
	case lyrics:
		short = append(short, hint("l", "lyrics"))
	case scope:
		short = append(short, hint("v", "waveform"))
	default:
		short = append(short, hint("d", "devices"))
	}
	short = append(short, hint("?", "help"))

	second := []key.Binding{k.Shuffle, k.Repeat, k.Devices}
	if scope {
		second = append(second, k.Scope)
	}
	if lyrics {
		second = append(second, k.Lyrics)
	}
	if peek {
		second = append(second, k.Peek)
	}

	return tabKeys{
		short: short,
		full: [][]key.Binding{
			{k.PlayPause, k.Next, k.SeekFwd, k.VolUp},
			second,
			{k.NextTab, k.GoTab, k.Quit, k.QuitAll},
		},
	}
}

// forTab is the help for the screens that are lists. scope says whether the
// visualiser has room beside the queue's artwork; it is listed only where it can
// be drawn, and never changes the bar's height, which the layout depends on.
func (k keyMap) forTab(t tabID, scope bool) tabKeys {
	switch t {
	case tabQueue:
		short := []key.Binding{
			hint("↑↓", "select"),
			hint("enter", "play"),
			hint("x", "remove"),
			hint("j/k", "move"),
		}
		second := []key.Binding{k.PlayPause, k.Next, k.NextTab, k.GoTab, k.Help}
		if scope {
			short = append(short, hint("v", "waveform"))
			second = append(second, k.Scope)
		}
		return tabKeys{
			short: append(short, hint("?", "help")),
			full: [][]key.Binding{
				{k.Down, k.Enter, k.Drop, k.MoveDn},
				second,
			},
		}

	case tabLibrary:
		short := []key.Binding{
			hint("↑↓", "select"),
			hint("enter", "play"),
			hint("o", "only this"),
			hint("a", "queue"),
			hint("esc", "back"),
		}
		second := []key.Binding{k.GoTab, k.PlayOne, k.PlayPause, k.Next, k.Help}
		if scope {
			short = append(short, hint("v", "waveform"))
			second = append(second, k.Scope)
		}
		return tabKeys{
			short: append(short, hint("?", "help")),
			full: [][]key.Binding{
				{k.Down, k.Enter, k.Enqueue, k.Back, k.NextTab},
				second,
			},
		}

	case tabSearch:
		return tabKeys{
			short: []key.Binding{
				hint("type", "to search"),
				hint("↑↓", "select"),
				hint("enter", "play"),
				hint("ctrl+a", "queue"),
			},
			full: [][]key.Binding{
				{k.Down, k.Enter, k.EnqueueTyped, k.Back, k.NextTab},
				{hint("ctrl+c", "quit"), k.Help},
			},
		}

	default:
		return tabKeys{}
	}
}
