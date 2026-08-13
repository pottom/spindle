package main

import "time"

// What each key does while a pass is running.

// key answers one press, and reports whether the session is over.
func (s *session) key(k byte) bool {
	switch k {
	case 'p':
		// Sent, not waited for. Every command to the daemon goes off on its own
		// so that a device which has stopped answering cannot take the keyboard
		// with it; what happened shows up in the panel a moment later either
		// way, because the position is polled regardless.
		apply := s.apply
		go func() {
			if err := playPause(); err != nil {
				hand(apply, func(s *session) { s.say("the daemon would not answer: %v", err) })
			}
		}()
		return false
	case 'f', 'b':
		// Forward and back by fifteen seconds. The interesting singing is not
		// always at the top of a record — a verse can run at two syllables a
		// second and the rap after it at eight — and tapping the wrong half of
		// a song answers a question nobody asked.
		by := 15 * time.Second
		if k == 'b' {
			by = -by
		}
		go func() {
			if err := seek(by); err != nil {
				hand(s.apply, func(s *session) { s.say("could not move it: %v", err) })
			}
		}()
		s.say("moved %+ds", int(by.Seconds()))
		return false
	case 'r':
		// The way out of a botched pass: the same record from the top, and
		// nothing kept from the attempt.
		s.begin()
		s.say("starting this one again")
		return false
	case 'n':
		s.keep()
		return s.next()
	}

	if s.between {
		// Anything at all moves on from the summary.
		s.between = false
		return s.next()
	}
	if s.finished {
		return false
	}

	at, ok := s.a.now()
	if !ok {
		s.say("nothing is playing — p starts it")
		return false
	}

	p := s.plan[s.track].passes[s.pass]
	switch k {
	case ' ':
		words := s.ly.words(s.line)
		w := ""
		if s.word < len(words) {
			w = words[s.word]
		}
		s.taps = append(s.taps, tap{at: at, line: s.line, word: s.word, words: w})
		s.word++
		s.say("%s  line %d %s", secs(at), s.line, w)
	case 'e':
		if p.kind != passEnds {
			return false
		}
		words := s.ly.words(s.line)
		last := ""
		if len(words) > 0 {
			last = words[len(words)-1]
		}
		s.taps = append(s.taps, tap{at: at, line: s.line, word: -1, words: last})
		s.say("%s  line %d ends", secs(at), s.line)
	case 127, 8: // backspace, however the terminal spells it
		// A measurement made by a hand needs this: without it one press at the
		// wrong moment costs the whole pass, and somebody who has just mistapped
		// is somebody about to mistap again.
		if len(s.taps) == 0 {
			return false
		}
		last := s.taps[len(s.taps)-1]
		s.taps = s.taps[:len(s.taps)-1]
		if last.word >= 0 && s.word > 0 {
			s.word--
		}
		s.say("took back %s", secs(last.at))
	case '\r', '\n':
		if p.kind == passWords {
			s.word++
		}
	}

	if done(p.kind, s.taps) >= p.goal {
		s.keep()
		s.between = true
		_ = pause()
	}
	return false
}
