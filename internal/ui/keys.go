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
	GoTab key.Binding
	Up    key.Binding
	Down  key.Binding
	// PageDown and PageUp move by whatever the list is showing, so a page is a
	// screenful rather than a number somebody picked.
	PageDown key.Binding
	PageUp   key.Binding
	// First and Last go to the ends. The letters that mean the same are separate
	// bindings rather than more keys on these two, because the search tab reads
	// every printable key as part of the query and must not act on them there.
	First    key.Binding
	Last     key.Binding
	FirstVim key.Binding
	LastVim  key.Binding
	Enter    key.Binding
	Back     key.Binding
	Devices  key.Binding
	Refresh  key.Binding
	Scope    key.Binding
	// Stage gives the whole terminal to the visualiser, and Tell puts the
	// record's own name up there without waiting for a solo to make room.
	Stage key.Binding
	Tell  key.Binding
	Marks key.Binding
	Face  key.Binding

	// Loose turns keeping time with the record off and on, so the two ways of
	// drawing can be put side by side on the one record.
	Loose  key.Binding
	Lyrics key.Binding
	Peek   key.Binding
	Mute   key.Binding

	// Queue editing. Only the tracks put there by hand can be moved or dropped,
	// so these do nothing on the rest of the list.
	// Actions opens the menu of verbs for the item under the cursor. It has a
	// second binding for the search tab, where every printable key is the query.
	Actions      key.Binding
	ActionsTyped key.Binding

	// SearchType starts typing a query, and is how the search tab is entered
	// from anywhere.
	SearchType key.Binding

	// SearchKind moves between the kinds of thing a query matched. Control keys
	// because every printable one on that tab is the query.
	SearchKind     key.Binding
	SearchKindBack key.Binding

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

	// HalfDown and HalfUp move by half a screen, which is how vim reads a long
	// list: far enough to make progress, near enough to keep your place.
	HalfDown key.Binding
	HalfUp   key.Binding

	// Restart puts the playback device through a stop and a start, for the
	// settings it only reads when it begins.
	Restart key.Binding

	// FindNext and FindPrev walk the rows a search inside the list matched.
	//
	// They had n and N, and the transport had ctrl+n and ctrl+p for it. That was
	// the wrong way round in use: skipping a track is wanted on every screen
	// there is and is the most pressed key after play, while walking matches is
	// wanted in a list that has just been searched. So the plain keys go to the
	// transport and these take semicolon and comma — which is what vim repeats a
	// search with, so the hand already knows them.
	FindNext key.Binding
	FindPrev key.Binding
}

func newKeyMap() keyMap {
	k := keyMap{
		PlayPause: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "play / pause"),
		),
		Next: key.NewBinding(
			// Plain, and the control ones kept beside them. Skipping is the most
			// pressed key on the transport after play, and it was the only one
			// asking for two fingers.
			key.WithKeys("n", "ctrl+n"),
			key.WithHelp("n / p", "next / previous"),
		),
		Prev: key.NewBinding(key.WithKeys("p", "ctrl+p")),
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
			key.WithKeys("1", "2", "3", "4", "5", "6"),
			key.WithHelp("1–6", "go to tab"),
		),
		Up:   key.NewBinding(key.WithKeys("up")),
		Down: key.NewBinding(key.WithKeys("down"), key.WithHelp("↑↓", "select")),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("pgdn / pgup", "page down / up"),
		),
		PageUp: key.NewBinding(key.WithKeys("pgup", "ctrl+b")),

		// Half a screen at a time, as vim moves. The whole-screen keys keep
		// theirs: ctrl+f and ctrl+b are the same pair in the same place.
		HalfDown: key.NewBinding(key.WithKeys("ctrl+d")),
		HalfUp:   key.NewBinding(key.WithKeys("ctrl+u")),

		Restart: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "restart the device"),
		),
		FindNext: key.NewBinding(
			// Semicolon and comma, which is what vim repeats a search with, and
			// what n and N were before skipping a track wanted them. A list is
			// the only place these mean anything, and skipping a track is
			// wanted in every place there is.
			key.WithKeys(";"),
			key.WithHelp("; / ,", "next / previous match"),
		),
		FindPrev: key.NewBinding(key.WithKeys(",")),
		First: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home / end", "first / last"),
		),
		Last:     key.NewBinding(key.WithKeys("end")),
		FirstVim: key.NewBinding(key.WithKeys("g")),
		LastVim:  key.NewBinding(key.WithKeys("G")),
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
		Stage: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "full screen"),
		),
		Tell: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "what is playing"),
		),
		Marks: key.NewBinding(
			// d for the dancers. It was m, which is mute everywhere else in the
			// program — and on the big screen it won, so the one screen where
			// you most want to silence the room was the one screen that could
			// not.
			//
			// d is the device picker everywhere else, and up here it is not
			// reachable. That trade is the opposite of the one it replaces:
			// silencing the room is something you do while watching, and
			// choosing which speakers to play through is something you go to the
			// player for.
			key.WithKeys("d"),
			key.WithHelp("d", "the dancers"),
		),
		Face: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "a face"),
		),
		Loose: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "keep time"),
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

		SearchType: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "find"),
		),
		SearchKind: key.NewBinding(
			key.WithKeys("ctrl+right", "ctrl+t"),
			key.WithHelp("ctrl+t", "kind"),
		),
		SearchKindBack: key.NewBinding(key.WithKeys("ctrl+left")),
		Actions: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp(".", "actions"),
		),
		ActionsTyped: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "actions"),
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

