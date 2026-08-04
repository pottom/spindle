package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// queued builds a queue of hand-added tracks followed by context ones, which is
// the shape Spotify always reports: what was asked for, then what was coming
// anyway.
func queueOf(handAdded int, ids ...string) []player.Track {
	out := make([]player.Track, 0, len(ids))
	for i, id := range ids {
		t := trackAt(id, id)
		t.Queued = i < handAdded
		out = append(out, t)
	}
	return out
}

// queueModel builds a queue tab with something playing, so the cursor starts on
// the now-playing row exactly as it does in use. queueRow(n) is the row of the
// nth queue entry.
func queueModel(handAdded int, ids ...string) Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "playing", Playing: true}
	m.tab = tabQueue
	m.queue = queueOf(handAdded, ids...)
	return m
}

func queueRowOf(n int) int { return n + 1 }

// Removing a track must take it off the screen at once. Waiting for the round
// trip would make every x look like it had missed.
func TestDropRemovesTheTrackImmediately(t *testing.T) {
	m := queueModel(2, "a", "b", "c")
	m.queuePane.cursor.cursor = queueRowOf(0)

	if cmd := m.dropQueued(); cmd == nil {
		t.Fatal("dropQueued() = nil, want the edit to be sent")
	}
	if len(m.queue) != 2 || m.queue[0].ID != "b" {
		t.Errorf("queue = %v, want a gone", ids(m.queue))
	}
}

// Every row can be taken out, whichever half of the list it came from: to the
// user there is one list, and a track in it is one they mean to hear.
func TestDropRemovesAContextTrackToo(t *testing.T) {
	dropped := make(chan string, 1)
	m := queueModel(1, "a", "b", "c")
	m.player = recordingEditor{Player: m.player, dropped: dropped}
	m.queuePane.cursor.cursor = queueRowOf(1)

	cmd := m.dropQueued()
	if cmd == nil {
		t.Fatal("dropQueued() = nil on a context track")
	}
	cmd()

	if got := <-dropped; got != "b" {
		t.Errorf("Drop(%q), want the track under the cursor", got)
	}
	if names := ids(m.queue); len(names) != 2 || names[0] != "a" || names[1] != "c" {
		t.Errorf("queue = %v, want b gone and the rest in order", names)
	}
}

func TestMoveSwapsWithinTheQueuedBlock(t *testing.T) {
	m := queueModel(3, "a", "b", "c", "d")
	m.queuePane.cursor.cursor = queueRowOf(0)

	if cmd := m.moveQueued(1); cmd == nil {
		t.Fatal("moveQueued(1) = nil, want the edit to be sent")
	}
	if got := ids(m.queue); got[0] != "b" || got[1] != "a" {
		t.Errorf("queue = %v, want a and b swapped", got)
	}
	// The cursor follows the track, or a second press would move a different one.
	if want := queueRowOf(1); m.queuePane.cursor.cursor != want {
		t.Errorf("cursor = %d, want it to follow the track to %d", m.queuePane.cursor.cursor, want)
	}
}

// A hand-queued track cannot be pushed past the context: the two halves keep
// their own order, and a move that looks like it did nothing is worse than one
// that is refused.
func TestMoveStopsAtTheContextBoundary(t *testing.T) {
	m := queueModel(1, "a", "b", "c")
	m.queuePane.cursor.cursor = queueRowOf(0)

	if cmd := m.moveQueued(1); cmd != nil {
		t.Error("moveQueued(1) crossed into the context tracks")
	}
	if got := ids(m.queue); got[0] != "a" {
		t.Errorf("queue = %v, want it untouched", got)
	}
}

// Only the hand-queued ids are sent: the device replaces that list wholesale,
// so including the context would make it queue the album a second time.
func TestOnlyQueuedTracksAreSent(t *testing.T) {
	sent := make(chan []string, 1)
	m := queueModel(2, "a", "b", "c")
	m.player = recordingEditor{Player: m.player, sent: sent}
	m.queuePane.cursor.cursor = queueRowOf(0)

	cmd := m.moveQueued(1)
	if cmd == nil {
		t.Fatal("moveQueued() = nil")
	}
	cmd()

	got := <-sent
	if len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Errorf("SetQueue(%v), want the two hand-queued tracks, swapped", got)
	}
}

