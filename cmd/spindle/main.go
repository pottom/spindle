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
	"github.com/pottom/spindle/internal/build"
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
	"notify": true, "daemon": true, "callback": true,
}

// usage is what --help prints.
//
// The flag package only knows about flags, and spindle is mostly subcommands:
// the default help listed three options and none of the eight things the
// program actually does, which reads as a program that does almost nothing.
func usage() {
	fmt.Fprintf(os.Stderr, "spindle %s — a Spotify player for the terminal\n", build.Version())
	fmt.Fprint(os.Stderr, `
    spindle                              open the interface
    spindle login [client id]            authorise, once
    spindle version                      which build this is

  The playback device, which keeps playing after the interface is closed:

    spindle daemon start | stop          start or stop it
    spindle daemon restart | status      restart it, or ask whether one runs

  Driving what is already playing, for a key binding or a status bar:

    spindle play | pause | toggle        resume, stop, or flip between them
    spindle next | prev                  move through the queue
    spindle status [--line | --format]   what is playing; --follow to keep going
    spindle queue                        the playing track and what follows
    spindle volume [0-100]               report the level, or set it
    spindle seek 90 | +30 | -15          to a position, or by an offset

  Settings, kept between runs:

    spindle quality [low|normal|high]    what to ask Spotify for
    spindle crossfade [seconds|off]      how long one track overlaps the next
    spindle notify on | off              announce each new track to the desktop
    spindle callback [port]              where the browser is sent back to when logging in

  --json on any of the driving commands prints the daemon's own answer.
  Exit codes: 0 done, 1 refused, 3 no daemon is running, 4 nothing is playing.

Options for the interface:
`)
	flag.PrintDefaults()
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
	// The flags are declared before anything else so that the help has them to
	// list: the word "help" is answered from the subcommands below, which used
	// to run before these existed, and the section headed "Options for the
	// interface" printed nothing under it for as long as it has been there.
	mock := flag.Bool("mock", false, "run against the offline mock backend, without auth or network")
	backend := flag.String("cover", "auto", "artwork backend: auto, kitty or halfblock")
	info := flag.Bool("cover-info", false, "report what the terminal supports and exit")
	fps := flag.Int("fps", 60, "how many frames a second the visualisers are drawn at, 15 to 120")
	flag.Usage = usage

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
		// The word as well as the flags: somebody looking for help types
		// whichever comes to mind first, and one of the two answering with the
		// interface is a poor joke.
		case "help", "--help", "-h":
			usage()
			return
		// Which spindle this is. It answers to the word and to the flag,
		// because whoever is asking has been surprised by a build already.
		case "version", "--version", "-v":
			fmt.Println("spindle", build.Version())
			return
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
		case "callback":
			if err := runCallback(os.Args[2:]); err != nil {
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

	flag.Parse()

	if *info {
		reportCoverSupport()
		return
	}

	// Before anything is drawn, and once: everything on the visual screens is
	// tuned per frame, and the conversion for the rate actually asked for is
	// worked out here. See internal/ui/pace.go.
	ui.SetFrameRate(*fps)

	ctx, stopWatching := context.WithCancel(context.Background())
	defer stopWatching()

	// Authorise before Bubble Tea takes the terminal: the browser flow needs to
	// print a URL and be readable while it waits.
	backendPlayer, err := openBackend(ctx, *mock)
	if err != nil {
		reportFatal(err)
	}

	found := cover.Probe(os.Stdout, os.Stdin)
	cell := cellSize(found)
	renderer, err := coverRenderer(*backend, cell, found)
	if err != nil {
		reportFatal(err)
	}
	loader := cover.NewLoader(renderer, &http.Client{Timeout: 15 * time.Second})
	loader.SetGraphics(found)

	final, err := tea.NewProgram(
		ui.New(backendPlayer, loader, cell).WithNotes(artistNotes()),
	).Run()

	// Whatever the debug bar wrote down goes with the session that wrote it.
	ui.ForgetDebug()

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
// The daemon is started if it is not already up, and not waited for. Failing to
// start one is not fatal: everything except queue editing and instant updates
// works without it, and refusing to run would be a worse answer than saying so.
// Nor is being slow to come up — see daemon.Spawn.
func openBackend(ctx context.Context, mock bool) (player.Player, error) {
	if mock {
		return player.NewMock(), nil
	}

	session, err := auth.NewSession(ctx, os.Stdout)
	if err != nil {
		return nil, err
	}
	web := player.NewSpotify(session.Client(ctx))

	if _, _, err := daemon.Spawn(); err != nil {
		fmt.Fprintln(os.Stderr, "spindle: no local playback device:", err)
		return web, nil
	}

	local := player.NewLocal(web, daemon.Addr(), nil)
	go local.Watch(ctx)
	return local, nil
}

// coverRenderer picks the artwork backend. The terminal is probed before Bubble
// Tea claims it, so the query and its reply cannot collide with the event loop's
// own input handling.
func coverRenderer(backend string, cell cover.CellSize, found cover.Graphics) (cover.Renderer, error) {
	switch backend {
	case "halfblock":
		return cover.NewHalfblock(cell), nil
	case "kitty":
		return cover.NewKitty(os.Stdout, cell), nil
	case "auto":
		if found.Backend() == "kitty" {
			return cover.NewKitty(os.Stdout, cell), nil
		}
		return cover.NewHalfblock(cell), nil
	default:
		return nil, fmt.Errorf("unknown cover backend %q: want auto, kitty or halfblock", backend)
	}
}

// cellSize is how big one cell is, from the terminal's own answer where it gave
// one and from the kernel's window size where it did not.
//
// The terminal first, because it knows. The window size is filled in over ssh by
// whatever the client said, and measured over ssh from a Windows client it said
// 5 × 19 px — a cell nearly four times taller than it is wide, which no font is,
// and which would draw every cover the wrong shape.
func cellSize(g cover.Graphics) cover.CellSize {
	if g.Cell.Measured {
		return g.Cell
	}
	return cover.DetectCellSize(os.Stdout)
}

// reportCoverSupport prints what the terminal was found to support, so a fallback
// to halfblock can be told apart from a kitty backend that is simply not drawing.
func reportCoverSupport() {
	g := cover.Probe(os.Stdout, os.Stdin)

	cell := cellSize(g)
	source := "measured via TIOCGWINSZ"
	switch {
	case g.Cell.Measured:
		source = "the terminal said so"
	case !cell.Measured:
		source = "assumed; nothing believable was reported"
	}

	says := g.Name
	if says == "" {
		says = "(it did not say)"
	}
	fmt.Printf("terminal:   TERM=%s TERM_PROGRAM=%s\n", os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"))
	fmt.Printf("says it is: %s\n", says)
	fmt.Printf("cell size:  %d × %d px (%s)\n", cell.Width, cell.Height, source)
	fmt.Printf("kitty:      %v\n", g.Kitty)
	fmt.Printf("placehold:  %v\n", g.Placeholders)
	fmt.Printf("backend:    %s\n", g.Backend())
	fmt.Printf("artwork:    %d × %d cells = %d × %d px\n",
		20, 10, 20*cell.Width, 10*cell.Height)
}
