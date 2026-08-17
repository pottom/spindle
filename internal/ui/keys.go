package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
)

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
	GoTab    key.Binding
	Up       key.Binding
	Down     key.Binding
	NextTile key.Binding
	PrevTile key.Binding
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
	Story  key.Binding

	// Close folds the band above the queue away a block at a time, so the list
	// has the rows. See queueRoom.
	Close key.Binding

	Peek key.Binding
	Mute key.Binding

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
	Like    key.Binding
	Fit     key.Binding
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
			key.WithKeys(keyPlayPause),
			key.WithHelp(keyPlayPause, "play / pause"),
		),
		Next: key.NewBinding(
			// The plain keys, and only those. Skipping is the most pressed key
			// on the transport after play, and it was the only one asking for
			// two fingers; ctrl+n and ctrl+p were kept beside them for a while
			// afterwards and did nothing the letters did not — including in the
			// one place they might have earned their keep, since a query being
			// typed swallows them rather than skipping a track.
			key.WithKeys(keyNext),
			key.WithHelp(pair(keyNext, keyPrev), "next / previous"),
		),
		Prev: key.NewBinding(key.WithKeys(keyPrev)),
		SeekFwd: key.NewBinding(
			key.WithKeys(keySeekFwd, keySeekFwdAlt, keySeekFwdAny),
			key.WithHelp("← → or < >", "seek ∓5s"),
		),
		SeekBack: key.NewBinding(key.WithKeys(keySeekBack, keySeekBackAlt, keySeekBackAny)),
		VolUp: key.NewBinding(
			key.WithKeys(keyVolUp, keyVolUpAlt, keyVolUpAny, keyVolUpMore),
			key.WithHelp("↑ ↓ or + −", "volume ±5"),
		),
		VolDown: key.NewBinding(key.WithKeys(keyVolDown, keyVolDownAlt, keyVolDownAny)),
		Shuffle: key.NewBinding(
			key.WithKeys(keyShuffle),
			key.WithHelp(keyShuffle, "shuffle"),
		),
		Repeat: key.NewBinding(
			key.WithKeys(keyRepeat),
			key.WithHelp(keyRepeat, "cycle repeat"),
		),
		Help: key.NewBinding(
			key.WithKeys(keyHelp),
			key.WithHelp(keyHelp, "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys(keyQuit, keyQuitAlt),
			key.WithHelp(keyQuit, "quit"),
		),
		QuitAll: key.NewBinding(
			key.WithKeys(keyQuitAll),
			key.WithHelp(keyQuitAll, "quit and stop playback"),
		),

		NextTab: key.NewBinding(
			key.WithKeys(keyNextTab),
			key.WithHelp(keyNextTab, "next tab"),
		),
		PrevTab: key.NewBinding(key.WithKeys(keyPrevTab)),
		GoTab: key.NewBinding(
			// One digit per screen there is, so a screen added or taken away
			// cannot leave a digit that goes nowhere.
			key.WithKeys(tabDigits()...),
			key.WithHelp(tabDigitRange("–"), "go to tab"),
		),
		Up:   key.NewBinding(key.WithKeys(keyUp)),
		Down: key.NewBinding(key.WithKeys(keyDown), key.WithHelp("↑↓", "select")),

		// The wall's own pair. Separate from the transport's, which answers to
		// the same two arrows off a list: a binding that matched both would move
		// the cursor when somebody asked to seek.
		NextTile: key.NewBinding(key.WithKeys(keyNextTile)),
		PrevTile: key.NewBinding(key.WithKeys(keyPrevTile)),
		PageDown: key.NewBinding(
			key.WithKeys(keyPageDown, keyPageDnVim),
			key.WithHelp("pgdn / pgup", "page down / up"),
		),
		PageUp: key.NewBinding(key.WithKeys(keyPageUp, keyPageUpVim)),

		// Half a screen at a time, as vim moves. The whole-screen keys keep
		// theirs: ctrl+f and ctrl+b are the same pair in the same place.
		HalfDown: key.NewBinding(key.WithKeys(keyHalfDown)),
		HalfUp:   key.NewBinding(key.WithKeys(keyHalfUp)),

		Restart: key.NewBinding(
			key.WithKeys(keyRestart),
			key.WithHelp(keyRestart, "restart the device"),
		),
		FindNext: key.NewBinding(
			// Semicolon and comma, which is what vim repeats a search with, and
			// what n and N were before skipping a track wanted them. A list is
			// the only place these mean anything, and skipping a track is
			// wanted in every place there is.
			key.WithKeys(keyFindNext),
			key.WithHelp(pair(keyFindNext, keyFindPrev), "next / previous match"),
		),
		FindPrev: key.NewBinding(key.WithKeys(keyFindPrev)),
		First: key.NewBinding(
			key.WithKeys(keyFirst),
			key.WithHelp(pair(keyFirst, keyLast), "first / last"),
		),
		Last:     key.NewBinding(key.WithKeys(keyLast)),
		FirstVim: key.NewBinding(key.WithKeys(keyFirstVim)),
		LastVim:  key.NewBinding(key.WithKeys(keyLastVim)),
		Enter: key.NewBinding(
			key.WithKeys(keyEnter),
			key.WithHelp(keyEnter, "play"),
		),
		Back: key.NewBinding(
			key.WithKeys(keyBack),
			key.WithHelp(keyBack, "back"),
		),
		Devices: key.NewBinding(
			key.WithKeys(keyDevices),
			key.WithHelp(keyDevices, "devices"),
		),
		Refresh: key.NewBinding(
			key.WithKeys(keyRefresh),
			key.WithHelp(keyRefresh, "refresh"),
		),
		Scope: key.NewBinding(
			key.WithKeys(keyScope),
			key.WithHelp(keyScope, "vis"),
		),
		Stage: key.NewBinding(
			key.WithKeys(keyStage),
			key.WithHelp(keyStage, "full vis"),
		),
		Tell: key.NewBinding(
			key.WithKeys(keyTell),
			key.WithHelp(keyTell, "what is playing"),
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
			key.WithKeys(keyMarks),
			key.WithHelp(keyMarks, "the dancers"),
		),
		Face: key.NewBinding(
			key.WithKeys(keyFace),
			key.WithHelp(keyFace, "a face"),
		),
		Loose: key.NewBinding(
			key.WithKeys(keyLoose),
			key.WithHelp(keyLoose, "keep time"),
		),
		Lyrics: key.NewBinding(
			key.WithKeys(keyLyrics),
			key.WithHelp(keyLyrics, "lyrics"),
		),
		Story: key.NewBinding(
			key.WithKeys(keyStory),
			key.WithHelp(keyStory, "about"),
		),

		Close: key.NewBinding(
			key.WithKeys(keyClose),
			key.WithHelp(keyClose, "fold the top away"),
		),
		Peek: key.NewBinding(
			key.WithKeys(keyPeek),
			key.WithHelp(keyPeek, "up next"),
		),
		Mute: key.NewBinding(
			key.WithKeys(keyMute),
			key.WithHelp(keyMute, "mute"),
		),

		SearchType: key.NewBinding(
			key.WithKeys(keyFind),
			key.WithHelp(keyFind, "find"),
		),
		SearchKind: key.NewBinding(
			key.WithKeys(keyKindNext, keyKindFwd, keyKind),
			key.WithHelp(pair(keyKindPrev, keyKindNext), "kind"),
		),
		SearchKindBack: key.NewBinding(key.WithKeys(keyKindPrev, keyKindBack)),
		Actions: key.NewBinding(
			key.WithKeys(keyActions),
			key.WithHelp(keyActions, "actions"),
		),
		ActionsTyped: key.NewBinding(
			key.WithKeys(keyActionsHeld),
			key.WithHelp(keyActionsHeld, "actions"),
		),
		Fit: key.NewBinding(
			key.WithKeys(keyFit),
			key.WithHelp(keyFit, "follow this track"),
		),
		Like: key.NewBinding(
			key.WithKeys(keyLike),
			key.WithHelp(keyLike, "like / unlike"),
		),
		Enqueue: key.NewBinding(
			key.WithKeys(keyEnqueue),
			key.WithHelp(keyEnqueue, "add to queue"),
		),
		PlayOne: key.NewBinding(
			key.WithKeys(keyPlayOne),
			key.WithHelp(keyPlayOne, "play only this"),
		),
		EnqueueTyped: key.NewBinding(
			key.WithKeys(keyEnqueueHeld),
			key.WithHelp(keyEnqueueHeld, "add to queue"),
		),
		Drop: key.NewBinding(
			key.WithKeys(keyDrop),
			key.WithHelp(keyDrop, "remove from queue"),
		),
		MoveUp: key.NewBinding(key.WithKeys(keyMoveUp)),
		MoveDn: key.NewBinding(
			key.WithKeys(keyMoveDn),
			key.WithHelp(pair(keyMoveDn, keyMoveUp), "move down / up"),
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

// hint builds a display-only binding, for the entries that are a picture of a
// key rather than the name of one: the arrows, the page glyphs, the caret
// shorthand, and "type", which is an instruction.
func hint(keys, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, desc))
}

