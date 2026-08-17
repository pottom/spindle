package ui

import (
	"context"
	"errors"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// Liking a track, and unliking it.
//
// The column of hearts has been drawn on every list of tracks since the library
// was built, and until now nothing could fill one in: Spotify refuses the
// library writes to an application registered after its 2024 clampdown, and
// refuses the question with them. It is the registration that decides — see
// docs/SPOTIFY-API.md — so this is offered where the backend can do it and
// stops being offered the moment Spotify says it cannot.
//
// The heart changes before the request goes out. Everything else on this screen
// answers a key at once and mends itself if the server disagrees (see
// DESIGN.md 4.2), and a heart that waits half a second for Spotify would be the
// one thing on the screen that feels remote.

// savedTook is the answer to a like or an unlike.
type savedTook struct {
	track string
	saved bool
	err   error
}

// canSave reports whether the like key is worth offering: a backend that can
// save at all, and a registration that has not already refused.
func (m Model) canSave() bool {
	if m.noSaving || !m.allows.Has(player.Collecting) {
		return false
	}
	_, ok := m.player.(player.Collector)
	return ok
}

// toggleSaved likes what the cursor is on, or takes it back out.
//
// The list's own idea of what is saved is changed here rather than waiting for
// the answer, and put back where the answer says otherwise.
func (m *Model) toggleSaved() tea.Cmd {
	if !m.canSave() {
		return nil
	}

	t := m.cursorTrack()
	if t == nil || t.ID == "" {
		return nil
	}

	want := !m.library.saved(t.ID)
	m.markSaved(*t, want)

	track := *t
	collector := m.player.(player.Collector)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		var err error
		if want {
			err = collector.Save(ctx, track.ID)
		} else {
			err = collector.Unsave(ctx, track.ID)
		}
		return savedTook{track: track.ID, saved: want, err: err}
	}
}

// tookSaved reads the answer. A refusal of the application is a fact about the
// whole run: the key goes away rather than failing under the reader's hand every
// time it is pressed.
func (m *Model) tookSaved(message savedTook) tea.Cmd {
	if message.err == nil {
		m.said, m.saidAt = savedSaid(message.saved), time.Now()
		return nil
	}

	m.markSaved(player.Track{ID: message.track}, !message.saved)

	if errors.Is(message.err, player.ErrNotPermitted) {
		m.noSaving = true
		m.said, m.saidAt = "Spotify does not allow this application to change your library", time.Now()
		return nil
	}
	return func() tea.Msg { return msg.Error{Err: message.err} }
}

// savedSaid is what the notice says. Short: the heart has already said which way
// it went, and this is the line that says it reached Spotify.
func savedSaid(saved bool) string {
	if saved {
		return "Liked"
	}
	return "Removed from your liked songs"
}

// markSaved is what the screen believes about one track.
//
// Both halves of it: the set the hearts are drawn from, and the list of saved
// tracks itself — which is a screen somebody may be looking at, and the row at
// the head of the library, whose count would otherwise disagree with the hearts
// under it. A newly saved track goes to the front, which is where Spotify puts
// it: the saved tracks are in the order they were saved, newest first.
func (m *Model) markSaved(t player.Track, saved bool) {
	if m.library.likedIDs == nil {
		m.library.likedIDs = map[string]bool{}
	}
	if m.library.saved(t.ID) == saved {
		return
	}

	if saved {
		m.library.likedIDs[t.ID] = true
		if t.Title != "" {
			m.library.liked = append([]player.Track{t}, m.library.liked...)
		}
	} else {
		delete(m.library.likedIDs, t.ID)
		m.library.liked = slices.DeleteFunc(m.library.liked, func(held player.Track) bool {
			return held.ID == t.ID
		})
	}
	m.refreshLikedRow()

	// The saved tracks opened as a list are a copy of that same collection, and
	// a heart pressed inside it has to take the row out from under the cursor —
	// otherwise the one screen the act is most visible on is the one that does
	// not show it.
	if page := m.open(); page != nil && isLiked(page.id) {
		page.tracks = slices.Clone(m.library.liked)
		page.cursor.move(0, len(page.tracks))
	}
}
