package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
)

const (
	rowCursor = "▸"
	nowMark   = "♪"
	// queuedMark says a track is already waiting. Pressing the key that puts it
	// there is otherwise an act with no visible result, and a list you have
	// been picking from is worth being able to read back. A dot rather than a
	// sign: it marks a state, and the column is read down rather than word by
	// word.
	queuedMark = "•"

	// The scrollbar: a lit thumb over a faint track, one cell wide.
	scrollThumb = "┃"
	scrollTrack = "│"

	// scrollCols is what a list gives up to carry the bar: the bar itself and a
	// blank column, so it does not touch the durations.
	scrollCols = 2

	// explicitMark is the badge Spotify puts on a track with explicit lyrics.
	explicitMark = "[E]"

	// ordinalCols is the width of the track number column in a list.
	ordinalCols = 2

	// factLabelCols is the label column of the detail panel.
	factLabelCols = 12

	// The rating: five stars, so each stands for twenty of Spotify's hundred.
	// Nothing scores zero stars — a track nobody plays is still a track.
	starFull  = "★"
	starEmpty = "☆"
	starCount = 5
	starStep  = 100 / starCount

	// hotMark flags a track most of Spotify is playing right now. hotStars is
	// where "most" begins, and hotCols is what the mark occupies: an emoji is
	// two cells wide, and the column is held open on the rows without one.
	hotMark  = "🔥"
	hotStars = 4

	// secondaryCols caps the middle column of a list row: the artist, or the
	// playlist's owner. It takes a third of the row up to this, so a narrow
	// screen keeps the title readable and a wide one stops cutting names that
	// would easily fit — a list of collaborations is mostly commas otherwise.
	secondaryCols = 44

	// trailingCols holds a duration or a count, right aligned.
	trailingCols = 8

	// unknownValue stands where a fact would be, for the ones that are pending
	// rather than absent.
	unknownValue = "—"

	// tempoCols holds a beat rate, right aligned. It is the first column to go
	// when the pane is narrow: a tempo is worth knowing, a title is worth more.
	tempoCols = 4
)

// listBlock is the shape every list screen shares: the cover and the details of
// whatever the cursor rests on across the top, a heading, then the list itself
// across the full width below. The list is the point of the screen, so it gets
// the width; the detail panel fills what would otherwise be empty beside the
// artwork.
//
// One shape for all of them is the point. The queue, the library and the search
// results are the same act — looking down a list of tracks — and three
// compositions for one act made the tabs read as three programs.
func (m Model) listBlock(l layout, rows int, opts listScreen) []string {
	w := queueBlockWidth(l)

	top := min(l.artHeight, rows)
	art := alignTop(strings.Split(m.artworkCells(), "\n"), l.artWidth, top)

	// The picture decides how far the panel beside it may reach: the box the
	// layout gives the artwork is a row or two taller than the cover drawn in
	// it, and a fact hanging below the picture's foot reads as a mistake.
	foot := min(l.artRows, top)

	// The trace hangs from the same column the artists below start at, so the
	// screen reads as two halves. Nothing under it moves when it appears: the
	// list starts where the artwork ends either way.
	detailWidth := l.infoWidth
	var trace []string
	if m.scopeVisible() {
		detailWidth = queueDetailWidth(l)
		trace = stack(m.scopeRender(queueScopeWidth(l)), queueScopeWidth(l), top)
	}
	detail := stack(opts.detail(detailWidth, foot), detailWidth, foot)
	for len(detail) < top {
		detail = append(detail, strings.Repeat(" ", detailWidth))
	}

	out := make([]string, 0, rows)
	gap := strings.Repeat(" ", columnGap)
	for i := range art {
		row := art[i] + gap + detail[i]
		if trace != nil {
			row += gap + trace[i]
		}
		out = append(out, row)
	}
	if len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}

	// The heading carries the count on the right, where a subtitle line would
	// otherwise cost a row the list could use.
	if len(out) < rows {
		out = append(out, spread(opts.heading(w), opts.subtitle(), w))
	}
	if len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}

	body := max(rows-len(out), 0)
	if opts.count == 0 && body > 0 {
		out = append(out, fit(m.styles.Empty.Render(opts.empty), w))
	}

	from, to := opts.state.window(opts.count, body)
	bar := m.scrollColumn(body, opts.count, opts.state.top)
	rowWidth := queueRowWidth(l)

	for i := from; i < to; i++ {
		row := fit(opts.row(i, rowWidth, i == opts.state.cursor), rowWidth)
		if bar != nil {
			row += " " + bar[i-from]
		}
		out = append(out, row)
	}

	for len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}
	return out[:rows]
}

