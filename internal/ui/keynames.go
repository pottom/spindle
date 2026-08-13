package ui

// The keys spindle reads, each written down once.
//
// A key used to be spelled twice: once in the binding that reads it and once in
// the hint that advertises it. The two drifted — the player screen offered t for
// the full-screen visualiser, which is on f and never was on t — and somebody
// reading a hint bar presses what it says. One name now, used on both sides, so
// the bar cannot promise a key nothing answers.
//
// Named for what the key does rather than for the letter it is, because that is
// what the call sites are saying. A letter that means different things on
// different screens gets a name per screen: d is the device picker on the player
// and the dancers on the full screen, and one of them may move without the
// other.
const (
	keyPlayPause = "space"
	keyNext      = "n"
	keyPrev      = "p"
	keySeekFwd   = "right"
	keySeekBack  = "left"
	keyVolUp     = "up"
	keyVolDown   = "down"
	keyShuffle   = "s"
	keyRepeat    = "r"
	keyMute      = "m"

	keyHelp    = "?"
	keyQuit    = "q"
	keyQuitAlt = "ctrl+c"
	keyQuitAll = "Q"

	keyNextTab   = "tab"
	keyPrevTab   = "shift+tab"
	keyUp        = "up"
	keyDown      = "down"
	keyPageDown  = "pgdown"
	keyPageUp    = "pgup"
	keyPageDnVim = "ctrl+f"
	keyPageUpVim = "ctrl+b"
	keyHalfDown  = "ctrl+d"
	keyHalfUp    = "ctrl+u"
	keyFirst     = "home"
	keyLast      = "end"
	keyFirstVim  = "g"
	keyLastVim   = "G"
	keyEnter     = "enter"
	keyBack      = "esc"

	keyDevices = "d"
	keyRefresh = "r"
	keyRestart = "R"

	// The visualiser: keyScope walks the ways of drawing it, keyStage gives it
	// the whole terminal.
	keyScope = "v"
	keyStage = "f"

	// The full screen's own keys. keyMarks is d up there, where the device
	// picker cannot be reached anyway, and keyTell puts the record's name up
	// without waiting for a solo to make room.
	keyTell  = "t"
	keyMarks = "d"
	keyFace  = "w"
	keyLoose = "b"

	keyLyrics = "l"

	keyPeek = "u"

	keyFind     = "/"
	keyFindNext = ";"
	keyFindPrev = ","

	// The search tab reads every printable key as part of the query, so the
	// verbs that have to work while typing take a control key as well.
	keyKind        = "ctrl+t"
	keyKindFwd     = "ctrl+right"
	keyKindBack    = "ctrl+left"
	keyActions     = "."
	keyActionsHeld = "ctrl+o"
	keyEnqueue     = "a"
	keyEnqueueHeld = "ctrl+a"

	keyPlayOne = "o"
	keyDrop    = "x"
	keyMoveUp  = "k"
	keyMoveDn  = "j"

	// keyDebug is ctrl and shift together, because ctrl alone is spoken for:
	// ctrl+d is half a page down in the lists.
	keyDebug = "ctrl+shift+d"
)

// pair spells the two keys of a pair the way a hint names them, so the bar and
// the expanded table say it the same way and neither has to spell the letters.
func pair(a, b string) string { return a + " / " + b }
