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

	// secondaryCols is the fixed middle column of a list row: the artist, or the
	// playlist's owner. Fixed so the trailing column always lines up.
	secondaryCols = 20

	// trailingCols holds a duration or a count, right aligned.
	trailingCols = 8
)

// browsePane is the right-hand side of the playlists and search tabs: a heading,
// a blank line, then as many rows as fit.
func (m Model) browsePane(l layout, rows int) []string {
	switch m.tab {
	case tabPlaylists:
		return m.playlistPaneView(l, rows)
	case tabSearch:
		return m.searchPaneView(l, rows)
	default:
		return nil
	}
}

// queueBlock is the whole queue screen: the cover and the details of whatever
// the cursor rests on across the top, the list itself across the full width
// below. The list is the point of the screen, so it gets the width; the detail
// panel fills what would otherwise be empty beside the artwork.
func (m Model) queueBlock(l layout, rows int) []string {
	w := l.artWidth + columnGap + l.infoWidth

	top := min(l.artHeight, rows)
	art := alignTop(strings.Split(m.artworkCells(), "\n"), l.artWidth, top)
	detail := stack(m.trackDetail(l.infoWidth), l.infoWidth, top)

	out := make([]string, 0, rows)
	gap := strings.Repeat(" ", columnGap)
	for i := range art {
		out = append(out, art[i]+gap+detail[i])
	}
	if len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}

	// The heading carries the count on the right, where a subtitle line would
	// otherwise cost a row the list could use.
	if len(out) < rows {
		out = append(out, spread(
			m.styles.Title.Render("Queue"),
			m.styles.Album.Render(queueSubtitle(m.queue)),
			w,
		))
	}
	if len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}

	rowsOf := m.queueRows()
	body := max(rows-len(out), 0)
	if len(rowsOf) == 0 && body > 0 {
		out = append(out, fit(m.styles.Empty.Render("Nothing is queued."), w))
	}

	// The playing track heads the list and is not numbered: it is not waiting
	// for its turn, so a place in the running order would be a lie.
	_, playing := m.nowPlayingRow()
	from, to := m.queuePane.cursor.window(len(rowsOf), body)
	bar := m.scrollColumn(body, len(rowsOf), m.queuePane.cursor.top)
	rowWidth := w
	if bar != nil {
		rowWidth = w - scrollCols
	}

	for i := from; i < to; i++ {
		number := i + 1
		if playing {
			number = i
		}
		row := fit(m.queueRow(rowsOf[i], rowWidth, i == m.queuePane.cursor.cursor, number), rowWidth)
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
func (m Model) trackDetail(w int) []string {
	t := m.queuedTrack()
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
	_, hasNow := m.nowPlayingRow()
	for _, f := range trackFacts(*t, hasNow && m.queuePane.cursor.cursor == 0) {
		lines = append(lines, m.fact(f.label, f.value, w))
	}
	if t.Popularity != nil {
		lines = append(lines, m.fact("Popularity", m.stars(*t.Popularity), w))
	}
	return lines
}

// stars renders a rating as five of them, filled in fifths. Spotify's number is
// out of a hundred, which reads as a measurement; a row of stars reads as an
// opinion, which is what it is.
func (m Model) stars(popularity int) string {
	filled := min(max((popularity+starStep-1)/starStep, 1), starCount)
	return m.styles.StarOn.Render(strings.Repeat(starFull, filled)) +
		m.styles.StarOff.Render(strings.Repeat(starEmpty, starCount-filled))
}

// fact is one label-and-value row of the detail panel. The label column is
// fixed so the values line up into a second column of their own.
func (m Model) fact(label, value string, w int) string {
	return fit(m.styles.FactLabel.Render(padRight(label, factLabelCols))+value, w)
}

type trackFact struct{ label, value string }

// trackFacts is what is worth saying about a track, in the order it is worth
// saying it. Anything Spotify left blank is left out rather than shown empty.
func trackFacts(t player.Track, playing bool) []trackFact {
	facts := []trackFact{{"Album", t.Album}}

	if year := releaseYear(t.Released); year != "" {
		// A single or a compilation is worth knowing and costs no room of its
		// own; on a plain album it would be saying what the label already said.
		if t.AlbumType != "" && t.AlbumType != "album" {
			year += " · " + t.AlbumType
		}
		facts = append(facts, trackFact{"Released", year})
	}
	if t.TrackNumber > 0 {
		place := fmt.Sprintf("%d", t.TrackNumber)
		if t.TotalTracks > 0 {
			place = fmt.Sprintf("%d of %d", t.TrackNumber, t.TotalTracks)
		}
		if t.DiscNumber > 1 {
			place += fmt.Sprintf(", disc %d", t.DiscNumber)
		}
		facts = append(facts, trackFact{"Track", place})
	}
	facts = append(facts, trackFact{"Length", formatDuration(t.Duration)})
	if playing {
		facts = append(facts, trackFact{"Status", "playing now"})
	}
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
	// The mark stands in the same columns the track number would, or the titles
	// beside it would sit one indent out.
	return m.row(w, selected,
		m.styles.Cursor.Render(padLeft(nowMark, ordinalCols))+"  "+primary.Render(t.Title),
		m.styles.RowSecondary.Render(strings.Join(t.Artists, ", ")),
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
		return m.listPane(l, rows,
			[]string{m.styles.Title.Render("Playlists"), m.styles.Album.Render(playlistSubtitle(m.playlists.items))},
			len(m.playlists.items), &m.playlists.cursor,
			"No playlists.",
			func(i, w int, selected bool) string { return m.playlistRow(m.playlists.items[i], w, selected) },
		)
	}

	open := *m.playlists.open
	subtitle := fmt.Sprintf("%s · %d tracks", open.Owner, open.Tracks)
	// Spotify does not report a playlist's length; only the mock knows it.
	if open.Duration > 0 {
		subtitle += " · " + formatSpan(open.Duration)
	}

	head := []string{
		m.styles.Title.Render(open.Name),
		m.styles.Album.Render(subtitle),
	}
	return m.listPane(l, rows, head,
		len(m.playlists.tracks), &m.playlists.inner,
		"This playlist is empty.",
		func(i, w int, selected bool) string {
			return m.trackRow(m.playlists.tracks[i], w, selected, i+1)
		},
	)
}

func (m Model) searchPaneView(l layout, rows int) []string {
	head := []string{m.searchField(l.infoWidth), ""}

	empty := "Type to search the catalogue."
	if strings.TrimSpace(m.search.input.Value()) != "" {
		empty = "Nothing matched."
	}
	return m.listPane(l, rows, head,
		len(m.search.results), &m.search.cursor, empty,
		func(i, w int, selected bool) string {
			return m.trackRow(m.search.results[i], w, selected, 0)
		},
	)
}

// listPane assembles a heading and a scrolling list into exactly artHeight rows,
// so the pane and the artwork beside it always end level.
func (m Model) listPane(l layout, height int, head []string, count int, state *listState, empty string, row func(i, w int, selected bool) string) []string {
	w := l.infoWidth
	lines := make([]string, 0, height)
	for _, h := range head {
		lines = append(lines, fit(h, w))
	}
	lines = append(lines, "")

	rows := max(height-len(lines), 0)
	if count == 0 {
		lines = append(lines, fit(m.styles.Empty.Render(empty), w))
	}

	from, to := state.window(count, rows)
	bar := m.scrollColumn(rows, count, state.top)
	rowWidth := w
	if bar != nil {
		rowWidth = w - scrollCols
	}

	for i := from; i < to; i++ {
		line := fit(row(i, rowWidth, i == state.cursor), rowWidth)
		if bar != nil {
			line += " " + bar[i-from]
		}
		lines = append(lines, line)
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines[:height]
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

	title := t.Title
	if number > 0 {
		title = padLeft(fmt.Sprintf("%d", number), ordinalCols) + "  " + t.Title
	}
	if m.ps != nil && m.ps.TrackID == t.ID {
		title = nowMark + " " + title
	}

	return m.row(w, selected,
		primary.Render(title),
		m.styles.RowSecondary.Render(strings.Join(t.Artists, ", ")),
		m.styles.RowTrailing.Render(formatDuration(t.Duration)),
	)
}

// row lays out the cursor gutter and the three columns of a list row. The two
// right-hand columns are dropped when the pane is too narrow to carry them,
// rather than squeezing every column into uselessness.
func (m Model) row(w int, selected bool, primary, secondary, trailing string) string {
	gutter := "  "
	if selected {
		gutter = m.styles.Cursor.Render(rowCursor) + " "
	}

	body := w - lipgloss.Width(gutter)
	if body < minInfoCols-6 {
		return gutter + fit(primary, max(body, 0))
	}

	second := min(secondaryCols, body/3)
	main := body - second - trailingCols - 2
	if main < 16 {
		main, second = body-trailingCols-1, 0
	}

	line := gutter + fit(primary, max(main, 0)) + " "
	if second > 0 {
		line += fit(secondary, second) + " "
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
