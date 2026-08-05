package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// daemonStatus is what the command line reads out of the daemon's /status
// document.
//
// It is declared here rather than shared with internal/player because the two
// want different things — the interface needs artwork, tempo and repeat modes,
// a shell wants a line of text — and a single struct would drag one's needs
// through the other.
type daemonStatus struct {
	DeviceName  string       `json:"device_name"`
	Stopped     bool         `json:"stopped"`
	Paused      bool         `json:"paused"`
	Volume      int          `json:"volume"`
	VolumeSteps int          `json:"volume_steps"`
	Track       *daemonTrack `json:"track"`
}

type daemonTrack struct {
	Name        string   `json:"name"`
	ArtistNames []string `json:"artist_names"`
	AlbumName   string   `json:"album_name"`
	Position    int64    `json:"position"`
	Duration    int64    `json:"duration"`
}

// idle reports whether the daemon has nothing loaded. A stopped daemon is a
// working daemon with no music in it, which is why it is not an error.
func (s *daemonStatus) idle() bool { return s.Stopped || s.Track == nil }

// steps is how finely the daemon counts its volume. It picks the number itself,
// and a daemon that reported none would otherwise make every level zero.
func (s *daemonStatus) steps() int {
	if s.VolumeSteps <= 0 {
		return 100
	}
	return s.VolumeSteps
}

// volumePercent rescales the daemon's volume onto the 0–100 everything else
// speaks.
func (s *daemonStatus) volumePercent() int {
	return min(max(s.Volume*100/s.steps(), 0), 100)
}

// status prints what is playing.
//
// It is one request and no waiting, so a prompt or a status bar can run it as
// often as it redraws.
func (c *cli) status(ctx context.Context, args []string) error {
	args, shape, err := takeLineFlags(args)
	if err != nil {
		return err
	}
	if err := expectNoArgs("status", args); err != nil {
		return err
	}

	// Following never ends and never fails: a bar wants a line when something
	// changes and silence otherwise, including while the daemon is restarting.
	if shape.follow {
		return c.follow(ctx, shape)
	}
	if shape.format != "" {
		raw, st, err := c.readStatus(ctx)
		if err != nil {
			return err
		}
		if c.json {
			fmt.Fprintln(c.out, string(raw))
		}
		if st.idle() {
			return errIdle
		}
		if !c.json {
			fmt.Fprintln(c.out, render(shape.format, st))
		}
		return nil
	}

	raw, st, err := c.readStatus(ctx)
	if err != nil {
		return err
	}

	if c.json {
		// The daemon's own document, verbatim: --json exists for jq, and jq
		// wants the fields this command chose not to print as much as the ones
		// it did. It is printed even when nothing is playing, because "stopped"
		// is an answer a script can branch on.
		fmt.Fprintln(c.out, string(raw))
		if st.idle() {
			return errIdle
		}
		return nil
	}

	if st.idle() {
		// Nothing playing prints nothing: a status bar showing this output
		// should go empty rather than say so, and the exit code carries the
		// explanation for anything that cares.
		return errIdle
	}

	state := "playing"
	if st.Paused {
		state = "paused"
	}
	fmt.Fprintf(c.out, "state:    %s\n", state)
	fmt.Fprintf(c.out, "title:    %s\n", st.Track.Name)
	fmt.Fprintf(c.out, "artist:   %s\n", joinArtists(st.Track.ArtistNames))
	fmt.Fprintf(c.out, "album:    %s\n", st.Track.AlbumName)
	fmt.Fprintf(c.out, "position: %s\n", clock(millis(st.Track.Position)))
	fmt.Fprintf(c.out, "duration: %s\n", clock(millis(st.Track.Duration)))
	fmt.Fprintf(c.out, "volume:   %d\n", st.volumePercent())
	fmt.Fprintf(c.out, "device:   %s\n", st.DeviceName)
	return nil
}

// readStatus fetches the status, handing back the raw document alongside the
// parsed one so --json can print what the daemon actually said.
func (c *cli) readStatus(ctx context.Context) ([]byte, *daemonStatus, error) {
	raw, err := c.remote.get(ctx, "/status")
	if err != nil {
		return nil, nil, err
	}

	var st daemonStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, nil, fmt.Errorf("decode daemon status: %w", err)
	}
	return raw, &st, nil
}
