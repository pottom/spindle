package daemon

import (
	"testing"
	"time"
)

// The three keys spindle answers, and the ones it leaves alone.
func TestWhichKeysAreAnswered(t *testing.T) {
	now := time.Now()
	active := now.Add(-time.Second).UnixMilli()

	for key, want := range map[int]string{
		keyPlay: "/player/playpause",
		keyNext: "/player/next",
		keyPrev: "/player/prev",
	} {
		got, ok := mediaKeyPath(key, active, now)
		if !ok || got != want {
			t.Errorf("key %d = %q, %v; want %q", key, got, ok, want)
		}
	}

	// The volume keys belong to the system: the music has a level of its own,
	// and taking these would take that choice away.
	for _, key := range []int{0, 1, 7} {
		if _, ok := mediaKeyPath(key, active, now); ok {
			t.Errorf("key %d was taken, want it left to the system", key)
		}
	}
}

// Play, pause, play is one gesture and has to reach one player. Taking the keys
// only while something plays meant the first press paused, the daemon reported
// itself stopped, and the second press went past to the system — which started
// Apple Music.
func TestTheKeysStayOursAfterThePause(t *testing.T) {
	now := time.Now()

	justStopped := now.Add(-time.Minute).UnixMilli()
	if _, ok := mediaKeyPath(keyPlay, justStopped, now); !ok {
		t.Error("the play key was passed on a minute after the music stopped")
	}

	longSince := now.Add(-mediaKeysHold - time.Minute).UnixMilli()
	if _, ok := mediaKeyPath(keyPlay, longSince, now); ok {
		t.Error("the keys were still taken long after the listener moved on")
	}

	if _, ok := mediaKeyPath(keyPlay, 0, now); ok {
		t.Error("the keys were taken before this daemon had ever played anything")
	}
}
