package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/style"
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

	// likedMark stands beside the saved tracks in the library, which are a
	// collection rather than a playlist.
	//
	// The variation selector asks for the text form of the heart. Without it a
	// terminal is free to draw the emoji instead, which is two cells wide where
	// one was measured, and the name beside it lands a column out of line with
	// every other name in the list.
	likedMark = "♥\ufe0e"

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
	//
	// Wide enough for the unit as well as the number. A bare 103 sitting next to
	// a 4:10 reads as a second time, or as nothing at all — nobody asked what it
	// was because there was nothing there to ask about.
	tempoCols = 7
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

	// Which of the two blocks above the list are open. Everywhere but the queue
	// that is both of them; there the key walks the four ways of arranging
	// them, and the last of them is no band at all. See queueRoom.
	room := m.bandRoom()

	top := min(m.listBandRows(l), rows)
	var art []string
	if room.showsNow() {
		art = m.place(m.artBlock(false), l.artWidth, top)
	}

	// The picture decides how far the panel beside it may reach: the box the
	// layout gives the artwork is a row or two taller than the cover drawn in
	// it, and a fact hanging below the picture's foot reads as a mistake.
	foot := min(l.artRows, top)

	// The right-hand half of the band, hanging from the same column the artists
	// below start at, so the screen reads as two halves. Nothing under it moves
	// when it appears: the list starts where the artwork ends either way.
	//
	// The queue puts the trace there; the library, which has no trace, puts
	// what is playing there instead — otherwise half of that band is a blank
	// rectangle on the widest screens, which is where a player should look its
	// best.
	detailWidth := m.bandDetailWidth(l)
	var right []string
	switch {
	case m.scopeVisible() && room.showsTrace():
		right = m.place(m.traceBlock(), queueScopeWidth(l), top)
	}

	// With the player folded away the picture has the band to itself, which is
	// what a picture wants: the strip beside a cover is a glance, and the same
	// picture across the whole width is something to watch.
	if right != nil && !room.showsNow() {
		right = m.place(m.traceBlock(), w, top)
	}

	var detail []string
	if room.showsNow() {
		detail = stackLift(opts.detail(detailWidth, foot), detailWidth, foot, m.detailLift())
		for len(detail) < top {
			detail = append(detail, strings.Repeat(" ", detailWidth))
		}
		detail = m.outline(detail, detailWidth, "detail")
	}

	out := make([]string, 0, rows)
	gap := strings.Repeat(" ", columnGap)
	band := max(len(art), len(right))
	for i := range band {
		var row string
		if art != nil {
			row = art[i] + gap + detail[i]
		}
		if right != nil {
			if row != "" {
				row += gap
			}
			row += right[i]
		}
		out = append(out, row)
	}
	// The air under the band, where there is a band. Folded away, the heading
	// stands at the top of the screen rather than a row down from nothing.
	if band > 0 && len(out) < rows {
		out = append(out, strings.Repeat(" ", w))
	}

	// The room the field a list is searched in stands in. It is drawn over the
	// screen once everything is laid out — see finder.go — and what it needs from
	// here is the rows, so that the table steps down for it rather than being
	// covered by it.
	for range m.finderTakes() {
		if len(out) < rows {
			out = append(out, strings.Repeat(" ", w))
		}
	}

	// The heading carries the count on the right, where a subtitle line would
	// otherwise cost a row the list could use. A page on its way turns the
	// spinner beside it: the rows already on screen are not all of them, and
	// nothing else on the screen would say so.
	if len(out) < rows {
		subtitle := opts.subtitle()
		if m.listLoading() && opts.count > 0 {
			subtitle += " " + m.spinner.View()
		}

		// Set against the heading, or at the left where there is no heading to
		// set it against. The queue gave its name up — the tab it is under is
		// called queue — and what was left was a count alone at the far end of an
		// empty row, which reads as something dropped in a corner rather than as
		// the line that says what the table below holds.
		head := opts.heading(w)
		if lipgloss.Width(head) == 0 {
			out = append(out, fit(subtitle, w))
		} else {
			out = append(out, spread(head, subtitle, w))
		}
	}
	// The blank under the heading names the columns, where the list has any and
	// has anything in them. Over an empty list it would be a header for nothing.
	named := opts.columns != nil && opts.count > 0 && m.listBodyRows(rows, m.listBandRows(l)) > 0
	if len(out) < rows {
		head := strings.Repeat(" ", w)
		if named {
			head = fit(opts.columns(queueRowWidth(l)), w)
		}
		out = append(out, head)
	}

	// And a line under them, which is what tells a heading from a row: without it
	// the names are the first entry in the list, set in a lighter grey.
	//
	// It stands in the row that was held over the heading for a field to search
	// the list, which the field no longer wants: it is given rows of its own above
	// the heading while it is open, rather than a row kept empty against the day.
	if len(out) < rows {
		rule := strings.Repeat(" ", w)
		if named {
			rule = fit(m.columnRule(queueRowWidth(l)), w)
		}
		out = append(out, rule)
	}

	// What is left is the list. It is asked for rather than counted off what has
	// been drawn, because the page keys have to move by the same number and
	// cannot see this function.
	body := m.listBodyRows(rows, m.listBandRows(l))

	if opts.count == 0 && body > 0 {
		// A list that has not arrived is not an empty list, and saying so is
		// the difference between a slow answer and a wrong one. The pages are
		// small — the search's limit is ten, and Spotify refuses eleven — so
		// this is on screen often enough to be worth being exact about.
		empty := m.styles.Empty.Render(opts.empty)
		if m.listLoading() {
			empty = m.spinner.View() + " " + m.styles.Empty.Render(opts.waiting)
		}
		out = append(out, fit(empty, w))
	}

	from, to := opts.state.window(opts.count, body)
	bar := m.scrollColumn(body, opts.count, opts.state.top)
	rowWidth := queueRowWidth(l)

	listFrom := len(out)
	for i := from; i < to; i++ {
		row := fit(opts.row(i, rowWidth, i == opts.state.cursor), rowWidth)
		if bar != nil {
			row += " " + bar[i-from]
		}
		out = append(out, row)
	}
	if lines := m.outline(out[listFrom:], w, "list"); len(lines) == len(out)-listFrom {
		copy(out[listFrom:], lines)
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

	// columns names the cells of a row, drawn in the blank under the heading.
	// Nil where a list has nothing worth naming.
	columns func(w int) string

	count int
	state *listState

	// empty is what to say when the list is empty, and waiting what to say
	// while it might not be — a list still arriving looks exactly like one with
	// nothing in it, and only one of the two is worth waiting for.
	empty   string
	waiting string

	row func(i, w int, selected bool) string
}