// listScreen is what one list screen puts into that shape.
type listScreen struct {
	// detail is the panel beside the artwork, given its width and the rows it
	// may fill. heading and subtitle are the line under it, the subtitle set to
	// the right.
	detail   func(w, rows int) []string
	heading  func(w int) string
	subtitle func() string

	count int
	state *listState
	empty string
	row   func(i, w int, selected bool) string
}

// queueBlock is the queue screen. The playing track heads the list and is not
// numbered: it is not waiting for its turn, so a place in the running order
// would be a lie.
func (m Model) queueBlock(l layout, rows int) []string {
	rowsOf := m.queueRows()
	_, playing := m.nowPlayingRow()

	return m.listBlock(l, rows, listScreen{
		detail:   m.trackDetail,
		heading:  func(int) string { return m.styles.Title.Render("Queue") },
		subtitle: func() string { return m.styles.Album.Render(queueSubtitle(m.queue)) },
		count:    len(rowsOf),
		state:    &m.queuePane.cursor,
		empty:    "Nothing is queued.",
		row: func(i, w int, selected bool) string {
			number := i + 1
			if playing {
				number = i
			}
			return m.queueRow(rowsOf[i], w, selected, number)
		},
	})
}

// scrollColumn is the thin bar down the right of a list, one glyph per visible
// row. It is nil when the whole list fits: a bar that can never move says
// nothing, and the row it would sit beside is worth more.
func (m Model) scrollColumn(rows, total, offset int) []string {
	if rows <= 0 || total <= rows {
		return nil
	}

	start, size := scrollRange(rows, total, offset)
	out := make([]string, rows)
	for i := range out {
		if i >= start && i < start+size {
			out[i] = m.styles.ScrollThumb.Render(scrollThumb)
			continue
		}
		out[i] = m.styles.ScrollTrack.Render(scrollTrack)
	}
	return out
}

// trackDetail is everything Spotify will say about one track, laid out beside
// the cover. It follows the player screen: the name first and large, the facts
// beneath it in a quiet column, so the two screens read as the same program.
// rows is how tall the panel is; the playhead is only drawn where it fits.
func (m Model) trackDetail(w, rows int) []string {
	t := m.cursorTrack()
	if t == nil {
		return nil
	}
	s := m.styles

	title := s.Title.Render(t.Title)
	if t.Explicit {
		title += "  " + s.FactLabel.Render(explicitMark)
	}

	lines := []string{
		title,
		s.Artist.Render(strings.Join(t.Artists, ", ")),
		"",
	}

	// The playhead and the clock either side of it, for the one track they can
	// belong to. Their rows are kept whether this is that track or not: without
	// that the panel would shift every time the cursor passed the track
	// playing, and a bar is not worth making the facts beside it move. They go
	// in only where there is room for them at all, because those facts are the
	// point of the panel.
	facts := trackFacts(*t)
	if rows >= len(facts)+6 {
		bar, times := "", ""
		if m.ps != nil && m.ps.TrackID == t.ID {
			bar = m.progressLine(w)
			times = spread(
				s.Time.Render(formatDuration(m.elapsed())),
				s.Time.Render(formatDuration(m.ps.Duration)),
				w,
			)
		}
		lines = append(lines, bar, times)
		if rows >= len(facts)+7 {
			// The air under the clock is the first thing to go: better a
			// tighter panel than no playhead at all.
			lines = append(lines, "")
		}
	}

	for _, f := range facts {
		lines = append(lines, m.fact(f.label, f.value, w))
	}
	if t.Popularity != nil {
		lines = append(lines, m.fact("Popularity", m.stars(*t.Popularity), w))
	}
	return lines
}

// starsFor is how many of the five a rating earns. Nothing scores none: a track
// nobody plays is still a track.
func starsFor(popularity int) int {
	return min(max((popularity+starStep-1)/starStep, 1), starCount)
}

