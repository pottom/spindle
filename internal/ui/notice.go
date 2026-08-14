package ui

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
)

const (
	warnGlyph  = "⚠"
	errorGlyph = "✕"
)

// notice is the one line between the player and the help bar. It clears itself
// when the cause resolves; it never needs acknowledging. SCREENS.md 4.6.
//
// Precedence matters: what just happened outranks what is standing. A skipped
// track and a throttle are both news about this moment; the explanations behind
// them are true either way and will still be there.
func (m Model) notice() (string, lipgloss.Style, bool) {
	if m.said != "" && time.Since(m.saidAt) < saidWindow {
		return m.said, m.styles.Detail, true
	}

	if name, ok := m.unplayableNotice(); ok {
		return fmt.Sprintf("%s Spotify would not play %s here — skipped", warnGlyph, name),
			m.styles.Warning, true
	}

	switch {
	// Above the rest of these because it is about all of them: while the daemon
	// is stuck, everything else on the screen is what was true a while ago, and
	// saying anything about it without saying that first would be saying it
	// about the past.
	case m.stalled():
		return warnGlyph + " The player has stopped responding — this is what it last said",
			m.styles.Warning, true

	case m.throttled():
		left := time.Until(m.rateLimitedUntil).Round(time.Second)
		return fmt.Sprintf("%s Rate limited — pausing for %s", warnGlyph, left),
			m.styles.Warning, true

	case m.noPremium:
		return errorGlyph + " Playback control requires Spotify Premium",
			m.styles.Error, true

	case m.err != nil:
		return errorGlyph + " " + m.err.Error(), m.styles.Error, true

	case m.ranOut():
		return warnGlyph + " The list has run out — nothing follows this track",
			m.styles.Warning, true
	}
	return "", lipgloss.Style{}, false
}

// hasNotice is what the layout needs: whether a line has to be reserved.
func (m Model) hasNotice() bool {
	_, _, ok := m.notice()
	return ok
}

// stalled reports that the daemon is answering with what it last said rather
// than with what is true now.
//
// It has to be asked for rather than inferred: the daemon speaks only when
// something happens, so a quiet stretch is an ordinary record playing through,
// and reading silence as a fault once froze the picture for half a minute at a
// time — see the resync, which asks. A socket ping is not enough either, because
// the API answers that on a goroutine of its own while the session sits still.
// The daemon says so itself, in the header on the answer it served out of the
// cupboard.
func (m Model) stalled() bool { return m.ps != nil && m.ps.Stalled }

// ranOut reports that the device stopped because it had nothing left to play.
//
// Spotify's own clients follow a finished list with a radio station; ours is
// refused one, so playback simply stops at the top of the last track with an
// empty queue behind it. On screen that is a still progress bar and a silent
// room, which reads as a program that has crashed. It has not: it has finished.
func (m Model) ranOut() bool {
	if m.ps == nil || m.ps.Playing || m.ps.TrackID == "" || len(m.queue) > 0 {
		return false
	}
	// At the top of a track rather than partway through it: a listener who
	// paused in the middle of the last track knows perfectly well what they did.
	return m.elapsed() < ranOutSlack
}