// queueBlock is the queue screen. The playing track heads the list and is not
// numbered: it is not waiting for its turn, so a place in the running order
// would be a lie.
func (m Model) queueBlock(l layout, rows int) []string {
	rowsOf := m.queueRows()
	_, playing := m.nowPlayingRow()

	return m.listBlock(l, rows, listScreen{
		detail: m.trackDetail,
		// No heading. The tab it is under is called queue, in the row across the
		// top of the screen, and a list titled after the tab it is the only thing
		// on is a label for something nobody was in any doubt about.
		heading:  func(int) string { return "" },
		subtitle: func() string { return m.styles.Album.Render(queueSubtitle(m.queue)) },
		columns:  func(w int) string { return m.trackColumns(w, true) },
		count:    len(rowsOf),
		state:    &m.queuePane.cursor,
		empty:    "Nothing is queued.",
		waiting:  "Reading the queue…",
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
// bandRoom is what the band over a list is showing. The queue is the one screen
// that can fold half of it away; everywhere else both halves stand.
func (m Model) bandRoom() queueRoom {
	if m.tab == tabQueue && m.open() == nil && !m.devices.open {
		return m.queuePane.room
	}
	return queueRoomBoth
}

// bandDetailWidth is how wide the panel beside the cover is drawn. Narrower on
// the queue, where the trace stands beside it.
//
// One function, read by the drawing and by the pointer: a bar the eye can see
// and the pointer cannot find is worse than no bar, and two copies of this
// arithmetic is how that happens.
func (m Model) bandDetailWidth(l layout) int {
	if m.scopeVisible() && m.bandRoom().showsTrace() {
		return queueDetailWidth(l)
	}
	return l.infoWidth
}

func (m Model) trackDetail(w, rows int) []string {
	lines, _ := m.trackDetailAt(w, rows)
	return lines
}

// trackDetailAt is that panel and which of its rows carries the playhead, or
// -1 where it carries none. The pointer needs the row and the drawing needs the
// lines, and they are worked out once so they cannot disagree.
func (m Model) trackDetailAt(w, rows int) ([]string, int) {
	t := m.cursorTrack()
	if t == nil {
		return nil, -1
	}

	// In the colour of the cover it is standing beside. Everywhere else on the
	// screen the accent is the sounding record's — see tone.go — and this is the
	// one block that is explicitly about another one: the frame round it says so,
	// and the colour inside it says which.
	s := m.coverStyles

	title := s.Title.Render(t.Title)
	if t.Explicit {
		title += "  " + s.FactLabel.Render(explicitMark)
	}

	// The rating goes under the name rather than in the facts, and stands
	// alone: a row of stars says what it is, and "Popularity" beside them is a
	// label on a picture. It sits with the name because it is the one fact about
	// a track that is an opinion rather than a measurement — and under it,
	// because a title is what the eye should land on first.
	// A row of air over the name, so the panel does not begin hard against the
	// top of the picture beside it.
	lines := []string{
		"",
		title,
		s.Artist.Render(strings.Join(t.Artists, ", ")),
	}
	if t.Popularity != nil {
		lines = append(lines, m.starsIn(s, *t.Popularity))
	}

	// A row of air under the name, but only where a playhead follows it: it is
	// what keeps the bar off the rating, and with no bar it is a gap.
	if m.ps != nil && m.ps.TrackID == t.ID {
		lines = append(lines, "")
	}

	// The playhead and the clock either side of it, for the one track they can
	// belong to. Their rows are kept whether this is that track or not: without
	// that the panel would shift every time the cursor passed the track
	// playing, and a bar is not worth making the facts beside it move. They go
	// in only where there is room for them at all, because those facts are the
	// point of the panel.
	facts := trackFacts(*t)
	if rows < len(facts)+6 {
		for _, f := range facts {
			lines = append(lines, fit(f.value, w))
		}
		return lines, -1
	}

	// The playhead, and only for the one track it can belong to.
	//
	// The rows were held open for every other track so that nothing moved as the
	// cursor passed the one playing, and what that bought was two empty rows
	// through the middle of every other panel — a hole with a duration hanging
	// off it, saying a thing the row in the list below already says. A block
	// that closes up is worth more than a block that never moves.
	var bar, times string
	if m.ps != nil && m.ps.TrackID == t.ID {
		bar = m.progressLine(w)
		times = spread(
			s.Time.Render(formatDuration(m.elapsed())),
			s.Time.Render(formatDuration(m.ps.Duration)),
			w,
		)
	}

	// The album on its own row and everything else on one under it, the way the
	// player screen sets the same facts.
	//
	// Three rows of one short value each is what put a hole in the middle of
	// this block: the playhead has to sit on the picture's middle, the facts sit
	// under it, and what was left between them was rows of nothing. Set as a
	// caption they come to two, and the block closes up — the air goes outside
	// it, where air belongs, rather than through the middle of it.
	var under []string
	if times != "" {
		under = append(under, times)
	}
	var rest []string
	for _, f := range facts {
		if f.key == "album" {
			under = append(under, fit(s.Album.Render(f.value), w))
			continue
		}
		rest = append(rest, f.value)
	}
	if len(rest) > 0 {
		under = append(under, fit(s.Detail.Render(strings.Join(rest, " · ")), w))
	}

	// The playhead is the middle row of the panel, and the panel is centred in
	// the band — so the bar lands on the band's own middle, which is where the
	// picture beside it is drawn from. Two things that answer the same track
	// should stand on one line; measured before this, the bar sat a row above
	// the picture's middle and the eye caught it.
	//
	// The air goes above rather than below: under the name it reads as room,
	// and between the clock and the facts it reads as a gap.
	//
	// Only where there is a playhead to place. Without one the block is what it
	// is and the band centres it, which is where a block with nothing to line up
	// against belongs.
	if bar == "" {
		return append(lines, under...), -1
	}
	for len(lines) < len(under) {
		lines = append(lines, "")
	}
	at := len(lines)
	return append(append(lines, bar), under...), at
}

// detailLift is how far above the middle of its band the panel beside the cover
// sits.
//
// A row, unless there is a playhead in it. A block of words centred exactly
// beside a picture reads as sitting low, because a picture's weight is even and
// a block of words is heavier at the top. With a playhead the middle is not a
// choice: the bar has to land on the picture's own middle, which is what puts
// the two on one line.
func (m Model) detailLift() int {
	if t := m.cursorTrack(); t != nil && m.ps != nil && m.ps.TrackID == t.ID {
		return 0
	}
	return 1
}

// starsFor is how many of the five a rating earns. Nothing scores none: a track
// nobody plays is still a track.
func starsFor(popularity int) int {
	return min(max((popularity+starStep-1)/starStep, 1), starCount)
}

// stars renders a rating as five of them, filled in fifths. Spotify's number is
// out of a hundred, which reads as a measurement; a row of stars reads as an
// opinion, which is what it is.
func (m Model) stars(popularity int) string { return m.starsIn(m.styles, popularity) }

// starsIn is the same in a given set, so the panel beside a cover can wear that
// cover's colour rather than the sounding record's.
func (m Model) starsIn(s style.Styles, popularity int) string {
	filled := starsFor(popularity)
	return s.StarOn.Render(strings.Repeat(starFull, filled)) +
		s.StarOff.Render(strings.Repeat(starEmpty, starCount-filled))
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
//
// Only where the row is too narrow for the rating itself. The flag was how a
// rating reached every row while the number was on the detail panel alone; on a
// row wide enough to carry the stars it is the same thing said twice, and the
// stars say it better — three of five and four of five are both a flag.
func (m Model) withMark(t player.Track, title string, w int) string {
	if hot(t) && !m.showsStars(w) {
		title += " " + hotMark
	}
	return title
}

// showsStars reports whether a row of the given width has earned the column the
// rating is drawn in.
func (m Model) showsStars(w int) bool {
	gutter := rowGutter
	if m.rowsAreFlush {
		gutter = 0
	}
	return rowWidths(w-gutter).stars > 0
}

// queuedColumn is the narrow band between the ordinal and the title, carrying
// the mark for a track already waiting in the queue. A column of its own rather
// than a flag after the title: what is worth reading here is how much of a list
// has been picked over, and marks scattered along the titles cannot be counted
// at a glance the way a column can.
func (m Model) queuedColumn(t player.Track) string {
	if m.rowsAreTheQueue {
		// Nothing in front of the title. Everything here is in the queue by
		// definition, so a column of dots saying so is a column that says
		// nothing — and the heart it was spent on instead is a column of the
		// table now, where it is named and where every other list carries it.
		// A mark and a column both would be the same fact twice in one row.
		return ""
	}
	if m.tab == tabQueue {
		// Same again, and nothing to put in its place: the queue screen is a
		// list to work on, and its own marks are already in this column.
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

// blankQueuedColumn is that column with nothing in it, which is what the row
// naming the columns needs: the width the marks take, without a mark.
func (m Model) blankQueuedColumn() string {
	return m.queuedColumn(player.Track{})
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

// trackFact is one thing known about a track: which fact it is, and what it
// says. Both screens draw the same facts from here, so they are gathered once
// and rendered twice.
//
// Neither of them names the facts any more. A column of labels beside three
// short values is a table with one row in every column, and every one of them
// says what it is without being told: an album is an album, a year is a year,
// and a tempo carries its unit.
type trackFact struct {
	key   string
	value string
}

// trackFacts is what is worth saying about a track, in the order it is worth
// saying it. Anything Spotify left blank is left out rather than shown empty.
func trackFacts(t player.Track) []trackFact {
	facts := []trackFact{{"album", t.Album}}

	if year := releaseYear(t.Released); year != "" {
		// A single or a compilation is worth knowing and costs no room of its
		// own; on a plain album it would be saying what the label already said.
		if t.AlbumType != "" && t.AlbumType != "album" {
			year += " · " + t.AlbumType
		}
		facts = append(facts, trackFact{"released", year})
	}
	// The row is always here, even with nothing to put in it. A tempo takes a
	// dozen seconds to measure, so it arrives mid-track — and a row appearing
	// then would push everything under it down while it is being read.
	tempo := unknownValue
	if t.Tempo > 0 {
		tempo = fmt.Sprintf("%.0f bpm", t.Tempo)
	}
	facts = append(facts, trackFact{"tempo", tempo})
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
	return m.drawRow(w, selected, rowCells{
		primary:   m.leadIn(m.styles.Cursor.Render(nowMark)) + m.withMark(t, m.lit(primary, t.Title), w),
		secondary: m.lit(m.styles.RowSecondary, strings.Join(t.Artists, ", ")),
		stars:     m.starsCell(t),
		liked:     m.likedCell(t),
		tempo:     m.tempoCell(t),
		trailing:  m.styles.RowTrailing.Render(formatDuration(t.Duration)),
	})
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

func (m Model) libraryPaneView(l layout, rows int) []string {
	// The tab itself is a wall of covers. What is opened from it is a list, and
	// comes back through here as one — see openPageView.
	return m.libraryPaneGrid(l, rows)
}

// libraryKinds is the tab bar of the library's own kinds: the four lists the tab
// holds, the one on screen lit and underlined.
//
// Drawn the way the tabs across the top of the screen are drawn, because it is
// the same thing one level down — the underline is the whole of the chrome, and
// somebody who has learned the top row has learned this one. It stands at the
// left, where a heading used to say "Library" over a tab already called library.
//
// It costs no rows: the labels take the heading's row and the underline the
// blank under it, which was there to hold the heading clear of the wall.
func (m Model) libraryKinds() (labels, rule string) {
	var names, marks strings.Builder
	for i, label := range m.kindLabels() {
		if i > 0 {
			names.WriteString(kindGap)
			marks.WriteString(kindGap)
		}

		if libraryOrder[i] == m.library.kind {
			names.WriteString(m.styles.TabActive.Render(label))
			marks.WriteString(m.styles.TabRule.Render(strings.Repeat(meterFull, lipgloss.Width(label))))
			continue
		}
		names.WriteString(m.styles.TabIdle.Render(label))
		marks.WriteString(strings.Repeat(" ", lipgloss.Width(label)))
	}
	return names.String(), marks.String()
}

// kindGap is the air between two of those labels, and what a click steps by.
const kindGap = "   "

// kindLabels is what each of them says: the name of the kind, and how many of
// it have been read where anything has.
//
// The one list of them, so that what is drawn and what a click is measured
// against cannot be two different lists. See kindSpans.
func (m Model) kindLabels() []string {
	out := make([]string, len(libraryOrder))
	for i, kind := range libraryOrder {
		label := kind.String()
		if n := m.library.countOf(kind); n > 0 {
			count := fmt.Sprintf("%d", n)
			if m.library.pages[kind].more {
				// What has been read, not what exists.
				count += "+"
			}
			label += " " + count
		}
		out[i] = label
	}
	return out
}

// kindSpans is where those labels sit on the row, for a click to be answered.
// The bar is set flush left, against the margin every block on the screen keeps.
func (m Model) kindSpans() []span {
	return labelSpans(m.kindLabels(), len(kindGap), leftMargin)
}

// libraryOrder is the order the kinds are drawn and walked in.
var libraryOrder = []libraryKind{libraryPlaylists, libraryAlbums, libraryArtists, libraryRecent}

// libraryDetail is the panel beside the cover: whichever kind is on screen, it
// describes the row under the cursor.
func (m Model) libraryDetail(w, rows int) []string {
	switch {
	case m.library.atAlbum() != nil:
		return m.albumDetail(m.library.atAlbum(), w)
	case m.library.atArtist() != nil:
		return m.artistDetail(m.library.atArtist(), w)
	default:
		return m.playlistDetailOf(m.library.selected(), w, rows)
	}
}

// openEmpty is what stands where the tracks would be when there are none.
//
// A list somebody else owns is refused outright to an application registered
// since Spotify's 2024 clampdown, and what comes back is empty rather than
// angry. Saying "nothing here" about a playlist with three hundred tracks in it
// is the program telling a lie it could have avoided.
func openEmpty(page openPage) string {
	if page.refused {
		return "Spotify will not hand this list to the application spindle is using. " +
			"The one it ships with can read it — see the settings screen."
	}
	return "Nothing here."
}

// openPageView is whatever has been opened: a playlist, an album, or an
// artist's records. The same composition as every other list, because it is the
// same act — the heading says what was opened and the panel describes the row
// under the cursor.
func (m Model) openPageView(l layout, rows int) []string {
	page := *m.open()

	screen := listScreen{
		detail:   m.trackDetail,
		heading:  func(w int) string { return fit(m.styles.Title.Render(page.name), max(w/2, 1)) },
		subtitle: func() string { return m.styles.Album.Render(m.openSubtitle(page)) },
		count:    page.count(),
		state:    &m.stack[len(m.stack)-1].cursor,
		columns:  func(w int) string { return m.trackColumns(w, true) },
		empty:    openEmpty(page),
		waiting:  "Reading it…",
		row: func(i, w int, selected bool) string {
			// An album's tracks are numbered by the record; a playlist's by
			// where they sit in it. The two happen to be written the same way.
			return m.trackRow(page.tracks[i], w, selected, i+1)
		},
	}

	if page.holdsAlbums() {
		screen.columns = func(w int) string { return m.albumColumns(w, "") }
		// The panel on an artist page is about the artist, where anything is
		// known about them, and about the record under the cursor otherwise.
		screen.detail = m.notesPanel
		screen.empty, screen.waiting = "No records here.", "Reading their records…"
		screen.row = func(i, w int, selected bool) string {
			return m.albumRow("", page.albums[i], w, selected)
		}
	}
	return m.listBlock(l, rows, screen)
}

// openAlbumDetail is the panel on an artist page, describing the record under
// the cursor.
func (m Model) openAlbumDetail(w, rows int) []string {
	return m.albumDetail(m.cursorAlbum(), w)
}

// openSubtitle is what the page says about itself beside its name: what it is
// made of, and how much of it there is.
func (m Model) openSubtitle(page openPage) string {
	if page.holdsAlbums() {
		n := len(page.albums)
		switch {
		case n == 0:
			return page.subtitle
		case page.subtitle == "":
			return fmt.Sprintf("%d releases", n)
		default:
			return fmt.Sprintf("%s · %d releases", page.subtitle, n)
		}
	}

	// The running time is only added up once every track is in: a total that
	// grew as the reader scrolled would read as a mistake rather than as
	// progress.
	out := fmt.Sprintf("%s · %d tracks", page.subtitle, len(page.tracks))
	if page.subtitle == "" {
		out = fmt.Sprintf("%d tracks", len(page.tracks))
	}
	if page.pages.more || len(page.tracks) == 0 {
		return out
	}

	var total time.Duration
	for _, t := range page.tracks {
		total += t.Duration
	}
	return out + " · " + formatSpan(total)
}

// playlistDetailOf is the same panel for a playlist from anywhere, so a search
// result reads exactly as the library's own row does.
func (m Model) playlistDetailOf(p *player.Playlist, w, rows int) []string {
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
	// A count of nothing is not a count: the saved tracks are read a page at a
	// time and their number is not known until the last page has come in, and
	// Spotify has been known to leave a playlist's total out altogether.
	if p.Tracks > 0 {
		lines = append(lines, m.fact("Tracks", fmt.Sprintf("%d", p.Tracks), w))
	}
	if p.Duration > 0 {
		lines = append(lines, m.fact("Length", formatSpan(p.Duration), w))
	}
	return lines
}

// searchFieldWidth is how wide the field the catalogue is searched from is
// drawn, wherever on the screen it stands. Named so that a press can be measured
// against the same number.
func searchFieldWidth(l layout) int { return max(queueBlockWidth(l)/3, 8) }

func (m Model) searchPaneView(l layout, rows int) []string {
	empty := "Type to search the catalogue."
	if strings.TrimSpace(m.search.input.Value()) != "" {
		empty = "Nothing matched."
	}

	found := m.search.current()

	// With nothing found there is nothing for the panel to describe and no
	// cover to show, so the band they live in is not reserved: the field goes
	// to the top of the screen, where a search box belongs, rather than sitting
	// halfway down behind a wall of blank rows.
	if found.count() == 0 {
		w := queueBlockWidth(l)

		// A query still in flight looks exactly like one that matched nothing,
		// and the difference is worth a line: Spotify answers ten results at a
		// time, so this screen is waiting more often than any other.
		line := m.styles.Empty.Render(empty)
		if m.listLoading() {
			line = m.spinner.View() + " " + m.styles.Empty.Render("Asking Spotify…")
		}

		out := []string{
			fit(m.searchField(searchFieldWidth(l)), w),
			strings.Repeat(" ", w),
			fit(line, w),
		}
		for len(out) < rows {
			out = append(out, strings.Repeat(" ", w))
		}
		return out[:rows]
	}
	return m.listBlock(l, rows, listScreen{
		detail: m.searchDetail,
		// The field is the heading: it is what the screen is about, and it has
		// to be where the eye already goes for a heading rather than in a band
		// of its own above one.
		heading:  func(int) string { return m.searchField(searchFieldWidth(l)) },
		subtitle: m.searchKinds,
		columns:  m.searchColumns,
		count:    found.count(),
		state:    &found.cursor,
		empty:    empty,
		waiting:  "Asking Spotify…",
		row:      m.searchRow,
	})
}

// searchKinds is what else the query matched, set against the field.
//
// It is the whole argument for showing one kind at a time: the counts say that
// three artists and eighteen albums are there, which is what a screen of mixed
// sections would otherwise have to spend rows saying.
func (m Model) searchKinds() string {
	if m.search.current().count() == 0 && len(m.search.found) == 0 {
		return ""
	}

	var parts []string
	for _, kind := range player.SearchKinds {
		found := m.search.of(kind)
		if found.count() == 0 {
			continue
		}

		count := fmt.Sprintf("%d", found.count())
		if found.pages.more {
			// What has been read, not what exists: Spotify's totals are not
			// worth carrying through three layers for a heading.
			count += "+"
		}

		style := m.styles.Album
		if kind == m.search.kind {
			style = m.styles.Title
		}
		parts = append(parts, style.Render(kind.String()+" "+count))
	}
	return strings.Join(parts, m.styles.Album.Render(" · "))
}

// searchRow draws one hit, whichever kind is on screen.
func (m Model) searchRow(i, w int, selected bool) string {
	found := m.search.current()
	switch m.search.kind {
	case player.SearchAlbums:
		return m.albumRow("", found.albums[i], w, selected)
	case player.SearchArtists:
		return m.artistRow("", found.artists[i], w, selected)
	case player.SearchPlaylists:
		return m.playlistRow(found.playlists[i], w, selected)
	default:
		return m.trackRow(found.tracks[i], w, selected, 0)
	}
}

// searchColumns names the columns of whichever kind of result is on screen.
func (m Model) searchColumns(w int) string {
	switch m.search.kind {
	case player.SearchAlbums:
		return m.albumColumns(w, "")
	case player.SearchArtists:
		return m.artistColumns(w, "")
	case player.SearchPlaylists:
		return m.playlistColumns(w)
	default:
		// Search results carry no ordinal: what a track's place in a list of
		// answers is worth is nothing.
		return m.trackColumns(w, false)
	}
}

// albumRow is a record: its name, who made it, and how many tracks it holds.
func (m Model) albumRow(lead string, a player.Album, w int, selected bool) string {
	primary := m.styles.RowPrimary
	if selected {
		primary = m.styles.RowSelected
	}

	year := releaseYear(a.Released)
	if a.AlbumType != "" && a.AlbumType != "album" {
		year = strings.TrimSpace(year + " " + a.AlbumType)
	}
	return m.row(w, selected,
		lead+m.lit(primary, a.Name),
		m.lit(m.styles.RowSecondary, strings.Join(a.Artists, ", ")),
		m.styles.RowTrailing.Render(year),
	)
}

// artistRow is a name and how many people follow it, which is the only measure
// of an artist that arrives with the list.
func (m Model) artistRow(lead string, a player.Artist, w int, selected bool) string {
	primary := m.styles.RowPrimary
	if selected {
		primary = m.styles.RowSelected
	}
	// Genres and a follower count are what an artist row is made of, and
	// neither is always sent: measured against a live account, the followed
	// list answers with a name and a picture and nothing else. The columns stay
	// empty rather than saying nought followers, which is a number and not an
	// absence.
	followers := ""
	if a.Followers > 0 {
		followers = m.styles.RowTrailing.Render(formatCount(a.Followers))
	}
	return m.row(w, selected,
		lead+m.lit(primary, a.Name),
		m.lit(m.styles.RowSecondary, strings.Join(a.Genres, ", ")),
		followers,
	)
}

// searchDetail is the panel beside the cover: whichever kind is on screen, it
// describes the row under the cursor.
func (m Model) searchDetail(w, rows int) []string {
	found := m.search.current()
	switch m.search.kind {
	case player.SearchAlbums:
		return m.albumDetail(atAlbum(found.albums, found.cursor.cursor), w)
	case player.SearchArtists:
		return m.artistDetail(atArtist(found.artists, found.cursor.cursor), w)
	case player.SearchPlaylists:
		return m.playlistDetailOf(atPlaylist(found.playlists, found.cursor.cursor), w, rows)
	default:
		return m.trackDetail(w, rows)
	}
}

func (m Model) albumDetail(a *player.Album, w int) []string {
	if a == nil {
		return nil
	}
	lines := []string{
		m.styles.Title.Render(a.Name),
		m.styles.Artist.Render(strings.Join(a.Artists, ", ")),
		"",
	}
	if year := releaseYear(a.Released); year != "" {
		lines = append(lines, m.fact("Released", year, w))
	}
	if a.AlbumType != "" {
		lines = append(lines, m.fact("Kind", a.AlbumType, w))
	}
	if a.Tracks > 0 {
		lines = append(lines, m.fact("Tracks", fmt.Sprintf("%d", a.Tracks), w))
	}
	return lines
}

func (m Model) artistDetail(a *player.Artist, w int) []string {
	if a == nil {
		return nil
	}
	lines := []string{
		m.styles.Title.Render(a.Name),
		"",
	}
	if len(a.Genres) > 0 {
		lines = append(lines, m.fact("Genres", strings.Join(a.Genres, ", "), w))
	}
	if a.Followers > 0 {
		lines = append(lines, m.fact("Followers", formatCount(a.Followers), w))
	}
	return lines
}

// formatCount is a follower count a person can read at a glance rather than
// count the digits of.
func formatCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (m Model) playlistRow(p player.Playlist, w int, selected bool) string {
	primary := m.styles.RowPrimary
	if selected {
		primary = m.styles.RowSelected
	}

	// The saved tracks carry a heart where the others carry nothing: the row is
	// a different kind of thing from the playlists under it, and one glyph says
	// so without a heading and without a row of its own. The column is held
	// open on the rows without one, the way the queue holds its marks, so every
	// name in the list starts in the same place.
	name := m.libraryMark(p) + m.lit(primary, p.Name)

	// A count of nothing is not a count. The saved tracks arrive a page at a
	// time and their number is not known until the last of them has, so the
	// column stays empty rather than saying zero.
	//
	// The number alone: the column is named now, and "30 tracks" under a heading
	// reading tracks is the word twice — which is what it was, except that the
	// column is too narrow for it and it came out as "30 trac…".
	count := ""
	if p.Tracks > 0 {
		count = m.styles.RowTrailing.Render(fmt.Sprintf("%d", p.Tracks))
	}

	return m.row(w, selected, name, m.lit(m.styles.RowSecondary, p.Owner), count)
}

// libraryMark is the column in front of a library row, which says what kind of
// thing the row is. Only the saved tracks have anything to say there; the rest
// keep the column blank rather than closing it, so the names stay in line.
func (m Model) libraryMark(p player.Playlist) string {
	if isLiked(p.ID) {
		return m.styles.Cursor.Render(likedMark) + " "
	}
	return blankMark
}

// blankMark is that column held open. Every library row carries it, whichever
// kind is on screen, so turning between them moves no name sideways.
const blankMark = "  "

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

	title := m.queuedColumn(t) + m.withMark(t, m.lit(primary, t.Title), w)
	switch {
	case m.tab == tabQueue && m.ps != nil && m.ps.TrackID == t.ID:
		// On the queue the playing track is marked rather than numbered: it is
		// not waiting its turn, so a place in the running order would be a lie.
		title = m.leadIn(m.styles.Cursor.Render(nowMark)) + title
	case number > 0:
		title = m.leadIn(primary.Render(fmt.Sprintf("%d", number))) + title
	}

	return m.drawRow(w, selected, rowCells{
		primary:   title,
		secondary: m.lit(m.styles.RowSecondary, strings.Join(t.Artists, ", ")),
		album:     m.lit(m.styles.RowTrailing, t.Album),
		stars:     m.starsCell(t),
		liked:     m.likedCell(t),
		tempo:     m.tempoCell(t),
		trailing:  m.styles.RowTrailing.Render(formatDuration(t.Duration)),
	})
}

// starsCell is a track's rating for a list row, and nothing at all for a track
// Spotify sent no rating for: five empty stars would be a rating of none, which
// is a different thing from not knowing.
//
// The filled ones only. Beside the cover, where one rating stands alone, the
// empty stars are the scale it is out of and are worth their room; down a column
// of thirty rows they are a wall of grey that says the same thing on every line.
// What a column wants is the length, which is the whole of the reading — so the
// stars start on one line and run as far as the rating goes, like any other bar.
func (m Model) starsCell(t player.Track) string {
	if t.Popularity == nil {
		return ""
	}
	return m.styles.StarOn.Render(strings.Repeat(starFull, starsFor(*t.Popularity)))
}

// likedCell is the heart on the rows that are saved.
//
// Only where the saved tracks are known — see libraryPane.saved, which can only
// answer for what has been read of them. Nothing is drawn for the rest, because
// a blank says "not in the part I have read" and a mark on every row nobody
// checked would say something stronger and wrong.
func (m Model) likedCell(t player.Track) string {
	if !m.library.saved(t.ID) {
		return ""
	}
	return m.styles.Queued.Render(likedMark)
}

// rowsCarryTheSplit reports whether something in the band is hung from the
// column a row's artists begin at.
//
// The queue's trace is: it starts on that column so the screen reads as two
// halves rather than four columns, and where it is up the title's width is not
// the row's business alone — it keeps whatever the division gives it. Where it is
// not, the title takes a ceiling and the row keeps its columns together.
//
// The same question listBlock asks to decide what to draw there, asked the same
// way: the two must agree, or the rows line up with a panel that is not there.
func (m Model) rowsCarryTheSplit() bool {
	room := queueRoomBoth
	if m.tab == tabQueue && m.open() == nil && !m.devices.open {
		room = m.queuePane.room
	}
	return m.scopeVisible() && room.showsTrace()
}

// tempoCell is the beat rate for a list row, or nothing for a track that has
// never been played: a tempo has to be heard to exist.
func (m Model) tempoCell(t player.Track) string {
	if t.Tempo <= 0 {
		return ""
	}
	return m.styles.Quality.Render(fmt.Sprintf("%.0f bpm", t.Tempo))
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
// The optional columns come and go in one order, so a terminal being resized
// walks a list of layouts rather than jumping between arrangements. Narrowing,
// the tempo goes first and the artists second; widening, the album is last in
// because it is the column people scan for least.
const (
	// albumCols is the album's share. Enough for most album names and not so
	// much that it competes with the title.
	albumCols = 28

	// albumFrom is the row width at which the album earns those columns. Below
	// it the row has nothing spare, and taking them would come out of the title.
	albumFrom = 2*secondaryCols + albumCols + tempoCols + trailingCols

	// titleCols is the most the title takes, however wide the terminal is.
	//
	// Without a ceiling the title and the artists divide everything left over
	// between them, and on a wide screen that is a lot: measured at 240 columns,
	// the title's column came to 98 cells and the longest title in it used 15. A
	// row like that is two words at opposite ends of the screen, and reading it
	// means travelling. Sixty-four is past all but the most parenthesised titles.
	titleCols = 64

	// starsCols is the rating's column, which is five stars wide because a
	// rating is five stars, and likedCols the heart's — as wide as the word over
	// it rather than as the glyph in it, so the name is not the thing it names
	// cut in half.
	starsCols = starCount
	likedCols = 5

	// starsFrom is the width at which the two of them earn their columns. Before
	// the album, because between them they cost a third of what it does: the
	// order the columns arrive in is what they are worth against what they take.
	starsFrom = 2*secondaryCols + tempoCols + trailingCols + starsCols + likedCols + 2
)

// rowSpan is how wide each of a row's columns is. A zero is a column this row
// has not earned and will not draw.
type rowSpan struct {
	main   int
	second int
	gap    int
	album  int
	stars  int
	liked  int
	beat   int
}

func rowWidths(body int) rowSpan {
	var span rowSpan

	// Last in, first considered: the columns on the right are taken off the top
	// so everything below divides what is genuinely left, rather than being
	// squeezed after the fact.
	if body >= starsFrom {
		span.stars, span.liked = starsCols, likedCols
		body -= span.stars + span.liked + 2
	}
	if body >= albumFrom {
		span.album = albumCols
		body -= span.album + 1
	}

	span.second = min(body/3, secondaryCols)

	span.beat = tempoCols
	span.main = body - span.second - span.beat - trailingCols - 2
	if span.main < 16 {
		// Too narrow for everything: the tempo goes first, then the artist.
		span.beat = 0
		span.main = body - span.second - trailingCols - 2
	}
	if span.main < 16 {
		return rowSpan{main: body - trailingCols - 1}
	}

	free := body - span.beat - trailingCols - 2
	if grown := min(secondaryCols+(body-shareAbove)/2, free/2); grown > span.second {
		span.second, span.main = grown, free-grown
	}

	return span
}

// capped is the same widths with a ceiling on the title, and what neither it nor
// the artists can use put between the pair and the columns to the right of them.
//
// The artists never wider than the title, because past that point the row reads
// as being about whoever made the record rather than about the record. What the
// two give up becomes one gap rather than air spread through the row, so the
// title and the artists stay a readable distance apart and the album, the rating
// and the clock stay a block at the edge.
//
// Not always applied: on the queue the column the artists begin at is the line
// the trace hangs from, and moving it moves the picture above the list. See
// rowsCarryTheSplit.
func (s rowSpan) capped() rowSpan {
	pair := s.main + s.second
	s.main = min(s.main, titleCols)
	s.second = min(s.second, s.main)
	s.gap = pair - s.main - s.second
	return s
}

// rowSecondaryAt is the column a track row's artists begin at, counted from the
// start of the row. The queue hangs its trace from the same line, so the screen
// reads as two columns rather than four.
func rowSecondaryAt(w int) int {
	span := rowWidths(w - rowGutter)
	if span.second == 0 {
		return w
	}
	return rowGutter + span.main + 1
}

// rowCells is what one row of a list holds, column by column. Every cell is
// passed whether or not the row is wide enough to draw it: which columns fit is
// the row's business, not the caller's.
type rowCells struct {
	primary   string
	secondary string
	album     string
	stars     string
	liked     string
	tempo     string
	trailing  string
}

// rowCols is a row of three columns, for the lists whose rows are a name, a
// second name and a number.
func (m Model) rowCols(w int, selected bool, primary, secondary, tempo, trailing string) string {
	return m.drawRow(w, selected, rowCells{
		primary: primary, secondary: secondary, tempo: tempo, trailing: trailing,
	})
}

// drawRow lays out the cursor's gutter and the columns of one row.
func (m Model) drawRow(w int, selected bool, c rowCells) string {
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
		return gutter + fit(c.primary, max(body, 0))
	}

	span := rowWidths(body)
	if m.rowsMainAt > 0 && m.rowsMainAt < body-trailingCols {
		// Widened or narrowed to line the second column up with something
		// elsewhere on the screen. What the first column gives up or takes goes
		// to the second, so the columns to the right of it stay where they are.
		span.second = max(span.second+span.main-m.rowsMainAt, 0)
		span.main = m.rowsMainAt
	}
	if !m.rowsCarryTheSplit() {
		span = span.capped()
	}

	line := gutter + fit(c.primary, max(span.main, 0)) + " "
	if span.second > 0 {
		line += fit(c.secondary, span.second) + " "
	}
	if span.gap > 0 {
		line += strings.Repeat(" ", span.gap)
	}
	if span.album > 0 {
		line += fit(c.album, span.album) + " "
	}
	if span.stars > 0 {
		line += fit(c.stars, span.stars) + " " + fit(c.liked, span.liked) + " "
	}
	if span.beat > 0 {
		line += padLeft(c.tempo, span.beat)
	}
	return line + padLeft(c.trailing, trailingCols)
}

// searchField renders the query line, keeping the prompt in the accent.
// searchField is the query line. It shows whether the keyboard belongs to it:
// a prompt in the accent while typing, quiet otherwise, so there is never a
// question about where a keystroke will go.
func (m Model) searchField(w int) string {
	in := m.search.input
	in.SetWidth(max(w-4, 8))
	if !m.search.typing {
		if v := in.Value(); v != "" {
			return m.styles.Album.Render("⌕ " + fit(v, max(w-2, 4)))
		}
		return m.styles.Empty.Render("⌕ press / to search")
	}
	return in.View()
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
		case "album":
			// The album has a line of its own.
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
