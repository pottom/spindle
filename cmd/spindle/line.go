package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// A status bar wants one line and no work.
//
// The default output is a field per line, which is right for a person reading
// it and wrong for everything else: a bar has to spawn jq or head every second
// to get a sentence out of it. --line prints that sentence, --format writes
// one, and --follow prints a fresh one whenever something actually changes,
// which costs nothing at all while nothing does.
const (
	lineFlag   = "--line"
	formatFlag = "--format"
	followFlag = "--follow"

	// defaultLine is what --line prints: enough to know what is on, short
	// enough for a bar that also has a clock and a battery in it.
	defaultLine = "{icon} {title} — {artist}"

	// followRetry is how long to wait before dialling the daemon again. It is
	// on loopback: a failure means the daemon is restarting, not that the
	// network is unhappy.
	followRetry = time.Second
)

// statusFormat is how one status is to be printed, and how often.
type statusFormat struct {
	// format is the template, empty for the field-per-line output.
	format string

	// follow keeps printing, one line per change, until the shell stops it.
	follow bool
}

// takeLineFlags pulls the formatting flags out of the arguments, wherever they
// were written, so they read as modifiers rather than as positional arguments.
func takeLineFlags(args []string) ([]string, statusFormat, error) {
	kept := make([]string, 0, len(args))
	var out statusFormat

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == lineFlag:
			out.format = defaultLine
		case args[i] == followFlag:
			out.follow = true
		case args[i] == formatFlag:
			if i+1 >= len(args) {
				return nil, out, fmt.Errorf("%s wants a template, like %s %q", formatFlag, formatFlag, defaultLine)
			}
			out.format = args[i+1]
			i++
		case strings.HasPrefix(args[i], formatFlag+"="):
			out.format = strings.TrimPrefix(args[i], formatFlag+"=")
		default:
			kept = append(kept, args[i])
		}
	}
	return kept, out, nil
}

// render fills a template from a status. The fields are named in braces because
// a bar's configuration file is already full of dollars and percent signs.
//
// An unknown field is left as it was written rather than being reported: a
// status bar is not the place to discover a typo, and a line that reads
// "{titel}" says where to look.
func render(format string, st *daemonStatus) string {
	fields := map[string]string{
		"icon":     "⏸",
		"state":    "paused",
		"title":    "",
		"artist":   "",
		"album":    "",
		"position": clock(0),
		"duration": clock(0),
		"volume":   fmt.Sprintf("%d", st.volumePercent()),
		"device":   st.DeviceName,
	}
	if !st.Paused {
		fields["icon"], fields["state"] = "▶", "playing"
	}
	if st.Track != nil {
		fields["title"] = st.Track.Name
		fields["artist"] = joinArtists(st.Track.ArtistNames)
		fields["album"] = st.Track.AlbumName
		fields["position"] = clock(millis(st.Track.Position))
		fields["duration"] = clock(millis(st.Track.Duration))
	}

	out := format
	for name, value := range fields {
		out = strings.ReplaceAll(out, "{"+name+"}", value)
	}
	return out
}

// follow prints a line whenever the daemon says something changed, and one at
// once so a bar has something to draw before the first change.
//
// The daemon's events say only that something moved; what moved comes from
// /status, which is one request against loopback. A dropped connection is
// expected — the daemon restarts, or was not up yet — so it is retried quietly
// rather than reported: a bar that printed an error every second would be
// worse than one that went blank.
func (c *cli) follow(ctx context.Context, shape statusFormat) error {
	c.printStatus(ctx, shape)

	for ctx.Err() == nil {
		if err := c.followOnce(ctx, shape); err != nil && ctx.Err() == nil {
			select {
			case <-ctx.Done():
			case <-time.After(followRetry):
			}
		}
	}
	return nil
}

func (c *cli) followOnce(ctx context.Context, shape statusFormat) error {
	conn, _, err := websocket.Dial(ctx, c.remote.events(), nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow() //nolint:errcheck // closing a dead socket says nothing

	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return err
		}
		c.printStatus(ctx, shape)
	}
}

// printStatus writes one line for whatever is playing now, and nothing at all
// when nothing is. Nothing rather than a word: a bar showing this output should
// go empty, which is what the one-shot command does too.
func (c *cli) printStatus(ctx context.Context, shape statusFormat) {
	raw, st, err := c.readStatus(ctx)
	if err != nil {
		return
	}
	switch {
	case c.json:
		fmt.Fprintln(c.out, string(raw))
	case st.idle():
		fmt.Fprintln(c.out)
	default:
		fmt.Fprintln(c.out, render(shape.format, st))
	}
}
