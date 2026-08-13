package ui

import (
	"strings"
	"time"

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

	// lyricsAhead is the most a line may arrive in front of the voice that sings
	// it. Raised from 300ms after listening: a line lit at the moment its first
	// syllable sounds has already been passed by the time it is read, so the
	// words want to arrive slightly before the voice, the way a singer reads
	// ahead of themselves.
	lyricsAhead = 550 * time.Millisecond

	// lyricsLeadShare is how much of a line's own window that lead may take.
	//
	// Half a second is a comfortable head start on a ballad and a quarter of the
	// line on a rap. Fixed, it cost the fast records dearly: a line that gives
	// way 550 ms early has to be swept in whatever is left, and measured against
	// the tapped lines it finished a median of 446 ms before the singer did.
	// Fifteen per cent of the window brings that to 162 ms and the whole
	// screen's error from 414 to 233. Below ten per cent the slow records start
	// to suffer instead — the lead is what makes a lyric readable, and taking it
	// all away is not free.
	lyricsLeadShare = 0.15

	// lyricsMaxRows is the most shown at once. A whole lyric on screen is a
	// page of text to search through; a handful of lines around the one being
	// sung is something to follow.

	lyricsMaxRows = 11

	// lyricsLead is how many of those rows sit above the current line. Exactly
	// half: the fade falls away evenly on both sides, which is what makes the
	// block read as a surface curving away rather than as a list with a tail.
	lyricsLead = lyricsMaxRows / 2

	// lyricsDefaultLine is how long the last line of a lyric is assumed to last,
	// there being nothing after it to measure against.
	lyricsDefaultLine = 4 * time.Second

	// lyricsSungShare and lyricsSungMost are how much of a line's window is
	// actually sung. See lyricsSung.
	lyricsSungShare = 0.85
	lyricsSungMost  = 3 * time.Second

	// lyricsStampsEarly is how far a line's timestamp sits in front of the
	// voice it belongs to.
	//
	// A stamp is written by somebody watching a bar go past, and a singer comes
	// in a moment after the bar does; every transcriber leans the same way, and
	// the lyric is timed to the music rather than to the mouth. It shows in the
	// tapping: the same hand landed +346, +350 and +430 ms after the stamps on
	// three different sections, and a hand answering a sound it is waiting for
	// is worth about two hundred of that. What is left over is the stamp.
	//
	// Only the lighting is moved by it. Which line is on screen still follows
	// the stamp, because that is a matter of reading and reading wants to be
	// early. See lyricsVoice.
	lyricsStampsEarly = 200 * time.Millisecond
)

// lyricsSung is how long a line is sung for, which is not how long it is on
// screen.
//
// A lyric is timed by the line and never by the word, so the only thing a line
// carries is when it begins; when it stops has to be guessed, and spreading it
// evenly until the next line begins is the guess this made for a long time.
// Measured, that guess is a second out: 36 lines tapped by ear against the
// playhead — a slow ballad, a Hungarian verse and the rap from the same record —
// put the end of the singing at a median of 0.68, 0.76 and 0.82 of the way to
// the next line. The rest of each window is the band playing on alone.
//
// Eighty-five per cent of the window, and never more than three seconds,
// answers all three: a median error of 211 ms against 734 for running the whole
// window, and it needs neither the tempo nor a syllable count — the syllables
// were tried and are worth nothing here, because the same singer takes 442 ms
// over one in the verse and 141 in the rap. See FINDINGS.md, and cmd/spindle-tap
// for how the tapping was measured.
//
// Both numbers were fitted on those 36 lines and nothing above 160 bpm has been
// checked, so the ceiling is the part to doubt first.
func lyricsSung(window time.Duration) time.Duration {
	return min(time.Duration(float64(window)*lyricsSungShare), lyricsSungMost)
}

// lyricsHeadStart is how far in front of its own stamp a line takes the screen.
func lyricsHeadStart(window time.Duration) time.Duration {
	return min(lyricsAhead, time.Duration(float64(window)*lyricsLeadShare))
}

// lyricsWindow is how long a line has before the next one is due, which is not
// how long it is sung for — see lyricsSung.
func (m Model) lyricsWindow(i int) time.Duration {
	if i < 0 || i >= len(m.lyrics.lines) {
		return 0
	}
	if i+1 >= len(m.lyrics.lines) {
		return lyricsDefaultLine
	}
	return time.Duration(m.lyrics.lines[i+1].At-m.lyrics.lines[i].At) * time.Millisecond
}

