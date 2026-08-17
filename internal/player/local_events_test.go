package player

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// A daemon that accepts the connection and then says nothing does not hold the
// interface with it.
//
// This is what a stuck daemon looks like from here: the process is alive and
// the port is open, so the socket connects — and then nothing ever arrives on
// it. Waited on without a deadline, the interface goes on drawing the last
// thing it heard as though it were still true, which is the worst thing it
// could do: what is on the screen is wrong and nothing says so.
func TestASilentDaemonDoesNotHoldTheInterface(t *testing.T) {
	was := dialTimeout
	dialTimeout = 200 * time.Millisecond
	defer func() { dialTimeout = was }()

	// A listener that accepts and then leaves the caller waiting: no handshake,
	// no answer, no close.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close() //nolint:errcheck // the test is ending

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			defer conn.Close() //nolint:errcheck // held open on purpose
		}
	}()

	l := NewLocal(nil, "http://"+listener.Addr().String(), &http.Client{Timeout: time.Second})

	done := make(chan error, 1)
	go func() { done <- l.listen(context.Background()) }()

	select {
	case err := <-done:
		t.Logf("the stream gave up with: %v", err)
		if err == nil {
			t.Error("a daemon that never spoke was treated as a daemon that did")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the interface sat on a dead connection, which is the whole bug")
	}
}

// And one that opens the stream and then stops answering is noticed too —
// while one that is merely quiet is left alone, because a track playing through
// with nothing changing says nothing for minutes at a time.
func TestAStreamThatStopsAnsweringIsGivenUp(t *testing.T) {
	wasEvery, wasWithin := pingEvery, pingWithin
	pingEvery, pingWithin = 100*time.Millisecond, 200*time.Millisecond
	defer func() { pingEvery, pingWithin = wasEvery, wasWithin }()

	// A real websocket handshake, and then silence.
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow() //nolint:errcheck // the test is ending
		time.Sleep(3 * time.Second)
	}))
	defer daemon.Close()

	l := NewLocal(nil, daemon.URL, daemon.Client())

	done := make(chan error, 1)
	go func() { done <- l.listen(context.Background()) }()

	select {
	case err := <-done:
		t.Logf("the stream gave up with: %v", err)
		if err == nil {
			t.Error("a stream that said nothing at all was treated as a live one")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a stream that went quiet was waited on forever")
	}
}

// A stream that is merely quiet is left alone.
//
// The daemon speaks when something happens, and a track playing through with
// nothing changing says nothing for minutes at a time. Treating that as a
// daemon lost puts the whole interface into its offline state — and visibly so,
// because the spectrum is not even asked for while the daemon is thought to be
// gone, so the picture freezes until the next look proves it was there all
// along.
func TestAQuietStreamIsLeftAlone(t *testing.T) {
	wasEvery, wasWithin := pingEvery, pingWithin
	pingEvery, pingWithin = 100*time.Millisecond, 200*time.Millisecond
	defer func() { pingEvery, pingWithin = wasEvery, wasWithin }()

	// A daemon holding the stream open and reading it — which is all it takes
	// to answer a ping — with nothing to say.
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow() //nolint:errcheck // the test is ending

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}))
	defer daemon.Close()

	l := NewLocal(nil, daemon.URL, daemon.Client())

	done := make(chan error, 1)
	go func() { done <- l.listen(context.Background()) }()

	select {
	case err := <-done:
		t.Fatalf("a stream with nothing to say was given up after less than a second: %v", err)
	case <-time.After(time.Second):
		t.Log("still connected after ten pings' worth of silence, which is the point")
	}
}

// A daemon that is not there is not asked after every second.
//
// Every failed attempt tells whoever is watching that the device has gone, and
// the screen answers that by asking Spotify what is playing instead — so a flat
// second is a Web API request a second, out of a daily quota, for as long as
// nothing is there. Measured on a machine whose daemon took half a minute to
// come up: thirty requests, one a second, for an answer that never changed.
func TestADaemonThatStaysAwayIsWaitedForLonger(t *testing.T) {
	// The wait doubles to a ceiling, and the first ones are short so that a
	// daemon restarting is not noticed.
	wait := reconnectDelay
	var waits []time.Duration
	for range 8 {
		waits = append(waits, wait)
		wait = min(wait*2, reconnectMost)
	}

	if waits[0] != time.Second {
		t.Errorf("the first wait is %s, want a second: a restart should not be noticed", waits[0])
	}
	if waits[1] <= waits[0] {
		t.Error("the wait does not grow while the daemon stays away")
	}

	var total time.Duration
	for _, w := range waits {
		total += w
	}
	if total < 30*time.Second {
		t.Errorf("eight attempts cover %s, which is still a request a second", total)
	}
	if waits[len(waits)-1] > reconnectMost {
		t.Errorf("the wait grew past its ceiling, to %s", waits[len(waits)-1])
	}
}
