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
	was, wasSilent := dialTimeout, silentFor
	dialTimeout, silentFor = 200*time.Millisecond, 200*time.Millisecond
	defer func() { dialTimeout, silentFor = was, wasSilent }()

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

// And one that opens the stream and then goes quiet is noticed too.
func TestAStreamThatGoesQuietIsGivenUp(t *testing.T) {
	was := silentFor
	silentFor = 200 * time.Millisecond
	defer func() { silentFor = was }()

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