// lyricsShows is the moment a line takes the screen: its own stamp, less the
// head start it is allowed. Everything about a line is measured from here — when
// the one before it gives way, and how much time the sweep has.
func (m Model) lyricsShows(i int) int64 {
	if i < 0 || i >= len(m.lyrics.lines) {
		return 0
	}
	return m.lyrics.lines[i].At - lyricsHeadStart(m.lyricsWindow(i)).Milliseconds()
}

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

	// language is what the words are in, which is how a syllable is counted in
	// them. See syllables.go.
	language string

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

// lyricsFit is how many rows the words may take.
//
// Never past the foot of the picture: the two columns are one composition, and
// text hanging below the sleeve reads as spilling out of it. On a screen with
// no picture there is nothing to stay level with, so the cap is the window.
func (m Model) lyricsFit(l layout) int {
	if !l.hasArt() {
		return lyricsMaxRows
	}
	// Measured against where the picture ends, not against the box it was given.
	// The box is a row or two taller than a square cover drawn in it, so
	// budgeting by artHeight ran the words past the sleeve's foot — which
	// layout.go already says reads as a mistake.
	return min(max(l.artRows-len(m.infoBlock(l.infoWidth))-1, 0), lyricsMaxRows)
}

// lyricsRows is how many rows the words have to work with, which is what
// decides whether they are worth offering at all.
func (m Model) lyricsRows(l layout) int {
	if !l.hasArt() {
		return max(l.bodyHeight-len(m.infoBlock(l.infoWidth))-1, 0)
	}
	return m.lyricsFit(l)
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

	// Words with no timings do not follow the music, and a screen that does not
	// say so reads as a screen that has stopped working. Measured against a
	// live account: Spotify has the words for plenty of tracks and the timings
	// for fewer, and which you get is decided per track by whoever supplied
	// them — there is nothing to fix and nothing to wait for.
	if !m.lyrics.synced {
		return m.lyricsPlain(w, rows)
	}

	// Not clamped at either end: the line being sung keeps its place on screen
	// from the first line to the last, and the rows beyond simply run out. A
	// window that stopped scrolling near an end would slide the lit line under
	// the eye that is following it — which is why the opening of a track looked
	// unlike the rest of it.
	from := at - lyricsLead

	out := make([]string, 0, rows)
	for i := from; i < 0 && len(out) < rows; i++ {
		out = append(out, strings.Repeat(" ", w))
	}
	from = max(from, 0)
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
		parts := wrapWords(words, w)

		if i != at {
			for _, part := range parts {
				if len(out) == rows {
					break
				}
				out = append(out, fit(style.Render(part), w))
			}
			continue
		}

		// The line being sung is swept through as it goes. The words are not
		// timed individually — only lines are — so this claims no more than the
		// progress bar does about a track: not which word, but how far in.
		cut := m.lyricsSweep(i, words)
		seen := 0
		for _, part := range parts {
			if len(out) == rows {
				break
			}
			runes := []rune(part)
			split := min(max(cut-seen, 0), len(runes))
			seen += len(runes) + 1 // the space the wrap ate

			out = append(out, fit(
				style.Render(string(runes[:split]))+
					m.styles.LyricAhead.Render(string(runes[split:])), w))
		}
	}
	for len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}
	return out[:rows]
}

