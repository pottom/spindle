package ui

import (
	"context"
	"image/color"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
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

// A track from the album or the playlist moves like any other. Reordering it
// means lifting it out of the context, and everything ahead of it has to travel
// too, or it would be pushed behind the tracks that moved.
func TestMoveCrossesIntoTheContext(t *testing.T) {
	reordered := make(chan []string, 1)
	m := queueModel(1, "a", "b", "c")
	m.player = recordingEditor{Player: m.player, reordered: reordered}
	m.queuePane.cursor.cursor = queueRowOf(2) // c, a context track

	if cmd := m.moveQueued(-1); cmd == nil {
		t.Fatal("moveQueued(-1) = nil on a context track")
	}
	settle(t, &m)

	if got := <-reordered; len(got) != 3 || got[0] != "a" || got[1] != "c" || got[2] != "b" {
		t.Errorf("Reorder(%v), want the whole run down to the edit, in its new order", got)
	}
	if got := ids(m.queue); got[1] != "c" || got[2] != "b" {
		t.Errorf("queue = %v, want c ahead of b", got)
	}
	// They are the queue's now, so the next press can move them freely.
	for i, track := range m.queue {
		if !track.Queued {
			t.Errorf("track %d is still marked as the context's after being reordered", i)
		}
	}
	if want := queueRowOf(1); m.queuePane.cursor.cursor != want {
		t.Errorf("cursor = %d, want it to follow the track to %d", m.queuePane.cursor.cursor, want)
	}
}

// The run stops at the deepest edit: sending the whole list would drag tracks
// out of the context that nobody asked to move.
func TestMoveSendsOnlyAsFarAsTheEdit(t *testing.T) {
	reordered := make(chan []string, 1)
	m := queueModel(0, "a", "b", "c", "d")
	m.player = recordingEditor{Player: m.player, reordered: reordered}
	m.queuePane.cursor.cursor = queueRowOf(1)

	if cmd := m.moveQueued(-1); cmd == nil {
		t.Fatal("moveQueued(-1) = nil")
	}
	settle(t, &m)

	if got := <-reordered; len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Errorf("Reorder(%v), want the two tracks that moved and no more", got)
	}
	if m.queue[3].Queued {
		t.Error("a track past the edit was lifted out of the context")
	}
}

// Only the hand-queued ids are sent: the device replaces that list wholesale,
// so including the context would make it queue the album a second time.
func TestOnlyQueuedTracksAreSent(t *testing.T) {
	sent := make(chan []string, 1)
	m := queueModel(2, "a", "b", "c")
	m.player = recordingEditor{Player: m.player, sent: sent}
	m.queuePane.cursor.cursor = queueRowOf(0)

	if cmd := m.moveQueued(1); cmd == nil {
		t.Fatal("moveQueued() = nil")
	}
	settle(t, &m)

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
	sent      chan []string
	reordered chan []string
	dropped   chan string
}

// settle runs the edit the move keys left waiting, as the debounce would once
// the list stopped moving.
func settle(t *testing.T, m *Model) {
	t.Helper()
	cmd := m.sendOrder()
	if cmd == nil {
		t.Fatal("nothing was waiting to be sent")
	}
	cmd()
}

