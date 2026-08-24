package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/daemon"
	"github.com/gofrs/flock"

	"github.com/pottom/spindle/internal/build"
	"github.com/pottom/spindle/internal/xdg"
)

const (
	// DefaultPort is where the local API listens. The TUI looks here first.
	DefaultPort = 3678

	// DefaultHost keeps the API off the network: it can start and stop
	// playback, so it has no business being reachable from anywhere else.
	DefaultHost = "127.0.0.1"

	// deviceName is what shows up in every Spotify client's device list.
	deviceName = "spindle"

	// authCallbackPort receives librespot's own OAuth redirect. It is not the
	// same authorisation as the Web API token: playing audio needs the
	// streaming scope, which a Web API app cannot ask for.
	authCallbackPort = 8899
)

// ErrAlreadyRunning reports that another daemon holds the lock. It is not a
// failure — it means the device the caller wanted already exists.
var ErrAlreadyRunning = errors.New("a spindle daemon is already running")

// Options configures a daemon run. The zero value is usable.
type Options struct {
	// Port overrides DefaultPort.
	Port int

	// Quality is what to ask Spotify for. The empty value means DefaultQuality.
	Quality Quality

	// Crossfade is how long one track overlaps the next. Zero is gapless.
	Crossfade time.Duration

	// Notify says whether each new track is announced to the desktop.
	Notify bool

	// Log receives the daemon's own logging.
	Log io.Writer
}

// Run starts the Connect device and blocks until ctx is cancelled.
//
// It takes an exclusive lock first: two daemons would fight over the same
// credentials and register the same device twice, so the second one leaves.
func Run(ctx context.Context, opts Options) error {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return err
	}

	lock := flock.New(filepath.Join(dir, "daemon.lock"))
	held, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("lock daemon: %w", err)
	}
	if !held {
		return ErrAlreadyRunning
	}
	defer lock.Unlock() //nolint:errcheck // the process is ending anyway

	if err := writePID(); err != nil {
		return err
	}
	defer removePID()

	port := opts.Port
	if port == 0 {
		port = DefaultPort
	}
	out := opts.Log
	if out == nil {
		out = io.Discard
	}
	quality := opts.Quality
	if quality == "" {
		quality = DefaultQuality
	}
	// The sign-in reads the log on its way past, because the one line a first
	// run needs a person to act on is written there and nowhere else. See
	// signin.go.
	entry := newSignIn()
	// A link outlives nothing: a daemon that has gone is not waiting to be
	// signed in, whatever it left on disk on its way past.
	defer entry.forget()
	log := newLogger(out, entry.notice)
	// Which spindle is playing. The daemon outlives the interface that started
	// it, so the two can be different builds, and this is the only place that
	// says so.
	log.Infof("spindle %s starting", build.Version())

	cacheDir, err := xdg.CacheDir()
	if err != nil {
		// Without somewhere to keep them, tempos last only as long as this run.
		cacheDir = ""
	}

	api, err := daemon.NewApiServer(log, DefaultHost, port, "", "", "")
	if err != nil {
		return fmt.Errorf("start local api: %w", err)
	}

	app, err := daemon.New(&daemon.Options{
		Logger:     log,
		StateStore: newStore(filepath.Join(dir, "daemon.json")),
		APIServer:  api,
		Config:     playbackConfig(quality, opts.Crossfade, cacheDir),
	})
	if err != nil {
		return fmt.Errorf("create daemon: %w", err)
	}
	defer app.Close() //nolint:errcheck // nothing useful to do while shutting down

	// The announcer talks to the API this daemon has just started, over
	// loopback, so it goes up with it rather than being wired into the player:
	// what is playing is a question the API already answers.
	if opts.Notify {
		go watchAndNotify(ctx, port)
	}

	// The keys on the keyboard belong to the daemon for the same reason the
	// notifications do: it is what goes on playing when the interface is
	// closed, and pausing from the keyboard is the one thing anybody wants to
	// do to a player they cannot see.
	watchMediaKeys(ctx, fmt.Sprintf("http://%s:%d", DefaultHost, port), func(format string, args ...any) {
		log.Infof("spindle: "+format, args...)
	})

	// Run in a goroutine of its own so that a wedged playback loop cannot hold
	// the process here: app.Run does not come back from one, which is the whole
	// difficulty — see watchdog.go. When the watchdog gives up, this returns and
	// the stuck goroutine goes with the process.
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()

	return leave(ctx, done, wedged(ctx, port, entry, log), log)
}

