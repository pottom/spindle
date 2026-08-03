package ui

import (
	"context"
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

func queueModel(handAdded int, ids ...string) Model {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Playing: true}
	m.tab = tabQueue
	m.queue = queueOf(handAdded, ids...)
	return m
}

// Removing a track must take it off the screen at once. Waiting for the round
// trip would make every x look like it had missed.
func TestDropRemovesTheTrackImmediately(t *testing.T) {
	m := queueModel(2, "a", "b", "c")
	m.queuePane.cursor.cursor = 0

	if cmd := m.dropQueued(); cmd == nil {
		t.Fatal("dropQueued() = nil, want the edit to be sent")
	}
	if len(m.queue) != 2 || m.queue[0].ID != "b" {
		t.Errorf("queue = %v, want a gone", ids(m.queue))
	}
}

// The context tracks are not the queue's to edit: they belong to whatever album
// or playlist is playing.
func TestDropRefusesContextTracks(t *testing.T) {
	m := queueModel(1, "a", "b", "c")
	m.queuePane.cursor.cursor = 1

	if cmd := m.dropQueued(); cmd != nil {
		t.Error("dropQueued() edited a track that came from the context")
	}
	if len(m.queue) != 3 {
		t.Errorf("queue = %v, want it untouched", ids(m.queue))
	}
}

func TestMoveSwapsWithinTheQueuedBlock(t *testing.T) {
	m := queueModel(3, "a", "b", "c", "d")
	m.queuePane.cursor.cursor = 0

	if cmd := m.moveQueued(1); cmd == nil {
		t.Fatal("moveQueued(1) = nil, want the edit to be sent")
	}
	if got := ids(m.queue); got[0] != "b" || got[1] != "a" {
		t.Errorf("queue = %v, want a and b swapped", got)
	}
	// The cursor follows the track, or a second press would move a different one.
	if m.queuePane.cursor.cursor != 1 {
		t.Errorf("cursor = %d, want it to follow the track to 1", m.queuePane.cursor.cursor)
	}
}

// A hand-queued track cannot be pushed past the context: the two halves keep
// their own order, and a move that looks like it did nothing is worse than one
// that is refused.
func TestMoveStopsAtTheContextBoundary(t *testing.T) {
	m := queueModel(1, "a", "b", "c")
	m.queuePane.cursor.cursor = 0

	if cmd := m.moveQueued(1); cmd != nil {
		t.Error("moveQueued(1) crossed into the context tracks")
	}
	if got := ids(m.queue); got[0] != "a" {
		t.Errorf("queue = %v, want it untouched", got)
	}
}

// Only the hand-queued ids are sent: the daemon replaces that list wholesale,
// so including the context would make it queue the album a second time.
func TestOnlyQueuedTracksAreSent(t *testing.T) {
	sent := make(chan []string, 1)
	m := queueModel(2, "a", "b", "c")
	m.player = recordingEditor{Player: m.player, sent: sent}
	m.queuePane.cursor.cursor = 0

	cmd := m.dropQueued()
	if cmd == nil {
		t.Fatal("dropQueued() = nil")
	}
	cmd()

	got := <-sent
	if len(got) != 1 || got[0] != "b" {
		t.Errorf("SetQueue(%v), want only the remaining hand-queued track", got)
	}
}

// Pressing enter on a queue entry jumps to it, keeping whatever it belongs to.
func TestEnterPlaysFromTheQueue(t *testing.T) {
	m := queueModel(1, "a", "b")
	m.queuePane.cursor.cursor = 1

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
	sent chan []string
}

func (r recordingEditor) SetQueue(_ context.Context, ids []string) error {
	r.sent <- ids
	return nil
}

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

	out := m.render()
	if !strings.Contains(out, "Queue") {
		t.Error("render() has no queue heading")
	}
	if !strings.Contains(out, queuedMark) {
		t.Error("render() does not mark the hand-queued track")
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
	for _, want := range []string{"Album", "Released", "Track", "Length", "Source"} {
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
