package cover

import (
	"strings"
	"testing"
)

// Speaking the kitty graphics protocol is not one question, and the terminals
// that can be drawn on are named rather than the ones that cannot.
//
// WezTerm answers the graphics query and does not do the Unicode placeholder
// mode the Kitty renderer draws in. Measured on a live window: the placeholder
// cells came out as rows of tofu, the cover was drawn twice at the cursor, and
// the text under it was shifted. iTerm was reported drawing no artwork at all,
// which is the same fault.
//
// Naming the broken ones is wrong by default for every terminal nobody has
// tried, and it is wrong in the direction that breaks the screen. Named this
// way, an unknown terminal gets a cover made of coloured blocks — a picture
// rather than a mess.
func TestATerminalMaySpeakTheProtocolWithoutItsPlaceholders(t *testing.T) {
	kitty := []byte("\x1b_Gi=31;OK\x1b\\\x1bP>|kitty 0.42.0\x1b\\\x1b[?62;c")
	wez := []byte("\x1b_Gi=31;OK\x1b\\\x1bP>|WezTerm 20240203\x1b\\\x1b[?62;c")
	plain := []byte("\x1b[?62;c")

	for _, c := range []struct {
		what         string
		reply        []byte
		kitty, place bool
		backend      string
	}{
		{"kitty", kitty, true, true, "kitty"},
		{"WezTerm", wez, true, false, "halfblock"},
		{"one that says nothing", plain, false, false, "halfblock"},
		{"Ghostty", []byte("\x1b_Gi=31;OK\x1b\\\x1bP>|ghostty 1.3.1\x1b\\\x1b[?62;c"), true, true, "kitty"},
		// Word for word what Rio 0.5.21 answered, capital R and all, which is
		// why the match is made on a lowercased name.
		{"Rio", []byte("\x1b_Gi=31;OK\x1b\\\x1bP>|Rio 0.5.21\x1b\\\x1b[?62;c"), true, true, "kitty"},
		// Answers the query, is not on the list, and so is drawn the safe way.
		{"one nobody has tried", []byte("\x1b_Gi=31;OK\x1b\\\x1bP>|Ninetendo 3\x1b\\\x1b[?62;c"), true, false, "halfblock"},
	} {
		g := readReplyAs(c.reply)
		if g.Kitty != c.kitty || g.Placeholders != c.place {
			t.Errorf("%s: kitty=%v placeholders=%v, want %v and %v",
				c.what, g.Kitty, g.Placeholders, c.kitty, c.place)
		}
		if got := g.Backend(); got != c.backend {
			t.Errorf("%s: backend %q, want %q", c.what, got, c.backend)
		}
	}
}

// The terminal is asked what it is, rather than being taken at the environment's
// word.
//
// TERM_PROGRAM is inherited: measured, Alacritty started from a Ghostty window
// reports TERM_PROGRAM=ghostty. Going by that, a Ghostty window opened from
// WezTerm would be taken for WezTerm and lose its pictures — the wrong way
// round, and silently.
func TestTheTerminalIsAskedWhatItIs(t *testing.T) {
	for _, c := range []struct{ reply, want string }{
		{"\x1bP>|WezTerm 20240203-110809\x1b\\", "WezTerm 20240203-110809"},
		{"\x1bP>|ghostty 1.0.1\x1b\\", "ghostty 1.0.1"},
		{"\x1bP>|kitty(0.42.0)\a", "kitty(0.42.0)"},
		{"\x1b[?62;c", ""},
		{"", ""},
		{"\x1bP>|no terminator", ""},
	} {
		if got := terminalName([]byte(c.reply)); got != c.want {
			t.Errorf("%q gave %q, want %q", c.reply, got, c.want)
		}
	}
}

// The query cleans up after itself.
//
// A terminal that does not know a sequence should swallow it; Alacritty prints
// the payload. Measured, --cover-info there opened with
// `_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA` written across the top of the screen. The
// probe runs before the interface takes the terminal, so nothing else would
// clear it.
func TestTheQueryPutsTheCursorBackAndWipesWhatWasPrinted(t *testing.T) {
	if tidyBefore != "\x1b[s" {
		t.Errorf("the position is not saved before asking: %q", tidyBefore)
	}
	// Back to where it was, then erase forward — forward rather than the line,
	// because a reply longer than the terminal is wide leaves a wrapped tail.
	if tidyAfter != "\x1b[u\x1b[J" {
		t.Errorf("what was printed is not wiped after asking: %q", tidyAfter)
	}
}

// The terminal is asked how big a cell is, and believed only when the answer
// could be a terminal's.
func TestTheCellSizeIsReadFromTheReply(t *testing.T) {
	for _, c := range []struct {
		name  string
		reply string
		want  CellSize
	}{
		{"a terminal that answers", "\x1b[6;19;9t\x1b[?62c", CellSize{Width: 9, Height: 19, Measured: true}},
		{"one that does not", "\x1b[?62c", CellSize{}},
		{"one that answers nonsense", "\x1b[6;19;5t\x1b[?62c", CellSize{}},
		{"a cell nobody could read", "\x1b[6;2;1t\x1b[?62c", CellSize{}},
		{"a reply cut short", "\x1b[6;19;", CellSize{}},
		{"a reply with a part missing", "\x1b[6;19t\x1b[?62c", CellSize{}},
	} {
		if got := readReplyAs([]byte(c.reply)).Cell; got != c.want {
			t.Errorf("%s: cell = %+v, want %+v", c.name, got, c.want)
		}
	}
}

// And the query asks for it, or nothing above can answer.
func TestTheQueryAsksForTheCellSize(t *testing.T) {
	if !strings.Contains(graphicsQuery, "\x1b[16t") {
		t.Errorf("the query does not ask the terminal its cell size: %q", graphicsQuery)
	}
}

// A cell size the kernel reports is refused on the same terms.
func TestAnImpossibleCellIsRefused(t *testing.T) {
	for _, c := range []struct {
		cell CellSize
		want bool
	}{
		{CellSize{Width: 9, Height: 19}, true},
		{CellSize{Width: 10, Height: 20}, true},
		{CellSize{Width: 20, Height: 40}, true},
		{CellSize{Width: 7, Height: 15}, true},
		{CellSize{Width: 5, Height: 19}, false},  // measured over ssh, Windows client
		{CellSize{Width: 19, Height: 19}, false}, // square is not a cell
		{CellSize{Width: 0, Height: 0}, false},
	} {
		if got := c.cell.plausible(); got != c.want {
			t.Errorf("%+v: plausible = %v, want %v", c.cell, got, c.want)
		}
	}
}
