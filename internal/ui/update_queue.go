package ui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// queueKey handles the queue tab.
func (m *Model) queueKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(k, m.keys.Down), key.Matches(k, m.keys.Up):
		delta := 1
		if key.Matches(k, m.keys.Up) {
			delta = -1
		}
		m.queuePane.cursor.move(delta, len(m.queueRows()))
		return m.previewCover(), true

	case key.Matches(k, m.keys.Enter):
		// The top row is already playing; restarting it is not what "play"
		// means to anyone pressing enter on the track they are listening to.
		at := m.queueIndex()
		if at < 0 {
			return nil, true
		}
		return m.playRow(at), true

	case key.Matches(k, m.keys.Drop):
		return m.dropQueued(), true

	case key.Matches(k, m.keys.MoveUp):
		return m.moveQueued(-1), true

	case key.Matches(k, m.keys.MoveDn):
		return m.moveQueued(1), true

	case key.Matches(k, m.keys.Back):
		return m.switchTab(tabPlayer), true
	}
	return nil, false
}

// dropQueued takes the row under the cursor out of the list. Every row can go,
// including the one playing: not wanting to hear the rest of a track is the
// commonest reason to reach for the key at all, and skipping it is what taking
// it out of the list means while it is sounding.
func (m *Model) dropQueued() tea.Cmd {
	at := m.queueIndex()
	if at < 0 {
		p := m.player
		cmd := m.skip("skip to next track", p.Next)
		if next := m.takeFromQueue(); next != nil {
			m.showTrack(next)
			return tea.Batch(cmd, m.syncCover())
		}
		return cmd
	}

	editor, ok := m.player.(player.QueueEditor)
	if !ok {
		return nil
	}

	id := deviceID(m.queue[at])
	rest := make([]player.Track, 0, len(m.queue)-1)
	rest = append(rest, m.queue[:at]...)
	rest = append(rest, m.queue[at+1:]...)
	m.queue = rest
	m.clampQueueCursor()

	return controlCmd("remove from queue", func(ctx context.Context) error {
		return editor.Drop(ctx, id)
	})
}

// moveQueued shifts the track under the cursor by one place. It refuses to move
// a hand-queued track past the context tracks: those keep their own order, and
// swapping across the boundary would look like a move that did not take.
func (m *Model) moveQueued(delta int) tea.Cmd {
	at := m.queueIndex()
	if at < 0 || !m.queue[at].Queued {
		return nil
	}

	to := at + delta
	if to < 0 || to >= len(m.queue) || !m.queue[to].Queued {
		return nil
	}

	next := make([]player.Track, len(m.queue))
	copy(next, m.queue)
	next[at], next[to] = next[to], next[at]
	m.queuePane.cursor.cursor += delta
	return m.commitQueue(next)
}

// commitQueue shows the new order at once and sends it. The list on screen is
// the one the user just edited, so waiting for Spotify to agree before drawing
// it would make every edit feel like it had missed.
func (m *Model) commitQueue(next []player.Track) tea.Cmd {
	editor, ok := m.player.(player.QueueEditor)
	if !ok {
		return nil
	}

	ids := make([]string, 0, len(next))
	for _, t := range next {
		if !t.Queued {
			// Only the leading run is the queue; the device keeps no more.
			break
		}
		ids = append(ids, deviceID(t))
	}

	m.queue = next
	m.clampQueueCursor()
	return controlCmd("edit queue", func(ctx context.Context) error {
		return editor.SetQueue(ctx, ids)
	})
}

// playRow brings a track forward: it starts playing, and every other track
// keeps its place in the list and moves up one.
//
// Seeking to it would have been the other reading of "play", and is what the
// official client does, but it throws away every track in between — and those
// are in the list precisely because they are meant to be heard.
func (m *Model) playRow(at int) tea.Cmd {
	target := m.queue[at]
	id, p := deviceID(target), m.player
	cmd := m.skip("play from queue", func(ctx context.Context) error {
		return p.PlayFrom(ctx, id)
	})

	// Show the result now rather than a round trip from now. The marks will be
	// redrawn by the refresh that follows the track change; the order will not
	// change, because this is the order the device was asked for.
	rest := make([]player.Track, 0, len(m.queue)-1)
	rest = append(rest, m.queue[:at]...)
	rest = append(rest, m.queue[at+1:]...)
	m.queue = rest
	m.showTrack(&target)
	m.queuePane.cursor.reset()

	return tea.Batch(cmd, m.syncCover())
}

// deviceID is the id to speak to the playback device with, falling back to the
// Web API's when there is no device of ours to ask.
func deviceID(t player.Track) string {
	if t.DeviceID != "" {
		return t.DeviceID
	}
	return t.ID
}

// enqueue appends a track and asks for the queue again, so the queue tab shows
// it without waiting for the next track change.
func (m *Model) enqueue(trackID string) tea.Cmd {
	p := m.player
	return tea.Sequence(
		controlCmd("add to queue", func(ctx context.Context) error {
			return p.AddToQueue(ctx, trackID)
		}),
		fetchQueueCmd(p),
	)
}

// editable reports whether the queue can be reordered, which only the local
// device allows. Elsewhere the keys are still listed but do nothing, so the
// help has to stop offering them.
func (m Model) editable() bool {
	_, ok := m.player.(player.QueueEditor)
	return ok
}
