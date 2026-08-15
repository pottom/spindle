package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// queueKey handles the queue tab.
func (m *Model) queueKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if m.listKey(k, &m.queuePane.cursor, len(m.queueRows()), true) {
		return m.previewCover(), true
	}

	switch {
	case m.pressed(k, m.keys.Enter):
		// The top row is already playing; restarting it is not what "play"
		// means to anyone pressing enter on the track they are listening to.
		at := m.queueIndex()
		if at < 0 {
			return nil, true
		}
		return m.playRow(at), true

	case m.pressed(k, m.keys.Actions):
		m.openActions()
		return nil, true

	case m.pressed(k, m.keys.Drop):
		return m.dropQueued(), true

	case m.pressed(k, m.keys.MoveUp):
		return m.moveQueued(-1), true

	case m.pressed(k, m.keys.MoveDn):
		return m.moveQueued(1), true

	case m.pressed(k, m.keys.Back):
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

	id := m.queue[at].ID
	rest := make([]player.Track, 0, len(m.queue)-1)
	rest = append(rest, m.queue[:at]...)
	rest = append(rest, m.queue[at+1:]...)
	m.queue = rest
	m.clampQueueCursor()

	return controlCmd("remove from queue", func(ctx context.Context) error {
		return editor.Drop(ctx, id)
	})
}

// pendingOrder is a run of move keys waiting to go out. The list on screen is
// already the one being asked for; only the request waits.
type pendingOrder struct {
	// seq counts the presses, so only the last one's timer sends anything.
	seq int

	// live is whether there is an edit to send, deep how far down the list the
	// run reached, and lift whether any of it has to be lifted out of the
	// context rather than merely reordered inside the queue.
	live bool
	deep int
	lift bool
}

// moveQueued shifts the track under the cursor by one place. Any row can move,
// not only the hand-queued ones: an album or a playlist has one order of its
// own, and lifting a track out of it is the only way to hear it in another.
//
// The list moves at once; the request waits for the run of presses to end. A
// reorder is not something the device can be given twice a second — each one
// rewrites what is coming — and the second press would be describing a list it
// has not agreed to yet.
func (m *Model) moveQueued(delta int) tea.Cmd {
	at := m.queueIndex()
	if at < 0 {
		return nil
	}

	to := at + delta
	if to < 0 || to >= len(m.queue) {
		return nil
	}

	next := make([]player.Track, len(m.queue))
	copy(next, m.queue)
	next[at], next[to] = next[to], next[at]
	m.queue = next
	m.queuePane.cursor.cursor += delta

	m.order.seq++
	m.order.deep = max(m.order.deep, max(at, to))
	// A run that touches the context anywhere has to go out as a lift: the
	// cheaper edit cannot express it.
	m.order.lift = m.order.lift || !next[at].Queued || !next[to].Queued
	m.order.live = true
	return orderSettleCmd(m.order.seq)
}

// sendOrder dispatches the run of moves that has come to rest.
func (m *Model) sendOrder() tea.Cmd {
	order := m.order
	m.order = pendingOrder{seq: order.seq}
	if !order.live || len(m.queue) == 0 {
		return nil
	}

	if !order.lift {
		// Every track that moved was already the queue's to order, which is a
		// cheaper edit and leaves the context where it stands.
		return m.commitQueue(m.queue)
	}
	return m.commitOrder(m.queue, min(order.deep, len(m.queue)-1))
}

// commitOrder shows the new order at once and sends the run that has to travel
// with it: every track from the front of the list down to the deepest edit.
// Anything left out would be pushed behind the run rather than staying put.
func (m *Model) commitOrder(next []player.Track, deepest int) tea.Cmd {
	editor, ok := m.player.(player.QueueEditor)
	if !ok {
		return nil
	}

	ids := make([]string, 0, deepest+1)
	for i := range deepest + 1 {
		ids = append(ids, next[i].ID)
		// They are lifted out of the context to be reordered, and the list has
		// to say so: they can be moved freely from here on.
		next[i].Queued = true
	}

	m.queue = next
	m.clampQueueCursor()
	return controlCmd("reorder queue", func(ctx context.Context) error {
		return editor.Reorder(ctx, ids)
	})
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
		ids = append(ids, t.ID)
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
	id, p := target.ID, m.player
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

// enqueueMost is how much of a playlist is taken when the whole of it is added
// to the queue. Some playlists run to thousands of tracks; a queue that long is
// not a queue, and the tail of it would be replaced long before it was reached.
const enqueueMost = 200

// enqueueList puts a whole playlist or album at the back of the queue.
//
// The tracks are fetched first and then sent as one order where the device
// takes one: appending them one at a time is a request per track, which is how
// the rate limiter is met and how half a playlist ends up queued. Only where
// the queue cannot be rewritten — anything that is not our own daemon — does it
// fall back to appending, and there the cap is what keeps it honest.
func (m *Model) enqueueList(kind openKind, id, name string) tea.Cmd {
	p := m.player
	queued := m.handQueued()

	m.said, m.saidAt = "Adding "+name+" to the queue", time.Now()
	return tea.Sequence(queueListCmd(p, kind, id, queued), fetchQueueCmd(p))
}

// queueListCmd is the request itself: read the list, then put it at the back of
// the queue.
func queueListCmd(p player.Player, kind openKind, id string, queued []string) tea.Cmd {
	return controlWithin("queue list", bulkTimeout, func(ctx context.Context) error {
		ids, err := listTrackIDs(ctx, p, kind, id)
		if err != nil || len(ids) == 0 {
			return err
		}

		if editor, ok := p.(player.QueueEditor); ok {
			return editor.SetQueue(ctx, append(queued, ids...))
		}
		for _, track := range ids {
			if err := p.AddToQueue(ctx, track); err != nil {
				return err
			}
		}
		return nil
	})
}

// handQueued is what has been queued by hand and is still waiting, which the
// device holds as one run at the front of the list. Rewriting the queue means
// naming it in full, so anything already put there has to be carried across.
func (m Model) handQueued() []string {
	ids := make([]string, 0, len(m.queue))
	for _, t := range m.queue {
		if !t.Queued {
			break
		}
		ids = append(ids, t.ID)
	}
	return ids
}

// editable reports whether the queue can be reordered, which only the local
// device allows. Elsewhere the keys are still listed but do nothing, so the
// help has to stop offering them.
func (m Model) editable() bool {
	_, ok := m.player.(player.QueueEditor)
	return ok
}
