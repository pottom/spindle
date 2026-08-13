package main

import (
	"fmt"
	"strings"
	"time"
)

// The panel. It is redrawn in place rather than scrolled: a page that scrolls
// is a page you read, and this one is glanced at with one hand on the space bar
// while the ears do the work. Nothing on it moves except the line being sung and
// the count.

const (
	home  = "\x1b[H"  // cursor to the top left
	clear = "\x1b[K"  // clear to the end of the line
	rest  = "\x1b[J"  // clear everything below
	dim   = "\x1b[2m" //nolint:unused // kept beside its pair
	off   = "\x1b[0m"
	bold  = "\x1b[1m"
)

// draw paints the panel.
func (s *session) draw() { fmt.Print(s.panel()) }

// panel is what draw prints — built rather than written out, so that what the
// tapper is looking at can be read back in a test.
func (s *session) panel() string {
	w := s.plan[s.track]
	p := w.passes[s.pass]

	var b strings.Builder
	line := func(format string, args ...any) {
		fmt.Fprintf(&b, format+clear+"\r\n", args...)
	}

	b.WriteString(home)
	line("%stap%s   record %d of %d   pass %d of %d", bold, off, s.track+1, len(s.plan), s.pass+1, len(w.passes))
	line("")
	line("  %s%s — %s%s", bold, w.artist, w.title, off)
	line("  %s", w.why)
	line("")

	switch {
	case s.finished:
		line("  %sAll of it is done. q writes the last of it and leaves.%s", bold, off)
	case s.between:
		line("  %s%s%s", bold, s.said, off)
		line("")
		line("  any key starts the next one")
	default:
		line("  %s%s%s        %d of %d lines", bold, p.asks, off, done(p.kind, s.taps), p.goal)
		line("")
		if !s.playing {
			line("  … nothing is playing. p starts it.")
		} else if l := s.ly.words(s.line); len(l) > 0 {
			line("  ▸ %s", strings.Join(l, " "))
		} else {
			line("  ▸ …")
		}
		line("")
		line("  %s", s.said)
	}

	line("")
	line("  %s", p.keys)
	line("  backspace undo   r restart   f/b move ±15s   n skip on   p play/pause   q quit")
	b.WriteString(rest)
	return b.String()
}

// say puts one line at the foot of the panel — what was just taken, or what
// went wrong. It is not a log: the next thing that happens replaces it.
func (s *session) say(format string, args ...any) {
	s.said = fmt.Sprintf(format, args...)
}

func secs(d time.Duration) string { return fmt.Sprintf("%.2fs", d.Seconds()) }
