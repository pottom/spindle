package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// daemonQueue is what the daemon's /player/queue document holds: what is
// playing, and what is coming after it in order.
type daemonQueue struct {
	Current *daemonQueueTrack  `json:"current"`
	Tracks  []daemonQueueTrack `json:"tracks"`
}

type daemonQueueTrack struct {
	Name        string   `json:"name"`
	ArtistNames []string `json:"artist_names"`
	Duration    int64    `json:"duration"`
}

// queue prints the playing track and everything queued behind it, one track per
// line, numbered from the next one.
func (c *cli) queue(ctx context.Context, args []string) error {
	if err := expectNoArgs("queue", args); err != nil {
		return err
	}

	raw, err := c.remote.get(ctx, "/player/queue")
	if err != nil {
		return err
	}

	var q daemonQueue
	if err := json.Unmarshal(raw, &q); err != nil {
		return fmt.Errorf("decode daemon queue: %w", err)
	}

	if c.json {
		fmt.Fprintln(c.out, string(raw))
		if q.Current == nil {
			return errIdle
		}
		return nil
	}

	if q.Current == nil {
		return errIdle
	}

	// "now" rather than "0": the playing track is not the zeroth thing coming,
	// and numbering from the next one is how far ahead each track actually is.
	c.printQueueTrack("now", *q.Current)
	for i, track := range q.Tracks {
		c.printQueueTrack(strconv.Itoa(i+1), track)
	}
	return nil
}

func (c *cli) printQueueTrack(label string, track daemonQueueTrack) {
	fmt.Fprintf(c.out, "%-4s %s — %s  %s\n",
		label, track.Name, joinArtists(track.ArtistNames), clock(millis(track.Duration)))
}
