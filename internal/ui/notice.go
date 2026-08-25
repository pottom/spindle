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

	// Above everything it causes. When the network goes, the Web API fails too
	// and m.err fills with whatever the transport said; "dial tcp: no such
	// host" is the same news told worse. Below the two above it because those
	// are about this very moment, and this is a state that lasts.
	case m.outOfTouch() >= outOfTouchAfter:
		if m.soundAtStake() {
			return fmt.Sprintf("%s Out of touch with Spotify for %s — playing what is here, and trying to get back",
				warnGlyph, howLong(m.outOfTouch())), m.styles.Warning, true
		}
		return fmt.Sprintf("%s Out of Spotify Connect's reach for %s — the device is trying to get back",
			warnGlyph, howLong(m.outOfTouch())), m.styles.Warning, true

	case m.noPremium:
		return errorGlyph + " Playback control requires Spotify Premium",
			m.styles.Error, true

	case m.err != nil:
		return errorGlyph + " " + m.err.Error(), m.styles.Error, true

	case m.ranOut():
		return warnGlyph + " The list has run out — nothing follows this track",
			m.styles.Warning, true

	// Last, because it is the least like news: a device that has gone deaf will
	// still be deaf in a second, and everything above is about this moment.
	//
	// It still plays and still answers, so nothing else on the screen looks
	// wrong — it is simply out of Spotify Connect's reach, where no phone can
	// drive it, until it is started again. Nothing here can mend it, so the line
	// says the one thing that can. See player.State.Deaf.
	case m.deviceDeaf():
		return warnGlyph + " The device has lost touch with Spotify — restart it from the settings screen",
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

// deviceDeaf reports that the device playing has lost an input it cannot get
// back. Only our own daemon says so; every other backend leaves it empty.
func (m Model) deviceDeaf() bool { return m.ps != nil && len(m.ps.Deaf) > 0 }

// outOfTouchAfter is how long a connection has to have been gone before it is
// worth a line. Measured from the daemon's own log: an ordinary reconnection
// lands in under two seconds and happens dozens of times a day, and none of
// those is news. What this is for is the outage that lasts.
const outOfTouchAfter = 20 * time.Second

// outOfTouch is how long the device has been without one of its connections to
// Spotify, counting the one that has been gone longest. Zero when it has them
// all — or when the backend is not our own daemon, which is the only one that
// can say.
func (m Model) outOfTouch() time.Duration {
	if m.ps == nil {
		return 0
	}
	var worst time.Duration
	for _, since := range m.ps.OutOfTouch {
		worst = max(worst, since)
	}
	return worst
}

// soundAtStake reports that what is missing is the connection the audio itself
// comes down. The device plays out what it has already fetched and can start
// nothing new; the other one only carries the remote control.
func (m Model) soundAtStake() bool {
	return m.ps != nil && m.ps.OutOfTouch["accesspoint"] >= outOfTouchAfter
}

// howLong spells a stretch of time the way somebody would say it: seconds up to
// a minute, then whole minutes, then hours and minutes. Never a decimal — the
// number is here to say roughly how long this has been going on.
func howLong(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case int(d.Minutes())%60 == 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
