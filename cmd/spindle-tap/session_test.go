package main

import (
	"strings"
	"testing"
	"time"
)

func testSession(t *testing.T) *session {
	// Nothing here touches the daemon: a test that walks the plan would
	// otherwise put seven records on whatever is listening.
	t.Cleanup(func() { fetchWords, putOn = wordsFor, play })
	fetchWords = func(string) (lyrics, error) { return lyrics{Synced: true}, nil }
	putOn = func(string, time.Duration) error { return nil }

	s := &session{plan: []work{
		{id: "a", artist: "A", title: "One", why: "because", passes: []pass{starts, ends}},
		{id: "b", artist: "B", title: "Two", why: "because", passes: []pass{starts}},
	}, line: 0}
	s.ly.Synced = true
	for i := range 20 {
		s.ly.Lines = append(s.ly.Lines, struct {
			At    int64  `json:"at"`
			Words string `json:"words"`
		}{At: int64(i) * 2500, Words: "one two three"})
	}
	s.a = anchor{at: 0, when: time.Now(), ok: true}
	return s
}

// A pass counts what it is actually asking for. Counting every tap instead
// would end the ends pass at six lines, having asked for twelve.
func TestAPassCountsTheLinesItAsked(t *testing.T) {
	var taps []tap
	for i := range 6 {
		taps = append(taps, tap{line: i, word: 0}, tap{line: i, word: -1})
	}

	if got := done(passStarts, taps); got != 12 {
		t.Errorf("the starts pass counted %d, want every tap", got)
	}
	if got := done(passEnds, taps); got != 6 {
		t.Errorf("the ends pass counted %d, want the lines that were closed", got)
	}
	if got := done(passWords, taps); got != 6 {
		t.Errorf("the words pass counted %d, want the lines it touched", got)
	}
}

// The session walks every pass of every record, in order, and stops after the
// last one rather than running off the end of the plan.
func TestTheSessionWalksThePlan(t *testing.T) {
	s := testSession(t)

	var seen []string
	for range 3 {
		seen = append(seen, s.plan[s.track].id+":"+s.plan[s.track].passes[s.pass].kind)
		if over := s.next(); over {
			break
		}
	}
	want := []string{"a:lines", "a:bounds", "b:lines"}
	if strings.Join(seen, " ") != strings.Join(want, " ") {
		t.Errorf("the session went %v, want %v", seen, want)
	}
	if !s.finished {
		t.Error("the plan ran out and the session did not say so")
	}
}

// A tap while nothing is playing is refused and said out loud. Swallowing it is
// how an evening gets spent tapping into a file that stays empty — which is
// exactly what happened the first time this tool was used.
func TestATapWithNothingPlayingSaysSo(t *testing.T) {
	s := testSession(t)
	s.a.ok = false

	s.key(' ')
	if len(s.taps) != 0 {
		t.Error("a tap was written down with no position to write")
	}
	if !strings.Contains(s.said, "p starts it") {
		t.Errorf("it said %q, want it to say how to start the music", s.said)
	}
}

// Backspace takes the last tap back, and the word cursor with it.
func TestBackspaceTakesTheLastTapBack(t *testing.T) {
	s := testSession(t)
	s.key(' ')
	s.key(' ')
	if len(s.taps) != 2 || s.word != 2 {
		t.Fatalf("two taps left %d taps and the cursor at %d", len(s.taps), s.word)
	}

	s.key(127)
	if len(s.taps) != 1 || s.word != 1 {
		t.Errorf("backspace left %d taps and the cursor at %d, want 1 and 1", len(s.taps), s.word)
	}
	s.key(127)
	s.key(127) // one too many, and nothing to take back
	if len(s.taps) != 0 || s.word != 0 {
		t.Errorf("backspace past the start left %d taps and the cursor at %d", len(s.taps), s.word)
	}
}

// The panel says which record, which pass, how far along, and the line being
// sung — the four things somebody with one hand on the space bar looks for.
func TestThePanelSaysWhereYouAre(t *testing.T) {
	s := testSession(t)
	s.playing = true
	s.key(' ')

	panel := s.panel()
	for _, want := range []string{"record 1 of 2", "pass 1 of 2", "A — One", "1 of 12", "one two three"} {
		if !strings.Contains(panel, want) {
			t.Errorf("the panel does not say %q:\n%s", want, panel)
		}
	}
}
