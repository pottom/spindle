package cover

import (
	"bytes"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
)

// graphicsQuery asks the terminal three things in one breath: whether it speaks
// the kitty graphics protocol, what it calls itself, and — last, because every
// terminal answers it — its device attributes, which is how the reading knows
// the terminal has finished talking. A reply with only the last is a definitive
// "no" to the first. DESIGN.md 5.3.
//
// The name is asked for because speaking the protocol is not one question. See
// Graphics.Placeholders.
const graphicsQuery = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\\x1b[>q\x1b[c"

// tidyQuery puts the cursor back and wipes anything the terminal printed
// because it did not recognise what it was asked.
//
// A terminal that does not know a sequence is supposed to swallow it. Alacritty
// prints the payload instead: measured, `--cover-info` in Alacritty opens with
// `_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA` written across the top of the screen. The
// probe runs before the interface takes the terminal, so there is nothing else
// to clear it, and the same rubbish would greet anybody starting spindle there.
//
// Save the position, ask, then come back and erase forward. Erasing forward
// rather than the line because the reply can be longer than the terminal is
// wide, and a wrapped tail left behind is the same mess one row down.
const (
	tidyBefore = "\x1b[s"
	tidyAfter  = "\x1b[u\x1b[J"
)

// detectTimeout bounds the wait. The program must never block on this.
const detectTimeout = 200 * time.Millisecond

// Graphics is what the terminal said about drawing pictures.
type Graphics struct {
	// Kitty is whether it answered the graphics query at all.
	Kitty bool

	// Placeholders is whether it can also do the Unicode placeholder mode the
	// Kitty renderer draws in — the image transmitted once and a rectangle of
	// U+10EEEE cells left for the terminal to fill.
	//
	// Not the same question as Kitty, and there is no query for it. See
	// placeholderers, which is the list of terminals known to do it.
	Placeholders bool

	// Name is what the terminal called itself, or empty if it did not say.
	Name string
}

// Backend is the renderer this terminal should be given.
func (g Graphics) Backend() string {
	if g.Kitty && g.Placeholders {
		return "kitty"
	}
	return "halfblock"
}

// placeholderers are the terminals known to do the kitty protocol's Unicode
// placeholder mode. Everything else gets half blocks.
//
// A list rather than a test, because the protocol has no way to ask.
//
// Named the right way round, which took two goes. It was a list of the ones
// known *not* to do placeholders, and that is wrong by default for every
// terminal nobody has tried: a new one gets a broken screen — rows of tofu, the
// cover drawn twice, the text shifted — until somebody reports it. WezTerm sat
// there for months, and iTerm was reported drawing no artwork at all, which is
// the same fault. Named this way, the worst an unknown terminal gets is a cover
// made of coloured blocks, which is a picture rather than a mess, and a terminal
// that does support placeholders is one line to add.
//
// Researched 2026-08-12. Terminals that speak the protocol at all: kitty,
// Ghostty, Konsole, st (patched), Warp, wayst, WezTerm, iTerm2, xterm.js, Rio.
// Confirmed to do placeholders: these two, and kitty's own documentation says
// nothing about the rest. Confirmed not to: WezTerm, measured here, and
// xterm.js, which is VS Code's terminal.
//
// Matched against what the terminal says it is when asked directly, rather than
// against TERM_PROGRAM: that variable is inherited, and measured, Alacritty
// started from a Ghostty window reports TERM_PROGRAM=ghostty.
var placeholderers = []string{"kitty", "ghostty"}

// Probe asks the terminal what it can do. It must be called before Bubble Tea
// takes over the terminal.
//
// Where the output has been redirected, the terminal itself is opened and asked
// instead. Without that, `spindle --cover-info > somewhere` writes the query into
// the file, nothing answers it, and the report says the terminal can do nothing —
// which is the one question that report exists to answer, given wrong, in the
// exact circumstance somebody would be capturing it to send on.
func Probe(out, in *os.File) Graphics {
	if out == nil || in == nil {
		return Graphics{}
	}
	if !term.IsTerminal(out.Fd()) || !term.IsTerminal(in.Fd()) {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return Graphics{}
		}
		defer tty.Close() //nolint:errcheck // nothing useful to do here
		out, in = tty, tty
	}
	if !term.IsTerminal(in.Fd()) {
		return Graphics{}
	}
	return readReplyAs(askTerminal(out, in))
}

// readReplyAs is Probe's reading of what came back, apart from the asking, so
// the reading can be tested without a terminal.
func readReplyAs(reply []byte) Graphics {
	g := Graphics{Kitty: bytes.Contains(reply, []byte("\x1b_G")), Name: terminalName(reply)}
	if g.Kitty {
		name := strings.ToLower(g.Name)
		for _, good := range placeholderers {
			if strings.Contains(name, good) {
				g.Placeholders = true
			}
		}
	}
	return g
}

// SupportsKitty reports whether the terminal can be drawn on the way the Kitty
// renderer draws.
func SupportsKitty(out, in *os.File) bool {
	return Probe(out, in).Backend() == "kitty"
}

// terminalName pulls the terminal's own name out of its XTVERSION reply, which
// arrives as DCS > | name version ST.
func terminalName(reply []byte) string {
	const head = "\x1bP>|"
	i := bytes.Index(reply, []byte(head))
	if i < 0 {
		return ""
	}
	rest := reply[i+len(head):]
	end := bytes.Index(rest, []byte("\x1b\\"))
	if end < 0 {
		if end = bytes.IndexByte(rest, 0x07); end < 0 {
			return ""
		}
	}
	return strings.TrimSpace(string(rest[:end]))
}
