package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/devgianlu/go-librespot/daemon"
	"github.com/gofrs/flock"

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
	log := newLogger(out)

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
		Config: &daemon.Config{
			DeviceName:   deviceName,
			DeviceType:   "computer",
			AudioBackend: audioBackend,
			Bitrate:      quality.Bitrate(),
			// go-librespot takes it in milliseconds, and applies it to the
			// transitions it prefetches: a track running into the next one,
			// not a skip, where an overlap would just delay the answer.
			CrossfadeDuration: int(opts.Crossfade / time.Millisecond),
			VolumeSteps:       100,
			InitialVolume:     50,
			Credentials: daemon.CredentialsConfig{
				Type:        "interactive",
				Interactive: daemon.InteractiveCredentials{CallbackPort: authCallbackPort},
			},
			// Not for the audio, which is left to stream: this is where the
			// measured tempos live, and they are only worth measuring once.
			Cache: daemon.CacheConfig{Dir: cacheDir},
		},
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

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("run daemon: %w", err)
	}
	return nil
}
