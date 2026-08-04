package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

const (
	// lyricsMinRows is the least the words are worth showing in. Fewer than
	// this and the line being sung has no context around it, which is most of
	// the point.
	lyricsMinRows = 6

	// lyricsMaxRows is the most shown at once. A whole lyric on screen is a
	// page of text to search through; a handful of lines around the one being
	// sung is something to follow.
	lyricsMaxRows = 11

	// lyricsLead is how many of those rows sit above the current line. Exactly
	// half: the fade falls away evenly on both sides, which is what makes the
	// block read as a surface curving away rather than as a list with a tail.
	lyricsLead = lyricsMaxRows / 2
)

// lyricsState is the words of the track playing, and whether they are on screen.
type lyricsState struct {
	// on is what the key toggles. Off to begin with: the words are a mode, not
	// the default way to look at a player.
	on bool

	// forTrack is the track lines belong to, so a fetch that lands after a skip
	// can be thrown away rather than captioning the wrong song.
	forTrack string
	lines    []player.Lyric
	synced   bool

	// missing records that the track was asked about and has none, which is a
	// different thing from not having asked yet.
	missing bool

	// fetching stops a second request going out for the same track while the
	// first is still in the air.
	fetching string
}

// lyricsAvailable reports whether the words can be shown without moving
// anything: they go into the rows below the transport line, which the player
// screen only has when the artwork is not filling the body.
func (m Model) lyricsAvailable() bool {
	if m.tab != tabPlayer || m.noDevice || m.ps == nil {
		return false
	}
	return m.lyricsRows(m.layout()) >= lyricsMinRows
}

// lyricsVisible reports whether the words are on screen right now.
func (m Model) lyricsVisible() bool { return m.lyrics.on && m.lyricsAvailable() }

// artTop is the row the picture begins on, which is as high as the track
// information is allowed to go.
func (m Model) artTop(l layout, rows int) int {
	if !l.hasArt() {
		return 0
	}
	return max((rows-l.artHeight)/2, 0)
}

// lyricsRows is how many rows are free under the information block once it has
// been raised to the top of the picture.
func (m Model) lyricsRows(l layout) int {
	return max(l.bodyHeight-m.artTop(l, l.bodyHeight)-len(m.infoBlock(l.infoWidth))-1, 0)
}

// lyricsBlock is the words, laid out to fill the rows given.
//
// The line being sung sits a third of the way down rather than in the middle:
// what comes next is worth more than what has gone, and reading ahead is what
// anyone does with a lyric.
func (m Model) lyricsBlock(w, rows int) []string {
	if rows <= 0 {
		return nil
	}

	switch {
	case m.lyrics.forTrack != m.ps.TrackID:
		return m.lyricsNote(w, rows, "Looking for the words…")
	case m.lyrics.missing:
		return m.lyricsNote(w, rows, "No lyrics for this track.")
	case len(m.lyrics.lines) == 0:
		return m.lyricsNote(w, rows, "Looking for the words…")
	}

	at := m.lyricsAt()
	// Not clamped to the end of the lyric: the line being sung keeps its place
	// on screen all the way through, and the rows below it simply run out. A
	// window that stops scrolling near the end would slide the lit line down
	// under the eye that is following it.
	from := max(at-lyricsLead, 0)

	out := make([]string, 0, rows)
	for i := from; i < len(m.lyrics.lines) && len(out) < rows; i++ {
		words := strings.TrimSpace(m.lyrics.lines[i].Words)
		if words == "" || words == "♪" {
			// The provider marks instrumental breaks with a note. A blank row
			// carries the pause better than a symbol repeated down the screen.
			out = append(out, strings.Repeat(" ", w))
			continue
		}

		// Wrapped, not cut: a line of a song that stops before its last word is
		// not worth showing. The continuation carries the same strength, so a
		// long line still reads as one line.
		style := m.lyricStyle(i - at)
		for _, part := range wrapWords(words, w) {
			if len(out) == rows {
				break
			}
			out = append(out, fit(style.Render(part), w))
		}
	}
	for len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}
	return out[:rows]
}

