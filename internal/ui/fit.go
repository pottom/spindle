package ui

import (
	"context"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// Putting what is coming into an order that follows what is playing.
//
// Not a new list and not a shorter one: every track that was in the queue is
// still in it, and every one of them is still played once. What changes is
// which comes first — the ones that go with the record sounding now, then the
// rest in the order they were already in. Nothing is dropped, so a queue
// somebody spent a minute building is never damaged by asking.
//
// What "goes with" means is Spotify's opinion rather than ours, and it is about
// artists rather than tracks. Two questions: who else sounds like this artist,
// and what would Spotify play after this track. Both name artists, and an
// artist is what a queued track can be matched against — matching tracks would
// answer almost never, because what Spotify suggests is by definition not what
// is already in the queue.
//
// Audio features would have been the obvious way to do this, and Spotify closed
// them to everybody in 2024. See docs/SPOTIFY-API.md.

const (
	// fitSame is what a track by the artist already playing scores, fitNear one
	// by an artist Spotify compares them to, and fitSuggested one by an artist
	// it would have played next.
	//
	// Three steps rather than two because they are three different strengths of
	// the same claim, and the difference is audible: more of the same artist is
	// not the same thing as more of the same corner of the catalogue.
	fitSame      = 3
	fitNear      = 2
	fitSuggested = 1

	// fitSuggestions is how many suggestions are asked for. Enough artists to
	// recognise a corner of the catalogue by; more would be a longer answer
	// saying the same thing.
	fitSuggestions = 50
)

// fitTook carries the opinion back: the artists worth being near.
//
// By name rather than by id. The names are the one thing every source here
// has — what the device reports as playing, what the daemon says is queued,
// what Spotify calls a related artist — and an id is missing from at least one
// of those on every path. Folded to lower case, which is as far as matching
// names can honestly be pushed.
type fitTook struct {
	near      map[string]bool
	suggested map[string]bool
	err       error
}

// fitAvailable reports whether the queue can be put in order at all: something
// playing to order it around, something to order, and a backend allowed to be
// asked and able to be told.
func (m Model) fitAvailable() bool {
	if m.ps == nil || m.ps.TrackID == "" || len(m.queue) < 2 {
		return false
	}
	if !m.allows.Has(player.Suggesting) {
		return false
	}
	if _, ok := m.player.(player.QueueEditor); !ok {
		return false
	}
	_, suggests := m.player.(player.Suggester)
	_, relates := m.player.(player.RelatedArtists)
	return suggests || relates
}

// askFit puts the question in a box rather than acting on it.
//
// One press of a key must not rearrange a list somebody spent a minute
// building. Nothing is lost either way — every track stays — but the order they
// were in is theirs, and it is not recoverable once it has gone, which is the
// whole test of whether something should ask first.
//
// The box is the menu's own: a title, and what can be done about it. Choosing
// this from the menu does not ask again, because opening a menu and picking a
// line is already two deliberate acts, and asking twice teaches people to press
// through questions without reading them.
func (m *Model) askFit() bool {
	if !m.fitAvailable() {
		return false
	}

	x, y := m.cursorPoint(m.layout())
	m.actions = actionsPane{
		open:     true,
		title:    "Order what is coming?",
		subtitle: "to follow " + m.ps.Title,
		x:        x, y: y,
		verbs: []verb{{
			label: "Order it — nothing is dropped, only the order changes",
			do:    func(m *Model) tea.Cmd { return m.fitQueue() },
		}, {
			label: "Leave it as it is",
			do:    func(*Model) tea.Cmd { return nil },
		}},
	}
	return true
}

// fitQueue asks what goes with what is playing.
//
// The asking is where the time goes — two requests, both of them catalogue
// answers the gate keeps for an hour — so the ordering itself happens when the
// answer lands. See tookFit.
func (m *Model) fitQueue() tea.Cmd {
	if !m.fitAvailable() {
		return nil
	}

	p := m.player
	seed := m.ps.TrackID

	// The related artists need an id, which the device's own report of what is
	// playing does not carry; the queue's copy of the same track does, where the
	// two agree. Without one that half is simply skipped — the suggestions are
	// seeded by the track, which is always known.
	var artist string
	if now, ok := m.nowPlayingRow(); ok && len(now.ArtistIDs) > 0 {
		artist = now.ArtistIDs[0]
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		near := map[string]bool{}
		if source, ok := p.(player.RelatedArtists); ok && artist != "" {
			found, err := source.RelatedArtists(ctx, artist)
			if err != nil {
				return fitTook{err: err}
			}
			for _, a := range found {
				near[foldName(a.Name)] = true
			}
		}

		suggested := map[string]bool{}
		if source, ok := p.(player.Suggester); ok {
			found, err := source.Suggestions(ctx, seed, fitSuggestions)
			if err != nil {
				return fitTook{err: err}
			}
			for _, t := range found {
				for _, name := range t.Artists {
					suggested[foldName(name)] = true
				}
			}
		}
		return fitTook{near: near, suggested: suggested}
	}
}

// tookFit puts the queue in order and sends it.
func (m *Model) tookFit(message fitTook) tea.Cmd {
	if message.err != nil {
		return func() tea.Msg { return msg.Error{Err: message.err} }
	}
	if m.ps == nil || len(m.queue) < 2 {
		return nil
	}

	ordered := fitOrder(m.queue, m.ps.Artists, message.near, message.suggested)
	if slices.EqualFunc(ordered, m.queue, func(a, b player.Track) bool { return a.ID == b.ID }) {
		m.said, m.saidAt = "What is coming already follows this", time.Now()
		return nil
	}

	// On screen at once, as every other edit to this list is. What was sent is
	// what is drawn; a queue that rearranged itself half a second later would
	// look like something else had done it.
	m.queue = ordered
	m.clampQueueCursor()
	m.said, m.saidAt = "What is coming now follows "+m.ps.Title, time.Now()

	editor, ok := m.player.(player.QueueEditor)
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(ordered))
	for _, t := range ordered {
		ids = append(ids, t.ID)
	}
	return controlCmd("order what is coming", func(ctx context.Context) error {
		return editor.Reorder(ctx, ids)
	})
}

// fitOrder is the new order: what goes with the record playing first, and
// everything else after it in the order it was already in.
//
// Stable, and it keeps every track. A sort that dropped a row, or that shuffled
// the rest while it was at it, would be a different list from the one somebody
// built — see the queue's whole reason for existing.
func fitOrder(queue []player.Track, playing []string, near, suggested map[string]bool) []player.Track {
	same := map[string]bool{}
	for _, name := range playing {
		same[foldName(name)] = true
	}

	scored := make([]player.Track, len(queue))
	copy(scored, queue)

	slices.SortStableFunc(scored, func(a, b player.Track) int {
		return fitScore(b, same, near, suggested) - fitScore(a, same, near, suggested)
	})
	return scored
}

// fitScore is how well one track goes with what is playing: the strongest claim
// any of its artists can make.
func fitScore(t player.Track, same, near, suggested map[string]bool) int {
	best := 0
	for _, name := range t.Artists {
		switch folded := foldName(name); {
		case same[folded]:
			best = max(best, fitSame)
		case near[folded]:
			best = max(best, fitNear)
		case suggested[folded]:
			best = max(best, fitSuggested)
		}
	}
	return best
}

// foldName is how two names are compared. Case only: anything cleverer —
// dropping punctuation, folding accents — would join artists who are not the
// same, and this decides an order rather than an identity.
func foldName(name string) string { return strings.ToLower(strings.TrimSpace(name)) }
