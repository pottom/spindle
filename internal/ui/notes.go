package ui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/notes"
)

// What the other databases say, on the screen.
//
// Spotify knows a name, a picture and a follower count. Who this is, where they
// are from, what they are called when they are not on a stage, and who was in
// the band in which years — none of that is anywhere in what a player is handed,
// and all of it is free elsewhere. See internal/notes.
//
// It is an addition and never a requirement. The panel is drawn from what
// arrived; where nothing arrived the screen is exactly what it was before any of
// this existed, which is also what happens with no network, no key and no luck.

// notesTook is an answer coming back. It carries the id it was asked about
// rather than only the answer, because by the time it lands the cursor may be
// somewhere else entirely.
type notesTook struct {
	artist string
	got    notes.Artist
}

// askNotes sends for what is known about an artist, once.
//
// Once: an answer already held — including an answer that turned out to be
// nothing — is not asked for again. The whole of this is affordable only because
// the second visit to a record costs no requests at all.
func (m *Model) askNotes(id, name string) tea.Cmd {
	if m.notes == nil || id == "" {
		return nil
	}
	if _, held := m.artists[id]; held {
		return nil
	}
	if m.asking == nil {
		m.asking = map[string]bool{}
	}
	if m.asking[id] {
		return nil
	}
	m.asking[id] = true

	source := m.notes
	return func() tea.Msg {
		// Its own budget, and a generous one: this walks two or three databases
		// at a request a second and nothing on screen is waiting for it.
		ctx, cancel := context.WithTimeout(context.Background(), notesWait)
		defer cancel()

		got, err := source.Artist(ctx, notes.Key{SpotifyArtist: id, Name: name})
		if err != nil {
			// Not an error anybody needs telling about: it means the panel is
			// the panel it was. Asking again is the tick's business, not this
			// one's.
			return notesTook{artist: id}
		}
		return notesTook{artist: id, got: got}
	}
}

// tookNotes files an answer.
func (m *Model) tookNotes(msg notesTook) {
	if m.artists == nil {
		m.artists = map[string]notes.Artist{}
	}
	m.artists[msg.artist] = msg.got
	delete(m.asking, msg.artist)
}

// syncNotes asks after whatever artist the screen is about. Only an artist page:
// that is the one screen with the room for a paragraph and the one where
// somebody has asked about a person rather than about a record.
func (m *Model) syncNotes() tea.Cmd {
	page := m.open()
	if page == nil || page.kind != openArtist {
		return nil
	}
	return m.askNotes(page.id, page.name)
}

// artistNotes is what is known about the artist whose page is open, and whether
// anything is.
func (m Model) artistNotes() (notes.Artist, bool) {
	page := m.open()
	if page == nil || page.kind != openArtist {
		return notes.Artist{}, false
	}
	got, held := m.artists[page.id]
	return got, held && got.Known()
}

// notesPanel is the band on an artist page: who this is, and a paragraph about
// them.
//
// It stands where the record under the cursor used to be described. That panel
// said three things, two of which — when it came out, and that it is an album —
// are already in the row the cursor is on and in the column above it. This is a
// page about a person; the panel should be about the person.
func (m Model) notesPanel(w, rows int) []string {
	got, ok := m.artistNotes()
	if !ok {
		// Nothing known yet, or nothing to know. The screen is what it was.
		return m.openAlbumDetail(w, rows)
	}

	lines := []string{m.styles.Title.Render(fit(got.Name, w))}

	// The line under the name: what a database keeps for telling two artists of
	// the same name apart, and where they are from.
	//
	// Not the years. MusicBrainz has them and MusicBrainz had them wrong on the
	// first Hungarian artist they were tried against — 1977 against a Wikidata
	// and a Wikipedia that both say 1979. The description carries a date of its
	// own where it matters, written by whoever wrote the paragraph.
	var about []string
	if got.Line != "" {
		about = append(about, got.Line)
	}
	if got.Area != "" && !strings.Contains(strings.ToLower(got.Line), strings.ToLower(got.Area)) {
		about = append(about, got.Area)
	}
	if len(about) > 0 {
		lines = append(lines, m.styles.Album.Render(fit(strings.Join(about, " · "), w)))
	}

	// The names they are filed under elsewhere. On a stage name this is the one
	// place the given name appears at all.
	if len(got.Aliases) > 0 {
		lines = append(lines, m.styles.Detail.Render(fit(strings.Join(got.Aliases, " · "), w)))
	}

	// Who else people listening to this listen to, and how many of them there
	// are. Not from a catalogue: this is the one line here that says something
	// about how a record is heard rather than about how it was made, and it is
	// the only one that works for a Hungarian artist. See internal/notes.
	if len(got.Similar) > 0 || got.Listeners > 0 {
		// The count first, because it is short and never worth losing, and the
		// names after it, because they are what gets cut on a narrow panel.
		var like []string
		if got.Listeners > 0 {
			like = append(like, formatCount(got.Listeners)+" listeners")
		}
		like = append(like, got.Similar[:min(len(got.Similar), 3)]...)
		lines = append(lines, m.styles.Detail.Render(fit(strings.Join(like, " · "), w)))
	}

	// And the paragraph, in whatever room is left. Wrapped rather than cut: a
	// sentence with its end missing is worse than a sentence fewer.
	if got.Note != "" && len(lines)+2 < rows {
		lines = append(lines, "")

		wrapped := wrapWords(got.Note, w)
		for i, line := range wrapped {
			if len(lines) >= rows {
				break
			}
			// The last row it fits in says that it did not fit. A paragraph
			// stopping in the middle of a word reads as a bug; the same
			// paragraph with a mark after it reads as a paragraph that goes on.
			if len(lines) == rows-1 && i < len(wrapped)-1 {
				line = fit(line, max(w-1, 1)) + "…"
			}
			lines = append(lines, m.styles.Detail.Render(line))
		}
	}
	return lines
}

// notesWait is how long the walk may take before it is given up on. Long,
// because it is several databases at a request a second each and nothing is
// waiting for it; the screen is already drawn.
const notesWait = 30 * time.Second