// Pressing enter on a queue entry jumps to it, keeping whatever it belongs to.
func TestEnterPlaysFromTheQueue(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.queuePane.cursor.cursor = queueRowOf(1)

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no command")
	}
	if got := tm.(Model); got.awaitingTrack != "now" {
		t.Errorf("awaitingTrack = %q, want the track being left", got.awaitingTrack)
	}
}

type recordingEditor struct {
	player.Player
	sent    chan []string
	dropped chan string
}

func (r recordingEditor) SetQueue(_ context.Context, ids []string) error {
	r.sent <- ids
	return nil
}

func (r recordingEditor) Drop(_ context.Context, id string) error {
	if r.dropped != nil {
		r.dropped <- id
	}
	return nil
}

// plain strips the styling, so a test can assert on what the screen says rather
// than on how it was coloured.
func plain(s string) string { return ansiOff(s) }

func ids(tracks []player.Track) []string {
	out := make([]string, 0, len(tracks))
	for _, t := range tracks {
		out = append(out, t.ID)
	}
	return out
}

// The queue tab has to draw: a pane that panics or comes out the wrong height
// would break the whole screen, not just itself.
func TestQueuePaneRenders(t *testing.T) {
	m := queueModel(1, "a", "b", "c")
	m.width, m.height = 100, 30
	m.resize()

	out := plain(m.render())
	if !strings.Contains(out, "Queue") {
		t.Error("render() has no queue heading")
	}
	// One list, numbered in the order it will be heard: no row carries a badge
	// for how it got there.
	if !strings.Contains(out, " 1  a") || !strings.Contains(out, " 2  b") {
		t.Errorf("render() does not number the queue:\n%s", out)
	}
	if !strings.Contains(out, "3 tracks") {
		t.Errorf("render() has no count in the subtitle:\n%s", out)
	}
	// Every tab has to come out the same height, or switching would make the
	// screen jump.
	player := m
	player.tab = tabPlayer
	want := len(strings.Split(player.render(), "\n"))
	if got := len(strings.Split(out, "\n")); got != want {
		t.Errorf("render() = %d lines, want %d as on the player tab", got, want)
	}
}

// The panel beside the cover describes whatever the cursor is on, so moving the
// cursor has to change it.
func TestDetailFollowsTheCursor(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.queue[0].Album, m.queue[1].Album = "First Album", "Second Album"
	m.queuePane.cursor.cursor = queueRowOf(0)
	m.width, m.height = 100, 44
	m.resize()

	if got := strings.Join(m.trackDetail(40), "\n"); !strings.Contains(got, "First Album") {
		t.Errorf("trackDetail() = %q, want the first track", got)
	}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := strings.Join(tm.(Model).trackDetail(40), "\n"); !strings.Contains(got, "Second Album") {
		t.Errorf("trackDetail() = %q, want the second track after moving down", got)
	}
}

// Everything Spotify supplied is worth showing; everything it left blank is
// worth leaving out, rather than printing a label with nothing after it.
func TestFactsSkipWhatSpotifyDidNotSay(t *testing.T) {
	full := player.Track{
		Album: "Hot Space", Released: "1982-05-21",
		TrackNumber: 3, TotalTracks: 11, DiscNumber: 2, Duration: 4 * time.Minute,
	}
	got := factLines(trackFacts(full))
	for _, want := range []string{"Album", "Released", "Track", "Length"} {
		if !strings.Contains(got, want) {
			t.Errorf("trackFacts() = %q, want a %s row", got, want)
		}
	}
	if !strings.Contains(got, "3 of 11, disc 2") {
		t.Errorf("trackFacts() = %q, want the disc named when there is more than one", got)
	}
	if !strings.Contains(got, "1982") || strings.Contains(got, "1982-05-21") {
		t.Errorf("trackFacts() = %q, want just the year", got)
	}

	bare := player.Track{Album: "Unknown", Duration: time.Minute}
	if got := factLines(trackFacts(bare)); strings.Contains(got, "Released") || strings.Contains(got, "Track ") {
		t.Errorf("trackFacts() = %q, want no empty rows", got)
	}

}

func factLines(facts []trackFact) string {
	var b strings.Builder
	for _, f := range facts {
		b.WriteString(f.label + " " + f.value + "\n")
	}
	return b.String()
}

