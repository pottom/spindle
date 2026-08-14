package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

func peekModel() Model {
	m := lyricsModel(120, 44)
	for i, n := range []string{"first", "second", "third", "fourth", "fifth", "sixth"} {
		m.queue = append(m.queue, player.Track{ID: fmt.Sprint(i), Title: n,
			Artists: []string{"someone"}, Duration: time.Duration(180+i*9) * time.Second})
	}
	return m
}

// The glance goes into the band above the artwork, which was blank. Nothing
// below it moves — the same promise the words and the trace make.
func TestPeekMovesNothing(t *testing.T) {
	m := peekModel()
	m.peek.on = false
	off := strings.Split(plain(m.render()), "\n")

	m.peek.on = true
	on := strings.Split(plain(m.render()), "\n")
	if !m.peekVisible() {
		t.Fatal("the glance is not showing")
	}

	// Only the band itself may differ. It begins one row above the body, in the
	// blank the frame keeps under the header — which is where every other
	// screen's first line sits, and is why this one used to read a row low.
	first := tabBarHeight - 1
	for i := range off {
		if i >= first && i < first+peekRows+peekChrome {
			continue
		}
		if off[i] != on[i] {
			t.Errorf("row %d moved\n  off: %q\n  on:  %q", i, off[i], on[i])
		}
	}
}

// It shows what is coming, in order, and no more than a glance's worth.
func TestPeekShowsWhatIsNext(t *testing.T) {
	m := peekModel()
	m.peek.on = true

	out := plain(m.render())
	for _, want := range []string{"Up next", "first", "second", "third", "fourth"} {
		if !strings.Contains(out, want) {
			t.Errorf("the glance does not show %q", want)
		}
	}
	if strings.Contains(out, "fifth") {
		t.Errorf("the glance shows more than %d tracks", peekRows)
	}
}

// It is off to begin with, and the key is what brings it out.
func TestPeekKey(t *testing.T) {
	m := peekModel()
	if m.peek.on {
		t.Error("the glance is on before being asked for")
	}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if !tm.(Model).peekVisible() {
		t.Fatal("u did not bring out the glance")
	}
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if tm.(Model).peekVisible() {
		t.Error("u did not put it away")
	}
}

