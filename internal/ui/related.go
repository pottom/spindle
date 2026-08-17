package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// Who else sounds like this one.
//
// The panel beside an artist has carried that line since last.fm went in, and it
// was the one thing there that needed a key nobody has by default. Spotify
// answers the same question to an application it has not clamped down on — see
// player.Suggesting — so where there is no key, or where last.fm has never heard
// of somebody, this fills the same line.
//
// Spotify's answer is artists rather than names, and the ids are what make them
// somewhere to go: the menu over an artist offers each of them, so "who else
// sounds like this" is a door rather than a fact. Last.fm's are names only, so
// they stay a line of text — which is the honest difference between the two.

// relatedTook is an answer arriving.
type relatedTook struct {
	artist string
	found  []player.Artist
}

// askRelated asks who else sounds like the artist on screen, once each.
//
// Only where the page is an artist: it is the one screen with room for the
// answer and the one where the question is being asked. An answer of nothing is
// an answer and is kept, because most of a real library is artists Spotify has
// nobody to compare with.
func (m *Model) askRelated(artistID string) tea.Cmd {
	if artistID == "" || !m.allows.Has(player.Suggesting) {
		return nil
	}
	if _, asked := m.related[artistID]; asked {
		return nil
	}

	source, ok := m.player.(player.RelatedArtists)
	if !ok {
		return nil
	}
	if m.related == nil {
		m.related = map[string][]player.Artist{}
	}
	// Marked before the answer, so a screen redrawn while it is in flight does
	// not ask again.
	m.related[artistID] = nil

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		found, err := source.RelatedArtists(ctx, artistID)
		if err != nil {
			// Nothing to report: the line simply stays as last.fm left it, or
			// empty. A refusal has already been recorded against the ability.
			return relatedTook{artist: artistID}
		}
		return relatedTook{artist: artistID, found: found}
	}
}

// tookRelated files an answer.
func (m *Model) tookRelated(message relatedTook) {
	if m.related == nil {
		m.related = map[string][]player.Artist{}
	}
	m.related[message.artist] = message.found
}

// relatedArtists is who Spotify says sounds like this one, with the ids that
// make them somewhere to go rather than a fact. See actionsForArtist.
func (m Model) relatedArtists(artistID string) []player.Artist {
	return m.related[artistID]
}

// relatedTo is who else sounds like an artist, from whichever source has an
// answer. Last.fm first where there is one: it answers about how a record is
// heard rather than about a catalogue, and on a Hungarian artist it is the only
// one that answers at all.
func (m Model) relatedTo(artistID string, fromNotes []string) []string {
	if len(fromNotes) > 0 {
		return fromNotes
	}

	found := m.related[artistID]
	names := make([]string, 0, len(found))
	for _, a := range found {
		names = append(names, a.Name)
	}
	return names
}