// The queue is read downwards from what is sounding now, so that is the first
// row — marked, unnumbered, and not something that can be edited out.
func TestQueueLeadsWithThePlayingTrack(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.width, m.height = 100, 44
	m.resize()

	rows := m.queueRows()
	if len(rows) != 3 || rows[0].ID != "now" {
		t.Fatalf("queueRows() = %v, want the playing track first", ids(rows))
	}
	if m.queueIndex() != -1 {
		t.Errorf("queueIndex() = %d on the playing row, want -1", m.queueIndex())
	}
	if cmd := m.moveQueued(1); cmd != nil {
		t.Error("moveQueued() tried to move the track that is playing")
	}

	out := plain(m.render())
	if !strings.Contains(out, nowMark+"  playing") {
		t.Errorf("render() does not mark the playing track:\n%s", out)
	}
}

// Nothing playing means no leading row, and the queue is the whole list.
func TestQueueWithoutAPlayingTrack(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.ps = nil

	if rows := m.queueRows(); len(rows) != 2 || rows[0].ID != "a" {
		t.Errorf("queueRows() = %v, want just the queue", ids(rows))
	}
	if m.queueIndex() != 0 {
		t.Errorf("queueIndex() = %d, want the cursor to address the queue directly", m.queueIndex())
	}
}

// Bringing a track forward puts it on screen at once, and every other track
// keeps its place and moves up one. Nothing may be dropped: the tracks in the
// list are there to be heard, and choosing one to hear sooner is not a decision
// about the others.
func TestPlayingARowMovesItToTheTop(t *testing.T) {
	m := queueModel(1, "a", "b", "c", "d")
	m.queuePane.cursor.cursor = queueRowOf(2) // "c"

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no command")
	}

	got := tm.(Model)
	if got.ps.Title != "c" {
		t.Errorf("title = %q, want the chosen track playing at once", got.ps.Title)
	}
	if names := ids(got.queue); len(names) != 3 || names[0] != "a" || names[1] != "b" || names[2] != "d" {
		t.Errorf("queue = %v, want the rest kept in their order", names)
	}
	if got.elapsed() > 100*time.Millisecond {
		t.Errorf("elapsed = %v, want the playhead reset", got.elapsed())
	}
}

// Only the leading run of hand-queued tracks is the queue; the device keeps no
// more than that, so sending a context track further down would queue the whole
// album a second time.
func TestOnlyTheLeadingRunIsSent(t *testing.T) {
	sent := make(chan []string, 1)
	m := queueModel(1, "a", "b", "c")
	m.player = recordingEditor{Player: m.player, sent: sent}
	m.queue[2].Queued = true // a mark past a context track, which cannot be a queue

	cmd := m.commitQueue(m.queue)
	if cmd == nil {
		t.Fatal("commitQueue = nil")
	}
	cmd()

	if got := <-sent; len(got) != 1 || got[0] != "a" {
		t.Errorf("SetQueue(%v), want only the run at the front", got)
	}
}

// The bar says two things at a glance: how much of the list is on screen, and
// where in it you are. Both have to be true at the ends, which is where an
// off-by-one shows up as a list that looks unfinished.
func TestScrollRange(t *testing.T) {
	// Ten rows on screen out of forty: a thumb a quarter of the way down the
	// track, at the top when unscrolled and flush with the bottom at the end.
	start, size := scrollRange(10, 40, 0)
	if size != 2 || start != 0 {
		t.Errorf("scrollRange(10, 40, 0) = %d, %d; want a thumb of 2 at the top", start, size)
	}
	if start, size := scrollRange(10, 40, 30); start+size != 10 {
		t.Errorf("scrollRange(10, 40, 30) = %d, %d; want it flush with the bottom", start, size)
	}
	// A list barely longer than the pane still gets a thumb that can be seen.
	if _, size := scrollRange(20, 400, 0); size < 1 {
		t.Error("scrollRange lost the thumb on a long list")
	}
	// And one that cannot scroll never leaves the top.
	if start, size := scrollRange(10, 11, 0); start != 0 || size < 1 {
		t.Errorf("scrollRange(10, 11, 0) = %d, %d", start, size)
	}
}