// stars renders a rating as five of them, filled in fifths. Spotify's number is
// out of a hundred, which reads as a measurement; a row of stars reads as an
// opinion, which is what it is.
func (m Model) stars(popularity int) string {
	filled := starsFor(popularity)
	return m.styles.StarOn.Render(strings.Repeat(starFull, filled)) +
		m.styles.StarOff.Render(strings.Repeat(starEmpty, starCount-filled))
}

// hot reports whether a track is worth flagging in a list. The rating is on the
// detail panel for whatever the cursor rests on; the mark is what carries it to
// every other row at once.
func hot(t player.Track) bool {
	return t.Popularity != nil && starsFor(*t.Popularity) >= hotStars
}

// leadIn is the track's place in the list. It arrives already styled: a running
// order is part of the row and is drawn like one, while the mark for the track
// playing belongs to the cursor.
func (m Model) leadIn(place string) string {
	// A list of four never reaches double figures, so the second column of the
	// ordinal is space that only pushes the rows out of line with their heading.
	width := ordinalCols
	if m.rowsAreFlush {
		width = 1
	}

	// One space where a mark column follows, two where the title comes next.
	// Four columns between the ordinal and the title is a gap you read across
	// rather than past.
	gap := "  "
	if m.tab != tabQueue {
		gap = " "
	}
	return padLeft(place, width) + gap
}

// withMark is a title with the flag for a track most of Spotify is playing.
// After the title rather than in a column of its own: a column keeps the marks
// in line with each other, but it indents every title to make room for
// something most rows do not have.
func (m Model) withMark(t player.Track, title string) string {
	if hot(t) {
		title += " " + hotMark
	}
	return title
}

// queuedColumn is the narrow band between the ordinal and the title, carrying
// the mark for a track already waiting in the queue. A column of its own rather
// than a flag after the title: what is worth reading here is how much of a list
// has been picked over, and marks scattered along the titles cannot be counted
// at a glance the way a column can.
func (m Model) queuedColumn(t player.Track) string {
	if m.tab == tabQueue {
		// Everything here is in the queue by definition; a column saying so
		// would be a column of dots.
		return ""
	}

	switch {
	case m.ps != nil && m.ps.TrackID == t.ID:
		// The track sounding takes the same column, and keeps its place in the
		// list: a library is numbered so it can be counted, and a row that
		// gives up its ordinal to say what it is doing leaves a hole where the
		// number should be.
		return m.styles.Cursor.Render(nowMark) + " "
	case m.isQueued(t.ID):
		return m.styles.Queued.Render(queuedMark) + " "
	default:
		return "  "
	}
}

// isQueued reports whether a track is one of those waiting by hand. Only those:
// everything the album or playlist supplies is in the list as well, and marking
// all of it would say nothing about what was chosen.
func (m Model) isQueued(id string) bool {
	if id == "" {
		return false
	}
	for _, t := range m.queue {
		if !t.Queued {
			// The hand-added tracks are the leading run; past them is the
			// context, which nobody put there.
			break
		}
		if t.ID == id {
			return true
		}
	}
	return false
}

// fact is one label-and-value row of the detail panel. The label column is
// fixed so the values line up into a second column of their own.
func (m Model) fact(label, value string, w int) string {
	return fit(m.styles.FactLabel.Render(padRight(label, factLabelCols))+value, w)
}

// trackFact is one thing known about a track: what it is called, and what it
// says. Both screens draw the same facts — the queue names them, the player
// does not — so they are gathered once and rendered twice.
type trackFact struct {
	key   string
	label string
	value string
}

// trackFacts is what is worth saying about a track, in the order it is worth
// saying it. Anything Spotify left blank is left out rather than shown empty.
func trackFacts(t player.Track) []trackFact {
	facts := []trackFact{{"album", "Album", t.Album}}

	if year := releaseYear(t.Released); year != "" {
		// A single or a compilation is worth knowing and costs no room of its
		// own; on a plain album it would be saying what the label already said.
		if t.AlbumType != "" && t.AlbumType != "album" {
			year += " · " + t.AlbumType
		}
		facts = append(facts, trackFact{"released", "Released", year})
	}
	if t.TrackNumber > 0 {
		place := fmt.Sprintf("%d", t.TrackNumber)
		if t.TotalTracks > 0 {
			place = fmt.Sprintf("%d of %d", t.TrackNumber, t.TotalTracks)
		}
		if t.DiscNumber > 1 {
			place += fmt.Sprintf(", disc %d", t.DiscNumber)
		}
		facts = append(facts, trackFact{"track", "Track", place})
	}
	facts = append(facts, trackFact{"length", "Length", formatDuration(t.Duration)})

	// The row is always here, even with nothing to put in it. A tempo takes a
	// dozen seconds to measure, so it arrives mid-track — and a row appearing
	// then would push everything under it down while it is being read.
	tempo := unknownValue
	if t.Tempo > 0 {
		tempo = fmt.Sprintf("%.0f bpm", t.Tempo)
	}
	facts = append(facts, trackFact{"tempo", "Tempo", tempo})
	return facts
}

