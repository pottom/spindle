package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// The transport, as five things the program does rather than five keys it
// answers.
//
// A key pressed and a glyph clicked are the same act, and the only way to keep
// them the same act is for there to be one of it. These used to be written out
// inside the key switch, where a second caller would have had to copy them —
// and a copy of "pause, but pin the clock first" is a copy that will be right
// on one of the two paths.

// togglePlay starts or stops what is loaded.
func (m *Model) togglePlay() tea.Cmd {
	// Pin the position before the flag flips. Pausing stops the clock from
	// being carried forward, so the anchor has to already hold everything that
	// had accumulated, or the playhead drops back to wherever the last poll
	// left it.
	m.setProgress(m.elapsed())
	m.ps.Playing = !m.ps.Playing
	m.hold()

	p, play := m.player, m.ps.Playing
	return tea.Batch(m.spinDevice(), controlCmd("toggle playback", func(ctx context.Context) error {
		if play {
			return p.Play(ctx)
		}
		return p.Pause(ctx)
	}))
}

// skipNext goes to the next track, and puts it on screen without waiting to be
// told: the queue already says what is coming, so show it now instead of half a
// second from now. The confirming fetch demotes to a check.
func (m *Model) skipNext() tea.Cmd {
	cmd := m.skip("skip to next track", m.player.Next)
	if next := m.takeFromQueue(); next != nil {
		m.showTrack(next)
		return tea.Batch(cmd, m.syncCover())
	}
	return cmd
}

// skipPrev goes back.
func (m *Model) skipPrev() tea.Cmd {
	return m.skip("skip to previous track", m.player.Previous)
}

// toggleShuffle and turnRepeat set what they say, on screen first and on the
// device after: both are answered by a poll a second later, and a control that
// waited for it would feel like a control that had missed.
func (m *Model) toggleShuffle() tea.Cmd {
	m.ps.Shuffle = !m.ps.Shuffle
	m.hold()

	p, on := m.player, m.ps.Shuffle
	return controlCmd("set shuffle", func(ctx context.Context) error {
		return p.SetShuffle(ctx, on)
	})
}

func (m *Model) turnRepeat() tea.Cmd {
	m.ps.Repeat = nextRepeat(m.ps.Repeat)
	m.hold()

	p, mode := m.player, m.ps.Repeat
	return controlCmd("set repeat", func(ctx context.Context) error {
		return p.SetRepeat(ctx, mode)
	})
}