// A list that fits gets no bar: furniture that can never move is worth less
// than the column it takes from the durations.
func TestScrollbarOnlyWhenItCanScroll(t *testing.T) {
	m := queueModel(0, "a", "b", "c")
	m.width, m.height = 96, 30
	m.resize()

	if got := plain(m.render()); strings.Contains(got, scrollThumb) {
		t.Errorf("render() drew a scrollbar for a list that fits:\n%s", got)
	}

	for i := 0; i < 40; i++ {
		m.queue = append(m.queue, trackAt(fmt.Sprintf("x%d", i), "filler"))
	}
	out := plain(m.render())
	if !strings.Contains(out, scrollThumb) || !strings.Contains(out, scrollTrack) {
		t.Errorf("render() drew no scrollbar for a list that overflows:\n%s", out)
	}
}

// A rating of zero is a rating: an obscure catalogue track really does score
// nothing, and hiding the row for it would read as the backend having gone
// quiet. Only a backend that does not rate tracks at all leaves it out.
func TestPopularityShownEvenAtZero(t *testing.T) {
	zero, fifty := 0, 50
	m := queueModel(0, "a", "b")
	m.width, m.height = 100, 44
	m.resize()

	m.queue[0].Popularity = &zero
	m.queuePane.cursor.cursor = queueRowOf(0)
	if got := plain(strings.Join(m.trackDetail(40), "\n")); !strings.Contains(got, "Popularity") {
		t.Errorf("trackDetail() = %q, want a rating of zero shown", got)
	}

	m.queue[1].Popularity = &fifty
	m.queuePane.cursor.cursor = queueRowOf(1)
	if got := plain(strings.Join(m.trackDetail(40), "\n")); !strings.Contains(got, strings.Repeat(starFull, 3)) {
		t.Errorf("trackDetail() = %q, want three of five stars for a rating of fifty", got)
	}

	// A backend that does not rate tracks says nothing rather than zero.
	m.queue[1].Popularity = nil
	if got := plain(strings.Join(m.trackDetail(40), "\n")); strings.Contains(got, "Popularity") {
		t.Errorf("trackDetail() = %q, want no rating when the backend does not give one", got)
	}
}

// Five stars over Spotify's hundred: each stands for twenty. The boundaries are
// where a rating looks wrong if they are off by one, so they are pinned here.
func TestStars(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)

	cases := []struct {
		popularity int
		want       int
	}{
		{0, 1}, {1, 1}, {20, 1},
		{21, 2}, {40, 2},
		{41, 3}, {49, 3}, {60, 3},
		{61, 4}, {80, 4},
		{81, 5}, {100, 5},
	}
	for _, c := range cases {
		got := strings.Count(plain(m.stars(c.popularity)), starFull)
		if got != c.want {
			t.Errorf("stars(%d) = %d filled, want %d", c.popularity, got, c.want)
		}
		if total := len([]rune(plain(m.stars(c.popularity)))); total != starCount {
			t.Errorf("stars(%d) = %d stars, want %d", c.popularity, total, starCount)
		}
	}
}

// The rating is on the detail panel for one track at a time. The mark is what
// carries it to every row at once, so it has to appear exactly where the stars
// reach four and nowhere else.
func TestHotMark(t *testing.T) {
	m := queueModel(0, "a", "b", "c")
	m.width, m.height = 96, 30
	m.resize()

	three, four, five := 60, 61, 100
	m.queue[0].Popularity = &three
	m.queue[1].Popularity = &four
	m.queue[2].Popularity = &five

	for _, line := range strings.Split(plain(m.render()), "\n") {
		marked := strings.Contains(line, hotMark)
		switch {
		case strings.Contains(line, " a"):
			if marked {
				t.Errorf("three stars was marked: %q", line)
			}
		case strings.Contains(line, " b"):
			if !marked {
				t.Errorf("four stars was not marked: %q", line)
			}
		case strings.Contains(line, " c"):
			if !marked {
				t.Errorf("five stars was not marked: %q", line)
			}
		}
	}

	// A backend that does not rate tracks marks nothing.
	m.queue[1].Popularity, m.queue[2].Popularity = nil, nil
	if strings.Contains(plain(m.render()), hotMark) {
		t.Error("tracks with no rating were marked")
	}
}

