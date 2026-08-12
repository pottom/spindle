package player

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A daemon answering out of a cupboard is not a daemon that is up to date.
//
// Its API and its playback session run on one goroutine. When that goroutine
// stops moving the API goes on answering — with the last answer it gave, and a
// Warning saying so. Everything else this end has to tell the two apart is
// silence, and silence is not a fault here: the daemon speaks only when
// something happens, and the socket's ping is answered by the API's own
// goroutine, so a session that has seized up still says "yes, I am here".
func TestALastAnswerIsNotTakenForANewOne(t *testing.T) {
	var stale bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stale {
			w.Header().Set("Warning", `110 - "Response is Stale"`)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_id":"d","volume":50,"volume_steps":100,
			"track":{"uri":"spotify:track:one","name":"One","position":30000,"duration":200000}}`))
	}))
	defer srv.Close()

	l := &Local{addr: srv.URL, http: srv.Client()}
	ctx := context.Background()

	// Fresh: the position is carried forward by this end's clock, because the
	// daemon only speaks when something happens.
	if err := l.refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	st := l.localState()
	if st == nil {
		t.Fatal("no state at all")
	}
	if st.Stalled {
		t.Error("a fresh answer was taken for a stale one")
	}
	l.snapshotAt = time.Now().Add(-4 * time.Second)
	if got := l.localState().Progress; got < 33*time.Second {
		t.Errorf("the clock is not being carried forward: %v after four seconds", got)
	}

	// Stale: the position stays where the daemon left it. Carrying it forward
	// assumes nothing has changed since the daemon last spoke, and a daemon
	// serving its last answer is exactly the case where that is wrong — the
	// screen would count out a record that stopped a minute ago.
	stale = true
	if err := l.refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if !l.localState().Stalled {
		t.Error("the Warning was not noticed")
	}
	l.snapshotAt = time.Now().Add(-4 * time.Second)
	if got := l.localState().Progress; got != 30*time.Second {
		t.Errorf("the clock ran on while the daemon was stuck: %v, want 30s", got)
	}

	// And it clears itself when the daemon comes back.
	stale = false
	if err := l.refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if l.localState().Stalled {
		t.Error("it stayed stuck after the daemon started answering again")
	}
}
