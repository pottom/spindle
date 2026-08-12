package cover

import "testing"

// Speaking the kitty graphics protocol is not one question.
//
// WezTerm answers the graphics query and does not do the Unicode placeholder
// mode the Kitty renderer draws in. Measured on a live window: the placeholder
// cells came out as rows of tofu, the cover was drawn twice at the cursor, and
// the text under it was shifted. There is no query for placeholders, so the ones
// known not to do them are named.
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
