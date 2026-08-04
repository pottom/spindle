package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/pottom/spindle/internal/daemon"
)

// Exit codes. A shell script has to be able to tell these apart: no daemon is
// something it can fix by starting one, nothing playing is not a failure at all
// and usually means "print nothing", and everything else is a real complaint.
const (
	exitOK       = 0
	exitFailed   = 1
	exitNoDaemon = 3
	exitIdle     = 4
)

// errIdle reports that the daemon is running with nothing loaded.
var errIdle = errors.New("nothing is playing")

// jsonFlag turns every command's output into the daemon's own JSON.
const jsonFlag = "--json"

// controlCommands are the subcommands that drive a running daemon. main.go
// matches against this so a new one only has to be added in a single place.
var controlCommands = map[string]bool{
	"play": true, "pause": true, "toggle": true,
	"next": true, "prev": true,
	"status": true, "queue": true,
	"volume": true, "seek": true,
}

// cli is one invocation of the command line.
type cli struct {
	remote *remote
	out    io.Writer
	errOut io.Writer
	json   bool
}

// runControl runs one daemon-driving command and returns the process exit code.
//
// It reports its own errors rather than handing them to reportFatal, because
// which failure it was is part of the answer and reportFatal only knows how to
// say 1.
func runControl(ctx context.Context, name string, args []string) int {
	c := &cli{remote: newRemote(daemon.Addr()), out: os.Stdout, errOut: os.Stderr}
	return c.run(ctx, name, args)
}

func (c *cli) run(ctx context.Context, name string, args []string) int {
	args, c.json = takeJSONFlag(args)

	switch err := c.dispatch(ctx, name, args); {
	case err == nil:
		return exitOK
	case errors.Is(err, errNoDaemon):
		// One line, and a way out of it: this is what a script or a person hits
		// first when nothing is playing music yet.
		fmt.Fprintf(c.errOut, "spindle: %v — start one with spindle daemon\n", err)
		return exitNoDaemon
	case errors.Is(err, errIdle):
		fmt.Fprintln(c.errOut, "spindle: nothing is playing")
		return exitIdle
	default:
		fmt.Fprintln(c.errOut, "spindle:", err)
		return exitFailed
	}
}

func (c *cli) dispatch(ctx context.Context, name string, args []string) error {
	switch name {
	case "status":
		return c.status(ctx, args)
	case "queue":
		return c.queue(ctx, args)
	case "play":
		return c.command(ctx, name, args, "/player/resume", "playing")
	case "pause":
		return c.command(ctx, name, args, "/player/pause", "paused")
	case "next":
		return c.command(ctx, name, args, "/player/next", "next")
	case "prev":
		return c.command(ctx, name, args, "/player/prev", "previous")
	case "toggle":
		return c.toggle(ctx, args)
	case "volume":
		return c.volume(ctx, args)
	case "seek":
		return c.seek(ctx, args)
	}
	return fmt.Errorf("unknown command %q", name)
}

// command sends one of the plain playback commands, having first checked that
// there is anything to send it about.
//
// The check costs a second request on the loopback interface and buys an exit
// code that means something: the daemon accepts a resume it has nothing to
// resume and answers 200, which would otherwise be reported as success.
func (c *cli) command(ctx context.Context, name string, args []string, path, done string) error {
	if err := expectNoArgs(name, args); err != nil {
		return err
	}
	if _, err := c.fetchStatus(ctx); err != nil {
		return err
	}
	if err := c.remote.post(ctx, path, nil); err != nil {
		return err
	}
	return c.say(done, map[string]string{"command": name})
}

// toggle flips between playing and paused, and says which it ended on.
//
// The daemon has a playpause endpoint that would do this in one request, but it
// answers nothing, leaving the caller to guess what it did. Reading the state
// first costs a loopback request and lets the answer be true.
func (c *cli) toggle(ctx context.Context, args []string) error {
	if err := expectNoArgs("toggle", args); err != nil {
		return err
	}

	st, err := c.fetchStatus(ctx)
	if err != nil {
		return err
	}

	path, state := "/player/pause", "paused"
	if st.Paused {
		path, state = "/player/resume", "playing"
	}
	if err := c.remote.post(ctx, path, nil); err != nil {
		return err
	}
	return c.say(state, map[string]string{"state": state})
}

