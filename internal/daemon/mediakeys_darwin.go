package daemon

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework CoreGraphics -framework CoreFoundation
int startMediaKeys(void);
void runMediaKeys(void);
int mediaKeysAllowed(void);
void askForMediaKeys(void);
*/
import "C"

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
)

// The keys on the keyboard itself.
//
// A player that cannot be paused from the row of keys made for pausing players
// is a player people quietly stop using. macOS routes them to whichever
// application it believes is playing, and an application is exactly what
// spindle is not — so the events are read where they pass, which is what every
// terminal player that answers these keys ends up doing.
//
// The cost is a permission: the terminal spindle was started from has to be
// allowed to monitor input. Until it is, the tap cannot be created, and the one
// thing worth doing about that is saying so plainly.

const (
	keyPlay = 16
	keyNext = 19
	keyPrev = 20
)

// keys is what the tap talks to: where the daemon's own API is, whether it is
// worth taking a key at all, and where to log.
var keys struct {
	addr string
	logf func(format string, args ...any)

	// active is when this daemon last had music of its own — playing or
	// paused. A key is taken while that is recent, and passed on otherwise.
	active atomic.Int64
}

// mediaKeysHold is how long after the music stops the keys still belong to
// spindle.
//
// Measured the hard way: taking the keys only while something plays means the
// first press pauses, the daemon then reports itself stopped, and the second
// press goes past us to the system — which starts Apple Music. Play, pause,
// play is one gesture and has to reach one player, so the keys stay ours for a
// while after the music stops, and go back to the system once the listener has
// plainly moved on.
const mediaKeysHold = 30 * time.Minute

//export spindleMediaKey
func spindleMediaKey(key C.int) C.int {
	path, ok := mediaKeyPath(int(key), keys.active.Load(), time.Now())
	if !ok {
		return 0
	}

	// Sent from a goroutine of its own: this runs on the event tap's thread,
	// and a request made there would hold up every key the system delivers.
	go post(keys.addr + path)
	return 1
}

// mediaKeyPath decides what a press means: which command it is, and whether it
// is ours to answer at all.
//
// A key is taken only while the music here is somebody's current business.
// Otherwise it is passed on: a browser or Spotify's own client is as likely to
// be what the press was for, and a player that swallows everybody else's keys
// is worse than one that answers none.
func mediaKeyPath(key int, activeAt int64, now time.Time) (string, bool) {
	if activeAt == 0 || now.Sub(time.UnixMilli(activeAt)) > mediaKeysHold {
		return "", false
	}

	switch key {
	case keyPlay:
		return "/player/playpause", true
	case keyNext:
		return "/player/next", true
	case keyPrev:
		return "/player/prev", true
	default:
		return "", false
	}
}

// watchMediaKeys listens for the media keys until ctx is cancelled.
func watchMediaKeys(ctx context.Context, addr string, logf func(string, ...any)) {
	keys.addr, keys.logf = addr, logf

	go followPlaying(ctx, addr)

	go func() {
		// The tap belongs to the thread that created it, and so does the run
		// loop it is fed by. Nothing else may be scheduled there.
		runtime.LockOSThread()

		// Asked for rather than assumed. Without this permission the tap is
		// created and never hears a thing, which looks exactly like a keyboard
		// nobody is touching — and is why this was worth a system call rather
		// than a guess.
		if C.mediaKeysAllowed() == 0 {
			logf("the media keys need permission: allow spindle under System Settings › " +
				"Privacy & Security › Input Monitoring, then start the daemon again")
			C.askForMediaKeys()
			return
		}

		if C.startMediaKeys() == 0 {
			logf("the media keys could not be listened for")
			return
		}
		logf("listening for the media keys")
		C.runMediaKeys()
	}()
}

// followPlaying keeps the one fact the tap needs — whether this daemon is the
// thing playing — close at hand, because the tap cannot wait for an answer.
func followPlaying(ctx context.Context, addr string) {
	status := addr + "/status"
	for ctx.Err() == nil {
		// Loaded is enough, playing is not required: a paused track is still
		// what the keys are for.
		if now, err := nowPlaying(ctx, status); err == nil && (now.Uri != "" || !now.Stopped) {
			keys.active.Store(time.Now().UnixMilli())
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(playingPoll):
		}
	}
}

// playingPoll is how often that fact is refreshed. Often enough that the keys
// start working within a moment of the music starting, cheap enough to be
// nothing: it is one request to a server on this machine.
const playingPoll = 2 * time.Second
