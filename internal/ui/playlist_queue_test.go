package ui

import (
	"context"
	"strconv"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
)

// libraryModel is the library tab at its top level, with the playlists loaded
// and the cursor on the first of them.
func libraryModel(t *testing.T) Model {
	t.Helper()

	p := player.NewMock()
	page, err := p.PlaylistsPage(context.Background(), 0)
	if err != nil {
		t.Fatalf("PlaylistsPage: %v", err)
	}

	m := New(p, nil, defaultTestCell)
	m.tab = tabLibrary
	m.library.playlists = page.Items
	m.width, m.height = 100, 40
	m.resize()
	return m
}

// A whole playlist goes to the back of the queue in one order, behind whatever
// was already waiting there. One order rather than one request per track: a
// playlist is long enough that appending it a track at a time is how the rate
// limiter is met halfway through.
func TestQueueingAPlaylistSendsOneOrder(t *testing.T) {
	m := libraryModel(t)
	sent := make(chan []string, 1)
	m.player = recordingEditor{Player: m.player, sent: sent}

	pl := m.library.playlists[0]
	want, err := listTrackIDs(context.Background(), m.player, openPlaylist, pl.ID)
	if err != nil {
		t.Fatalf("listTrackIDs: %v", err)
	}
	if len(want) < 2 {
		t.Fatalf("the mock playlist has %d tracks, want a few to queue", len(want))
	}

	queueListCmd(m.player, openPlaylist, pl.ID, []string{"waiting"})()

	got := <-sent
	if len(got) != len(want)+1 || got[0] != "waiting" {
		t.Fatalf("SetQueue(%v), want the track already waiting and then the playlist", got)
	}
	for i, id := range want {
		if got[i+1] != id {
			t.Errorf("SetQueue()[%d] = %q, want %q — the playlist keeps its order", i+1, got[i+1], id)
		}
	}
}

// The key says what it does on the tracks inside a playlist, and it has to mean
// the same thing on the playlists themselves: the cursor is on a whole list, so
// the whole list is what is queued.
func TestTheQueueKeyTakesAWholePlaylist(t *testing.T) {
	m := libraryModel(t)
	name := m.library.playlists[0].Name

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if cmd == nil {
		t.Fatal("a produced no command on a playlist")
	}
	if said := tm.(Model).said; said != "Adding "+name+" to the queue" {
		t.Errorf("said = %q, want it to name the playlist being queued", said)
	}
}

// The menu over a playlist offers what can be done to all of it, and says which
// list it is about.
func TestTheMenuOverAPlaylistActsOnAllOfIt(t *testing.T) {
	m := libraryModel(t)
	if !m.openActions() {
		t.Fatal("no menu over a playlist")
	}
	if m.actions.title != m.library.playlists[0].Name {
		t.Errorf("the menu is headed %q, want the playlist under the cursor", m.actions.title)
	}

	var labels []string
	for _, v := range m.actions.verbs {
		labels = append(labels, v.label)
	}
	if len(labels) != 3 {
		t.Fatalf("the menu offers %v, want playing it, queueing it and its link", labels)
	}
}

// A long playlist is not the whole queue: what is taken is capped, and it is
// taken from the top rather than from wherever the paging stopped.
func TestQueueingALongPlaylistStopsAtTheCap(t *testing.T) {
	ids, err := listTrackIDs(context.Background(), endlessPlaylist{}, openPlaylist, "p")
	if err != nil {
		t.Fatalf("listTrackIDs: %v", err)
	}
	if len(ids) != enqueueMost {
		t.Errorf("took %d tracks, want the cap of %d", len(ids), enqueueMost)
	}
	if ids[0] != "t0" {
		t.Errorf("the first track queued is %q, want the first of the playlist", ids[0])
	}
}

// endlessPlaylist answers with a page of tracks and always has more.
type endlessPlaylist struct{ player.Player }

func (endlessPlaylist) PlaylistTracksPage(_ context.Context, _ string, offset int) (player.Page[player.Track], error) {
	const size = 50
	items := make([]player.Track, 0, size)
	for i := range size {
		items = append(items, player.Track{ID: "t" + strconv.Itoa(offset+i)})
	}
	return player.Page[player.Track]{Items: items, More: true, Next: offset + size}, nil
}
