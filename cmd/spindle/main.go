// Command spindle is a Spotify Connect remote control for the terminal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/auth"
	"github.com/pottom/spindle/internal/daemon"
	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui"
	"github.com/pottom/spindle/internal/ui/cover"
)

// reportFatal prints an error and leaves. A missing client id is not really an
// error so much as an unfinished setup, so it gets the instructions instead of a
// one-line complaint.
// controlCommandIn finds the daemon-driving command in an argument list, and
// hands back everything else as its arguments — wherever the flags were
// written.
//
// It stops at the first word that is a command of another kind. "spindle daemon
// status" asks the daemon command about itself and has nothing to do with the
// status of what is playing; without this it was answered by the wrong one of
// the two.
func controlCommandIn(args []string) (name string, rest []string, ok bool) {
	for i, arg := range args {
		if otherCommands[arg] {
			return "", nil, false
		}
		if !controlCommands[arg] {
			continue
		}
		rest = append(rest, args[:i]...)
		return arg, append(rest, args[i+1:]...), true
	}
	return "", nil, false
}

// otherCommands are the subcommands that are not about a running daemon's
// playback. They own every word that follows them.
var otherCommands = map[string]bool{
	"login": true, "quality": true, "crossfade": true,
	"notify": true, "daemon": true,
}

func reportFatal(err error) {
	if errors.Is(err, auth.ErrNoClientID) {
		fmt.Fprintln(os.Stderr, auth.SetupHelp())
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "spindle:", err)
	os.Exit(1)
}

func main() {
	// Subcommands come before flags so "spindle login" reads the way it looks.
	if len(os.Args) > 1 {
		// The daemon-driving commands leave through os.Exit rather than
		// reportFatal: they answer with an exit code of their own, so a script
		// can tell a missing daemon from a silent one.
		//
		// The command is looked for past any leading flags, because "spindle
		// --json status" is how everybody writes it the first time and falling
		// through to the interface for it is a baffling answer.
		if name, rest, ok := controlCommandIn(os.Args[1:]); ok {
			os.Exit(runControl(context.Background(), name, rest))
		}

		switch os.Args[1] {
		case "login":
			if err := runLogin(context.Background(), os.Args[2:]); err != nil {
				reportFatal(err)
			}
			return
		case "quality":
			if err := runQuality(os.Args[2:]); err != nil {
				reportFatal(err)
			}
			return
		case "crossfade":
			if err := runCrossfade(os.Args[2:]); err != nil {
				reportFatal(err)
			}
			return
		case "notify":
			if err := runNotify(os.Args[2:]); err != nil {
				reportFatal(err)
			}
			return
		case "daemon":
			if err := runDaemon(os.Args[2:]); err != nil {
				reportFatal(err)
			}
			return
		}
	}

	mock := flag.Bool("mock", false, "run against the offline mock backend, without auth or network")
	backend := flag.String("cover", "auto", "artwork backend: auto, kitty or halfblock")
	info := flag.Bool("cover-info", false, "report what the terminal supports and exit")
	flag.Parse()

	if *info {
		reportCoverSupport()
		return
	}

	ctx, stopWatching := context.WithCancel(context.Background())
	defer stopWatching()

	// Authorise before Bubble Tea takes the terminal: the browser flow needs to
	// print a URL and be readable while it waits.
	backendPlayer, err := openBackend(ctx, *mock)
	if err != nil {
		reportFatal(err)
	}

	cell := cover.DetectCellSize(os.Stdout)
	renderer, err := coverRenderer(*backend, cell)
	if err != nil {
		reportFatal(err)
	}
	loader := cover.NewLoader(renderer, &http.Client{Timeout: 15 * time.Second})

	final, err := tea.NewProgram(ui.New(backendPlayer, loader, cell)).Run()
	if err != nil {
		reportFatal(err)
	}

	// Quitting leaves the music playing; Q asks for it to stop as well. Acting
	// on it here rather than inside Update keeps the shutdown out of a UI that
	// is still drawing.
	if m, ok := final.(ui.Model); ok && m.StopDaemonRequested() {
		pauseBeforeLeaving(backendPlayer)
		stopWatching()
		if err := daemon.Stop(context.Background()); err != nil && !errors.Is(err, daemon.ErrNoDaemon) {
			fmt.Fprintln(os.Stderr, "spindle:", err)
		}
	}
}

// pauseBeforeLeaving stops the music before the device that was playing it
// disappears.
//
// Measured: a device that vanishes mid-track leaves Spotify with no session at
// all — it answers "no active playback device" and remembers no position — so
// coming back starts the track from the top. Pausing first ends the session
// tidily, and Spotify keeps the place: the next start comes up where the
// listener left off.
//
// A failure here is not worth a word: the device is about to be stopped anyway.
func pauseBeforeLeaving(p player.Player) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = p.Pause(ctx)
}

// openBackend picks between the offline mock, the local daemon and the bare
// Web API.
//
// The daemon is started if it is not already up. Failing to start one is not
// fatal: everything except queue editing and instant updates works without it,
// and refusing to run would be a worse answer than saying so.
func openBackend(ctx context.Context, mock bool) (player.Player, error) {
	if mock {
		return player.NewMock(), nil
	}

	session, err := auth.NewSession(ctx, os.Stdout)
	if err != nil {
		return nil, err
	}
	web := player.NewSpotify(session.Client(ctx))

	if !daemon.Running() {
		if err := detachDaemon(); err != nil {
			fmt.Fprintln(os.Stderr, "spindle: no local playback device:", err)
			return web, nil
		}
	}

	local := player.NewLocal(web, daemon.Addr(), nil)
	go local.Watch(ctx)
	return local, nil
}

// coverRenderer picks the artwork backend. The terminal is probed before Bubble
// Tea claims it, so the query and its reply cannot collide with the event loop's
// own input handling.
func coverRenderer(backend string, cell cover.CellSize) (cover.Renderer, error) {
	switch backend {
	case "halfblock":
		return cover.NewHalfblock(cell), nil
	case "kitty":
		return cover.NewKitty(os.Stdout, cell), nil
	case "auto":
		if cover.SupportsKitty(os.Stdout, os.Stdin) {
			return cover.NewKitty(os.Stdout, cell), nil
		}
		return cover.NewHalfblock(cell), nil
	default:
		return nil, fmt.Errorf("unknown cover backend %q: want auto, kitty or halfblock", backend)
	}
}

// reportCoverSupport prints what the terminal was found to support, so a fallback
// to halfblock can be told apart from a kitty backend that is simply not drawing.
func reportCoverSupport() {
	cell := cover.DetectCellSize(os.Stdout)
	source := "measured via TIOCGWINSZ"
	if !cell.Measured {
		source = "assumed; the terminal reported no pixel size"
	}

	kitty := cover.SupportsKitty(os.Stdout, os.Stdin)
	backend := "halfblock"
	if kitty {
		backend = "kitty"
	}

	fmt.Printf("terminal:   TERM=%s TERM_PROGRAM=%s\n", os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	fmt.Printf("cell size:  %d × %d px (%s)\n", cell.Width, cell.Height, source)
	fmt.Printf("kitty:      %v\n", kitty)
	fmt.Printf("backend:    %s\n", backend)
	fmt.Printf("artwork:    %d × %d cells = %d × %d px\n",
		20, 10, 20*cell.Width, 10*cell.Height)
}
