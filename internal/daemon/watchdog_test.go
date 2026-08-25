package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// serveOn starts a stub API on DefaultHost, on a port the test then hands to
// the watchdog. The watchdog asks the loopback address by port, which is the
// one thing about it worth not faking.
func serveOn(t *testing.T, h http.Handler) int {
	t.Helper()
	l, err := net.Listen("tcp", DefaultHost+":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := httptest.NewUnstartedServer(h)
	srv.Listener.Close() //nolint:errcheck // replaced by ours
	srv.Listener = l
	srv.Start()
	t.Cleanup(srv.Close)

	port, err := strconv.Atoi(strings.Split(l.Addr().String(), ":")[1])
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	return port
}

func quickly(t *testing.T) {
	t.Helper()
	every, patience, wedge, start := watchEvery, watchPatience, wedgeAfter, startPatience
	t.Cleanup(func() {
		watchEvery, watchPatience, wedgeAfter, startPatience = every, patience, wedge, start
	})
	watchEvery, watchPatience, wedgeAfter = 10*time.Millisecond, 50*time.Millisecond, 60*time.Millisecond
	startPatience = 30 * time.Millisecond
}

// silent is a sign-in nothing has been written to: a device coming up with
// credentials already stored, which is every start after the first.
func silent() *signIn { return &signIn{open: func(string) bool { return false }} }

// signedIn is a daemon that is not waiting on the network to sign in, which is
// every case but the one TestItSaysWhenItCannotReachSpotify is about.
func signedIn() (time.Duration, bool) { return 0, false }

// A daemon whose playback loop has wedged answers 503 to everything — that is
// the guard in the API doing its job — and never recovers. After long enough
// the watchdog gives up on it, which is what ends the process and lets a fresh
// device take over.
func TestTheWatchdogGivesUpOnAStuckDevice(t *testing.T) {
	quickly(t)

	// Counted rather than timed: the first two requests are answered and every
	// one after that is not. A sleep here would be a test that fails on a busy
	// machine, which is the kind that gets deleted rather than believed.
	var served atomic.Int32
	port := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if served.Add(1) > 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := watch(ctx, port, silent(), signedIn, func(string, ...any) {}); !errors.Is(err, ErrWedged) {
		t.Errorf("the watchdog answered %v, want it to give up", err)
	}
}

// A device that stops answering and comes back is not a wedge. This is the
// ordinary case — a laptop that slept, a network that blinked — and ending the
// daemon over it would be the cure being worse than the illness.
func TestTheWatchdogWaitsForOneThatComesBack(t *testing.T) {
	quickly(t)

	// Two answers, then two refusals — less than the patience — then answers
	// again, all counted so the run is the same on any machine.
	var served atomic.Int32
	port := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if n := served.Add(1); n == 3 || n == 4 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 20*watchEvery)
	defer cancel()
	if err := watch(ctx, port, silent(), signedIn, func(string, ...any) {}); err != nil {
		t.Errorf("the watchdog gave up on a device that came back: %v", err)
	}
}

// And a daemon that has not come up yet is not one that has stopped: the
// counting starts at the first good answer, not at the first request.
func TestTheWatchdogWaitsToBeStarted(t *testing.T) {
	quickly(t)

	port := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 4*wedgeAfter)
	defer cancel()
	if err := watch(ctx, port, silent(), signedIn, func(string, ...any) {}); err != nil {
		t.Errorf("the watchdog gave up on a device that had never answered: %v", err)
	}
}

// Waiting to be started is not the same as being left waiting. A daemon that
// never answers at all holds the port, holds the lock, and plays nothing, and
// what it used to write about that was every reason but the reason.
func TestTheWatchdogSaysWhyItNeverCameUp(t *testing.T) {
	quickly(t)

	port := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	entry := silent()
	entry.notice(signInPrefix + "https://accounts.spotify.com/authorize?x=1")

	said := make(chan string, 8)
	ctx, cancel := context.WithTimeout(context.Background(), 8*startPatience)
	defer cancel()
	if err := watch(ctx, port, entry, signedIn, func(format string, args ...any) {
		select {
		case said <- fmt.Sprintf(format, args...):
		default:
		}
	}); err != nil {
		t.Fatalf("the watchdog gave up on a device that had never answered: %v", err)
	}

	close(said)
	var lines []string
	for line := range said {
		lines = append(lines, line)
	}
	if len(lines) != 1 {
		t.Fatalf("said %d things about a device that never came up, want one:\n%s",
			len(lines), strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], "signed in") || !strings.Contains(lines[0], "authorize?x=1") {
		t.Errorf("said %q, want the sign-in and the link it is waiting on", lines[0])
	}
}

// The other reason a device stays away with nothing wrong with it: Spotify
// cannot be reached, and it is waiting on the network rather than on a person.
// It comes up on its own when the network does — so the line says what is
// happening, and does not ask anybody to do anything about it.
func TestItSaysWhenItCannotReachSpotify(t *testing.T) {
	quickly(t)

	port := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	said := make(chan string, 8)
	ctx, cancel := context.WithTimeout(context.Background(), 8*startPatience)
	defer cancel()

	trying := func() (time.Duration, bool) { return 7 * time.Minute, true }
	if err := watch(ctx, port, silent(), trying, func(format string, args ...any) {
		select {
		case said <- fmt.Sprintf(format, args...):
		default:
		}
	}); err != nil {
		t.Fatalf("the watchdog gave up on a device that was still trying: %v", err)
	}

	close(said)
	var lines []string
	for line := range said {
		lines = append(lines, line)
	}
	if len(lines) != 1 {
		t.Fatalf("said %d things, want one:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], "cannot reach Spotify") || !strings.Contains(lines[0], "7m") {
		t.Errorf("said %q, want the reason and how long it has been going on", lines[0])
	}
}

// And once the device answers, that link is spent: a later silence has some
// other reason, and offering the old one as the reason would be a lie.
func TestTheSignInIsSpentOnceTheDeviceAnswers(t *testing.T) {
	quickly(t)

	var served atomic.Int32
	port := serveOn(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if served.Add(1) > 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	entry := silent()
	entry.notice(signInPrefix + "https://accounts.spotify.com/authorize?x=1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := watch(ctx, port, entry, signedIn, func(string, ...any) {}); !errors.Is(err, ErrWedged) {
		t.Fatalf("the watchdog answered %v, want it to give up", err)
	}
	if link := entry.pending(); link != "" {
		t.Errorf("the device is still said to be waiting on %q after it answered", link)
	}
}
