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
	for i := from; i < to; i++ {
		lines = append(lines, fit(row(i, w, i == state.cursor), w))
	}

	// A hint that the list runs on past the bottom of the pane.
	if to < count && len(lines) > 0 {
		lines[len(lines)-1] = fit(m.styles.Empty.Render(fmt.Sprintf("  … %d more", count-to+1)), w)
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
		title = fmt.Sprintf("%2d  %s", number, t.Title)
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