// terse re-words a binding for the short bar, which needs shorter wording than
// the expanded table to survive a 64-column terminal — "cycle repeat" has to
// become "repeat", and enter is "open" in a list of records.
//
// The key comes from the binding rather than from the call, so the bar cannot
// name a key that binding does not read. Spelling it twice is how the bar came
// to offer t for the full-screen visualiser, which is on f: both spellings were
// written by hand, and only one of them was ever pressed.
func terse(b key.Binding, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(b.Keys()...), key.WithHelp(b.Help().Key, desc))
}

// tight is terse with the air taken out of a pair: "n / p" is three columns
// wider than "n/p", and the bar is one line that is already full.
func tight(b key.Binding, desc string) key.Binding {
	out := terse(b, desc)
	out.SetHelp(strings.ReplaceAll(b.Help().Key, " / ", "/"), desc)
	return out
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
		out = append(out, hint(pair(keyFirstVim, keyLastVim), "first / last"))
	}
	return out
}

// forNoDevice is the help while nothing is playing: the transport keys have
// nothing to act on, so offering them would be a lie.
func (k keyMap) forNoDevice() tabKeys {
	return tabKeys{
		short: []key.Binding{
			hint("↑↓", "select"),
			terse(k.Enter, "play here"),
			terse(k.Refresh, "refresh"),
			terse(k.Quit, "quit"),
		},
		full: [][]key.Binding{
			{k.Down, k.Enter, k.Refresh},
			{k.NextTab, k.Quit},
			k.moveKeys(true),
		},
	}
}

