package ui

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// The rules every screen obeys, checked at every size rather than argued about.
//
// This file exists because the answer to "does it still use the room?" was an
// afternoon of squinting at a terminal, and the answer changes every time
// anything is added to a tab. Adding a column, a caption, a band — any of it can
// take the room back without anybody noticing until a screenshot arrives.
//
// So: after any change to what a tab holds, this is what says whether the
// responsiveness survived it. If a rule here starts failing, the layout is what
// moved, not the test.

// probeSizes runs from the smallest terminal that draws at all up to a very
// large one. The large ones matter most: shrinking the font is how somebody asks
// for more of a screen, and it is where a fixed-size element shows up.
var probeSizes = [][2]int{
	{minWidth, minHeight}, {80, 24}, {100, 30}, {132, 36},
	{160, 44}, {200, 50}, {240, 60}, {300, 70}, {400, 100},
}

// Nothing on the player screen may be a fixed height: the picture takes the rows
// the trace does not, so the body is never mostly blank however tall the
// terminal is.
func TestThePlayerGrowsIntoTheRoomItIsGiven(t *testing.T) {
	for _, s := range probeSizes {
		l := computeLayout(s[0], s[1], 2, false, modePlayer, defaultTestCell)
		if !l.hasArt() {
			continue
		}
		// Once the picture has reached its ceiling the rest of the room is blank
		// on purpose, so the rule only holds below it. With slack for the
		// squaring off: a cover has to come out a whole number of cells both
		// ways, and cells are not square, so the last few columns of the
		// allowance are unreachable.
		if ceiling := maxArtPx / defaultTestCell.Width; l.artWidth >= ceiling-3 {
			continue
		}
		// And the ceiling counted in rows, which is the one that binds on a
		// display that does not report two pixels to the point. Same rule: a
		// picture that has stopped growing because it was told to is not a
		// picture that stopped growing by accident.
		if l.artHeight >= maxArtRows {
			continue
		}
		spare := l.bodyHeight - l.artHeight
		if spare > playerBelowArt {
			t.Errorf("%dx%d: %d rows spare under the picture, want no more than %d — "+
				"something on this screen has stopped growing with the terminal",
				s[0], s[1], spare, playerBelowArt)
		}
	}
}

// And it grows sideways too, into the picture rather than into the line length.
func TestThePlayerFillsTheWidthItIsGiven(t *testing.T) {
	var wasArt int
	for _, s := range probeSizes {
		l := computeLayout(s[0], s[1], 2, false, modePlayer, defaultTestCell)
		if !l.hasArt() {
			continue
		}
		// The frame reaches the edge of what the terminal offers, the same as
		// every list does. A player centred in a blank band was the bug.
		if want := s[0]; l.interior != want {
			t.Errorf("%dx%d: the player is %d wide, want %d — it is leaving a blank band down the sides",
				s[0], s[1], l.interior, want)
		}
		if used := leftMargin + l.artWidth + columnGap + l.infoWidth + rightMargin; used < l.interior {
			t.Errorf("%dx%d: the two columns come to %d inside a frame of %d",
				s[0], s[1], used, l.interior)
		}
		if l.artWidth < wasArt {
			t.Errorf("%dx%d: the picture shrank to %d from %d on a wider terminal",
				s[0], s[1], l.artWidth, wasArt)
		}
		wasArt = l.artWidth
		if l.interior > s[0] {
			t.Errorf("%dx%d: the frame is %d wide, past the terminal", s[0], s[1], l.interior)
		}
	}
}

// The lists take the width they are given, all of it. Nothing caps it: somebody
// who shrinks the type is buying room, and a margin hands it back.
func TestTheListsTakeTheWidth(t *testing.T) {
	for _, s := range probeSizes {
		l := computeLayout(s[0], s[1], 2, false, modeList, defaultTestCell)
		want := s[0]
		if l.interior != want {
			t.Errorf("%dx%d: the list is %d wide, want %d", s[0], s[1], l.interior, want)
		}
	}
}

