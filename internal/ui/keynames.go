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
	keyNextTile  = "right"
	keyPrevTile  = "left"

	// The transport's second pair, for the screens where the arrows belong to
	// what is on them. A list took the up and down arrows for its cursor and the
	// volume has been unreachable there ever since; the wall takes left and
	// right as well. Shift gives both back, everywhere and by one rule.
	keySeekFwdAlt  = "shift+right"
	keySeekBackAlt = "shift+left"
	keyVolUpAlt    = "shift+up"
	keyVolDownAlt  = "shift+down"

	// And a set that needs no arrow at all. The arrows belong to whatever list
	// is on screen — the wall walks by them, so does the queue — which left the
	// transport holding the shift key on every tab but one, for the two things
	// somebody reaches for most often.
	//
	// These are the marks every player has used for them since tape: plus and
	// minus for the level, the two chevrons for winding through a track. They
	// are free on every screen here, and on the one screen where a printable key
	// belongs to a search field the arrows above still answer.
	keyVolUpAny    = "+"
	keyVolUpMore   = "="
	keyVolDownAny  = "-"
	keySeekFwdAny  = ">"
	keySeekBackAny = "<"
	keySeekBack    = "left"
	keyVolUp       = "up"
	keyVolDown     = "down"
	keyShuffle     = "s"
	keyRepeat      = "r"
	keyMute        = "m"

	keyHelp = "?"
	// Leaving takes a modifier. q was the key for it, and q is a letter: a
	// screen with a field on it — the search, the find box over a list — turns
	// every letter into a character, and the one that closed the program was one
	// keystroke away from every one of those. A program should not be able to
	// end by accident.
	keyQuit    = "ctrl+q"
	keyQuitAlt = "ctrl+c"
	keyQuitAll = "ctrl+shift+q"

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

	// keyLike saves the track under the cursor, or takes it back out. h for
	// heart, which is what the column it fills in is drawn with. Offered only
	// where the backend can do it at all: an application Spotify registered
	// after 2024 is refused the whole family of library writes, and a key that
	// cannot work is worse than no key. See player.Collector.
	keyLike = "h"

	// keyStory opens what somebody wrote about the record that is playing. It is
	// offered only where there is something behind it — see storyAvailable — and
	// i is what every program uses for "tell me about this".
	keyStory = "i"

	// keyClose folds the band above the queue away, a block at a time, so the
	// list has the rows. Not z, which is where y sits on a Hungarian keyboard
	// and is awkward on both — a reason that has since gone: a binding is now
	// matched by the key a press came from as well as by the letter it sent, so
	// a letter here is the same key on any layout. See keypress.go.
	keyClose = "c"

	// keyCovers turns a list of answers into a wall of covers and back. The same
	// key as the queue's fold, because it is the same act read one way: what the
	// screen spends its room on. They are on different tabs and never both
	// answer a press.
	keyCovers = "c"
	keyPeek  = "u"

	keyFind     = "/"
	keyFindNext = ";"
	keyFindPrev = ","

	// The search tab reads every printable key as part of the query, so the
	// verbs that have to work while typing take a control key as well.
	// The kinds a tab holds — the library's four lists, the search's four sorts
	// of answer — are walked with the two keys next to each other on any
	// keyboard. Which letters they send does not matter: a binding is matched by
	// the key a press came from as well. See keypress.go.
	keyKindPrev = "["
	keyKindNext = "]"

	keyKind        = "ctrl+t"
	keyKindFwd     = "ctrl+right"
	keyKindBack    = "ctrl+left"
	keyActions     = "."
	keyActionsHeld = "ctrl+o"
	keyEnqueue     = "a"
	keyEnqueueHeld = "ctrl+a"

	// keyFit puts what is coming into an order that follows what is playing.
	// w for "what goes with this", and it is free on every screen that has a
	// queue: the letter's other use is the full screen's own, which swallows
	// keys of its own. See fit.go.
	keyFit = "w"

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
