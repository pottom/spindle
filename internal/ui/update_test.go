package ui

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A poll dispatched before a key press can land after it, still carrying the old
// playback flags. Inside the optimistic window that must not undo the local
// change, but new metadata should still come through. See DESIGN.md 4.2.
func TestAdoptInsideOptimisticWindow(t *testing.T) {
	m := Model{
		ps:              &player.State{Title: "old", Playing: true, Progress: time.Minute},
		progressAt:      time.Now(),
		optimisticUntil: time.Now().Add(optimisticWindow),
	}
	m.ps.Playing = false // the user just hit pause

	m.adopt(&player.State{Title: "new", Playing: true, Progress: 2 * time.Minute})

	if m.ps.Playing {
		t.Error("playback flag was overwritten inside the optimistic window")
	}
	if m.ps.Title != "new" {
		t.Errorf("metadata not adopted: title = %q, want %q", m.ps.Title, "new")
	}
	if got := m.ps.Progress; got != time.Minute {
		t.Errorf("progress anchor = %v, want %v", got, time.Minute)
	}
}

func TestAdoptAfterOptimisticWindow(t *testing.T) {
	m := Model{
		ps:              &player.State{Title: "old", Playing: false},
		optimisticUntil: time.Now().Add(-time.Second),
	}

	m.adopt(&player.State{Title: "new", Playing: true, Progress: 2 * time.Minute})

	if !m.ps.Playing {
		t.Error("server state was not adopted once the window had closed")
	}
	if got := m.ps.Progress; got != 2*time.Minute {
		t.Errorf("progress anchor = %v, want %v", got, 2*time.Minute)
	}
}

func TestNextRepeatCycles(t *testing.T) {
	want := []string{player.RepeatContext, player.RepeatTrack, player.RepeatOff}
	mode := player.RepeatOff
	for i, w := range want {
		mode = nextRepeat(mode)
		if mode != w {
			t.Errorf("step %d: mode = %q, want %q", i, mode, w)
		}
	}
}

// The position on screen is the last reported one carried forward by the clock,
// not a counter of its own. A counter and a poll are two answers to the same
// question, and they disagree by whatever the tick has drifted — which showed
// up as the playhead jumping backwards whenever the poll won.
func TestElapsedCarriesForwardFromTheAnchor(t *testing.T) {
	m := Model{
		ps:         &player.State{Playing: true, Progress: time.Minute, Duration: 5 * time.Minute},
		progressAt: time.Now().Add(-2 * time.Second),
	}

	got := m.elapsed()
	if got < 61500*time.Millisecond || got > 62500*time.Millisecond {
		t.Errorf("elapsed = %v, want about 62s", got)
	}
}

// Ticks used to add a second each. They no longer add anything, so an adopted
// snapshot cannot disagree with them.
func TestTicksDoNotAdvanceThePosition(t *testing.T) {
	var tm tea.Model = Model{
		ps:         &player.State{Playing: true, Progress: time.Minute, Duration: 5 * time.Minute},
		progressAt: time.Now(),
	}

	for range 5 {
		tm, _ = tm.Update(msg.Tick{})
	}

	if got := tm.(Model).ps.Progress; got != time.Minute {
		t.Errorf("anchor = %v, want it untouched by ticks", got)
	}
}

// Pausing stops the clock being carried forward, so the anchor has to already
// hold everything that had accumulated — otherwise the playhead drops back to
// wherever the last poll left it.
func TestPauseFreezesWhereItWas(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Playing: true, Progress: time.Minute, Duration: 5 * time.Minute}
	m.progressAt = time.Now().Add(-3 * time.Second)

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	got := tm.(Model)
	if got.ps.Playing {
		t.Fatal("space did not pause")
	}
	if elapsed := got.elapsed(); elapsed < 62500*time.Millisecond {
		t.Errorf("elapsed = %v, want the three seconds already played to be kept", elapsed)
	}
}

// Paused, it stays put however long it sits there.
func TestPausedPositionDoesNotDrift(t *testing.T) {
	m := Model{
		ps:         &player.State{Playing: false, Progress: time.Minute, Duration: 5 * time.Minute},
		progressAt: time.Now().Add(-time.Hour),
	}

	if got := m.elapsed(); got != time.Minute {
		t.Errorf("elapsed = %v, want it unchanged at 1m", got)
	}
}

// The names come off the audio stream and there is not always one, but the
// queue still knows what is playing. A blank screen over music that is plainly
// playing is the one answer that helps nobody.
func TestABlankStateIsFilledFromTheQueue(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.QueueFetched{
		Current: []player.Track{{
			ID: "now", Title: "Sandokan", Artists: []string{"Neoton Familia"},
			Album: "Sandokan", CoverURL: "http://cover", Duration: 3*time.Minute + 27*time.Second,
		}},
	})

	got := tm.(Model)
	if got.ps.Title != "Sandokan" || got.ps.TrackID != "now" {
		t.Errorf("state = %q/%q, want the queue's answer", got.ps.TrackID, got.ps.Title)
	}
	if got.ps.Duration == 0 || got.ps.CoverURL == "" {
		t.Errorf("duration = %v, cover = %q — want both borrowed too", got.ps.Duration, got.ps.CoverURL)
	}
}

// What the device does say is never overwritten by the queue: the device is the
// one that knows, and the queue lags it.
func TestTheQueueDoesNotOverwriteWhatTheDeviceSaid(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "now", Title: "the device's answer", Playing: true}

	var tm tea.Model = m
	tm, _ = tm.Update(msg.QueueFetched{
		Current: []player.Track{{ID: "now", Title: "the queue's answer"}},
	})

	if got := tm.(Model).ps.Title; got != "the device's answer" {
		t.Errorf("title = %q, want the device's", got)
	}
}