// lyricsPlain is the words when nobody timed them: all of them, from the top,
// in one weight, under a line saying why nothing here is going to move.
//
// The alternative was to light the first line and leave it lit, which is what
// this screen used to do — and it reads as a screen that has stopped working
// rather than as a track nobody has timed. Measured against a live account:
// Spotify has the words for plenty of tracks and the timings for fewer, and
// which of the two arrives is decided per track by whoever supplied them.
func (m Model) lyricsPlain(w, rows int) []string {
	out := []string{
		fit(m.styles.Empty.Render("Not timed — these words will not follow the music."), w),
		strings.Repeat(" ", w),
	}

	for _, line := range m.lyrics.lines {
		words := strings.TrimSpace(line.Words)
		if words == "" || words == "♪" {
			out = append(out, strings.Repeat(" ", w))
			continue
		}
		for _, part := range wrapWords(words, w) {
			if len(out) >= rows {
				break
			}
			out = append(out, fit(m.lyricStyle(1).Render(part), w))
		}
		if len(out) >= rows {
			break
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

// lyricsAt is the line being sung, or -1 before the first one. Words that were
// never timed have no such line at all, and are drawn by lyricsPlain instead.
func (m Model) lyricsAt() int {
	if !m.lyrics.synced {
		return 0
	}

	// Songs open with a bar or two of music. Lighting the first line through it
	// says it is being sung when it is not.
	pos := m.elapsed().Milliseconds()
	at := -1
	for i := range m.lyrics.lines {
		if m.lyricsShows(i) > pos {
			break
		}
		at = i
	}
	return at
}

// lyricsSweep is how much of the current line has been reached, spread evenly
// over the time the line is sung, and rounded to a whole word.
//
// Evenly is a guess — nobody sings at a constant rate — and the guess is why it
// moves a word at a time rather than a letter at a time. A letter-by-letter
// sweep claims to know which syllable is in the air, and is visibly wrong for
// most of every word; a word lights as its turn comes and stays lit, which is
// the same claim a progress bar makes about a track and is what the eye follows
// anyway.
//
// Over the time the line is sung, not the time until the next one: those are
// different by about a second, and it is the second that used to be given away.
// See lyricsSung.
func (m Model) lyricsSweep(i int, line string) int {
	length := len([]rune(line))
	if !m.lyrics.synced || i < 0 || i >= len(m.lyrics.lines) {
		return length
	}

	start := m.lyrics.lines[i].At
	window := m.lyricsWindow(i)
	onScreen := time.Duration(m.lyricsShows(i+1)-start) * time.Millisecond
	if i+1 >= len(m.lyrics.lines) {
		onScreen = window
	}

	// Never longer than the line is the lit one. The next line takes over a
	// lead ahead of its own stamp — that is what makes it readable — so a line
	// on a short window gives way before 85% of it has passed, and the last word
	// of it would never light at all. Measured on screen: below about three and
	// a half seconds of window, which is most of a fast verse.
	//
	// The singing is being cut short here rather than the truth being told, and
	// that is the right way round: a word that lights a little early is a small
	// lie, and a word that never lights is a broken screen.
	sung := min(lyricsSung(window), onScreen-lyricsStampsEarly)
	if sung <= 0 {
		return length
	}
	end := start + sung.Milliseconds()

	// Before the voice has reached the line, nothing of it is lit. The line is
	// on screen by then — it arrives early on purpose — and it stands there to
	// be read rather than pretending to be sung. See lyricsVoice.
	voice := m.lyricsVoice()
	if voice < start {
		return 0
	}

	frac := float64(voice-start) / float64(end-start)
	return sweepTo(line, m.lyrics.language, min(max(frac, 0), 1))
}

// lyricsVoice is the playhead the sweep is lit against: the one the singer is
// actually at, with nothing added.
//
// The lead belongs to the line and not to the word. A line wants to arrive
// before the voice — that is what makes it readable, and it is why lyricsClock
// exists — but a word lit half a second before it is sung is the screen saying
// something untrue about this moment, and it was the first thing anybody
// noticed once the ends were right. So the line comes early and unlit, and each
// word lights as it is sung.
func (m Model) lyricsVoice() int64 {
	return m.elapsed().Milliseconds() - int64(lyricsStampsEarly/time.Millisecond)
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

	// The track information and the transport, outlined on their own: they are
	// what somebody means by "the player part", and lumping them in with the
	// words made the one block on this screen that nobody could point at.
	for _, line := range m.place(m.playerBlock(), l.infoWidth, len(m.infoBlock(l.infoWidth))) {
		if len(out) == rows {
			return out
		}
		out = append(out, line)
	}

	// A blank row after the controls, so the words do not read as another line
	// of the track information.
	if len(out) < rows {
		out = append(out, strings.Repeat(" ", l.infoWidth))
	}
	out = append(out, m.place(m.wordsBlock(), l.infoWidth, min(rows-len(out), m.lyricsFit(l)))...)

	for len(out) < rows {
		out = append(out, strings.Repeat(" ", l.infoWidth))
	}
	return out[:rows]
}

// fetchLyrics asks for the words of the track playing, once. It returns nil
// when there is nothing to ask for, when the answer is already held, or when a
// request for the same track is already in the air.
func (m *Model) fetchLyrics() tea.Cmd {
	if m.ps == nil || m.ps.TrackID == "" {
		return nil
	}
	// The pane on the player is one place the words are wanted. The big
	// screen's lyric picture is the other, and it is reached by its own key —
	// without this it would wait for words nobody had sent for, and quietly
	// show every record as if it had none.
	if !m.lyrics.on && !(m.stage.on && m.scopeMode().words()) {
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
	m.lyrics.language = res.Language
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
