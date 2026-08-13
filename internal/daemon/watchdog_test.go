package daemon

import (
	"context"
	"errors"
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
	every, patience, wedge := watchEvery, watchPatience, wedgeAfter
	t.Cleanup(func() { watchEvery, watchPatience, wedgeAfter = every, patience, wedge })
	watchEvery, watchPatience, wedgeAfter = 10*time.Millisecond, 50*time.Millisecond, 60*time.Millisecond
}

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

	if err := watch(ctx, port, func(string, ...any) {}); !errors.Is(err, ErrWedged) {
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
	if err := watch(ctx, port, func(string, ...any) {}); err != nil {
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
	if err := watch(ctx, port, func(string, ...any) {}); err != nil {
		t.Errorf("the watchdog gave up on a device that had never answered: %v", err)
	}
}
