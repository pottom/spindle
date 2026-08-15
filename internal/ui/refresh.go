package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Keeping what is on screen true while somebody edits it somewhere else.
//
// A list read once and left alone is a list that is wrong the moment anybody
// touches it from a phone. Nothing in the Web API tells us that has happened, so
// what is on screen is asked for again from time to time — and only what is on
// screen, because every request is a request against somebody's quota.
//
// The queue is the exception in both directions. It comes from our own daemon
// over localhost rather than from Spotify, so asking costs nothing worth
// counting and it is asked far more often. See local_queue.go.

const (
	// staleAfter is how old a list may be before the screen it is on asks for it
	// again. Thirty seconds: long enough that reading a library is not a stream
	// of requests, short enough that a playlist edited on a phone catches up
	// while you are still looking at the screen it is on.
	staleAfter = 30 * time.Second

	// queueStaleAfter is the same for the queue while it is the daemon's rather
	// than Spotify's. Two seconds is a localhost round trip and no quota at all.
	queueStaleAfter = 2 * time.Second
)

// queueEvery is how often the queue may be asked for.
//
// Two seconds while it comes from the daemon, and the same as everything else
// when it does not. The queue is the daemon's only while the daemon is the
// device that is playing: paused, stopped, or playing somewhere else, the same
// call goes to the Web API instead — and asked every two seconds that is thirty
// requests a minute against somebody's quota, which is a rate limit within the
// minute. Measured the hard way.
func (m Model) queueEvery() time.Duration {
	if local, ok := m.player.(interface{ QueueIsLocal() bool }); ok && local.QueueIsLocal() {
		return queueStaleAfter
	}
	return staleAfter
}

// refreshOnScreen asks again for whatever the screen is showing, once it is old
// enough. It is called from the tick, so the age is checked once a second and
// the request goes out on the first tick past the age.
func (m *Model) refreshOnScreen() tea.Cmd {
	if m.player == nil || m.devices.open || !fitsMinimum(m.width, m.height) {
		return nil
	}
	if m.throttled() {
		// Spotify has asked to be left alone. Asking anyway is how a short
		// silence becomes a long one — the same reason the state poll stops.
		return nil
	}

	// What is open is the screen, whichever tab it was opened from.
	if page := m.openMut(); page != nil {
		if !page.pages.stale(staleAfter) {
			return nil
		}
		page.pages.loading = true
		return fetchOpenCmd(m.player, page.kind, page.id, 0)
	}

	switch m.tab {
	case tabQueue:
		if time.Since(m.queueAt) < m.queueEvery() {
			return nil
		}
		m.queueAt = time.Now()
		return fetchQueueCmd(m.player)

	case tabLibrary:
		paging := m.library.paging()
		if !paging.stale(staleAfter) {
			return nil
		}
		paging.loading = true
		return fetchLibraryCmd(m.player, m.library.kind, 0)
	}
	return nil
}
