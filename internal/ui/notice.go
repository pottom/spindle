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
// Precedence matters: a throttle is transient and tells you why nothing is
// moving, so it outranks the standing explanations behind it.
func (m Model) notice() (string, lipgloss.Style, bool) {
	switch {
	case m.throttled():
		left := time.Until(m.rateLimitedUntil).Round(time.Second)
		return fmt.Sprintf("%s Rate limited — pausing for %s", warnGlyph, left),
			m.styles.Warning, true

	case m.noPremium:
		return errorGlyph + " Playback control requires Spotify Premium",
			m.styles.Error, true

	case m.err != nil:
		return errorGlyph + " " + m.err.Error(), m.styles.Error, true
	}
	return "", lipgloss.Style{}, false
}

// hasNotice is what the layout needs: whether a line has to be reserved.
func (m Model) hasNotice() bool {
	_, _, ok := m.notice()
	return ok
}