// releaseYear takes the year off a Spotify release date, which may be a year, a
// month or a full date depending on how much the label bothered to record.
func releaseYear(date string) string {
	if len(date) < 4 {
		return ""
	}
	return date[:4]
}

// queueRow draws one row of the queue. A number of 0 means the track is the one
// playing, which is marked rather than numbered: it is not waiting its turn.
// Everything else is one list, numbered in the order it will be heard.
func (m Model) queueRow(t player.Track, w int, selected bool, number int) string {
	if number > 0 {
		return m.trackRow(t, w, selected, number)
	}

	primary := m.styles.RowPlaying
	if selected {
		primary = m.styles.RowSelected
	}
	return m.rowWithTempo(w, selected,
		m.leadIn(m.styles.Cursor.Render(nowMark))+m.withMark(t, primary.Render(t.Title)),
		m.styles.RowSecondary.Render(strings.Join(t.Artists, ", ")),
		m.tempoCell(t),
		m.styles.RowTrailing.Render(formatDuration(t.Duration)),
	)
}

func queueSubtitle(tracks []player.Track) string {
	if len(tracks) == 0 {
		return ""
	}

	var total time.Duration
	for _, t := range tracks {
		total += t.Duration
	}
	return fmt.Sprintf("%d tracks · %s", len(tracks), formatSpan(total))
}

func (m Model) playlistPaneView(l layout, rows int) []string {
	if m.playlists.open == nil {
		return m.listBlock(l, rows, listScreen{
			detail:   m.playlistDetail,
			heading:  func(int) string { return m.styles.Title.Render("Library") },
			subtitle: func() string { return m.styles.Album.Render(playlistSubtitle(m.playlists.items)) },
			count:    len(m.playlists.items),
			state:    &m.playlists.cursor,
			empty:    "Nothing saved yet.",
			row: func(i, w int, selected bool) string {
				return m.playlistRow(m.playlists.items[i], w, selected)
			},
		})
	}

	open := *m.playlists.open
	return m.listBlock(l, rows, listScreen{
		detail:   m.trackDetail,
		heading:  func(w int) string { return fit(m.styles.Title.Render(open.Name), max(w/2, 1)) },
		subtitle: func() string { return m.styles.Album.Render(openPlaylistSubtitle(open, m.playlists.tracks)) },
		count:    len(m.playlists.tracks),
		state:    &m.playlists.inner,
		empty:    "This playlist is empty.",
		row: func(i, w int, selected bool) string {
			return m.trackRow(m.playlists.tracks[i], w, selected, i+1)
		},
	})
}

// playlistDetail is the panel beside the cover at the top level: what the
// cursor is resting on, laid out like a track's so the two levels of the tab
// read as one screen.
func (m Model) playlistDetail(w, rows int) []string {
	p := m.playlists.selected()
	if p == nil {
		return nil
	}
	s := m.styles

	lines := []string{
		s.Title.Render(p.Name),
		s.Artist.Render(p.Owner),
		"",
	}
	if p.Description != "" {
		// Room for what is left: the facts under it, and a blank either side.
		room := max(rows-len(lines)-3, 0)
		if wrapped := wrapWords(s.Detail.Render(p.Description), w); room > 0 && len(wrapped) > 0 {
			lines = append(lines, wrapped[:min(len(wrapped), room)]...)
			lines = append(lines, "")
		}
	}
	lines = append(lines, m.fact("Tracks", fmt.Sprintf("%d", p.Tracks), w))
	if p.Duration > 0 {
		lines = append(lines, m.fact("Length", formatSpan(p.Duration), w))
	}
	return lines
}