// forDevices is the help while the picker is open over the player.
// forFinding is the bar while a list is being searched.
//
// The keys of the search rather than of the list under it, because the search is
// what the keyboard is doing — and because the way out of it has to be written
// somewhere. Esc clears a query as readily as it abandons one being typed, and
// nothing said so: the field showed a count and no way to be rid of it.
func (k keyMap) forFinding(typing bool) tabKeys {
	short := []key.Binding{}
	if typing {
		short = append(short, hint("type", "search"), terse(k.Enter, "keep"))
	}
	short = append(short,
		terse(k.FindNext, "next match"),
		terse(k.Back, "clear"),
	)
	return tabKeys{
		short: short,
		full: [][]key.Binding{
			{k.Enter, k.FindNext, k.Back},
			k.moveKeys(true),
		},
	}
}

func (k keyMap) forDevices() tabKeys {
	return tabKeys{
		short: []key.Binding{
			hint("↑↓", "select"),
			terse(k.Enter, "play here"),
			terse(k.Refresh, "refresh"),
			terse(k.Back, "close"),
		},
		full: [][]key.Binding{
			{k.Down, k.Enter, k.Refresh, k.Back},
			k.moveKeys(true),
		},
	}
}

// forReadOnlyQueue is the help for a queue that can be walked but not edited.
func (k keyMap) forReadOnlyQueue(like bool) tabKeys {
	short := []key.Binding{
		selectHint,
		terse(k.Enter, "play"),
	}
	// Liking is nothing to do with whose device is playing: the collection is
	// the account's, and a queue that can only be read can still be liked from.
	if like {
		short = append(short, terse(k.Like, "like"))
	}
	return tabKeys{
		short: append(short, terse(k.NextTab, "switch"), terse(k.Help, "help")),
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

func (k keyMap) forPlayer(scope, lyrics, peek, story, like bool, width int) tabKeys {
	short := []key.Binding{
		terse(k.PlayPause, "play/pause"),
		tight(k.Next, "track"),
		hint("←→", "seek"),
	}
	short = append(short, hint("↑↓", "volume"), terse(k.Shuffle, "shuffle"), terse(k.Repeat, "repeat"))
	if lyrics {
		short = append(short, terse(k.Lyrics, "lyrics"))
	}
	if story {
		short = append(short, terse(k.Story, "about"))
	}
	if like {
		short = append(short, terse(k.Like, "like"))
	}
	if scope {
		short = append(short, terse(k.Scope, "vis"))
	}
	if peek {
		short = append(short, terse(k.Peek, "up next"))
	}
	// No "?" on it. The bar names everything this screen does, so the table
	// behind that key has nothing left to add here.
	short = append(short, terse(k.Stage, "full vis"), terse(k.Devices, "devices"))
	short = fitHints(short, width)

	second := []key.Binding{k.Shuffle, k.Repeat, k.Devices}
	if like {
		second = append(second, k.Like)
	}
	if scope {
		second = append(second, k.Scope)
	}
	if lyrics {
		second = append(second, k.Lyrics)
	}
	if story {
		second = append(second, k.Story)
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
func (k keyMap) forOpen(albums, scope, like bool) tabKeys {
	enter := terse(k.Enter, "play")
	if albums {
		enter = terse(k.Enter, "open")
	}

	short := []key.Binding{
		selectHint,
		enter,
		terse(k.SearchType, "find"),
		terse(k.Actions, "actions"),
		terse(k.Enqueue, "queue"),
		terse(k.Back, "back"),
	}
	second := []key.Binding{k.Actions, k.PlayOne, k.PlayPause, k.Next, k.Help}
	if scope {
		short = append(short, terse(k.Scope, "vis"))
		second = append(second, k.Scope)
	}
	if like && !albums {
		short = append(short, terse(k.Like, "like"))
		second = append(second, k.Like)
	}
	return tabKeys{
		short: append(short, terse(k.Help, "help")),
		full: [][]key.Binding{
			{k.Down, k.Enter, k.Enqueue, k.Back, k.NextTab},
			second,
			k.moveKeys(true),
		},
	}
}

func (k keyMap) forTab(t tabID, scope, like, fit bool) tabKeys {
	switch t {
	case tabQueue:
		short := []key.Binding{
			selectHint,
			terse(k.Enter, "play"),
			terse(k.SearchType, "find"),
			terse(k.Actions, "actions"),
			terse(k.Drop, "remove"),
			tight(k.MoveDn, "move"),
		}
		if fit {
			short = append(short, terse(k.Fit, "follow"))
		}
		second := []key.Binding{k.Actions, k.PlayPause, k.Next, k.NextTab, k.Help}
		if scope {
			short = append(short, terse(k.Scope, "vis"))
			second = append(second, k.Scope)
		}
		if like {
			short = append(short, terse(k.Like, "like"))
			second = append(second, k.Like)
		}
		return tabKeys{
			short: append(short, terse(k.Help, "help")),
			full: [][]key.Binding{
				{k.Down, k.Enter, k.Drop, k.MoveDn},
				second,
				k.moveKeys(true),
			},
		}

	case tabLibrary:
		short := []key.Binding{
			selectHint,
			terse(k.Enter, "open"),
			terse(k.SearchType, "find"),
			terse(k.SearchKind, "kind"),
			terse(k.Actions, "actions"),
			terse(k.Enqueue, "queue"),
		}
		second := []key.Binding{k.Actions, k.PlayOne, k.PlayPause, k.Next, k.Help}
		if scope {
			short = append(short, terse(k.Scope, "vis"))
			second = append(second, k.Scope)
		}
		return tabKeys{
			short: append(short, terse(k.Help, "help")),
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
				terse(k.Restart, "restart the device"),
				terse(k.Help, "help"),
			},
			full: [][]key.Binding{
				{k.Down, hint("← / →", "change the setting"), k.NextTab, k.Quit},
			},
		}

	case tabHelp:
		return tabKeys{
			short: []key.Binding{terse(k.GoTab, "go to a screen"), terse(k.NextTab, "next"), terse(k.Quit, "quit")},
			full:  [][]key.Binding{{k.NextTab, k.GoTab, k.Quit, k.QuitAll}},
		}

	case tabSearch:
		return tabKeys{
			short: []key.Binding{
				hint("type", "to search"),
				terse(k.SearchType, "search"),
				selectHint,
				terse(k.Enter, "play"),
				terse(k.SearchKind, "kind"),
				terse(k.Actions, "actions"),
			},
			full: [][]key.Binding{
				{k.SearchType, k.Down, k.Enter, k.SearchKind, k.Actions},
				// The held quit rather than q, which types here.
				{hint(keyQuitAlt, "quit"), k.Help},
				// No g and G here: the query has them.
				k.moveKeys(false),
			},
		}

	default:
		return tabKeys{}
	}
}