// A column that has earned its place never loses it on a wider row, and none
// appears before the one ahead of it. That is the whole of what "responsive"
// means here: widening walks one order of layouts, so a resize adds a column
// rather than rearranging the row.
func TestColumnsComeInOneOrderAndStay(t *testing.T) {
	type seen struct{ second, beat, album, stars bool }
	var was seen

	for body := 20; body <= 400; body++ {
		span := rowWidths(body)
		now := seen{span.second > 0, span.beat > 0, span.third > 0, span.stars > 0}

		for _, c := range []struct {
			name       string
			had, hasIt bool
		}{
			{"the artists", was.second, now.second},
			{"the tempo", was.beat, now.beat},
			{"the album", was.album, now.album},
			{"the rating", was.stars, now.stars},
		} {
			if c.had && !c.hasIt {
				t.Errorf("row %d: %s column went away on a wider row", body, c.name)
			}
		}

		// The order: nothing arrives before what stands ahead of it.
		if now.album && !now.second {
			t.Errorf("row %d: the album is shown but the artists are not", body)
		}
		if now.beat && !now.second {
			t.Errorf("row %d: the tempo is shown but the artists are not", body)
		}
		if now.album && !now.stars {
			t.Errorf("row %d: the album is shown but the rating is not", body)
		}

		// The heart comes with the rating and goes with it.
		if (span.stars > 0) != (span.liked > 0) {
			t.Errorf("row %d: the rating has %d columns and the heart %d", body, span.stars, span.liked)
		}

		// And the row still adds up to what it was given.
		used := span.main + span.second + span.beat + span.third + span.stars + span.liked + trailingCols
		if used > body {
			t.Errorf("row %d: the columns come to %d, past the row", body, used)
		}
		was = now
	}
}

// The title never gives away so much that it stops being the column the list is
// read by. A column arriving may take from it, but not to the point where the
// names are unreadable.
func TestTheTitleKeepsTheLargestShare(t *testing.T) {
	for body := 60; body <= 400; body++ {
		span := rowWidths(body)
		if span.main < span.second {
			t.Errorf("row %d: the artists have %d columns and the title %d", body, span.second, span.main)
		}
		if span.third > 0 && span.main < span.third {
			t.Errorf("row %d: the album has %d columns and the title %d", body, span.third, span.main)
		}
	}
}

// Every tab draws at every size without a row running past the frame. A row that
// overruns wraps in the terminal and pushes everything below it down, which is
// how a screen that looked fine at one size falls apart at another.
func TestNoTabDrawsPastItsFrame(t *testing.T) {
	for _, which := range []struct {
		name string
		tab  tabID
	}{
		{"player", tabPlayer},
		{"queue", tabQueue},
		{"library", tabLibrary},
		{"search", tabSearch},
		{"settings", tabSettings},
		{"help", tabHelp},
	} {
		for _, s := range probeSizes {
			m := sizedModel(which.tab, s[0], s[1])
			for i, line := range strings.Split(m.render(), "\n") {
				if w := len([]rune(stripStyles(line))); w > s[0] {
					t.Errorf("%s %dx%d: row %d is %d wide", which.name, s[0], s[1], i, w)
					break
				}
			}
		}
	}
}

func sizedModel(which tabID, w, h int) Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = w, h
	m.tab = which
	m.ps = &player.State{
		Title: "Valhalla Calling", Artists: []string{"Ragal Ironbull"},
		Album: "VALHALLA CALLING", Playing: true,
		Duration: 3 * time.Minute, Progress: time.Minute, Volume: 60,
	}
	m.queue = make([]player.Track, 30)
	for i := range m.queue {
		m.queue[i] = player.Track{
			ID:       fmt.Sprintf("t%02d", i),
			Title:    "A Track With A Fairly Long Name To Cut",
			Artists:  []string{"An Artist With A Long Name"},
			Album:    "An Album With A Long Name",
			Duration: time.Duration(150+i) * time.Second,
		}
	}
	return m
}