// selectHint is the movement entry every list's short bar opens with. The page
// keys ride along with the arrows rather than taking a slot of their own: the
// bar is one line and already full at the narrowest terminal, and an entry
// added to the end of it costs whatever was last — which is the help key.
var selectHint = hint("↑↓ ⇞⇟", "select")

// moveKeys is the movement block of the expanded help. It is a column of its
// own rather than rows added to an existing one so that offering it cannot make
// the bar taller: the layout is measured from the help's height before the bar
// is drawn, and a bar that grew afterwards would push the last row of every
// list off the bottom of the screen.
//
// vim says whether g and G are read on this screen. They are not offered where
// they do nothing.
func (k keyMap) moveKeys(vim bool) []key.Binding {
	out := []key.Binding{k.PageDown, hint("^d / ^u", "half a page"), k.First}
	if vim {
		out = append(out, hint("g / G", "first / last"))
	}
	return out
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
			k.moveKeys(true),
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
			k.moveKeys(true),
		},
	}
}

// forReadOnlyQueue is the help for a queue that can be walked but not edited.
func (k keyMap) forReadOnlyQueue() tabKeys {
	return tabKeys{
		short: []key.Binding{
			selectHint,
			hint("enter", "play"),
			hint("tab", "switch"),
			hint("?", "help"),
		},
		full: [][]key.Binding{
			{k.Down, k.Enter, k.NextTab, k.GoTab},
			{k.PlayPause, k.Next, k.Help, k.Quit},
			k.moveKeys(true),
		},
	}
}

// forPlayer is the player screen's help. The waveform key is only listed where
// there is room to draw one: advertising a key that does nothing is worse than
// a shorter bar.
// fitHints drops entries off the end of a hint bar until it fits.
//
// Whole entries, because the alternative is what happened when the bar was
// first filled out: the frame cut it mid-word and left an ellipsis. Off the end
// rather than out of the middle, so what a narrow terminal loses is what was
// reached for least — losing "quit" is better than being shown the wrong four.
func fitHints(short []key.Binding, width int) []key.Binding {
	if width <= 0 {
		return short
	}
	room := width - leftMargin - rightMargin
	used := 0
	for i, b := range short {
		w := len([]rune(b.Help().Key)) + 1 + len([]rune(b.Help().Desc))
		if i > 0 {
			w += 3 // the " · " between them
		}
		if used+w > room {
			return short[:max(i, 1)]
		}
		used += w
	}
	return short
}

