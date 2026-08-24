package daemon

import (
	"strings"
	"sync"
	"testing"
)

// launcher records where a browser was sent, and whether one could be started
// at all.
type launcher struct {
	mu   sync.Mutex
	ok   bool
	sent []string
}

func (l *launcher) open(url string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sent = append(l.sent, url)
	return l.ok
}

func (l *launcher) opened() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.sent...)
}

const stubLink = "https://accounts.spotify.com/authorize?client_id=abc&code_challenge=xyz"

// The line a first run turns on. go-librespot writes it and then blocks on the
// redirect coming back, so a link nobody opens is a device that never plays.
func TestTheSignInLinkOpensABrowser(t *testing.T) {
	launch := &launcher{ok: true}
	entry := &signIn{open: launch.open}

	say, ok := entry.notice(signInPrefix + stubLink)
	if !ok {
		t.Fatal("the sign-in line went past unread")
	}
	if sent := launch.opened(); len(sent) != 1 || sent[0] != stubLink {
		t.Errorf("sent a browser to %v, want %q once", sent, stubLink)
	}
	if !strings.Contains(say, stubLink) {
		t.Errorf("said %q, want the link in it: the launcher answers whether it started, not whether anything appeared", say)
	}
	if entry.pending() != stubLink {
		t.Errorf("the device is waiting on %q, want %q", entry.pending(), stubLink)
	}
}

// Over SSH, and on a machine with no session to open a window on, there is no
// browser to send anywhere. The link is the whole of the answer then.
func TestTheSignInLinkIsSaidWhenNoBrowserOpens(t *testing.T) {
	launch := &launcher{ok: false}
	entry := &signIn{open: launch.open}

	say, ok := entry.notice(signInPrefix + stubLink)
	if !ok {
		t.Fatal("the sign-in line went past unread")
	}
	if !strings.Contains(say, stubLink) {
		t.Errorf("said %q, want the link in it", say)
	}
	if !strings.Contains(say, "no browser") {
		t.Errorf("said %q, want it to say a browser could not be opened", say)
	}
}

// The same link twice is the same login, not a second one. A log that repeats
// itself must not put a second tab on the screen.
func TestTheSameSignInDoesNotOpenTwice(t *testing.T) {
	launch := &launcher{ok: true}
	entry := &signIn{open: launch.open}

	entry.notice(signInPrefix + stubLink)
	if say, ok := entry.notice(signInPrefix + stubLink); ok {
		t.Errorf("said %q the second time, want the repeat left alone", say)
	}
	if sent := launch.opened(); len(sent) != 1 {
		t.Errorf("sent a browser %d times for one link, want once", len(sent))
	}

	// A fresh attempt carries a fresh challenge, and the old link is dead: that
	// one is worth opening.
	other := stubLink + "&second=1"
	if _, ok := entry.notice(signInPrefix + other); !ok {
		t.Error("a new link went past unread")
	}
	if sent := launch.opened(); len(sent) != 2 || sent[1] != other {
		t.Errorf("sent a browser to %v, want the new link opened too", sent)
	}
}

// Everything else in the log is somebody else's line. Reading them is the price
// of go-librespot offering no hook, and mistaking one for a sign-in would put a
// browser on the screen of somebody who asked for music.
func TestOrdinaryLinesAreNotMistakenForASignIn(t *testing.T) {
	launch := &launcher{ok: true}
	entry := &signIn{open: launch.open}

	for _, line := range []string{
		"loaded track \"Heroes\"",
		"the player did not take a status request within 10s",
		"visit the following link: nowhere",
		signInPrefix,
	} {
		if say, ok := entry.notice(line); ok {
			t.Errorf("read %q as a sign-in and said %q", line, say)
		}
	}
	if sent := launch.opened(); len(sent) != 0 {
		t.Errorf("sent a browser to %v over lines that were not sign-ins", sent)
	}
	if entry.pending() != "" {
		t.Errorf("the device is said to be waiting on %q", entry.pending())
	}
}

// The link reaches the log through the logger, because that is the only thing
// go-librespot is given: it takes a logger and offers nowhere else to listen.
func TestTheLogCarriesTheSignInToTheReader(t *testing.T) {
	launch := &launcher{ok: true}
	entry := &signIn{open: launch.open}

	var out strings.Builder
	log := newLogger(&out, entry.notice)
	log.Infof("%s%s", signInPrefix, stubLink)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("wrote %d lines, want the day, librespot's line and ours:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[2], "spindle:") || !strings.Contains(lines[2], stubLink) {
		t.Errorf("the line after it is %q, want spindle saying what is waiting and on what", lines[2])
	}
}

// A logger with nobody reading it writes what it was given and no more, which
// is every logger in the tests and the one in a daemon that is already signed
// in.
func TestALogWithNoReaderIsUnchanged(t *testing.T) {
	var out strings.Builder
	log := newLogger(&out, nil)
	log.Infof("%s%s", signInPrefix, stubLink)

	if lines := strings.Split(strings.TrimSpace(out.String()), "\n"); len(lines) != 2 {
		t.Errorf("wrote %d lines, want the day and the one entry:\n%s", len(lines), out.String())
	}
}

// The link is left where the process that started this daemon can find it. The
// two share no channel but the filesystem: the one they would otherwise use is
// the API, and the API not answering is the thing being explained.
func TestTheLinkIsLeftForWhoeverStartedTheDaemon(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	entry := &signIn{open: func(string) bool { return true }}
	if link := Waiting(); link != "" {
		t.Fatalf("something is already waiting on %q", link)
	}

	entry.notice(signInPrefix + stubLink)
	if link := Waiting(); link != stubLink {
		t.Errorf("whoever started it would be told %q, want %q", link, stubLink)
	}

	// And once the device answers, it is taken away: a link nobody can use any
	// more must not be offered as the reason for some later silence.
	entry.done()
	if link := Waiting(); link != "" {
		t.Errorf("still offering %q after the device answered", link)
	}
}