// stripStyles removes the escape sequences so a row can be measured in cells.
func stripStyles(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// The outlines are a measuring tool, so the one thing they may never do is
// change what they measure. Turning them on must leave every row count and every
// row width exactly as it was — a border that added a column would move
// everything inside it, and the screen being inspected would stop being the
// screen that ships.
func TestOutlinesDoNotMoveAnything(t *testing.T) {
	for _, which := range []tabID{tabPlayer, tabQueue, tabLibrary, tabSearch, tabSettings, tabHelp} {
		for _, s := range probeSizes {
			plain := strings.Split(sizedModel(which, s[0], s[1]).render(), "\n")

			m := sizedModel(which, s[0], s[1])
			m.debug.level = debugOutlines
			drawn := strings.Split(m.render(), "\n")

			if len(plain) != len(drawn) {
				t.Errorf("tab %v %dx%d: %d rows with outlines, %d without",
					which, s[0], s[1], len(drawn), len(plain))
				continue
			}
			for i := range plain {
				was, now := len([]rune(stripStyles(plain[i]))), len([]rune(stripStyles(drawn[i])))
				if was != now {
					t.Errorf("tab %v %dx%d: row %d is %d cells with outlines, %d without",
						which, s[0], s[1], i, now, was)
					break
				}
			}
		}
	}
}

// And the border of a block is the block's own edge, not an extra one.
func TestAnOutlineKeepsTheBlocksShape(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.debug.level = debugOutlines

	rows := []string{"aaaaaaaa", "bbbbbbbb", "cccccccc", "dddddddd"}
	out := m.outline(rows, 8, "thing")

	if len(out) != len(rows) {
		t.Fatalf("outline returned %d rows for %d", len(out), len(rows))
	}
	for i, row := range out {
		if w := len([]rune(stripStyles(row))); w != 8 {
			t.Errorf("row %d came out %d cells wide, want 8", i, w)
		}
	}
	if bare := stripStyles(out[0]); !strings.HasPrefix(bare, outlineTL) || !strings.HasSuffix(bare, outlineTR) {
		t.Errorf("the top edge is %q", bare)
	}
	if bare := stripStyles(out[len(out)-1]); !strings.HasPrefix(bare, outlineBL) {
		t.Errorf("the foot is %q", bare)
	}
}

// Off, it does nothing at all — the check that keeps a debug tool out of the
// picture everybody else sees.
func TestOutlinesDrawNothingWhenOff(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	rows := []string{"aaaa", "bbbb", "cccc"}
	out := m.outline(rows, 4, "thing")
	for i := range rows {
		if out[i] != rows[i] {
			t.Errorf("row %d was drawn on with the outlines off: %q", i, out[i])
		}
	}
}

// The cover has a ceiling, and it is in pixels rather than cells: shrinking the
// terminal font makes a cell smaller and the same number of cells physically
// larger, which is how it kept growing on a screen that was not.
func TestTheCoverStopsGrowing(t *testing.T) {
	for _, s := range [][2]int{{160, 44}, {200, 50}, {240, 60}, {300, 70}, {400, 100}, {600, 160}} {
		l := computeLayout(s[0], s[1], 2, false, modePlayer, defaultTestCell)
		if px := l.artWidth * defaultTestCell.Width; px > maxArtPx {
			t.Errorf("%dx%d: the cover is %d pixels across, past the %d it may be",
				s[0], s[1], px, maxArtPx)
		}
	}
}

// No tab leaves a band down the sides, however wide the terminal is.
//
// The frame was capped at two hundred columns, so a small font on a large screen
// — which is somebody buying room — got a fifth of it back as margin. What is
// capped is the prose inside the frame, which is where the argument about line
// length belongs.
func TestNoTabLeavesABandDownTheSides(t *testing.T) {
	for _, w := range []int{140, 200, 260, 340} {
		for _, tab := range []tabID{tabPlayer, tabQueue, tabLibrary, tabSearch, tabSettings, tabHelp} {
			m := queueModel(0, "one", "two", "three", "four")
			m.ps = &player.State{TrackID: "now", Title: "sounding", Playing: true}
			m.tab = tab
			m.width, m.height = w, 40
			m.resize()

			if l := m.layout(); l.interior != w {
				t.Errorf("%d cells, %v: the frame is %d wide, want the whole terminal", w, tab, l.interior)
			}

			// And something is drawn out at the right-hand edge, rather than the
			// frame merely claiming the width and leaving it blank.
			var reach int
			for _, row := range strings.Split(m.render(), "\n") {
				reach = max(reach, lipgloss.Width(strings.TrimRight(ansiOff(row), " ")))
			}
			if want := w - rightMargin; reach < want {
				t.Errorf("%d cells, %v: nothing is drawn past column %d, want the screen used to %d",
					w, tab, reach, want)
			}
		}
	}
}

// On a wide row the title stops growing and what it gives up becomes one gap
// before the columns at the right, rather than air spread through the row.
func TestTheTitleStopsGrowing(t *testing.T) {
	for body := 60; body <= 400; body++ {
		span := rowWidths(body).capped()
		if span.main > titleCols {
			t.Errorf("row %d: the title has %d columns, past its ceiling of %d", body, span.main, titleCols)
		}
		if span.second > span.main {
			t.Errorf("row %d: the artists have %d columns and the title %d", body, span.second, span.main)
		}
		if span.gap < 0 {
			t.Errorf("row %d: the gap is %d", body, span.gap)
		}

		// The row still adds up to what it was given.
		used := span.main + span.second + span.gap + span.third + span.stars + span.liked + span.beat + trailingCols
		if used > body {
			t.Errorf("row %d: the columns come to %d, past the row", body, used)
		}
	}

	// And the ceiling is what opens the gap, not something else: on a row wide
	// enough, the pair is the ceiling and the rest is the gap.
	wide := rowWidths(400).capped()
	if wide.gap == 0 {
		t.Error("a 400-column row has no gap, so the pair took everything")
	}
	if narrow := rowWidths(100).capped(); narrow.gap != 0 {
		t.Errorf("a 100-column row has a gap of %d, want the pair to use it", narrow.gap)
	}
}

// A list of answers is laid out for being read rather than for being kept: it
// holds no column it cannot fill, and what it can fill arrives as early as the
// width allows. Measured against the same rules the queue's rows follow.
func TestAnswerRowsKeepTheirColumnsAtEveryWidth(t *testing.T) {
	var was struct{ second, third, stars bool }

	for body := 20; body <= 400; body++ {
		span := answerWidths(body)

		// No tempo, ever. A record nobody has played has no measured beat, and
		// nobody has played any of these.
		if span.beat > 0 {
			t.Fatalf("row %d: a list of answers holds %d columns for a tempo", body, span.beat)
		}

		now := struct{ second, third, stars bool }{span.second > 0, span.third > 0, span.stars > 0}
		if was.second && !now.second {
			t.Errorf("row %d: the artists went away on a wider row", body)
		}
		if was.third && !now.third {
			t.Errorf("row %d: the album went away on a wider row", body)
		}
		if was.stars && !now.stars {
			t.Errorf("row %d: the rating went away on a wider row", body)
		}
		if now.third && !now.stars {
			t.Errorf("row %d: the album is shown but the rating is not", body)
		}
		if used := span.main + span.second + span.third + span.stars + span.liked + trailingCols; used > body {
			t.Errorf("row %d: the columns come to %d, past the row", body, used)
		}
		if span.third > 0 && span.third > span.main {
			t.Errorf("row %d: the album has %d columns and the title %d", body, span.third, span.main)
		}
		was = now
	}

	// And they arrive earlier than on a list of tracks somebody has built,
	// which is the whole point: the columns are how one answer is told from
	// twenty that look like it.
	for _, c := range []struct {
		name  string
		of    func(rowSpan) int
		queue int
	}{
		{"the rating", func(s rowSpan) int { return s.stars }, firstWidth(t, func(b int) bool { return rowWidths(b).stars > 0 })},
		{"the album", func(s rowSpan) int { return s.third }, firstWidth(t, func(b int) bool { return rowWidths(b).third > 0 })},
	} {
		answers := firstWidth(t, func(b int) bool { return c.of(answerWidths(b)) > 0 })
		if answers >= c.queue {
			t.Errorf("%s arrives at %d on a list of answers and %d on the queue, want it sooner", c.name, answers, c.queue)
		}
	}

	// A list of things that are not tracks holds neither a rating nor a heart:
	// an artist's own records are rows, and no record has either.
	for body := 20; body <= 400; body++ {
		if span := widths(body, false, true); span.stars > 0 || span.liked > 0 {
			t.Fatalf("row %d: a list of records holds a rating column", body)
		}
	}
}

// firstWidth is the narrowest row at which something is true.
func firstWidth(t *testing.T, when func(body int) bool) int {
	t.Helper()
	for body := 20; body <= 400; body++ {
		if when(body) {
			return body
		}
	}
	t.Fatal("never true at any width")
	return 0
}