func (r recordingEditor) Reorder(_ context.Context, ids []string) error {
	if r.reordered != nil {
		r.reordered <- ids
	}
	return nil
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

// showOpen puts a page on screen as opening the row would, without the request
// that fills it: the tests that follow are about what the screen does with the
// contents, not about how they arrive.
func showOpen(m *Model, pl player.Playlist, tracks []player.Track) {
	page := openedPlaylist(pl)
	page.tracks = tracks
	m.stack = append(m.stack, page)
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

	if got := strings.Join(m.trackDetail(40, 20), "\n"); !strings.Contains(got, "First Album") {
		t.Errorf("trackDetail() = %q, want the first track", got)
	}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := strings.Join(tm.(Model).trackDetail(40, 20), "\n"); !strings.Contains(got, "Second Album") {
		t.Errorf("trackDetail() = %q, want the second track after moving down", got)
	}
}

// Everything Spotify supplied is worth showing; everything it left blank is
// worth leaving out, rather than showing a fact with nothing in it.
func TestFactsSkipWhatSpotifyDidNotSay(t *testing.T) {
	full := player.Track{
		Album: "Hot Space", Released: "1982-05-21",
		TrackNumber: 3, TotalTracks: 11, DiscNumber: 2, Duration: 4 * time.Minute,
	}
	got := factLines(trackFacts(full))
	for _, want := range []string{"Hot Space", "1982"} {
		if !strings.Contains(got, want) {
			t.Errorf("trackFacts() = %q, want %q in it", got, want)
		}
	}

	// Two are not among them any more. Where a track stands on its album is
	// what nobody reading a queue is asking, and how long it is is already
	// under the playhead, where it is read against the elapsed time.
	for _, gone := range []string{"3 of 11", "4:00"} {
		if strings.Contains(got, gone) {
			t.Errorf("trackFacts() = %q, want nothing saying %q", got, gone)
		}
	}
	if strings.Contains(got, "1982-05-21") {
		t.Errorf("trackFacts() = %q, want just the year", got)
	}

	bare := player.Track{Album: "Unknown", Duration: time.Minute}
	if got := factLines(trackFacts(bare)); strings.Contains(got, "1:00") || len(strings.Split(strings.TrimSpace(got), "\n")) > 2 {
		t.Errorf("trackFacts() = %q, want no empty rows", got)
	}
}

func factLines(facts []trackFact) string {
	var b strings.Builder
	for _, f := range facts {
		b.WriteString(f.value + "\n")
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

	// Stood over the name rather than named among the facts: a row of stars
	// says what it is, and a label beside them is a caption on a picture.
	m.queue[0].Popularity = &zero
	m.queuePane.cursor.cursor = queueRowOf(0)
	got := plain(strings.Join(m.trackDetail(40, 20), "\n"))
	if !strings.Contains(got, starEmpty) {
		t.Errorf("trackDetail() = %q, want a rating of zero shown", got)
	}
	if strings.Contains(got, "Popularity") {
		t.Errorf("trackDetail() = %q, want the stars to stand without a label", got)
	}
	// Under the name, not over it: a title is what the eye should land on
	// first, and the rating belongs to the track rather than to the panel.
	// A row of air, the name, the artist, and then the rating under them.
	rows := strings.Split(got, "\n")
	if len(rows) < 4 || !strings.Contains(rows[3], starEmpty) {
		t.Errorf("the panel begins %q, want the stars under the name and the artist", rows[:min(len(rows), 4)])
	}

	m.queue[1].Popularity = &fifty
	m.queuePane.cursor.cursor = queueRowOf(1)
	if got := plain(strings.Join(m.trackDetail(40, 20), "\n")); !strings.Contains(got, strings.Repeat(starFull, 3)) {
		t.Errorf("trackDetail() = %q, want three of five stars for a rating of fifty", got)
	}

	// A backend that does not rate tracks says nothing rather than zero.
	m.queue[1].Popularity = nil
	if got := plain(strings.Join(m.trackDetail(40, 20), "\n")); strings.Contains(got, "Popularity") {
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
	if !strings.Contains(got, unknownValue) {
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

// Holding the key must not send an edit per press. Each one rewrites what the
// device has coming, and the next would be describing a list it has not agreed
// to yet — which is what the reorder refusals were.
func TestMoveSendsOneEditForARunOfPresses(t *testing.T) {
	reordered := make(chan []string, 4)
	m := queueModel(0, "a", "b", "c", "d")
	m.player = recordingEditor{Player: m.player, reordered: reordered}
	m.queuePane.cursor.cursor = queueRowOf(3)

	for range 3 {
		m.moveQueued(-1)
	}
	if len(reordered) != 0 {
		t.Fatal("an edit went out while the list was still moving")
	}

	// The timers the earlier presses left behind have been overtaken.
	var tm tea.Model = m
	for seq := 1; seq < m.order.seq; seq++ {
		if _, cmd := tm.Update(msg.OrderSettled{Seq: seq}); cmd != nil {
			t.Errorf("the timer from press %d sent an edit of its own", seq)
		}
	}

	_, cmd := tm.Update(msg.OrderSettled{Seq: m.order.seq})
	if cmd == nil {
		t.Fatal("the last press sent nothing")
	}
	cmd()

	if got := <-reordered; len(got) != 4 || got[0] != "d" {
		t.Errorf("Reorder(%v), want the whole run in the order it ended up", got)
	}
	if len(reordered) != 0 {
		t.Errorf("%d further edits went out, want one for the run", len(reordered))
	}
}

// A poll landing mid-edit describes the list as the device still has it, and
// drawing that would undo the move under the user's hand.
func TestAPollDoesNotUndoAWaitingMove(t *testing.T) {
	m := queueModel(0, "a", "b", "c")
	m.queuePane.cursor.cursor = queueRowOf(1)
	m.moveQueued(-1)

	var tm tea.Model = m
	tm, _ = tm.Update(msg.QueueFetched{Tracks: queueOf(0, "a", "b", "c")})
	if got := ids(tm.(Model).queue); got[0] != "b" || got[1] != "a" {
		t.Errorf("queue = %v, want the move to have survived the poll", got)
	}
}

// The playhead belongs to the track sounding, and its row is kept for every
// other one: without that the panel would shift by a row each time the cursor
// passed the track playing.
func TestQueueDetailShowsThePlayhead(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.ps = &player.State{TrackID: "a", Playing: true, Duration: 2 * time.Minute}
	m.queue[0].Duration = 2 * time.Minute

	playing := plain(strings.Join(m.trackDetail(40, 20), "\n"))
	if !strings.Contains(playing, knob) {
		t.Errorf("detail = %q, want the playhead on the track sounding", playing)
	}
	if !strings.Contains(playing, "2:00") {
		t.Errorf("detail = %q, want the clock beside the playhead", playing)
	}

	m.queuePane.cursor.cursor = queueRowOf(1)
	other := m.trackDetail(40, 20)
	if got := plain(strings.Join(other, "\n")); strings.Contains(got, knob) {
		t.Errorf("detail = %q, want no playhead on a track that is not playing", got)
	}
	if len(other) != len(m.trackDetail(40, 20)) || len(other) != len(strings.Split(playing, "\n")) {
		t.Errorf("the panel is %d rows without the playhead and %d with it",
			len(other), len(strings.Split(playing, "\n")))
	}
}

// The picture keeps its own shape inside the box the layout gives it, so it can
// come out shorter. Nothing beside it may reach past its foot.
func TestQueueDetailStaysAboveTheArtworksFoot(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.ps = &player.State{TrackID: "a", Playing: true, Duration: 2 * time.Minute}
	m.width, m.height = 200, 45
	m.resize()

	// On a track with something to say about it, or there is nothing beside the
	// picture to overrun its foot.
	m.queuePane.cursor.cursor = queueRowOf(0)

	l := m.layout()
	if l.artRows >= l.artHeight {
		t.Skipf("the picture fills its box at this cell size (%d rows), so there is no foot to overrun", l.artRows)
	}
	short := l.artRows

	block := m.queueBlock(l, l.bodyHeight)
	// The track's own name, which is the one thing the panel always carries.
	var above bool
	for i := range short {
		above = above || strings.Contains(plain(block[i]), m.queue[0].Title)
	}
	if !above {
		t.Fatal("the panel is empty above the foot, so there is nothing to have overflowed")
	}
	for i := short; i < min(l.artHeight, len(block)); i++ {
		if strings.TrimSpace(plain(block[i])) != "" {
			t.Errorf("row %d is below the picture's foot at %d but carries %q", i, short, plain(block[i]))
		}
	}
}

// The playhead must not appear while the cover loads and go again once it has:
// what fits is decided by the box and the cell, not by whether a picture has
// arrived yet.
func TestQueuePlayheadDoesNotFlickerAsTheCoverLoads(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.ps = &player.State{TrackID: "a", Playing: true, Duration: 2 * time.Minute}
	m.width, m.height = 200, 45
	m.resize()

	l := m.layout()
	loading := len(m.trackDetail(queueDetailWidth(l), min(l.artRows, l.artHeight)))

	m.cover.art = strings.TrimRight(strings.Repeat(strings.Repeat("#", l.artWidth)+"\n", l.artRows), "\n")
	loaded := len(m.trackDetail(queueDetailWidth(l), min(l.artRows, l.artHeight)))

	if loading != loaded {
		t.Errorf("the panel is %d rows while the cover loads and %d once it has", loading, loaded)
	}
}

// Every list screen is the same act — looking down a list of tracks — and has
// to be the same composition: the cover and a detail panel across the top, the
// list across the full width below.
func TestPlaylistsUseTheSameShapeAsTheQueue(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.ps = &player.State{TrackID: "t01", Playing: true}
	m.tab = tabLibrary
	listPage, err := p.PlaylistsPage(t.Context(), 0)
	lists := listPage.Items
	if err != nil {
		t.Fatalf("Playlists: %v", err)
	}
	m.library.playlists = lists
	m.width, m.height = 200, 45
	m.resize()

	l := m.layout()
	block := m.libraryPaneView(l, l.bodyHeight)
	if len(block) != l.bodyHeight {
		t.Fatalf("the block is %d rows, want the whole body of %d", len(block), l.bodyHeight)
	}

	// The panel beside the cover describes what the cursor rests on.
	head := plain(strings.Join(block[:l.artRows], "\n"))
	if !strings.Contains(head, lists[0].Name) || !strings.Contains(head, "Tracks") {
		t.Errorf("the panel = %q, want the playlist under the cursor described", head)
	}

	// The list starts below the artwork and runs the whole width.
	row := plain(block[l.artHeight+3])
	if !strings.Contains(row, lists[0].Name) {
		t.Errorf("row = %q, want the list under the heading", row)
	}
	if at := strings.Index(row, lists[0].Owner); at < len(row)/3 {
		t.Errorf("the owner sits at column %d of %d, want it out where the artists are", at, len(row))
	}
}

// Inside a playlist the panel describes the track under the cursor, so the
// picture beside it has to be that track's.
func TestOpenPlaylistFollowsTheCursor(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabLibrary
	listPage, _ := p.PlaylistsPage(t.Context(), 0)
	lists := listPage.Items
	open := lists[0]
	trackPage, err := p.PlaylistTracksPage(t.Context(), open.ID, 0)
	tracks := trackPage.Items
	if err != nil || len(tracks) < 2 {
		t.Fatalf("PlaylistTracks: %v (%d tracks)", err, len(tracks))
	}
	showOpen(&m, open, tracks)

	if got := m.cursorTrack(); got == nil || got.ID != tracks[0].ID {
		t.Fatalf("cursorTrack = %v, want the first track", got)
	}
	if got := m.coverTarget(); got != tracks[0].CoverURL {
		t.Errorf("coverTarget = %q, want the track's own cover", got)
	}

	m.openMut().cursor.move(1, len(tracks))
	if got := m.cursorTrack(); got == nil || got.ID != tracks[1].ID {
		t.Errorf("cursorTrack = %v, want the panel to follow the cursor", got)
	}
}

// Enter plays the list from the track under the cursor, so the rest of it
// follows. o is the other reading: that one track and nothing after it.
func TestPlayOnlyThisTrack(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabLibrary
	listPage, _ := p.PlaylistsPage(t.Context(), 0)
	lists := listPage.Items
	open := lists[0]
	trackPage, _ := p.PlaylistTracksPage(t.Context(), open.ID, 0)
	tracks := trackPage.Items
	if len(tracks) < 2 {
		t.Fatalf("the mock playlist has %d tracks, want at least 2", len(tracks))
	}
	showOpen(&m, open, tracks)
	m.openMut().cursor.move(1, len(tracks))

	cmd, handled := m.openKey(tea.KeyPressMsg{Code: 'o', Text: "o"})
	if !handled || cmd == nil {
		t.Fatal("o did nothing inside a playlist")
	}
	runControls(cmd)

	st, err := p.State(t.Context())
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if st.TrackID != tracks[1].ID {
		t.Errorf("playing %q, want the track under the cursor %q", st.TrackID, tracks[1].ID)
	}

	// At the top level there is no track under the cursor, so the key is a
	// no-op rather than playing something the cursor is not on.
	m.closeOpen()
	if cmd, _ := m.libraryKey(tea.KeyPressMsg{Code: 'o', Text: "o"}); cmd != nil {
		t.Error("o played something from the list of playlists")
	}
}

// Adding a track to the queue is otherwise an act with no visible result, and a
// list you have been picking from is worth being able to read back.
func TestQueuedTracksAreMarkedWhereverTheyAreListed(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabLibrary
	listPage, _ := p.PlaylistsPage(t.Context(), 0)
	lists := listPage.Items
	open := lists[0]
	trackPage, _ := p.PlaylistTracksPage(t.Context(), open.ID, 0)
	tracks := trackPage.Items
	showOpen(&m, open, tracks)
	m.width, m.height = 180, 40
	m.resize()

	before := plain(m.render())
	if strings.Contains(before, queuedMark) {
		t.Fatal("something was marked as queued before anything was")
	}

	// The second track, put there by hand.
	queued := tracks[1]
	queued.Queued = true
	m.queue = []player.Track{queued, {ID: tracks[2].ID, Title: tracks[2].Title}}

	after := plain(m.render())
	// The mark stands in a column of its own, between the ordinal and the
	// title, so a list can be read down for how much of it has been picked.
	for _, line := range strings.Split(after, "\n") {
		if at := strings.Index(line, queuedMark); at >= 0 {
			if title := strings.Index(line, queued.Title); title < at {
				t.Errorf("the mark is at %d and the title at %d — want the mark ahead of it", at, title)
			}
		}
	}
	for _, line := range strings.Split(after, "\n") {
		if !strings.Contains(line, queuedMark) {
			continue
		}
		if !strings.Contains(line, queued.Title) {
			t.Errorf("row %q is marked, want only the track that was added by hand", strings.TrimSpace(line))
		}
	}
	if !strings.Contains(after, queuedMark) {
		t.Error("the track added by hand is not marked at all")
	}

	// What the playlist itself supplies is in the list too, and marking that
	// would say nothing about what was chosen.
	if m.isQueued(tracks[2].ID) {
		t.Error("a track the context supplied is marked as queued by hand")
	}
}

// The track sounding is marked in the library too, in the same column as the
// rest of the marks — and keeps its number, because a library is numbered so it
// can be counted and a missing ordinal reads as a missing track.
func TestTheLibraryMarksWhatIsPlaying(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabLibrary
	listPage, _ := p.PlaylistsPage(t.Context(), 0)
	lists := listPage.Items
	open := lists[0]
	trackPage, _ := p.PlaylistTracksPage(t.Context(), open.ID, 0)
	tracks := trackPage.Items
	showOpen(&m, open, tracks)
	m.ps = &player.State{TrackID: tracks[3].ID, Playing: true}
	m.width, m.height = 160, 40
	m.resize()

	var row string
	for _, line := range strings.Split(plain(m.render()), "\n") {
		if strings.Contains(line, tracks[3].Title) {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("the playing track is not listed at all")
	}
	if !strings.Contains(row, nowMark) {
		t.Errorf("row = %q, want the playing track marked", strings.TrimSpace(row))
	}
	if !strings.Contains(row, "4 ") {
		t.Errorf("row = %q, want it to keep its place in the list", strings.TrimSpace(row))
	}
}

// Search is the last screen that was laid out its own way. Every list is the
// same act, and three compositions for one act read as three programs.
func TestSearchUsesTheSameShapeAsTheQueue(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabSearch
	m.search.input.SetValue("queen")
	resPage, err := p.SearchPage(t.Context(), "queen", 0)
	res := resPage.Items
	if err != nil || len(res) < 2 {
		t.Fatalf("Search: %v (%d results)", err, len(res))
	}
	m.search.of(player.SearchTracks).tracks = res
	m.width, m.height = 150, 36
	m.resize()

	l := m.layout()
	block := m.searchPaneView(l, l.bodyHeight)
	if len(block) != l.bodyHeight {
		t.Fatalf("the block is %d rows, want the whole body of %d", len(block), l.bodyHeight)
	}

	// The panel beside the cover describes what the cursor rests on.
	head := plain(strings.Join(block[:l.artRows], "\n"))
	if !strings.Contains(head, res[0].Title) || !strings.Contains(head, res[0].Album) {
		t.Errorf("the panel = %q, want the result under the cursor described", head)
	}

	// The field is the heading, and what else the query matched is set against
	// it — which is the whole argument for showing one kind at a time.
	heading := plain(block[l.artHeight+1])
	if !strings.Contains(heading, "queen") || !strings.Contains(heading, "tracks") {
		t.Errorf("heading = %q, want the query and the kinds", heading)
	}

	// And the results run the whole width, artists out where they are elsewhere.
	row := plain(block[l.artHeight+3])
	if at := strings.Index(row, res[0].Artists[0]); at < len(row)/3 {
		t.Errorf("the artists sit at column %d of %d, want them out with the others", at, len(row))
	}
}

// Lists arrive fifty at a time and are read by scrolling, so scrolling is the
// only signal there is that more is wanted. Nobody should have to know the list
// was ever cut.
func TestScrollingSendsForTheNextPage(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSearch
	m.width, m.height = 120, 36
	m.resize()

	// A page that says there is more behind it.
	m.search.input.SetValue("queen")
	var tm tea.Model = m
	tm, _ = tm.Update(msg.SearchResults{
		Seq: m.search.seq, Query: "queen", Matched: true,
		Results: player.Results{Tracks: player.Page[player.Track]{
			Items: make([]player.Track, 12), More: true, Next: 12,
		}},
	})
	got := tm.(Model)
	if !got.search.current().pages.more || got.search.current().pages.next != 12 {
		t.Fatalf("the page's answer was not kept: %+v", got.search.current().pages)
	}

	// Near the top, nothing is asked for.
	if cmd := got.readAhead(); cmd != nil {
		t.Error("a fetch went out with the cursor at the top of the list")
	}

	// Near the end, the next page is sent for — once, however many keys follow.
	got.search.current().cursor.cursor = 11
	if cmd := got.readAhead(); cmd == nil {
		t.Fatal("the cursor reached the end of the list and nothing was fetched")
	}
	if cmd := got.readAhead(); cmd != nil {
		t.Error("a second fetch went out while the first was still in flight")
	}
}

// A later page is added to what is already read; only the first replaces it.
// Reading past fifty must not throw the reader back to the top.
func TestALaterPageIsAppended(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSearch
	m.search.of(player.SearchTracks).tracks = make([]player.Track, 12)
	m.search.of(player.SearchTracks).cursor.cursor = 11

	var tm tea.Model = m
	tm, _ = tm.Update(msg.SearchResults{
		Seq: m.search.seq, Query: "queen", Matched: true, Kind: player.SearchTracks,
		Offset:  12,
		Results: player.Results{Tracks: player.Page[player.Track]{Items: make([]player.Track, 8)}},
	})

	got := tm.(Model)
	if len(got.search.current().tracks) != 20 {
		t.Errorf("%d results after the second page, want the twelve plus the eight", len(got.search.current().tracks))
	}
	if got.search.current().cursor.cursor != 11 {
		t.Errorf("the cursor moved to %d when the page arrived, want it left alone", got.search.current().cursor.cursor)
	}
}

// A player gathers verbs faster than it has keys for them, and a key that works
// on one screen only is a key nobody finds.
func TestTheActionsMenuOffersWhatTheScreenAllows(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.queuePane.cursor.cursor = queueRowOf(0)
	m.width, m.height = 120, 36
	m.resize()

	if !m.openActions() {
		t.Fatal("the menu would not open over a track")
	}
	labels := func() string {
		var out []string
		for _, v := range m.actions.verbs {
			out = append(out, v.label)
		}
		return strings.Join(out, " | ")
	}
	if got := labels(); !strings.Contains(got, "Remove from the queue") {
		t.Errorf("on the queue the menu offers %q, want removing", got)
	}
	if !strings.Contains(plain(m.render()), "Copy the Spotify link") {
		t.Error("the menu is not on screen")
	}

	// The same track in the library cannot be taken out of a queue it is not in.
	lib := New(player.NewMock(), nil, defaultTestCell)
	lib.tab = tabLibrary
	showOpen(&lib, player.Playlist{ID: "p1", Name: "one"}, []player.Track{{ID: "t1", Title: "one"}})
	if !lib.openActions() {
		t.Fatal("the menu would not open in the library")
	}
	for _, v := range lib.actions.verbs {
		if strings.Contains(v.label, "Remove from the queue") {
			t.Error("the library offers removing from a queue the track is not in")
		}
	}
}

// While the menu is up it answers every key: nothing underneath may act on a
// screen that is not the one being looked at.
func TestTheMenuHoldsTheKeyboard(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.queuePane.cursor.cursor = queueRowOf(0)
	m.openActions()
	was := m.queuePane.cursor.cursor

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	got := tm.(Model)
	if got.queuePane.cursor.cursor != was {
		t.Error("the list moved under the menu")
	}
	if got.actions.state.cursor != 1 {
		t.Errorf("the menu's own cursor is at %d, want it to have moved", got.actions.state.cursor)
	}

	// And esc puts it away without doing anything.
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if tm.(Model).actions.open {
		t.Error("esc left the menu up")
	}
}

// One kind at a time, with the counts of the others beside the query: what else
// matched is visible without spending rows on it, and the cursor and the paging
// stay what they are on every other list.
func TestSearchShowsOneKindAndCountsTheRest(t *testing.T) {
	p := player.NewMock()
	m := New(p, nil, defaultTestCell)
	m.tab = tabSearch
	m.search.input.SetValue("queen")
	res, err := p.Search(t.Context(), "queen", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	m.applySearchResults(msg.SearchResults{Seq: m.search.seq, Query: "queen", Matched: true, Results: res})
	m.width, m.height = 150, 34
	m.resize()

	heading := plain(m.render())
	for _, want := range []string{"tracks", "albums", "artists", "playlists"} {
		if !strings.Contains(heading, want) {
			t.Errorf("the heading does not count %s", want)
		}
	}

	// Tracks to begin with, and the rows are tracks.
	if m.search.kind != player.SearchTracks {
		t.Fatalf("a query opens on %q, want tracks", m.search.kind)
	}
	if !strings.Contains(plain(m.render()), res.Tracks.Items[0].Title) {
		t.Error("the first track is not listed")
	}

	// Turning the kind changes the rows and the panel, and each kind keeps its
	// own place in its own list.
	m.search.of(player.SearchTracks).cursor.cursor = 3
	m.turnSearchKind(1)
	if m.search.kind != player.SearchAlbums {
		t.Fatalf("the kind turned to %q, want albums", m.search.kind)
	}
	shown := plain(m.render())
	if !strings.Contains(shown, res.Albums.Items[0].Name) {
		t.Error("the albums are not listed after turning to them")
	}
	if m.search.current().cursor.cursor != 0 {
		t.Error("the albums opened somewhere other than the top")
	}

	m.turnSearchKind(-1)
	if m.search.current().cursor.cursor != 3 {
		t.Error("coming back to the tracks lost the place in them")
	}
}

// A kind that matched nothing is skipped rather than shown empty: the counts
// beside the query already say it matched nothing.
func TestSearchSkipsEmptyKinds(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.tab = tabSearch
	m.search.of(player.SearchTracks).tracks = []player.Track{{ID: "t", Title: "t"}}
	m.search.of(player.SearchPlaylists).playlists = []player.Playlist{{ID: "p", Name: "p"}}

	m.turnSearchKind(1)
	if m.search.kind != player.SearchPlaylists {
		t.Errorf("turned to %q, want the next kind that matched anything", m.search.kind)
	}
}

// The band above the queue folds away a block at a time, and every row it gives
// up goes to the list.
//
// The list is the point of that tab, and on an ordinary terminal the band over
// it costs a third of the rows it could be read in. Four arrangements rather
// than a switch: two blocks make four, and a key that turned one of them off
// would need a second key for the other.
func TestTheQueueFoldsTheBandAway(t *testing.T) {
	m := scopeModel(120, 44)
	m.tab = tabQueue
	ids := make([]string, 40)
	for i := range ids {
		ids[i] = fmt.Sprintf("t%02d", i)
	}
	m.queue = queueOf(0, ids...)

	var tm tea.Model = m
	rooms := []struct {
		room                queueRoom
		showsNow, showsTrace bool
	}{
		{queueRoomBoth, true, true},
		{queueRoomNow, true, false},
		{queueRoomTrace, false, true},
		{queueRoomList, false, false},
	}

	was := -1
	for _, want := range rooms {
		got := tm.(Model)
		if got.queuePane.room != want.room {
			t.Fatalf("the queue is arranged as %v, want %v", got.queuePane.room, want.room)
		}
		if got.queuePane.room.showsNow() != want.showsNow || got.queuePane.room.showsTrace() != want.showsTrace {
			t.Errorf("%v shows the player %v and the picture %v", want.room,
				got.queuePane.room.showsNow(), got.queuePane.room.showsTrace())
		}

		// Every fold gives the list rows, and the last of them gives it the
		// band's whole height.
		rows := got.visibleListRows()
		if want.room == queueRoomList && rows <= was {
			t.Errorf("with nothing above it the list has %d rows, no more than the %d it had", rows, was)
		}
		if want.room != queueRoomList && rows != was && was >= 0 {
			// The middle two keep the band, so they keep the rows: what changes
			// there is what stands in it, not how tall it is.
			t.Errorf("%v changed the list from %d rows to %d, want the band the same height", want.room, was, rows)
		}
		if want.room != queueRoomList {
			was = rows
		}

		// And the screen draws without complaint in every one of them.
		if screen := fmt.Sprint(got.View()); screen == "" {
			t.Errorf("%v drew nothing", want.room)
		}
		tm, _ = tm.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	}

	// And round: four presses come back to where it started.
	if got := tm.(Model).queuePane.room; got != queueRoomBoth {
		t.Errorf("four presses left the queue at %v, want it back where it began", got)
	}
}

// And how it is arranged comes back tomorrow: it is a way of working rather
// than a passing look.
func TestTheQueueRemembersHowItWasFolded(t *testing.T) {
	m := scopeModel(120, 44)
	m.tab = tabQueue

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	set := tm.(Model)
	if set.queuePane.room != queueRoomTrace {
		t.Fatalf("two presses left it at %v", set.queuePane.room)
	}

	// Written the way the key writes it, read the way the file is read.
	raw, err := json.Marshal(prefs{
		Scope: append([]scopeMode(nil), set.scope.modes[:]...),
		Room:  int(set.queuePane.room),
	})
	if err != nil {
		t.Fatal(err)
	}
	var back prefs
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}

	fresh := scopeModel(120, 44)
	fresh.applyPrefs(back)
	if fresh.queuePane.room != queueRoomTrace {
		t.Errorf("it came back arranged as %v, want %v", fresh.queuePane.room, queueRoomTrace)
	}

	// And a file from a version that never had it leaves the queue as it was.
	older := scopeModel(120, 44)
	older.applyPrefs(prefs{Lyrics: true})
	if older.queuePane.room != queueRoomBoth {
		t.Errorf("a file with nothing to say about it left the queue at %v", older.queuePane.room)
	}
}

// On the queue, v is which picture and c is whether there is one.
//
// Both keys could once put it away, and two ways to the same place is one too
// many — the worse of them leaving the band standing with a hole where the
// picture was.
func TestOnTheQueueTheScopeKeyNeverTurnsItOff(t *testing.T) {
	m := scopeModel(120, 44)
	m.tab = tabQueue

	var tm tea.Model = m
	for press := range int(scopeModes) + 2 {
		got := tm.(Model)
		if got.scopeMode() == scopeOff {
			t.Fatalf("after %d presses of v the queue's picture was off", press)
		}
		if !got.queuePane.room.showsTrace() {
			t.Fatalf("v folded the picture away, which is c's job")
		}
		tm, _ = tm.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	}

	// A file written before c existed can still say off, and the tab does not
	// come up with a hole in it.
	older := scopeModel(120, 44)
	older.tab = tabQueue
	older.scope.modes[tabQueue] = scopeOff
	if older.scopeMode() == scopeOff {
		t.Error("a saved off left the queue's band with nothing in it")
	}
}

// The playhead stands on the same row as the picture beside it.
//
// Two things that answer the same track, a row apart, and the eye catches it.
// The panel is centred in the band, so the bar is put at the panel's own middle
// and lands on the band's — which is where the picture is drawn from, at any
// size and whatever the facts under it come to.
func TestThePlayheadLinesUpWithThePicture(t *testing.T) {
	pop := 61
	m := queueModel(0, "a", "b")
	m.tab = tabQueue

	for _, size := range [][2]int{{160, 40}, {200, 44}, {120, 50}} {
		m.width, m.height = size[0], size[1]
		m.resize()

		m.queue[0] = player.Track{
			ID: "a", Title: "Daddy Cool", Artists: []string{"Boney M."},
			Album: "Take The Heat Off Me", Released: "1976-01-01",
			Duration: 3*time.Minute + 29*time.Second, Tempo: 125, Popularity: &pop,
		}
		m.ps = &player.State{TrackID: "a", Duration: m.queue[0].Duration, Playing: true}
		m.setProgress(3 * time.Minute)
		m.queuePane.cursor.cursor = queueRowOf(0)

		l := m.layout()
		band := min(m.listBandRows(l), size[1])
		if band < 8 {
			continue
		}
		panel := stack(m.trackDetail(queueDetailWidth(l), min(l.artRows, band)), queueDetailWidth(l), band)

		at := -1
		for i, row := range panel {
			if strings.Contains(plain(row), knob) {
				at = i
			}
		}
		if at < 0 {
			t.Fatalf("%dx%d: the playhead was not drawn at all", size[0], size[1])
		}

		// The picture's middle, which is where a centred block's own middle is.
		if want := (band - 1) / 2; at < want-1 || at > want+1 {
			t.Errorf("%dx%d: the playhead is on row %d of %d, want the band's middle at %d",
				size[0], size[1], at, band, want)
		}
	}
}

// The colour belongs to the record that is sounding, wherever the cursor is.
//
// It used to come off whichever cover was in the main slot, and on the browsing
// tabs that slot follows the cursor — so walking down a queue repainted the
// tabs, the playhead and the meter in the colours of records nobody was
// listening to.
func TestTheAccentIsTheSoundingRecordsEverywhere(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabQueue
	m.ps = &player.State{TrackID: "a", Playing: true}

	sounding := color.RGBA{R: 200, G: 40, B: 40, A: 255}
	cursor := color.RGBA{R: 40, G: 40, B: 200, A: 255}

	m.toneTook(cover.Art{Accent: sounding, HasAccent: true})
	was := m.styles.Cursor.GetForeground()

	// The cursor moves onto a record of quite another colour, and the picture
	// under it changes — the program does not.
	m.cover.accent, m.cover.hasAccent = cursor, true
	m.restyle()
	if got := m.styles.Cursor.GetForeground(); got != was {
		t.Errorf("the cursor's cover repainted the program: %v, want %v", got, was)
	}

	// And when the record changes, everything follows it.
	m.tone.asked = "b"
	m.toneTook(cover.Art{Accent: cursor, HasAccent: true})
	if got := m.styles.Cursor.GetForeground(); got == was {
		t.Error("the record changed and the colour did not")
	}

	// With nothing sounding yet, the cover on screen is what there is: a
	// program with no accent at all reads as broken rather than as quiet.
	fresh := queueModel(0, "a")
	fresh.cover.accent, fresh.cover.hasAccent = cursor, true
	fresh.restyle()
	if got, ok := fresh.toneAccent(); !ok || got != cursor {
		t.Errorf("with no record sounding the accent is %v (ok=%v), want the cover's", got, ok)
	}
}

// The sounding record's cover is asked for once, not on every frame.
func TestTheSoundingColourIsAskedForOncePerRecord(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.ps = &player.State{TrackID: "a", CoverURL: "http://example/a.jpg", Playing: true}

	if cmd := m.toneFlow(); cmd == nil {
		t.Fatal("the sounding record's colour was never sent for")
	}
	for range 5 {
		if cmd := m.toneFlow(); cmd != nil {
			t.Error("the same record's colour was sent for twice")
		}
	}

	m.ps.TrackID, m.ps.CoverURL = "b", "http://example/b.jpg"
	if cmd := m.toneFlow(); cmd == nil {
		t.Error("the record changed and its colour was not sent for")
	}
}

// What the header says about the tempo, the row and the panel say too.
//
// The row for what is sounding is built two ways: from the queue's own record of
// it where there is one, and from the player's state where there is not. The
// second built it without the tempo, so the panel said the tempo was unknown
// while the header two lines above printed it. That branch is the first listen,
// which is exactly when nothing has it written down yet.
func TestTheSoundingTempoReachesTheRowAndThePanel(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.ps = &player.State{
		TrackID: "unheard", Title: "Supergirl", Artists: []string{"Dream Chaos"},
		Album: "Supergirl", Duration: 3*time.Minute + 17*time.Second,
		Tempo: 112, Playing: true,
	}
	m.nowQueued = nil

	now, ok := m.nowPlayingRow()
	if !ok {
		t.Fatal("nothing is sounding")
	}
	if now.Tempo != 112 {
		t.Errorf("the row for what is sounding says %.0f bpm, want the 112 the header says", now.Tempo)
	}

	m.queuePane.cursor.cursor = 0
	if got := plain(strings.Join(m.trackDetail(40, 20), "\n")); !strings.Contains(got, "112 bpm") {
		t.Errorf("the panel = %q, want the measured tempo in it", got)
	}

	// And where the queue has its own record, the live measurement still wins:
	// it is fresher than whatever was written down last time.
	m.nowQueued = &player.Track{ID: "unheard", Title: "Supergirl", Tempo: 90}
	if now, _ := m.nowPlayingRow(); now.Tempo != 112 {
		t.Errorf("the row says %.0f bpm, want the live 112 over the remembered 90", now.Tempo)
	}
}

// When the band at the top is about a different record from the one the list
// begins with, a frame round it and a line down to the row it belongs to.
//
// Nothing said so before: the picture changed as the cursor moved and the only
// way to know why was to have watched it change.
func TestTheBandIsMarkedWhenItIsNotTheSoundingTrack(t *testing.T) {
	m := queueModel(0, "one", "two", "three", "four", "five")
	m.ps = &player.State{TrackID: "now", Title: "sounding", Playing: true}
	m.width, m.height = 120, 34
	m.resize()

	// On the row that is sounding, the band is about the track the list begins
	// with, and a frame round it would be pointing at what it stands on.
	m.queuePane.cursor.cursor = 0
	if got := plain(fmt.Sprint(m.View())); strings.Contains(got, pointerTL) {
		t.Error("the band was marked while the cursor was on the sounding track")
	}

	// And off it, the frame and the line.
	m.queuePane.cursor.cursor = 3
	rows := strings.Split(plain(fmt.Sprint(m.View())), "\n")

	var head, elbow int = -1, -1
	for i, row := range rows {
		if strings.Contains(row, pointerTL) {
			head = i
		}
		if strings.Contains(row, pointerElbow) {
			elbow = i
		}
	}
	if head < 0 || elbow < 0 {
		t.Fatalf("the band was not marked: frame at %d, line ends at %d", head, elbow)
	}
	if elbow <= head {
		t.Errorf("the line ends at row %d, above the frame at %d", elbow, head)
	}

	// It ends on the row under the cursor, which is the whole point of it. The
	// sounding track is the first row, so the third queue entry is the fourth.
	if !strings.Contains(rows[elbow], "three") {
		t.Errorf("the line points at %q, want the row under the cursor", strings.TrimSpace(rows[elbow]))
	}

	// It costs no rows: the same screen with and without it.
	m.queuePane.cursor.cursor = 0
	plainRows := len(strings.Split(plain(fmt.Sprint(m.View())), "\n"))
	m.queuePane.cursor.cursor = 3
	if marked := len(strings.Split(plain(fmt.Sprint(m.View())), "\n")); marked != plainRows {
		t.Errorf("the screen is %d rows marked and %d unmarked", marked, plainRows)
	}
}

// With the player folded away there is nothing up there describing anything, so
// there is nothing to point at.
func TestNothingIsMarkedWhenTheBandIsFoldedAway(t *testing.T) {
	m := queueModel(0, "one", "two", "three")
	m.ps = &player.State{TrackID: "now", Title: "sounding", Playing: true}
	m.width, m.height = 120, 34
	m.resize()
	m.queuePane.cursor.cursor = 2

	for _, room := range []queueRoom{queueRoomTrace, queueRoomList} {
		m.queuePane.room = room
		if got := plain(fmt.Sprint(m.View())); strings.Contains(got, pointerTL) {
			t.Errorf("%v marked a band that is not there", room)
		}
	}
}

// The band is as tall as the picture in it, and the heading stands a row clear
// of it.
//
// The band was the box the layout gives the artwork, and a cover keeps its own
// shape inside that box — so it came out a row shorter, and the row under it
// lined up with nothing. Drawn round the band, the frame stood clear of the very
// thing it was drawn round.
func TestTheBandIsAsTallAsTheCover(t *testing.T) {
	m := queueModel(0, "one", "two", "three")
	m.ps = &player.State{TrackID: "now", Title: "sounding", Playing: true}

	for _, size := range [][2]int{{120, 34}, {160, 44}, {200, 50}} {
		m.width, m.height = size[0], size[1]
		m.resize()

		l := m.layout()
		if l.artRows <= 0 {
			continue
		}
		if got := m.listBandRows(l); got != l.artRows {
			t.Errorf("%dx%d: the band is %d rows and the picture in it %d",
				size[0], size[1], got, l.artRows)
		}

		// And what is left over is the list's, to the row.
		block := m.queueBlock(l, l.bodyHeight)
		if len(block) != l.bodyHeight {
			t.Errorf("%dx%d: the block is %d rows, want %d", size[0], size[1], len(block), l.bodyHeight)
		}
		if got, want := m.visibleListRows(), listBodyRows(l.bodyHeight, l.artRows); got != want {
			t.Errorf("%dx%d: the keys page by %d rows and the list draws %d", size[0], size[1], got, want)
		}
	}
}

// The row over the heading is left empty on purpose: it is where a field to
// search the list will go, and reserving it now means nothing moves on the day
// it arrives.
func TestTheHeadingStandsClearOfTheBand(t *testing.T) {
	m := queueModel(0, "one", "two")
	m.ps = &player.State{TrackID: "now", Title: "sounding", Playing: true}
	m.width, m.height = 120, 34
	m.resize()

	l := m.layout()
	block := m.queueBlock(l, l.bodyHeight)
	band := m.listBandRows(l)
	if band+2 >= len(block) {
		t.Fatal("there is no room to say anything about")
	}

	for _, blank := range []int{band, band + 1} {
		if got := strings.TrimSpace(plain(block[blank])); got != "" {
			t.Errorf("row %d over the heading carries %q, want it clear", blank, got)
		}
	}
	if !strings.Contains(plain(block[band+2]), "Queue") {
		t.Errorf("the heading is not two rows under the band: %q", plain(block[band+2]))
	}
}