// leave decides how Run ends: the playback loop coming back on its own, the
// watchdog giving up on it, or the daemon being asked to stop.
//
// The last of those is here because being asked to stop is not the same as
// stopping. go-librespot closes its API the moment the context ends and then
// goes on doing whatever it was doing, and one of the things it may be doing is
// waiting to be signed in — a wait that ignores the context it was given; see
// signin.go. The process was left with no API, its lock still held, and no way
// out.
//
// Measured, on a machine with no stored credentials: `spindle daemon restart`
// answered "the daemon stopped answering but did not let go within 5s" and
// started nothing at all. The daemon it had told to leave was still holding the
// lock the new one needed, and would have held it until somebody found it with
// pgrep. A stop that cannot stop is worse than no stop: it takes the device
// away and puts nothing back.
//
// So the loop is given a moment to come back tidily, and then it is left
// behind. It is the same bargain the wedged case makes, for the same reason:
// nothing outside a goroutine can unwedge it, the process is ending anyway, and
// the goroutine goes with the process.
func leave(ctx context.Context, done <-chan error, wedge <-chan error, log librespot.Logger) error {
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("run daemon: %w", err)
		}
		return nil
	case err := <-wedge:
		return err
	case <-ctx.Done():
		select {
		case <-done:
		case <-time.After(shutdownGrace):
			log.Warnf("the device was asked to stop and did not finish within %s; leaving without it", shutdownGrace)
		}
		return nil
	}
}

// playbackConfig is everything go-librespot is told about how to play.
//
// It is lifted out of Run so that it can be looked at without a Spotify account
// and a sound card: what is in here is a set of answers to somebody else's
// questions, and getting one of them wrong is silent — see audioDevice, which
// was not answered at all and cost a working device that played nothing.
func playbackConfig(quality Quality, crossfade time.Duration, cacheDir string) *daemon.Config {
	return &daemon.Config{
		DeviceName:       deviceName,
		DeviceType:       "computer",
		AudioBackend:     audioBackend,
		AudioDevice:      audioDevice,
		AudioBufferTime:  audioBufferTime,
		AudioPeriodCount: audioPeriodCount,
		Bitrate:          quality.Bitrate(),
		// go-librespot takes it in milliseconds, and applies it to the
		// transitions it prefetches: a track running into the next one, not a
		// skip, where an overlap would just delay the answer.
		CrossfadeDuration: int(crossfade / time.Millisecond),
		VolumeSteps:       100,
		InitialVolume:     50,
		Credentials: daemon.CredentialsConfig{
			Type:        "interactive",
			Interactive: daemon.InteractiveCredentials{CallbackPort: authCallbackPort},
		},
		// Not for the audio, which is left to stream: this is where the
		// measured tempos live, and they are only worth measuring once.
		Cache: daemon.CacheConfig{Dir: cacheDir},
	}
}

// wedged reports the watchdog's verdict, once.
func wedged(ctx context.Context, port int, entry *signIn, log librespot.Logger) <-chan error {
	out := make(chan error, 1)
	go func() {
		if err := watch(ctx, port, entry, func(format string, args ...any) { log.Warnf(format, args...) }); err != nil {
			log.Warnf("the device has not answered for %s; leaving so a fresh one can take over", wedgeAfter)
			out <- err
		}
	}()
	return out
}