// Playing one track and then another before the first has been confirmed must
// leave the mark on the second. The answer to the overtaken request arrives
// afterwards and names a track nobody is waiting for any more.
func TestAnOvertakenRequestDoesNotMoveTheMark(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "a", Playing: true}

	// Asked for b, then for c before b was confirmed.
	m.awaitTrackChange()
	m.showTrack(&player.Track{ID: "b", Title: "b"})
	m.awaitTrackChange()
	m.showTrack(&player.Track{ID: "c", Title: "c"})

	var tm tea.Model = m
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "b", Title: "b", Playing: true}})
	if got := tm.(Model).ps.TrackID; got != "c" {
		t.Errorf("the mark moved to %q, want the track asked for last", got)
	}

	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "c", Title: "c", Playing: true}})
	if got := tm.(Model).ps.TrackID; got != "c" {
		t.Errorf("the mark is on %q once the device agrees, want c", got)
	}

	// Once it has agreed, the device is in charge again: the next track starts
	// on its own and has to be adopted.
	tm, _ = tm.Update(msg.StateFetched{State: &player.State{TrackID: "d", Title: "d", Playing: true}})
	if got := tm.(Model).ps.TrackID; got != "d" {
		t.Errorf("the mark is on %q, want the device's own next track", got)
	}
}

// Mid-change the queue still names the track being left, so it must not be
// borrowed from: that is what put the mark on the wrong row.
func TestTheQueueIsNotBorrowedFromMidChange(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "a", Playing: true}
	m.awaitTrackChange()

	// The device has gone quiet about what it is playing while it loads.
	m.ps = &player.State{Playing: true}
	m.nowQueued = &player.Track{ID: "a", Title: "a"}
	m.fillFromQueue()

	if m.ps.TrackID == "a" {
		t.Error("the track being left was borrowed back from the queue")
	}
}

// Two requests to play cannot be in flight at once. They are absolute — "play
// this" — so overlapping ones can be applied in either order, and the device
// ends up on whichever was asked for first. A run of presses collapses to two
// requests, and the last press is the one that sounds.
func TestOnlyOnePlayRequestAtATime(t *testing.T) {
	sent := make(chan string, 8)
	m := New(&recordingPlayer{Player: player.NewMock(), played: sent}, nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "a", Playing: true}
	m.tab = tabPlaylists

	ask := func(id string) tea.Cmd {
		return m.startPlay(playRequest{
			action: "play track",
			track:  player.Track{ID: id, Title: id},
			call:   func(ctx context.Context, p player.Player) error { return p.PlayTrack(ctx, id) },
		})
	}

	first := ask("b")
	ask("c")
	ask("d") // c is overtaken before it ever goes out

	if got := m.ps.TrackID; got != "d" {
		t.Errorf("the mark is on %q, want the track asked for last", got)
	}
	if len(sent) != 0 {
		t.Fatal("a request went out before the model ran the command")
	}

	// The first goes out on its own; the answer releases the last.
	done := runPlay(t, first)
	if got := <-sent; got != "b" {
		t.Errorf("first request was for %q, want b", got)
	}

	var tm tea.Model = m
	_, cmd := tm.Update(done)
	if cmd == nil {
		t.Fatal("the answer released nothing, so the last press never went out")
	}
	runPlay(t, cmd)
	if got := <-sent; got != "d" {
		t.Errorf("second request was for %q, want the last press", got)
	}
	if len(sent) != 0 {
		t.Errorf("%d further requests went out, want the overtaken one dropped", len(sent))
	}
}

// runPlay runs the commands a play request returned and hands back its answer.
func runPlay(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	var out tea.Msg
	var run func(tea.Cmd)
	run = func(c tea.Cmd) {
		if c == nil {
			return
		}
		switch result := c().(type) {
		case tea.BatchMsg:
			for _, inner := range result {
				run(inner)
			}
		case msg.PlayDone:
			out = result
		}
	}
	run(cmd)
	return out
}

// recordingPlayer notes which track was asked for, in the order the requests
// actually went out.
type recordingPlayer struct {
	player.Player
	played chan string
}

func (r *recordingPlayer) PlayTrack(ctx context.Context, trackID string) error {
	r.played <- trackID
	return nil
}

// Hammering a play key must not turn into a burst of track starts. Each one
// asks Spotify for an audio key, and asking too fast is answered with refusals
// that outlast the burst — measured against a live account, which is where the
// floor comes from.
func TestPlayRequestsAreHeldToAFloor(t *testing.T) {
	sent := make(chan string, 16)
	m := New(&recordingPlayer{Player: player.NewMock(), played: sent}, nil, defaultTestCell)
	m.ps = &player.State{TrackID: "a", Title: "a", Playing: true}

	ask := func(id string) tea.Cmd {
		return m.startPlay(playRequest{
			action: "play track",
			track:  player.Track{ID: id, Title: id},
			call:   func(ctx context.Context, p player.Player) error { return p.PlayTrack(ctx, id) },
		})
	}

	runPlay(t, ask("b")) // the first goes out at once
	if got := <-sent; got != "b" {
		t.Fatalf("first request was for %q, want b", got)
	}

	// Everything within the floor is held, however much of it there is.
	for _, id := range []string{"c", "d", "e", "f"} {
		cmd := ask(id)
		if cmd == nil {
			continue
		}
		runPlay(t, cmd)
	}
	if len(sent) != 0 {
		t.Errorf("%d requests went out inside the floor, want none", len(sent))
	}
	if m.playPending == nil || m.playPending.track.ID != "f" {
		t.Error("the last press is not the one waiting")
	}
	if got := m.ps.TrackID; got != "f" {
		t.Errorf("the mark is on %q, want the last press", got)
	}
}
