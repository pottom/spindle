package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pottom/spindle/internal/browser"
	"github.com/pottom/spindle/internal/xdg"
)

// The sign-in nobody was told about.
//
// The daemon needs an authorisation of its own. The Web API token in
// token.json cannot ask for the streaming scope, so before it can play a note
// go-librespot signs in separately, once per machine, and keeps what it earns
// in daemon.json. Every start after that is silent.
//
// The first one is not. With nothing stored, go-librespot writes a link to its
// log and blocks on the redirect coming back — see session.Connect, which is
// sitting on `code := <-codeCh` the whole time. Nothing opened a browser at it
// and nothing else was told, so the link went into a file in ~/.local/state
// that its reader had no reason to open.
//
// Measured, on a Linux machine the program had never run on: the daemon came
// up, took the port, held the lock, and answered 503 to every request for as
// long as it was left alone. The interface next to it looked half alive —
// playlists and search worked, because those are the Web API — and the device
// simply was not in Spotify's list. The log filled with "the player did not
// take a status request within 10s", which says what the loop was not doing and
// nothing at all about why. The one line that mattered was two hundred lines up
// and phrased as though somebody were reading over its shoulder.
//
// So the line is read here as it goes past, a browser is sent to it, and it is
// said again in words that name the daemon and say what is waiting on what.

// signInPrefix is go-librespot's whole side of this conversation. Matching a
// log line is a poor sort of interface and it is the only one there is: the
// session takes a logger and offers no hook, and the alternative — waiting for
// the fork to grow one — leaves first runs broken in the meantime.
const signInPrefix = "to complete authentication visit the following link: "

// signIn is what is known about the authorisation the device is waiting on.
// The zero value is not usable; see newSignIn.
type signIn struct {
	// open is the launcher, so a test can watch where a browser was sent
	// without one opening on the machine running the test.
	open func(string) bool

	mu sync.Mutex
	// url is the link last written, and "" once the device has answered: a
	// session that is up is a sign-in that has been finished, and a stale link
	// offered as the reason for a later silence would be a lie.
	url string
	// sent is the link a browser has already been sent to, so that a line
	// repeated does not open a second tab. A fresh attempt carries a fresh
	// challenge, and that one is worth opening.
	sent string
}

func newSignIn() *signIn { return &signIn{open: browser.Open} }

// notice reads one line on its way to the log and answers with a line of its
// own where that one was the sign-in link. See newLogger, which calls it.
func (s *signIn) notice(text string) (string, bool) {
	url, found := strings.CutPrefix(text, signInPrefix)
	if !found || url == "" {
		return "", false
	}

	s.mu.Lock()
	s.url = url
	opened := s.sent == url
	s.sent = url
	s.mu.Unlock()

	s.keep(url)

	if opened {
		return "", false
	}

	// The link is said in full either way. The launcher answers whether it
	// started, not whether anything came of it: over SSH, and on a machine with
	// no session to open a window on, it starts happily and nothing appears.
	if s.open(url) {
		return "spindle: this device is not signed in to Spotify yet. A browser has been opened — " +
			"if nothing happens there, visit: " + url, true
	}
	return "spindle: this device is not signed in to Spotify yet, and no browser could be opened. " +
		"Visit: " + url, true
}

// pending is the link the device is waiting on, or "" when it is not waiting on
// one.
func (s *signIn) pending() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

// done says the device has answered, which can only have happened on the far
// side of the sign-in.
func (s *signIn) done() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.url = ""
	s.mu.Unlock()
	s.forget()
}

// The link, left where another process can find it.
//
// A daemon that is waiting to be signed in knows why it has not come up, and
// the process that started it does not: they share no channel but the
// filesystem, because the channel they would otherwise use is the API, and the
// API not answering is the thing being explained. Without this, `spindle daemon
// start` waited its twenty seconds and said "the daemon did not come up" about
// a daemon that had come up perfectly well and was waiting for a person.
//
// It is a file rather than a line in the log because a caller should not have
// to parse a log to find out what happened to the thing it just started.

// keep writes the link down for whoever started this daemon.
func (s *signIn) keep(url string) {
	path, err := signInPath()
	if err != nil {
		return // nothing to lose: the log still has it
	}
	_ = os.WriteFile(path, []byte(url), 0o600)
}

// forget takes it away again. Called when the device answers and when the
// daemon leaves, so that a link nobody can use any more is not offered to the
// next caller as the reason for something else.
func (s *signIn) forget() {
	if path, err := signInPath(); err == nil {
		_ = os.Remove(path)
	}
}

func signInPath() (string, error) {
	dir, err := xdg.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "signin.url"), nil
}

// Waiting is the link a daemon is waiting to be signed in with, or "" where no
// daemon is waiting on one. It is what turns "the daemon did not come up" into
// something a person can act on — see Start.
func Waiting() string {
	path, err := signInPath()
	if err != nil {
		return ""
	}
	url, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(url))
}
