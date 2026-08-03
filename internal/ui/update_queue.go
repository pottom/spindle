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

// dropQueued removes the track under the cursor. Only hand-queued tracks can go:
// the rest belong to the album or playlist that is playing, and dropping one
// there would mean rewriting the context itself.
func (m *Model) dropQueued() tea.Cmd {
	at := m.queueIndex()
	if at < 0 || !m.queue[at].Queued {
		return nil
	}

	next := make([]player.Track, 0, len(m.queue)-1)
	next = append(next, m.queue[:at]...)
	next = append(next, m.queue[at+1:]...)
	return m.commitQueue(next)
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

// playRow brings a track forward: it goes to the top of the queue, everything
// else slides down, and the device is told to advance into it.
//
// Nothing is discarded. Seeking to the track would have been the other reading
// of "play", and it is what the official client does, but it throws away every
// track between here and there — and those are in the list precisely because
// they are meant to be heard. A track that came from the album stays in the
// album as well, so it will come round again later; that is the price of not
// losing anything, and it is the cheaper one.
func (m *Model) playRow(at int) tea.Cmd {
	if !m.editable() {
		// Nothing else can rewrite a queue, so against another device this is
		// the only move left, skipped-over tracks and all.
		id, p := deviceID(m.queue[at]), m.player
		return tea.Batch(
			controlCmd("play from queue", func(ctx context.Context) error {
				return p.PlayFrom(ctx, id)
			}),
			m.awaitTrackChange(),
		)
	}

	reorder := m.commitQueue(queueWithFirst(m.queue, at))
	m.queuePane.cursor.reset()

	p := m.player
	skip := m.skip("play from queue", p.Next)
	if t := m.takeFromQueue(); t != nil {
		m.showTrack(t)
		skip = tea.Batch(skip, m.syncCover())
	}
	// The device has to be holding the new order before it is told to advance,
	// or it advances into the old one.
	return tea.Sequence(reorder, skip)
}

// queueWithFirst moves the track at index at to the front, marking it as
// hand-queued so that it survives being sent back as the queue.
func queueWithFirst(queue []player.Track, at int) []player.Track {
	first := queue[at]
	first.Queued = true

	next := make([]player.Track, 0, len(queue))
	next = append(next, first)
	for i, t := range queue {
		if i != at {
			next = append(next, t)
		}
	}
	return next
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