// volume reports the level, or sets it. Both speak percent, which is what every
// other volume control does; the daemon counts in steps of its own.
//
// This is the one command that works while nothing is playing: the level
// belongs to the device rather than to the track, and setting it before
// starting anything is exactly when it is most useful.
func (c *cli) volume(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("spindle volume takes one argument at most, got %d", len(args))
	}

	_, st, err := c.readStatus(ctx)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return c.say(strconv.Itoa(st.volumePercent()), map[string]int{"volume": st.volumePercent()})
	}

	pct, err := parseVolume(args[0])
	if err != nil {
		return err
	}
	if err := c.remote.post(ctx, "/player/volume", map[string]any{"volume": pct * st.steps() / 100}); err != nil {
		return err
	}
	return c.say(strconv.Itoa(pct), map[string]int{"volume": pct})
}

// seek moves the playhead, either to a position or by an offset from wherever
// it is now.
func (c *cli) seek(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("spindle seek needs one position: seconds, +seconds or -seconds")
	}

	offset, relative, err := parseSeek(args[0])
	if err != nil {
		return err
	}

	st, err := c.fetchStatus(ctx)
	if err != nil {
		return err
	}

	// A relative seek is left to the daemon, which counts from the playhead as
	// it stands rather than from a status read a moment ago. What is printed is
	// worked out here from that older reading, so it can be a few milliseconds
	// behind where the playhead actually landed.
	body := map[string]any{"position": offset.Milliseconds(), "relative": relative}
	if err := c.remote.post(ctx, "/player/seek", body); err != nil {
		return err
	}

	landed := offset
	if relative {
		landed = millis(st.Track.Position) + offset
	}
	landed = min(max(landed, 0), millis(st.Track.Duration))
	return c.say(clock(landed), map[string]int64{"position": landed.Milliseconds()})
}

// fetchStatus reads the daemon's status and refuses to go on when there is no
// music in it — which is what every command but volume needs, either to know it
// has something to act on or, for seek, to know how long the track is.
func (c *cli) fetchStatus(ctx context.Context) (*daemonStatus, error) {
	_, st, err := c.readStatus(ctx)
	if err != nil {
		return nil, err
	}
	if st.idle() {
		return nil, errIdle
	}
	return st, nil
}

// say prints a command's result: the plain word by default, the JSON document
// when asked. Commands print the value they were asked for and nothing else —
// only status and queue label their fields, because only they carry more than
// one thing.
func (c *cli) say(plain string, document any) error {
	if !c.json {
		fmt.Fprintln(c.out, plain)
		return nil
	}

	// The daemon answers a command with an empty body, so there is no answer of
	// its own to pass through and the command describes itself instead.
	encoded, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode answer: %w", err)
	}
	fmt.Fprintln(c.out, string(encoded))
	return nil
}

// takeJSONFlag pulls --json out of the arguments wherever it was written, so
// that it reads as a modifier rather than as a positional argument.
func takeJSONFlag(args []string) ([]string, bool) {
	kept := make([]string, 0, len(args))
	found := false
	for _, arg := range args {
		if arg == jsonFlag {
			found = true
			continue
		}
		kept = append(kept, arg)
	}
	return kept, found
}

func expectNoArgs(name string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("spindle %s takes no arguments, got %q", name, args[0])
	}
	return nil
}

// parseVolume reads a percentage. Out of range is refused rather than clamped:
// someone who typed 120 meant something, and quietly playing at 100 hides it.
func parseVolume(arg string) (int, error) {
	pct, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("cannot read %q as a volume: want a whole number from 0 to 100", arg)
	}
	if pct < 0 || pct > 100 {
		return 0, fmt.Errorf("volume %d is outside 0 to 100", pct)
	}
	return pct, nil
}

// parseSeek reads a seek argument: "90" is a position, "+30" and "-15" are
// offsets from wherever the playhead is.
func parseSeek(arg string) (time.Duration, bool, error) {
	seconds, err := strconv.Atoi(arg)
	if err != nil {
		return 0, false, fmt.Errorf("cannot read %q as a position: want seconds, +seconds or -seconds", arg)
	}

	// A sign is what makes it relative, so "+0" and "-0" are honest no-ops
	// while "0" rewinds to the start.
	relative := arg[0] == '+' || arg[0] == '-'
	if !relative && seconds < 0 {
		return 0, false, fmt.Errorf("position %d is before the start of the track", seconds)
	}
	return time.Duration(seconds) * time.Second, relative, nil
}
