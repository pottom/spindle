// Command spindle-tap asks somebody with ears when a word is actually sung.
//
//	go run ./cmd/spindle-tap
//
// Nothing else to type. It puts each record on itself, says what to listen for,
// counts what it has, and moves on to the next one when it has enough. The hand
// stays on four keys and the head stays on the music.
//
// # Why anybody has to listen at all
//
// Spotify times a lyric by the line and nobody times it by the word — measured
// on 26 records and 1327 lines, written up in FINDINGS.md — and a line's window
// turns out to say almost nothing about the words inside it: spread a line
// evenly across it and the last syllable lands a second late at the median. The
// missing fact is where the singing stops, and no source has it. So it is
// listened for.
//
// # The passes, and why the first one is not wasted
//
// A hand is late. Not by an unknown amount, though, and that is the trick of
// this tool: the starts pass taps the beginning of each line, and those are
// already known to the millisecond. The difference between a tap and the stamp
// it belongs to is the tapper's own lag, measured on the tapper, on that record,
// at that tempo. The ends pass then taps where the singing stops, and the lag
// comes off it.
//
// The words pass — every word, in order — is only worth running on a slow
// record. The rap here runs at six syllables a second, which is a test of the
// hand rather than a measurement.
//
// # The playhead
//
// Asking the daemon at the moment of a press would put a round trip inside the
// measurement. Instead the position is anchored: the status is polled in the
// background and a tap is read off the local clock since. Measured, that clock
// holds to within three milliseconds over three seconds. A seek breaks the
// anchor rather than smearing it, and says so.
//
// Run it in a window of its own. Space is play/pause in spindle's own interface
// and this takes the terminal raw, so whichever window has the focus gets the
// key and the other never sees it.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/x/term"
)

// session is where the whole thing has got to.
type session struct {
	plan  []work
	track int
	pass  int

	ly    lyrics
	tempo float64
	taps  []tap

	a       anchor
	line    int
	word    int
	playing bool
	broke   bool

	between  bool // a pass is done and the next one waits for a key
	finished bool
	said     string

	// apply is how a goroutine hands its result back: it sends what it wants
	// done and the loop does it. Nothing outside the loop touches the session,
	// so there is one writer and no lock to forget.
	apply chan func(*session)
}

func main() {
	if _, err := get[status](base + "/status"); err != nil {
		fmt.Fprintf(os.Stderr, "spindle-tap: no daemon at %s: %v\n", base, err)
		fmt.Fprintln(os.Stderr, "start one with: spindle daemon start")
		os.Exit(1)
	}

	fd := os.Stdin.Fd()
	old, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spindle-tap: this needs a terminal: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(fd, old) //nolint:errcheck // the process is ending

	s := &session{plan: plan, line: -1, apply: make(chan func(*session), 8)}
	s.begin()
	s.loop()

	fmt.Print("\x1b[2J" + home)
	fmt.Print(s.farewell())
}

// The two things beginning a pass does to the world outside this process, held
// here so that a test of the session's own bookkeeping does not put a record on
// somebody's speakers.
var (
	fetchWords = wordsFor
	putOn      = play
)

// begin starts whatever pass is current: the record from the top, the words
// fetched, the count back to nothing.
func (s *session) begin() {
	w := s.plan[s.track]
	s.taps, s.line, s.word, s.broke = nil, -1, 0, false
	s.a = anchor{}

	s.ly = lyrics{}
	s.say("putting it on…")

	// Off the loop, both of them: fetching the words is a request to Spotify by
	// way of the daemon, and putting a record on can take a moment. Neither is
	// worth a frozen panel.
	// Everything the goroutine needs is taken here, on this one thread: it
	// carries values rather than reaching back into anything that could be
	// changed underneath it.
	uri := "spotify:track:" + w.id
	apply, words, put, from := s.apply, fetchWords, putOn, w.from
	go func() {
		ly, err := words(uri)
		note := ""
		switch {
		case err != nil:
			note = fmt.Sprintf("no words came back for this one: %v — n skips it", err)
		case !ly.Synced || len(ly.Lines) == 0:
			note = "this one's words carry no timings — n skips it"
		}
		if err := put(uri, from); err != nil {
			note = fmt.Sprintf("the daemon would not put it on: %v — is it running?", err)
		}
		hand(apply, func(s *session) {
			s.ly = ly
			s.say("%s", note)
		})
	}()
}