func (k keyMap) forPlayer(scope, lyrics, peek bool, width int) tabKeys {
	short := []key.Binding{
		hint("space", "play/pause"),
		hint("n/p", "track"),
		hint("←→", "seek"),
	}
	short = append(short, hint("↑↓", "volume"), hint("s", "shuffle"), hint("r", "repeat"))
	if lyrics {
		short = append(short, hint("l", "lyrics"))
	}
	if scope {
		short = append(short, hint("v", "waveform"))
	}
	if peek {
		short = append(short, hint("u", "up next"))
	}
	// No "?" on it. The bar names everything this screen does, so the table
	// behind that key has nothing left to add here.
	short = append(short, hint("t", "big screen"), hint("d", "devices"))
	short = fitHints(short, width)

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
// forOpen is the help for an opened playlist, album or artist. An artist's list
// is of records, so what enter does there is open one rather than play it.
func (k keyMap) forOpen(albums, scope bool) tabKeys {
	enter := hint("enter", "play")
	if albums {
		enter = hint("enter", "open")
	}

	short := []key.Binding{
		selectHint,
		enter,
		hint("/", "find"),
		hint(".", "actions"),
		hint("a", "queue"),
		hint("esc", "back"),
	}
	second := []key.Binding{k.Actions, k.PlayOne, k.PlayPause, k.Next, k.Help}
	if scope {
		short = append(short, hint("v", "waveform"))
		second = append(second, k.Scope)
	}
	return tabKeys{
		short: append(short, hint("?", "help")),
		full: [][]key.Binding{
			{k.Down, k.Enter, k.Enqueue, k.Back, k.NextTab},
			second,
			k.moveKeys(true),
		},
	}
}

func (k keyMap) forTab(t tabID, scope bool) tabKeys {
	switch t {
	case tabQueue:
		short := []key.Binding{
			selectHint,
			hint("enter", "play"),
			hint("/", "find"),
			hint(".", "actions"),
			hint("x", "remove"),
			hint("j/k", "move"),
		}
		second := []key.Binding{k.Actions, k.PlayPause, k.Next, k.NextTab, k.Help}
		if scope {
			short = append(short, hint("v", "waveform"))
			second = append(second, k.Scope)
		}
		return tabKeys{
			short: append(short, hint("?", "help")),
			full: [][]key.Binding{
				{k.Down, k.Enter, k.Drop, k.MoveDn},
				second,
				k.moveKeys(true),
			},
		}

	case tabLibrary:
		short := []key.Binding{
			selectHint,
			hint("enter", "open"),
			hint("/", "find"),
			hint("ctrl+t", "kind"),
			hint(".", "actions"),
			hint("a", "queue"),
		}
		second := []key.Binding{k.Actions, k.PlayOne, k.PlayPause, k.Next, k.Help}
		if scope {
			short = append(short, hint("v", "waveform"))
			second = append(second, k.Scope)
		}
		return tabKeys{
			short: append(short, hint("?", "help")),
			full: [][]key.Binding{
				{k.Down, k.Enter, k.SearchKind, k.Enqueue, k.NextTab},
				{k.SearchType, k.FindNext, k.PageDown, k.FirstVim},
				second,
				k.moveKeys(true),
			},
		}

	case tabSettings:
		return tabKeys{
			short: []key.Binding{
				selectHint,
				hint("← / →", "change it"),
				hint("R", "restart the device"),
				hint("?", "help"),
			},
			full: [][]key.Binding{
				{k.Down, hint("← / →", "change the setting"), k.NextTab, k.Quit},
			},
		}

	case tabHelp:
		return tabKeys{
			short: []key.Binding{hint("1–6", "go to a screen"), hint("tab", "next"), hint("q", "quit")},
			full:  [][]key.Binding{{k.NextTab, k.GoTab, k.Quit, k.QuitAll}},
		}

	case tabSearch:
		return tabKeys{
			short: []key.Binding{
				hint("type", "to search"),
				hint("/", "search"),
				selectHint,
				hint("enter", "play"),
				hint("ctrl+t", "kind"),
				hint(".", "actions"),
			},
			full: [][]key.Binding{
				{k.SearchType, k.Down, k.Enter, k.SearchKind, k.Actions},
				{hint("ctrl+c", "quit"), k.Help},
				// No g and G here: the query has them.
				k.moveKeys(false),
			},
		}

	default:
		return tabKeys{}
	}
}