// lyricStyle is how a line is drawn, given how far it is from the one being
// sung. The fade is symmetric: a line two ahead reads as faint as one two
// behind, so nothing but the current line competes for attention.
func (m Model) lyricStyle(distance int) lipgloss.Style {
	fade := m.styles.LyricFade
	if len(fade) == 0 {
		return lipgloss.NewStyle()
	}
	if distance < 0 {
		distance = -distance
	}
	return fade[min(distance, len(fade)-1)]
}

// lyricsAt is the line being sung. Unsynced lyrics have no such line, so the
// first one stands in and the words simply scroll with nothing.
func (m Model) lyricsAt() int {
	if !m.lyrics.synced {
		return 0
	}

	pos := m.elapsed().Milliseconds()
	at := 0
	for i, line := range m.lyrics.lines {
		if line.At > pos {
			break
		}
		at = i
	}
	return at
}

// lyricsNote is what stands in the words' place when there are none to show.
func (m Model) lyricsNote(w, rows int, text string) []string {
	out := make([]string, rows)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
	}
	if rows > 1 {
		out[1] = fit(m.styles.Empty.Render(text), w)
	}
	return out
}

// infoWithLyrics is the player's right-hand column with the words under it.
//
// The track information goes to the top of the body and the words take
// everything below it. The picture does not move — it stays centred, exactly
// where it sits without the words — so only this column rearranges.
func (m Model) infoWithLyrics(l layout, rows int) []string {
	out := make([]string, 0, rows)

	// Only as far as the top of the picture, never past it: the two columns
	// belong together, and text starting above the sleeve reads as two screens
	// side by side rather than one.
	for range m.artTop(l, rows) {
		out = append(out, strings.Repeat(" ", l.infoWidth))
	}

	for _, line := range m.infoBlock(l.infoWidth) {
		if len(out) == rows {
			return out
		}
		out = append(out, fit(line, l.infoWidth))
	}

	// A blank row after the controls, so the words do not read as another line
	// of the track information.
	if len(out) < rows {
		out = append(out, strings.Repeat(" ", l.infoWidth))
	}
	out = append(out, m.lyricsBlock(l.infoWidth, min(rows-len(out), lyricsMaxRows))...)

	for len(out) < rows {
		out = append(out, strings.Repeat(" ", l.infoWidth))
	}
	return out[:rows]
}

// fetchLyrics asks for the words of the track playing, once. It returns nil
// when there is nothing to ask for, when the answer is already held, or when a
// request for the same track is already in the air.
func (m *Model) fetchLyrics() tea.Cmd {
	if !m.lyrics.on || m.ps == nil || m.ps.TrackID == "" {
		return nil
	}
	if m.lyrics.forTrack == m.ps.TrackID || m.lyrics.fetching == m.ps.TrackID {
		return nil
	}

	source, ok := m.player.(player.LyricSource)
	if !ok {
		return nil
	}

	m.lyrics.fetching = m.ps.TrackID
	return lyricsCmd(source, m.ps.TrackID)
}

// adoptLyrics takes an answer, if it is still the one being waited for.
func (m *Model) adoptLyrics(res msg.LyricsFetched) {
	if m.ps == nil || res.TrackID != m.ps.TrackID {
		// The track changed while the request was in the air. Captioning the
		// new song with the old song's words is the one thing worse than none.
		return
	}

	m.lyrics.fetching = ""
	m.lyrics.forTrack = res.TrackID
	m.lyrics.lines = res.Lines
	m.lyrics.synced = res.Synced
	m.lyrics.missing = len(res.Lines) == 0
}

// wrapWords breaks a line to fit a width, on spaces where it can and mid-word
// only when a single word is longer than the whole row.
func wrapWords(line string, w int) []string {
	if w <= 0 {
		return nil
	}

	var out []string
	var row strings.Builder
	for _, word := range strings.Fields(line) {
		switch {
		case row.Len() == 0:
			row.WriteString(word)
		case lipgloss.Width(row.String())+1+lipgloss.Width(word) <= w:
			row.WriteString(" ")
			row.WriteString(word)
		default:
			out = append(out, row.String())
			row.Reset()
			row.WriteString(word)
		}
	}
	if row.Len() > 0 {
		out = append(out, row.String())
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}