// openPlaylistSubtitle is what an open playlist says about itself beside its
// name: whose it is, how much of it there is.
func openPlaylistSubtitle(p player.Playlist, tracks []player.Track) string {
	out := p.Owner
	if p.Tracks > 0 {
		out += fmt.Sprintf(" · %d tracks", p.Tracks)
	}

	// Spotify does not report a playlist's length, so it is added up from the
	// tracks that have arrived — and only once they all have, because a growing
	// number beside a fixed one reads as a mistake rather than as progress.
	if len(tracks) == p.Tracks {
		var total time.Duration
		for _, t := range tracks {
			total += t.Duration
		}
		if total > 0 {
			out += " · " + formatSpan(total)
		}
	}
	return out
}

func (m Model) searchPaneView(l layout, rows int) []string {
	empty := "Type to search the catalogue."
	if strings.TrimSpace(m.search.input.Value()) != "" {
		empty = "Nothing matched."
	}

	return m.listBlock(l, rows, listScreen{
		detail: m.trackDetail,
		// The field is the heading: it is what the screen is about, and it has
		// to be where the eye already goes for a heading rather than in a band
		// of its own above one.
		heading:  func(w int) string { return m.searchField(max(w/2, 8)) },
		subtitle: func() string { return m.styles.Album.Render(searchSubtitle(m.search.results)) },
		count:    len(m.search.results),
		state:    &m.search.cursor,
		empty:    empty,
		row: func(i, w int, selected bool) string {
			return m.trackRow(m.search.results[i], w, selected, 0)
		},
	})
}

// searchSubtitle counts what came back, and says nothing before anything has.
func searchSubtitle(results []player.Track) string {
	if len(results) == 0 {
		return ""
	}
	if len(results) == 1 {
		return "1 result"
	}
	return fmt.Sprintf("%d results", len(results))
}

func (m Model) playlistRow(p player.Playlist, w int, selected bool) string {
	primary := m.styles.RowPrimary
	if selected {
		primary = m.styles.RowSelected
	}
	return m.row(w, selected,
		primary.Render(p.Name),
		m.styles.RowSecondary.Render(p.Owner),
		m.styles.RowTrailing.Render(fmt.Sprintf("%d tracks", p.Tracks)),
	)
}

// trackRow draws one track. A number of 0 omits the ordinal, which is what the
// search results want.
func (m Model) trackRow(t player.Track, w int, selected bool, number int) string {
	primary := m.styles.RowPrimary
	switch {
	case selected:
		primary = m.styles.RowSelected
	case m.ps != nil && m.ps.TrackID == t.ID:
		primary = m.styles.RowPlaying
	}

	title := m.queuedColumn(t) + m.withMark(t, primary.Render(t.Title))
	switch {
	case m.tab == tabQueue && m.ps != nil && m.ps.TrackID == t.ID:
		// On the queue the playing track is marked rather than numbered: it is
		// not waiting its turn, so a place in the running order would be a lie.
		title = m.leadIn(m.styles.Cursor.Render(nowMark)) + title
	case number > 0:
		title = m.leadIn(primary.Render(fmt.Sprintf("%d", number))) + title
	}

	return m.rowWithTempo(w, selected,
		title,
		m.styles.RowSecondary.Render(strings.Join(t.Artists, ", ")),
		m.tempoCell(t),
		m.styles.RowTrailing.Render(formatDuration(t.Duration)),
	)
}

// tempoCell is the beat rate for a list row, or nothing for a track that has
// never been played: a tempo has to be heard to exist.
func (m Model) tempoCell(t player.Track) string {
	if t.Tempo <= 0 {
		return ""
	}
	return m.styles.Quality.Render(fmt.Sprintf("%.0f", t.Tempo))
}

// row lays out the cursor gutter and the three columns of a list row. The two
// right-hand columns are dropped when the pane is too narrow to carry them,
// rather than squeezing every column into uselessness.
func (m Model) row(w int, selected bool, primary, secondary, trailing string) string {
	return m.rowCols(w, selected, primary, secondary, "", trailing)
}

const (
	// rowGutter is the column the cursor stands in, to the left of every row.
	rowGutter = 2

	// shareAbove is the row width at which the artists start taking a share of
	// what the title is not using. It is where the column reaches its cap: below
	// that the row has nothing spare to give away.
	shareAbove = 3 * secondaryCols
)

