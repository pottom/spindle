package ui

import (
	"context"
	"fmt"
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

// songTook is what was learned about a song coming back.
type songTook struct {
	track string
	got   notes.TrackNote
}

// askSong sends for what is written about the song that is playing, once.
func (m *Model) askSong(id, artist, title string) tea.Cmd {
	if m.notes == nil || id == "" || artist == "" || title == "" {
		return nil
	}
	if _, held := m.songs[id]; held {
		return nil
	}
	if m.askingSong == nil {
		m.askingSong = map[string]bool{}
	}
	if m.askingSong[id] {
		return nil
	}
	m.askingSong[id] = true

	source := m.notes
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), notesWait)
		defer cancel()

		got, err := source.Track(ctx, notes.TrackKey{Artist: artist, Title: title})
		if err != nil {
			return songTook{track: id}
		}
		return songTook{track: id, got: got}
	}
}

// tookSong files one.
func (m *Model) tookSong(msg songTook) {
	if m.songs == nil {
		m.songs = map[string]notes.TrackNote{}
	}
	m.songs[msg.track] = msg.got
	delete(m.askingSong, msg.track)
}

// syncSong asks after the record that is playing. Only that one: it is the one
// somebody might want the story of, and asking after every row a cursor passes
// would be a request a keystroke.
func (m *Model) syncSong() tea.Cmd {
	if m.ps == nil || m.ps.TrackID == "" {
		return nil
	}
	return m.askSong(m.ps.TrackID, firstOf(m.ps.Artists), m.ps.Title)
}

// songNote is what is written about the record that is playing, and whether
// anything is.
func (m Model) songNote() (notes.TrackNote, bool) {
	if m.ps == nil {
		return notes.TrackNote{}, false
	}
	got, held := m.songs[m.ps.TrackID]
	return got, held && got.Known()
}

// firstOf is the name a song is looked up under: the first artist on it.
// Somebody's guest is not who the song is filed under anywhere.
func firstOf(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// openArtistID is the Spotify id of the artist whose page is open, or empty
// where the screen is about something else.
func (m Model) openArtistID() string {
	page := m.open()
	if page == nil || page.kind != openArtist {
		return ""
	}
	return page.id
}

// syncNotes asks after whatever artist the screen is about. Only an artist page:
// that is the one screen with the room for a paragraph and the one where
// somebody has asked about a person rather than about a record.
func (m *Model) syncNotes() tea.Cmd {
	page := m.open()
	if page == nil || page.kind != openArtist {
		return nil
	}
	// And who else sounds like them, which is the same line of the panel from a
	// different source. See related.go.
	return tea.Batch(m.askNotes(page.id, page.name), m.askRelated(page.id))
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
	similar := m.relatedTo(m.openArtistID(), got.Similar)
	if len(similar) > 0 || got.Listeners > 0 {
		// The count first, because it is short and never worth losing, and the
		// names after it, because they are what gets cut on a narrow panel.
		var like []string
		if got.Listeners > 0 {
			like = append(like, formatCount(got.Listeners)+" listeners")
		}
		like = append(like, similar[:min(len(similar), 3)]...)
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

const (
	// storyWidth is how wide the words in the box are set, where there is room.
	//
	// Sixty is where a paragraph reads: much narrower and the eye jumps a line
	// every few words, much wider and it loses its place coming back. The same
	// argument as maxInfoCols, and the same number within a few columns.
	storyWidth = 60

	// storyLeast is the narrowest column of prose worth setting. Under this the
	// box gives up standing beside the picture and stands over it instead: a
	// paragraph in twenty columns is a paragraph nobody reads, and the cover is
	// worth less than the words while the words are being read.
	storyLeast = 36
)

// storyAt is where the box stands and how wide its words are set.
//
// Beside the picture rather than over it: the cover is the one thing on this
// screen that is not words, and a box of words on top of it is the wrong way
// round — what is being read is about the record you are looking at, so the
// record should still be there.
//
// The width follows from the place rather than the other way round. Asked for
// sixty columns where only forty are free, the box was pulled back to the left
// until it fitted — which put it exactly where it was not to go, hard against
// the cover. So the room decides, and the words are set to what is left.
func (m Model) storyPlace(l layout) (left, width int) {
	if l.hasArt() {
		left = leftMargin + l.artWidth + columnGap
		if room := l.interior - rightMargin - left; room >= storyLeast+4 {
			return left, min(storyWidth, room-4)
		}
	}
	return leftMargin + 2, min(storyWidth, l.interior-leftMargin-rightMargin-6)
}

// storyPopup is the box with what is written about the record that is playing.
//
// A box rather than a panel on the screen, because it is read once: it costs
// nothing while it is shut, it stands over whatever you were looking at, and the
// next thing you do puts it away. The same shape as the menu of verbs and the
// list of devices — a short thing about something named at the top of it.
func (m Model) storyPopup() popup {
	l := m.layout()
	left, width := m.storyPlace(l)

	title, sub := "", ""
	if m.ps != nil {
		title, sub = m.ps.Title, strings.Join(m.ps.Artists, ", ")
	}
	return popup{
		x: left, y: tabBarHeight + 1,
		title: title, subtitle: sub, rows: m.storyLines(width), plain: true,
		want: width + 4, at: m.storyAt,
	}
}

// storyAvailable reports whether there is anything to open. The key is not
// offered where there is nothing behind it: a box that opens on nothing is worse
// than a key that is not there.
func (m Model) storyAvailable() bool {
	got, ok := m.songNote()
	return ok && got.Note != ""
}

// storyLines is everything the box holds, in order: the words, and what the
// record is worth to the rest of the world under them.
//
// One list, because the box draws from it and the scroll is bounded by it. Two
// counts of the same rows is how a box comes to say there is more when there is
// not — which is what it did, because the end was worked out from the paragraph
// alone while the box also held three lines after it.
func (m Model) storyLines(width int) []string {
	got, ok := m.songNote()
	if !ok {
		return nil
	}

	var rows []string
	for _, line := range wrapWords(got.Note, width) {
		rows = append(rows, m.styles.Detail.Render(line))
	}

	// Not instead of the words: a number about a song nobody wrote anything
	// about is a box holding one line, which is not worth opening.
	if got.Listeners > 0 {
		rows = append(rows, "", m.styles.Album.Render(fmt.Sprintf(
			"%s listeners · %s plays", formatCount(got.Listeners), formatCount(got.Plays))))
	}
	if len(got.Tags) > 0 {
		rows = append(rows, m.styles.Empty.Render(strings.Join(got.Tags[:min(len(got.Tags), 5)], " · ")))
	}
	if got.NoteFrom != "" {
		rows = append(rows, "", m.styles.Empty.Render("from "+got.NoteFrom))
	}
	return rows
}

// storyRows is how many rows of it are on screen at once, and storyLast the
// furthest it can be scrolled to.
//
// The last is where its last line stands on the last row of the box rather than
// where the text runs out: scrolling past the end into empty rows is what a
// scroll that counts lines rather than screens does, and it always feels broken.
func (m Model) storyRows(l layout) int {
	return max(l.bodyHeight-menuChrome, 1)
}

func (m Model) storyLast(l layout) int {
	_, width := m.storyPlace(l)
	return max(len(m.storyLines(width))-m.storyRows(l), 0)
}