// A tempo is measured while a track sounds, so it exists for what has been
// heard and for nothing else. Showing "0 bpm" for the rest would be inventing
// a fact about them.
func TestTempoShownOnlyWhenMeasured(t *testing.T) {
	measured := player.Track{Album: "Hot Space", Duration: 4 * time.Minute, Tempo: 117.6}
	if got := factLines(trackFacts(measured)); !strings.Contains(got, "118 bpm") {
		t.Errorf("trackFacts() = %q, want the measured tempo", got)
	}

	// The row stays even with nothing in it: a tempo takes a dozen seconds to
	// measure, and a row appearing then would push everything under it down
	// while it is being read.
	never := player.Track{Album: "Hot Space", Duration: 4 * time.Minute}
	got := factLines(trackFacts(never))
	if strings.Contains(got, "bpm") {
		t.Errorf("trackFacts() = %q, want no number for a track never played", got)
	}
	if !strings.Contains(got, "Tempo "+unknownValue) {
		t.Errorf("trackFacts() = %q, want the tempo row held open", got)
	}
	if a, b := len(trackFacts(measured)), len(trackFacts(never)); a != b {
		t.Errorf("the panel is %d rows with a tempo and %d without, want the same height", a, b)
	}
}

// The live measurement is fresher than whatever was recorded when the track was
// last played, and on a first listen it is the only one there is.
func TestPlayingTrackPrefersTheLiveTempo(t *testing.T) {
	m := queueModel(0, "a")
	m.ps.Tempo = 128
	m.nowQueued = &player.Track{ID: "now", Title: "playing", Tempo: 96}

	got, ok := m.nowPlayingRow()
	if !ok {
		t.Fatal("no playing row")
	}
	if got.Tempo != 128 {
		t.Errorf("tempo = %.0f, want the live measurement", got.Tempo)
	}

	// With nothing measured yet, what was recorded before still stands.
	m.ps.Tempo = 0
	if got, _ := m.nowPlayingRow(); got.Tempo != 96 {
		t.Errorf("tempo = %.0f, want the remembered one while nothing is measured", got.Tempo)
	}
}

// The tempo column holds its width whether a track has been heard or not, so
// the durations stay in line down the list instead of stepping in and out.
func TestTempoColumnHoldsItsPlace(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.queue[0].Tempo = 118
	m.width, m.height = 100, 44
	m.resize()

	rows := strings.Split(plain(m.render()), "\n")
	var withTempo, without string
	for _, r := range rows {
		switch {
		case strings.Contains(r, " a "):
			withTempo = r
		case strings.Contains(r, " b "):
			without = r
		}
	}
	if withTempo == "" || without == "" {
		t.Fatalf("could not find both rows:\n%s", strings.Join(rows, "\n"))
	}

	// The duration is the last thing on the row, so its column is the test.
	at := func(line string) int { return strings.LastIndex(strings.TrimRight(line, " "), "3:00") }
	if a, b := at(withTempo), at(without); a != b {
		t.Errorf("durations sit at %d and %d, want the same column\n  %q\n  %q", a, b, withTempo, without)
	}
	if !strings.Contains(withTempo, "118") {
		t.Errorf("row = %q, want the tempo in it", withTempo)
	}
}

// The artist column grows with the row. Held at a fixed twenty it cut names
// that would easily have fitted on a wide screen, and a list of collaborations
// came out as mostly commas.
func TestArtistColumnGrowsWithTheRow(t *testing.T) {
	m := queueModel(0, "a")
	m.queue[0].Artists = []string{"DJ Alex Man", "Dj Diac", "Nomeli"}
	full := strings.Join(m.queue[0].Artists, ", ")

	wide := m
	wide.width, wide.height = 200, 45
	wide.resize()
	if got := plain(wide.render()); !strings.Contains(got, full) {
		t.Errorf("a 200-column screen still cut the artists:\n%s", got)
	}

	// A narrow one still puts the title first: the name is what gives way.
	narrow := m
	narrow.width, narrow.height = 80, 45
	narrow.resize()
	out := plain(narrow.render())
	if strings.Contains(out, full) {
		t.Error("an 80-column screen gave the artists room the title needed")
	}
	if !strings.Contains(out, "a") {
		t.Error("the title did not survive")
	}
}