// rowWidths divides a row's body between the title, the artists, the tempo and
// the duration. The tempo's column is held whether the track has one or not, so
// the durations stay in line down the list instead of stepping in and out.
//
// The second column grows past its usual cap once the row has room to spare: a
// title is rarely long enough to earn twice the artists' room, and on a wide
// terminal the gap it leaves in the middle of the row is the widest thing on the
// screen. The growth is eased in from the cap and stops at an even split, so a
// resize moves the column rather than jumping it.
func rowWidths(body int) (main, second, beat int) {
	second = min(body/3, secondaryCols)

	beat = tempoCols
	main = body - second - beat - trailingCols - 2
	if main < 16 {
		// Too narrow for everything: the tempo goes first, then the artist.
		beat = 0
		main = body - second - trailingCols - 2
	}
	if main < 16 {
		return body - trailingCols - 1, 0, 0
	}

	free := body - beat - trailingCols - 2
	if grown := min(secondaryCols+(body-shareAbove)/2, free/2); grown > second {
		second, main = grown, free-grown
	}
	return main, second, beat
}

// rowSecondaryAt is the column a track row's artists begin at, counted from the
// start of the row. The queue hangs its trace from the same line, so the screen
// reads as two columns rather than four.
func rowSecondaryAt(w int) int {
	main, second, _ := rowWidths(w - rowGutter)
	if second == 0 {
		return w
	}
	return rowGutter + main + 1
}

// rowWithTempo is row with a beat rate between the artist and the duration.
// An empty tempo still holds its column, so the durations stay in line whether
// a track has been heard before or not.
func (m Model) rowWithTempo(w int, selected bool, primary, secondary, tempo, trailing string) string {
	return m.rowCols(w, selected, primary, secondary, tempo, trailing)
}

func (m Model) rowCols(w int, selected bool, primary, secondary, tempo, trailing string) string {
	gutter := "  "
	if selected {
		gutter = m.styles.Cursor.Render(rowCursor) + " "
	}
	if m.rowsAreFlush {
		// Nothing on this list can be pointed at, so the column the cursor
		// would stand in is dead space that pushes every row out of line with
		// its own heading.
		gutter = ""
	}

	body := w - lipgloss.Width(gutter)
	if body < minInfoCols-6 {
		return gutter + fit(primary, max(body, 0))
	}

	main, second, beat := rowWidths(body)

	line := gutter + fit(primary, max(main, 0)) + " "
	if second > 0 {
		line += fit(secondary, second) + " "
	}
	if beat > 0 {
		line += padLeft(tempo, beat)
	}
	return line + padLeft(trailing, trailingCols)
}

// searchField renders the query line, keeping the prompt in the accent.
func (m Model) searchField(w int) string {
	in := m.search.input
	in.SetWidth(max(w-4, 8))
	return in.View()
}

func playlistSubtitle(items []player.Playlist) string {
	if len(items) == 0 {
		return ""
	}
	var total int
	for _, p := range items {
		total += p.Tracks
	}
	return fmt.Sprintf("%d playlists · %d tracks", len(items), total)
}

// formatSpan renders a long duration as hours and minutes, for playlist lengths
// where seconds would be noise.
func formatSpan(d time.Duration) string {
	total := int(d.Round(time.Minute).Minutes())
	if h := total / 60; h > 0 {
		return fmt.Sprintf("%dh %02dm", h, total%60)
	}
	return fmt.Sprintf("%dm", total)
}

// trackCaption is the same facts without their names, for the player screen.
//
// There the values sit right under the title and read as one caption: the album
// on its own line, then the rest on another. Naming them would turn a caption
// into a table, and the player screen is the one place with room to be quiet.
func (m Model) trackCaption(t player.Track, w int) []string {
	s := m.styles
	lines := []string{s.Album.Render(fit(t.Album, w))}

	var rest []string
	for _, f := range trackFacts(t) {
		switch f.key {
		case "album", "length":
			// The album has a line of its own; the length is already under the
			// progress bar, where it is read against the elapsed time.
			continue
		case "track":
			// A bare number in a run of facts could be anything. The queue has
			// a column to name it; here the value has to name itself.
			rest = append(rest, "track "+f.value)
			continue
		}
		rest = append(rest, f.value)
	}
	if len(rest) > 0 {
		lines = append(lines, s.Detail.Render(fit(strings.Join(rest, " · "), w)))
	}
	if t.Popularity != nil {
		lines = append(lines, m.stars(*t.Popularity))
	}
	return lines
}