// hand gives the loop something to do, and drops it rather than blocking if the
// loop has gone.
func hand(apply chan func(*session), do func(*session)) {
	select {
	case apply <- do:
	default:
	}
}

// loop reads the keyboard, keeps the anchor fresh and repaints, until q.
func (s *session) loop() {
	keys := make(chan byte, 8)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(keys)
				return
			}
			keys <- buf[0]
		}
	}()

	// The daemon is asked from a goroutine of its own, never from here. It can
	// take seconds to answer or never answer at all, and the loop that reads the
	// keyboard cannot be the one waiting: a frozen panel with the music stopped
	// is the moment a key matters most.
	news := make(chan reading, 4)
	go func() {
		var a anchor
		for {
			time.Sleep(pollEvery)
			next, jumped, tempo := a.refresh()
			a = next
			select {
			case news <- reading{a: a, jumped: jumped, tempo: tempo}:
			default: // the panel is behind; the next reading will do
			}
		}
	}()

	paint := time.NewTicker(100 * time.Millisecond)
	defer paint.Stop()

	s.draw()
	for {
		select {
		case k, open := <-keys:
			if !open || k == 'q' || k == 3 { // 3 is ctrl+c
				s.keep()
				return
			}
			if s.key(k) {
				return
			}
			s.draw()

		case do := <-s.apply:
			do(s)
			s.draw()

		case r := <-news:
			s.a = r.a
			if r.tempo > 0 {
				s.tempo = r.tempo
			}
			if r.jumped {
				s.broke = true
				s.say("the playhead jumped — r starts this record again")
			}
			s.follow()
			s.draw()

		case <-paint.C:
			s.follow()
			s.draw()
		}
	}
}

// reading is one look at the daemon, handed over from the goroutine that took it.
type reading struct {
	a      anchor
	jumped bool
	tempo  float64
}

// follow keeps the line on the panel in step with the record.
func (s *session) follow() {
	at, ok := s.a.now()
	if !ok {
		s.playing = false
		return
	}
	if !s.playing {
		s.playing = true
		s.line = -1
	}
	if now := s.ly.lineAt(at); now != s.line {
		s.line, s.word = now, 0
	}
}

// keep writes the pass down and says what it measured, which is the thing that
// makes the next pass worth doing.
func (s *session) keep() {
	if len(s.taps) == 0 {
		return
	}
	w := s.plan[s.track]
	p := w.passes[s.pass]
	path, err := save(w, p.kind, s.tempo, s.ly, s.taps)
	if err != nil {
		s.say("could not write it down: %v", err)
		return
	}

	switch p.kind {
	case passStarts:
		if med, lo, hi, n := lag(s.ly, s.taps); n >= 4 {
			s.say("done. your hand is %+d ms late (quartiles %+d..%+d, %d lines) — saved", med, lo, hi, n)
			return
		}
	case passEnds:
		if med, lo, hi, n := share(s.ly, s.taps); n >= 4 {
			s.say("done. the singing fills %.0f%% of the bar (quartiles %.0f–%.0f%%, %d lines) — saved", med*100, lo*100, hi*100, n)
			return
		}
	}
	s.say("done, %d taps saved to %s", len(s.taps), path)
}

// next moves to the pass after this one, and to the next record after that.
func (s *session) next() bool {
	s.pass++
	if s.pass >= len(s.plan[s.track].passes) {
		s.pass = 0
		s.track++
	}
	if s.track >= len(s.plan) {
		s.finished = true
		_ = pause()
		return true
	}
	s.begin()
	return false
}

// farewell is what is left on the terminal afterwards: where the taps went and
// what is still owed.
func (s *session) farewell() string {
	dir := "~/.config/spindle/spike/taps"
	if s.finished {
		return fmt.Sprintf("all seven records done. The taps are in %s\n", dir)
	}
	left := 0
	for i := s.track; i < len(s.plan); i++ {
		left += len(s.plan[i].passes)
	}
	return fmt.Sprintf("stopped at %s — %s. %d passes left, and what was taken is in %s\n",
		s.plan[s.track].artist, s.plan[s.track].title, left-s.pass, dir)
}