// Where the band above the artwork is too shallow the glance is not offered,
// and the key says so by doing nothing rather than by rearranging the screen.
func TestPeekNeedsTheBand(t *testing.T) {
	m := lyricsModel(100, minHeight+4)
	if m.peekAvailable() {
		t.Skip("this terminal has room after all")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if cmd != nil || tm.(Model).peekVisible() {
		t.Error("the glance appeared with no band to put it in")
	}
}

// The glance has no cursor and never reaches double figures, so it gives up
// both the column the cursor would stand in and the second digit of the
// ordinal: without that the whole list sits indented from its own heading.
func TestPeekRowsAreFlushWithTheHeading(t *testing.T) {
	m := peekModel()
	m.peek.on = true

	var heading, first int = -1, -1
	for _, row := range strings.Split(plain(m.render()), "\n") {
		switch {
		case strings.Contains(row, "Up next"):
			heading = strings.Index(row, "Up next")
		case first < 0 && strings.Contains(row, "first"):
			first = strings.Index(row, "1")
		}
	}
	if heading < 0 || first < 0 {
		t.Fatal("could not find the heading and the first row")
	}
	if heading != first {
		t.Errorf("the heading starts at column %d and the first row at %d, want them flush", heading, first)
	}

	// And the block sits a column in from the frame's margin, so it does not
	// run hard against the same edge as the artwork and the device name.
	var device int
	for _, row := range strings.Split(plain(m.render()), "\n") {
		if i := strings.Index(row, "◐"); i >= 0 {
			device = i
			break
		}
	}
	if heading != device+1 {
		t.Errorf("the glance starts at column %d and the device name at %d, want one column in", heading, device)
	}
}

// The glance carries a heart for the tracks that are saved, and nothing for the
// rest — in the table's own column, once.
//
// The column in front of the titles carried it before, and before that a dot on
// every row saying each was in the queue, which is the one thing every row of a
// list of what is queued can be relied on to be. It carries nothing now: the
// table names a column for this, and a mark and a column both would be the same
// fact twice in one row.
func TestTheGlanceMarksTheSavedTracks(t *testing.T) {
	m := lyricsModel(190, 44)
	for i, n := range []string{"first", "second", "third", "fourth", "fifth", "sixth"} {
		m.queue = append(m.queue, player.Track{ID: fmt.Sprint(i), Title: n,
			Artists: []string{"someone"}, Duration: time.Duration(180+i*9) * time.Second})
	}
	m.peek.on = true
	m.library.adoptLiked([]player.Track{{ID: "1", Title: "second"}}, false)

	rows := map[string]string{}
	for _, row := range strings.Split(plain(m.render()), "\n") {
		for _, title := range []string{"first", "second", "third"} {
			if strings.Contains(row, title) {
				rows[title] = row
			}
		}
	}
	if len(rows) != 3 {
		t.Fatalf("found %d of the glance's rows", len(rows))
	}

	if !strings.Contains(rows["second"], likedMark) {
		t.Errorf("the saved track has no heart: %q", rows["second"])
	}
	for _, title := range []string{"first", "third"} {
		if strings.Contains(rows[title], likedMark) {
			t.Errorf("%q is not saved and carries a heart: %q", title, rows[title])
		}
	}

	// The titles start at the same column whether a row is saved or not: the
	// heart is out in the table now and nothing in front of them moves.
	at := func(row, title string) int { return lipgloss.Width(row[:strings.Index(row, title)]) }
	if a, b := at(rows["first"], "first"), at(rows["second"], "second"); a != b {
		t.Errorf("the titles start at columns %d and %d, want them in line", a, b)
	}

	// And the heart is out there rather than in front of the title.
	if before := rows["second"][:strings.Index(rows["second"], "second")]; strings.Contains(before, likedMark) {
		t.Errorf("the heart is still in front of the title: %q", before)
	}
}

// Nothing has been read of the saved tracks until something asks: a blank column
// because nobody fetched the list reads exactly like a queue with nothing saved
// in it. Every list of tracks draws the hearts now, so it is asked for once and
// not per screen.
func TestTheGlanceSendsForTheSavedTracks(t *testing.T) {
	m := peekModel()
	if m.readSaved() == nil {
		t.Fatal("the saved tracks were never sent for")
	}

	m.peek.on = true
	if m.readSaved() == nil {
		t.Fatal("the glance is up and the saved tracks were never sent for")
	}

	// The answer fills the set the marks are read from. This is the wire
	// between asking and drawing, and a set that stayed empty would show as a
	// glance with no hearts in it — which is also what a queue of nothing saved
	// looks like, so nothing on screen would say it was broken.
	var tm tea.Model = m
	tm, _ = tm.Update(msg.OpenedFetched{ID: likedID, Tracks: []player.Track{{ID: "3"}}})
	if got := tm.(Model); !got.library.saved("3") {
		t.Error("the saved tracks arrived and the glance still does not know them")
	}

	// And only once: what has been read is not read again on every keystroke.
	m.library.adoptLiked(nil, true)
	if m.readSaved() != nil {
		t.Error("the saved tracks were sent for again after being read")
	}
}

// The artists in the glance begin on the same column the playing track's
// artists do, further down the screen. Measured rather than eyeballed: it was
// wrong twice by a couple of columns, once because the row divided itself and
// once because the arithmetic counted a cursor gutter this list does not have.
func TestTheGlanceLinesUpWithTheColumnBesideThePicture(t *testing.T) {
	m := peekModel()
	m.peek.on = true
	l := m.layout()

	var at = -1
	for _, row := range strings.Split(plain(m.render()), "\n") {
		if strings.Contains(row, "first") {
			at = strings.Index(row, "someone")
			break
		}
	}
	if at < 0 {
		t.Fatal("no artist in the glance to measure")
	}
	if want := leftMargin + l.artWidth + columnGap; at != want {
		t.Errorf("the glance's artists start at column %d, want %d — the column beside the picture",
			at, want)
	}
}

// The glance is set like every other list of tracks: its columns are named, and
// a line stands under the names.
func TestTheGlanceIsATable(t *testing.T) {
	m := lyricsModel(190, 44)
	pop := 80
	for i, n := range []string{"first", "second", "third", "fourth"} {
		m.queue = append(m.queue, player.Track{ID: fmt.Sprint(i), Title: n, Album: "Nightfall",
			Artists: []string{"someone"}, Popularity: &pop, Duration: time.Minute})
	}
	m.peek.on = true

	rows := strings.Split(plain(m.render()), "\n")
	head, rule, first := -1, -1, -1
	for i, row := range rows {
		if head < 0 && strings.Contains(row, "Up next") {
			head = i
		}
		if head >= 0 && rule < 0 && strings.HasPrefix(strings.TrimSpace(row), pointerH) {
			rule = i
		}
		if head >= 0 && first < 0 && strings.Contains(row, "first") {
			first = i
		}
	}
	if head < 0 {
		t.Fatal("the glance is not on the screen")
	}

	// Its name stands where a title would, and the other columns are named
	// beside it — so the label costs no row of a block four rows tall.
	for _, name := range []string{"artist", "album", "stars", "liked", "time"} {
		if !strings.Contains(rows[head], name) {
			t.Errorf("the glance does not name its %s column: %q", name, rows[head])
		}
	}
	if rule != head+1 || first != head+2 {
		t.Errorf("the names are on row %d, the line on %d and the first track on %d", head, rule, first)
	}

	// And the columns hold what they are named after.
	if got := rows[first]; !strings.Contains(got, "Nightfall") || !strings.Contains(got, starFull) {
		t.Errorf("the glance's rows carry no album and no rating: %q", got)
	}
}
