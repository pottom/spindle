package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/pottom/spindle/internal/player"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/msg"
)

// The library holds three lists and shows one at a time. A kind nobody has
// asked for costs no request until it is switched to.
func TestTheLibraryTurnsBetweenItsKinds(t *testing.T) {
	m := likedModel(t)
	if m.library.kind != libraryPlaylists {
		t.Fatalf("the library opens on %v, want the playlists", m.library.kind)
	}
	if len(m.library.albums) != 0 {
		t.Error("the saved albums were fetched before anybody asked for them")
	}

	var tm tea.Model = m
	tm, cmd := tm.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	got := tm.(Model)
	if got.library.kind != libraryAlbums {
		t.Fatalf("ctrl+t landed on %v, want the albums", got.library.kind)
	}
	if cmd == nil {
		t.Fatal("switching to a list never read asked for nothing")
	}

	// The answer fills that kind and leaves the others alone.
	albums, err := got.player.SavedAlbums(context.Background(), 0)
	if err != nil {
		t.Fatalf("SavedAlbums: %v", err)
	}
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryAlbums), Albums: albums.Items, More: albums.More, Next: albums.Next})
	got = tm.(Model)
	if got.library.count() != len(albums.Items) {
		t.Errorf("the albums list holds %d, want %d", got.library.count(), len(albums.Items))
	}
	if len(got.library.playlists) == 0 {
		t.Error("switching kinds threw the playlists away")
	}

	// Each kind keeps its own cursor, so coming back lands where it was left.
	got.library.cursors[libraryAlbums].move(1, got.library.count())
	got.library.kind = libraryPlaylists
	if got.library.cursor().cursor != 0 {
		t.Error("the playlists' cursor moved when the albums' did")
	}
}

// A saved album opens like any other, and lands on the same page a search
// result would.
func TestOpeningASavedAlbum(t *testing.T) {
	m := likedModel(t)
	albums, err := m.player.SavedAlbums(context.Background(), 0)
	if err != nil || len(albums.Items) == 0 {
		t.Fatalf("SavedAlbums: %v (%d albums)", err, len(albums.Items))
	}
	m.library.kind = libraryAlbums
	m.library.albums = albums.Items

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	page := tm.(Model).open()
	if page == nil || page.kind != openAlbum || page.id != albums.Items[0].ID {
		t.Fatalf("enter opened %v, want the album under the cursor", page)
	}
}

// The library's kinds are a tab bar of their own: each named, with how much of
// it has been read, and the one on screen underlined.
func TestTheLibraryNamesItsKinds(t *testing.T) {
	m := likedModel(t)
	labels, rule := m.libraryKinds()
	strip := plain(labels)

	for _, want := range []string{"playlists", "albums", "artists", "recent"} {
		if !strings.Contains(strip, want) {
			t.Errorf("the strip reads %q, want it to name %q", strip, want)
		}
	}
	if !strings.Contains(strip, "playlists "+itoa(len(m.library.playlists))) {
		t.Errorf("the strip reads %q, want the number of playlists read so far", strip)
	}

	// The rule stands under the one on screen and nowhere else, the way the
	// tabs across the top of the screen are marked.
	under := plain(rule)
	if strings.TrimSpace(under) == "" {
		t.Fatalf("nothing is underlined: %q", under)
	}
	if at := strings.Index(under, meterFull); at != strings.Index(strip, "playlists") {
		t.Errorf("the rule starts at column %d and the kind on screen at %d",
			at, strings.Index(strip, "playlists"))
	}
	if strings.Count(under, meterFull) != lipgloss.Width(plain(m.library.kind.String()))+2 {
		t.Errorf("the rule is %d cells long, want the label's width",
			strings.Count(under, meterFull))
	}

	// And the heading that used to say Library is gone: the tab is called
	// library already.
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	if screen := plain(fmt.Sprint(m.View())); strings.Contains(screen, "Library") {
		t.Error("the wall still says Library over a tab called library")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

// The history is one of the library's lists, and it is of tracks: a row there
// plays rather than opens, and the queue survives it.
func TestTheHistoryPlaysATrackAndKeepsTheQueue(t *testing.T) {
	m := likedModel(t)
	recent, err := m.player.RecentlyPlayed(context.Background(), recentMost)
	if err != nil || len(recent) < 2 {
		t.Fatalf("RecentlyPlayed: %v (%d tracks)", err, len(recent))
	}

	m.library.kind = libraryRecent
	m.library.recent = recent
	m.library.cursors[libraryRecent].move(1, len(recent))

	if got := m.cursorTrack(); got == nil || got.ID != recent[1].ID {
		t.Fatalf("cursorTrack = %v, want the row under the cursor", got)
	}

	cmd, handled := m.libraryKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !handled || cmd == nil {
		t.Fatal("enter did nothing on the history")
	}
	runControls(cmd)

	q, err := m.player.Queue(context.Background())
	if err != nil {
		t.Fatalf("Queue: %v", err)
	}
	if q.Current == nil || q.Current.ID != recent[1].ID {
		t.Errorf("playing %v, want the track that was chosen", q.Current)
	}
}

// The history is not a page: Spotify keeps a fixed few and walks them by
// timestamp, so what arrives replaces what was there rather than adding to it.
func TestTheHistoryIsNotPaged(t *testing.T) {
	m := likedModel(t)
	recent, _ := m.player.RecentlyPlayed(context.Background(), recentMost)

	var tm tea.Model = m
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryRecent), Tracks: recent})
	tm, _ = tm.Update(msg.LibraryFetched{Kind: int(libraryRecent), Tracks: recent})

	if got := tm.(Model).library.countOf(libraryRecent); got != len(recent) {
		t.Errorf("the history holds %d rows after two answers, want %d", got, len(recent))
	}
}

// A request that failed is not a request still in flight. Every list marks
// itself as reading so a run of cursor keys cannot ask for the same page a
// dozen times; a failure that left the mark set would end the list there for
// good, and it would read as a list that simply stops.
func TestAFailedPageDoesNotEndTheList(t *testing.T) {
	m := likedModel(t)
	m.library.pages[libraryPlaylists].loading = true
	m.library.pages[libraryPlaylists].more = true
	m.library.cursors[libraryPlaylists].cursor = len(m.library.playlists) - 1

	var tm tea.Model = m
	tm, _ = tm.Update(msg.Error{Err: errors.New("network is down")})
	got := tm.(Model)

	if got.library.pages[libraryPlaylists].loading {
		t.Error("the list is still waiting for a page that failed")
	}
	if got.readAhead() == nil {
		t.Error("the list will not ask again after a failure")
	}
}

// The library is a wall of covers with no band over it, so there is nothing to
// mark: every tile is already showing its own picture.
func TestTheWallIsNotMarked(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	for i, n := range []string{"Deep House", "Chill", "Runners"} {
		m.library.playlists = append(m.library.playlists,
			player.Playlist{ID: fmt.Sprintf("p%d", i), Name: n, Owner: "pottom", Tracks: 30})
	}

	// The wall's own tiles are ringed with the same pen, so what says a band was
	// marked is the tee the line down to a row hangs from — which only the
	// bracket round a band draws.
	if got := plain(fmt.Sprint(m.View())); strings.Contains(got, pointerTee) {
		t.Error("the wall was marked as though it had a band")
	}

	// What is opened from it is a record with its tracks under it, and that is
	// marked like any other.
	m.stack = append(m.stack, openPage{kind: openPlaylist, id: "p0", name: "Deep House",
		tracks: []player.Track{{ID: "t0", Title: "One"}, {ID: "t1", Title: "Two"}}})
	m.stack[0].cursor.cursor = 1
	if got := plain(fmt.Sprint(m.View())); !strings.Contains(got, pointerTee) {
		t.Error("a page opened from the library was not marked")
	}
}

// And so is the search's, for the same reason: the cover up there is whatever
// the cursor has just moved onto, and without the mark the picture changes and
// nothing says why.
//
// It was not marked while the field was the screen's heading — a bracket round
// the results while the question was still being written was an answer arriving
// early. The field has a box of its own now, and the band underneath is nothing
// but the row the cursor is on.
func TestTheSearchBandIsMarked(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabSearch
	m.search.of(player.SearchTracks).tracks = []player.Track{
		{ID: "t1", Title: "One"}, {ID: "t2", Title: "Two"},
	}
	m.search.input.SetValue("one")
	m.resize()

	// The tee is what the line down to the row hangs from, as it is on the wall.
	if got := plain(fmt.Sprint(m.View())); !strings.Contains(got, pointerTee) {
		t.Error("the search's band was not marked")
	}
}

// The wall is walked across as well as down, and it scrolls a whole row at a
// time: a row of covers split across two screenfuls is not a row any more.
func TestTheWallIsWalkedInTwoDirections(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()
	for i := range 24 {
		m.library.playlists = append(m.library.playlists,
			player.Playlist{ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom"})
	}

	l := m.layout()
	g := m.libraryShape(l, l.bodyHeight)
	if g.cols < 2 || g.rows < 1 {
		t.Fatalf("the wall is %d by %d", g.cols, g.rows)
	}

	press := func(name string, code rune) {
		t.Helper()
		if !m.libraryGridKey(tea.KeyPressMsg{Code: code}) {
			t.Fatalf("the wall did not take %s", name)
		}
	}
	at := func() int { return m.library.cursors[m.library.kind].cursor }

	press("right", tea.KeyRight)
	if at() != 1 {
		t.Errorf("right moved to %d, want the next tile", at())
	}
	press("down", tea.KeyDown)
	if want := 1 + g.cols; at() != want {
		t.Errorf("down moved to %d, want %d — a row down", at(), want)
	}
	press("left", tea.KeyLeft)
	if want := g.cols; at() != want {
		t.Errorf("left moved to %d, want %d", at(), want)
	}
	press("up", tea.KeyUp)
	if at() != 0 {
		t.Errorf("up moved to %d, want the first tile", at())
	}

	// The window scrolls by rows: whatever is at the top of the screen is the
	// start of a row.
	state := &m.library.cursors[m.library.kind]
	state.cursor = len(m.library.playlists) - 1
	from, _ := state.gridWindow(len(m.library.playlists), g)
	if from%g.cols != 0 {
		t.Errorf("the wall starts at tile %d, which is %d into a row", from, from%g.cols)
	}
}

// A list asked for again keeps the cursor on the thing it was on, wherever that
// has moved to. A refresh that sent the reader back to the top would make a
// library unreadable for longer than the refresh takes.
func TestARefreshKeepsTheCursorWhereItWas(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()

	lists := []player.Playlist{
		{ID: "p0", Name: "Deep House"}, {ID: "p1", Name: "Chill"}, {ID: "p2", Name: "Runners"},
	}
	m.library.adopt(msg.LibraryFetched{Kind: int(libraryPlaylists), Playlists: lists}, player.Playlist{ID: likedID})
	m.library.cursors[libraryPlaylists].moveTo(2, m.library.count()) // "Chill", after the liked row

	was := m.library.idAt(libraryPlaylists, m.library.cursors[libraryPlaylists].cursor)
	if was == "" {
		t.Fatal("the cursor is on nothing to begin with")
	}

	// The same page again, with something new at the front: the cursor follows
	// what it was on rather than staying at its index.
	m.library.adopt(msg.LibraryFetched{
		Kind:      int(libraryPlaylists),
		Playlists: append([]player.Playlist{{ID: "pNew", Name: "Made on the phone"}}, lists...),
	}, player.Playlist{ID: likedID})

	if now := m.library.idAt(libraryPlaylists, m.library.cursors[libraryPlaylists].cursor); now != was {
		t.Errorf("after the refresh the cursor is on %q, want %q", now, was)
	}

	// And something that has gone leaves it at the top rather than pointing at
	// whatever slid into its place.
	m.library.adopt(msg.LibraryFetched{Kind: int(libraryPlaylists), Playlists: lists[:1]},
		player.Playlist{ID: likedID})
	if got := m.library.cursors[libraryPlaylists].cursor; got != 0 {
		t.Errorf("the cursor is at %d after what it was on went, want the top", got)
	}
}

// A window of a different size divides the wall into tiles of a different size,
// and a cover rendered for the old one is the wrong picture. The resize has to
// send for them again, or some never arrive at all.
func TestTheWallAsksAgainWhenTheWindowChanges(t *testing.T) {
	m := New(player.NewMock(), cover.NewLoader(cover.NewHalfblock(defaultTestCell), nil), defaultTestCell)
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 12 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i),
			CoverURL: fmt.Sprintf("http://example.invalid/%d.jpg", i),
		})
	}

	if cmd := m.syncGridCovers(); cmd == nil {
		t.Fatal("the wall sent for no covers at all")
	}
	was := m.libraryShape(m.layout(), m.layout().bodyHeight)

	// Wider: bigger tiles, and every cover held is for the old size.
	var tm tea.Model = m
	tm, cmd := tm.Update(tea.WindowSizeMsg{Width: 220, Height: 50})
	m = tm.(Model)
	if now := m.libraryShape(m.layout(), m.layout().bodyHeight); now.tileW == was.tileW {
		t.Skipf("the tiles are %d cells wide at both sizes", now.tileW)
	}
	if cmd == nil {
		t.Fatal("the resize sent for nothing")
	}

	// What is held now is asked for at the new size.
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	for id, tile := range m.tiles {
		if tile.width != g.boxW || tile.height != g.boxH {
			t.Errorf("%s is held at %dx%d, want the new %dx%d",
				id, tile.width, tile.height, g.boxW, g.boxH)
			break
		}
	}
}

// The words under a tile start where its picture starts. The cursor's mark
// stands beside them in the air to the left, not in front of them.
func TestTheWallsNamesLineUpWithItsPictures(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 8 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom",
		})
	}
	// A picture, so there is an edge to line up with.
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	m.tiles = map[string]coverState{"p0": tileArt(g)}

	// In cells rather than in bytes: the cursor's mark is three bytes and one
	// column, which is exactly the difference this is looking for.
	column := func(row, of string) int {
		at := strings.Index(row, of)
		if at < 0 {
			return -1
		}
		return lipgloss.Width(row[:at])
	}

	rows := strings.Split(plain(fmt.Sprint(m.View())), "\n")
	art, name := -1, -1
	for _, row := range rows {
		if art < 0 && strings.Contains(row, "###") {
			art = column(row, "#")
		}
		if art >= 0 && name < 0 && strings.Contains(row, "List 0") {
			name = column(row, "List 0")
		}
	}
	if art < 0 || name < 0 {
		t.Fatalf("the picture is at %d and the name at %d", art, name)
	}
	if art != name {
		t.Errorf("the picture starts at column %d and its name at %d", art, name)
	}

	// And the ring round the tile under the cursor stands in the air to the left
	// of it rather than over it.
	for _, row := range rows {
		if at := column(row, pointerV); at >= 0 {
			if at >= name {
				t.Errorf("the frame is at column %d, want it left of the name at %d", at, name)
			}
			return
		}
	}
	t.Error("the tile under the cursor is not framed")
}

// The gaps between tiles are what the frame is drawn into, so a frame never
// stands on a picture and the wall does not move when the cursor arrives.
func TestTheFrameFitsInTheAirBetweenTiles(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 12 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom",
		})
	}

	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	art := map[string]coverState{}
	for i := range 12 {
		art[fmt.Sprintf("p%d", i)] = tileArt(g)
	}
	m.tiles = art

	// Every picture on screen is whole: the frame took none of their cells.
	whole := func() int {
		screen := plain(fmt.Sprint(m.View()))
		if !strings.Contains(screen, pointerV) {
			t.Fatal("no tile is framed")
		}
		return strings.Count(screen, "#")
	}

	m.library.cursors[libraryPlaylists].cursor = 0
	first := whole()
	m.library.cursors[libraryPlaylists].cursor = 1
	if second := whole(); second != first {
		t.Errorf("the wall shows %d cells of picture with the first tile framed and %d with the second",
			first, second)
	}

	// And that is all of them: every visible tile's cover, entire.
	from, to := m.library.cursors[libraryPlaylists].gridWindow(len(m.library.playlists), g)
	if want := (to - from) * g.tileW * g.artRows; first != want {
		t.Errorf("the wall shows %d cells of picture, want %d — the frame ate some", first, want)
	}
}

// The tile under the cursor is marked with four corners rather than a ring, and
// each arm walks out to the screen's own ground.
func TestTheTilesCornersFadeOut(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	m.ground = color.RGBA{R: 10, G: 10, B: 14, A: 255}
	m.restyle()
	for i := range 8 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom",
		})
	}

	screen := fmt.Sprint(m.View())
	plainScreen := plain(screen)

	// Corners, and no side drawn the whole way: the two corners of a side never
	// meet.
	for _, corner := range []string{pointerTL, pointerTR, pointerElbow, pointerBR} {
		if !strings.Contains(plainScreen, corner) {
			t.Errorf("the tile under the cursor has no %s corner", corner)
		}
	}
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	if strings.Contains(plainScreen, strings.Repeat(pointerH, g.tileW)) {
		t.Error("a side of the frame is drawn the whole way across the tile")
	}

	// The corner is the accent and the far end of its arm is the ground.
	r, _, _, _ := m.styles.Accent.RGBA()
	accent := fmt.Sprintf("\x1b[38;2;%d;", r>>8)
	if !strings.Contains(screen, accent+"") {
		t.Errorf("the corner is not drawn in the accent")
	}
	if !strings.Contains(screen, "\x1b[38;2;10;10;14m"+pointerH) {
		t.Error("the arm does not end in the screen's own ground")
	}
}

// The air between a tile and its corners measures the same on all four sides.
//
// A cell is about twice as tall as it is wide, so the same number of them is not
// the same amount of air: two columns and one row is the pair that matches.
func TestTheTilesCornersStandOffItEvenly(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 8 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom",
		})
	}
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	m.tiles = map[string]coverState{"p0": tileArt(g)}

	rows := strings.Split(plain(fmt.Sprint(m.View())), "\n")
	var left, right, art0, art1 = -1, -1, -1, -1
	for _, row := range rows {
		if !strings.Contains(row, "#") || !strings.Contains(row, pointerV) {
			continue
		}
		left = strings.Index(row, pointerV)
		right = strings.LastIndex(row, pointerV)
		art0 = strings.Index(row, "#")
		art1 = strings.LastIndex(row, "#")
		break
	}
	if left < 0 || art0 < 0 {
		t.Fatal("no row of the framed tile carries both the frame and its picture")
	}

	// In cells: the uprights are multi-byte, so byte offsets would lie.
	cells := func(s string, at int) int { return lipgloss.Width(s[:at]) }
	row := rows[0]
	for _, r := range rows {
		if strings.Contains(r, "#") && strings.Contains(r, pointerV) {
			row = r
			break
		}
	}
	before := cells(row, art0) - cells(row, left) - 1
	after := cells(row, right) - cells(row, art1) - 1
	if before != after {
		t.Errorf("the picture has %d columns of air on the left and %d on the right", before, after)
	}
	if before != frameCols-1 {
		t.Errorf("the picture has %d columns of air, want %d", before, frameCols-1)
	}
}

// A tile is as wide as the picture it holds really comes out, not as wide as it
// was offered: a square cover fills whole cells only, and the columns it cannot
// use would all sit on one side of it.
func TestATileIsAsWideAsItsPicture(t *testing.T) {
	for _, cell := range []cover.CellSize{
		{Width: 10, Height: 20}, // exactly twice as tall as wide
		{Width: 9, Height: 19},  // and the awkward ones
		{Width: 7, Height: 15},
		{Width: 16, Height: 32},
	} {
		g := gridFor(150, 40, 0, cell)
		if !g.ok() {
			t.Fatalf("%+v: no wall at all", cell)
		}

		// The picture's size is the renderer's own answer, not ours.
		if cols, rows := cover.FitCells(640, 640, g.boxW, g.boxH, cell); cols != g.tileW || rows != g.artRows {
			t.Errorf("%+v: the tile is %dx%d and the renderer draws %dx%d",
				cell, g.tileW, g.artRows, cols, rows)
		}

		// And the gap is never tighter than the frame needs.
		if g.gap < tileGap {
			t.Errorf("%+v: the gap is %d cells, under the %d the frame needs", cell, g.gap, tileGap)
		}

		// The wall still fits the width it was given.
		if used := g.cols*g.tileW + (g.cols-1)*g.gap; used > 150 {
			t.Errorf("%+v: the wall comes to %d cells of 150", cell, used)
		}
	}
}

// A thing with no artwork of its own is given the drawn stand-in, so the wall
// has no holes in it.
func TestATileWithNoCoverGetsTheDrawnOne(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	m.library.playlists = []player.Playlist{
		{ID: "p0", Name: "Made here", Owner: "you"},
		{ID: "p1", Name: "From Spotify", Owner: "them", CoverURL: "http://example.invalid/a.jpg"},
	}

	tiles := m.libraryTiles()
	if len(tiles) != 2 {
		t.Fatalf("the wall holds %d tiles", len(tiles))
	}
	if tiles[0].url != cover.NoneURL {
		t.Errorf("a playlist with no cover asks for %q, want the drawn one", tiles[0].url)
	}
	if tiles[1].url == cover.NoneURL {
		t.Error("a playlist with a cover was given the drawn one")
	}
}

// tileArt is a picture of the size a tile wants, filed the way an arriving one
// is: split and squared off once, which is what the wall draws from.
func tileArt(g gridShape) coverState {
	tile := coverState{url: "u", width: g.boxW, height: g.boxH}
	var art strings.Builder
	for range g.artRows {
		art.WriteString(strings.Repeat("#", g.tileW) + "\n")
	}
	tile.took(strings.TrimSuffix(art.String(), "\n"))
	return tile
}

// The tile is laid out against the picture the renderer really draws, not
// against the box it was asked for. A cover is square and fills whole cells
// only, so a box is nearly always a column wider or a row taller than what it
// holds — and here the tile is the picture.
func TestTheTileMatchesWhatIsDrawn(t *testing.T) {
	square := image.NewRGBA(image.Rect(0, 0, 640, 640))
	for y := range 640 {
		for x := range 640 {
			square.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	for _, cell := range []cover.CellSize{
		{Width: 10, Height: 20}, {Width: 9, Height: 19}, {Width: 8, Height: 17},
		{Width: 16, Height: 32}, {Width: 16, Height: 34}, {Width: 7, Height: 15},
		{Width: 12, Height: 26}, {Width: 11, Height: 24},
	} {
		g := gridFor(140, 37, 0, cell)
		if !g.ok() {
			t.Fatalf("%+v: no wall at all", cell)
		}

		art, err := cover.NewHalfblock(cell).Render(square, g.boxW, g.boxH, 1, 0)
		if err != nil {
			t.Fatalf("%+v: %v", cell, err)
		}

		lines := strings.Split(art, "\n")
		if got := lipgloss.Width(lines[0]); got != g.tileW {
			t.Errorf("%+v: the picture is %d cells wide and the tile %d", cell, got, g.tileW)
		}
		if got := len(lines); got != g.artRows {
			t.Errorf("%+v: the picture is %d rows tall and the tile keeps %d for it",
				cell, got, g.artRows)
		}
	}
}

// A tile is square in pixels, on every cell shape a terminal reports.
//
// This is the one that matters, and it is not about our own drawing. A picture
// is handed to the terminal as a rectangle of cells and the terminal scales it
// into that rectangle: a square cover in a rectangle that is not square leaves a
// band down one side which nothing in this program can see, measure or account
// for — and which stood between the frame and the picture it marks.
func TestATileIsSquareInPixels(t *testing.T) {
	for _, cell := range []cover.CellSize{
		{Width: 10, Height: 20}, {Width: 9, Height: 19}, {Width: 8, Height: 17},
		{Width: 16, Height: 32}, {Width: 16, Height: 34}, {Width: 7, Height: 15},
		{Width: 12, Height: 26}, {Width: 11, Height: 24}, {Width: 5, Height: 19},
	} {
		for _, width := range []int{60, 100, 140, 200, 300} {
			g := gridFor(width, 37, 0, cell)
			if !g.ok() {
				continue
			}
			// Within a cell of square. Whole cells cannot land on it exactly; what
			// matters is that the band left over is under one cell rather than the
			// one or two it used to be.
			across, down := g.tileW*cell.Width, g.artRows*cell.Height
			if out := across - down; out > cell.Width || out < -cell.Height {
				t.Errorf("cell %dx%d at %d columns: the tile is %dx%d px, out by %d",
					cell.Width, cell.Height, width, across, down, out)
			}
		}
	}
}

// The corners and the uprights of a frame stand in the same two columns, and
// both stand the same distance from the picture on either side.
//
// They were written at different times and the arithmetic of the arms did not
// follow when the frame moved from one column out to two: every right-hand
// corner sat one column inside its own upright.
func TestTheFramesCornersMeetItsUprights(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 8 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom"})
	}
	g := m.libraryShape(m.layout(), m.layout().bodyHeight)
	m.tiles = map[string]coverState{"p0": tileArt(g)}

	col := func(row, of string) int {
		i := strings.Index(row, of)
		if i < 0 {
			return -1
		}
		return lipgloss.Width(row[:i])
	}
	last := func(row, of string) int {
		i := strings.LastIndex(row, of)
		if i < 0 {
			return -1
		}
		return lipgloss.Width(row[:i])
	}

	var left, right, headL, headR, footL, footR, artL, artR = -1, -1, -1, -1, -1, -1, -1, -1
	for _, row := range strings.Split(plain(fmt.Sprint(m.View())), "\n") {
		if at := col(row, pointerTL); at >= 0 {
			headL, headR = at, last(row, pointerTR)
		}
		if at := col(row, pointerElbow); at >= 0 {
			footL, footR = at, last(row, pointerBR)
		}
		if at := col(row, pointerV); at >= 0 && left < 0 {
			left, right = at, last(row, pointerV)
		}
		if at := col(row, "#"); at >= 0 && artL < 0 {
			artL, artR = at, last(row, "#")
		}
	}
	if left < 0 || headL < 0 || footL < 0 || artL < 0 {
		t.Fatalf("the frame is not all there: uprights %d/%d, head %d/%d, foot %d/%d, picture %d/%d",
			left, right, headL, headR, footL, footR, artL, artR)
	}

	for _, c := range []struct {
		name      string
		got, want int
	}{
		{"the head's left corner", headL, left},
		{"the head's right corner", headR, right},
		{"the foot's left corner", footL, left},
		{"the foot's right corner", footR, right},
	} {
		if c.got != c.want {
			t.Errorf("%s is at column %d and its upright at %d", c.name, c.got, c.want)
		}
	}

	// And the same air on both sides of the picture.
	if before, after := artL-left, right-artR; before != after {
		t.Errorf("the frame stands %d columns from the picture on the left and %d on the right",
			before, after)
	}
}

// The name of the tile under the cursor is written in the accent, like the frame
// round it: one thing is being pointed at, and both marks say so in one voice.
func TestTheSelectedTilesNameWearsTheAccent(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	m.library.playlists = []player.Playlist{
		{ID: "p0", Name: "Deep House", Owner: "you"},
		{ID: "p1", Name: "Chill", Owner: "you"},
	}

	screen := fmt.Sprint(m.View())
	want := m.styles.Cursor.Bold(true).Render("Deep House")
	if !strings.Contains(screen, want) {
		t.Errorf("the selected name is not in the accent: want %q", want)
	}
	if other := m.styles.RowSelected.Render("Deep House"); strings.Contains(screen, other) {
		t.Error("the selected name is still in the text colour")
	}
	if rest := m.styles.RowPrimary.Render("Chill"); !strings.Contains(screen, rest) {
		t.Error("the names that are not selected changed as well")
	}
}

// The field a wall is searched in stands under the heading, and the wall steps
// down for it — the same bargain the tables make. Drawn over the covers it was a
// box floating in the middle of the pictures.
func TestTheWallStandsAsideForTheField(t *testing.T) {
	m := queueModel(0, "a", "b")
	m.tab = tabLibrary
	m.width, m.height = 150, 40
	m.resize()
	for i := range 24 {
		m.library.playlists = append(m.library.playlists, player.Playlist{
			ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("List %d", i), Owner: "pottom",
		})
	}

	quiet := strings.Split(plain(fmt.Sprint(m.View())), "\n")
	wasRows := m.libraryShape(m.layout(), m.layout().bodyHeight).rows

	m.startFind()
	m.find.query = "list"
	m.refind()
	busy := strings.Split(plain(fmt.Sprint(m.View())), "\n")

	// The field is on the screen, under the heading.
	head, box := -1, -1
	for i, row := range busy {
		if head < 0 && strings.Contains(row, "playlists") {
			head = i
		}
		if head >= 0 && box < 0 && strings.Contains(row, pointerTL) {
			box = i
		}
	}
	if head < 0 || box < 0 {
		t.Fatalf("the heading is on row %d and the field on %d", head, box)
	}
	if box != head+gridChromeRows {
		t.Errorf("the field is on row %d, want %d — under the heading and its blank", box, head+gridChromeRows)
	}

	// And it covers no picture: the first tile's name has moved down by the
	// height of the box rather than being drawn over.
	rowOf := func(rows []string, of string) int {
		for i, row := range rows {
			if strings.Contains(row, of) {
				return i
			}
		}
		return -1
	}
	before, after := rowOf(quiet, "List 0"), rowOf(busy, "List 0")
	if before < 0 || after < 0 {
		t.Fatalf("the first tile is on row %d without the field and %d with it", before, after)
	}
	if after != before+finderRows {
		t.Errorf("the first tile moved from row %d to %d, want %d down", before, after, finderRows)
	}

	// The wall keeps its shape, a row shorter where the rows were tight.
	if now := m.libraryShape(m.layout(), m.layout().bodyHeight).rows; now > wasRows {
		t.Errorf("the wall holds %d rows of tiles with the field up and %d without", now, wasRows)
	}
}

// The two keys next to each other walk the library's kinds, and the search's.
// Which letters they send does not matter: a binding is matched by the key a
// press came from as well.
func TestTheBracketsWalkTheKinds(t *testing.T) {
	m := likedModel(t)
	m.width, m.height = 150, 40
	m.tab = tabLibrary
	m.resize()

	was := m.library.kind
	next, _ := m.handleKey(tea.KeyPressMsg{Code: ']', Text: "]"})
	if next.library.kind == was {
		t.Errorf("] did not move off %v", was)
	}
	back, _ := next.handleKey(tea.KeyPressMsg{Code: '[', Text: "["})
	if back.library.kind != was {
		t.Errorf("[ landed on %v, want back on %v", back.library.kind, was)
	}

	// The same key on a Hungarian keyboard, which sends something else from it.
	hungarian, _ := m.handleKey(tea.KeyPressMsg{Code: 'ő', Text: "ő", BaseCode: '['})
	if hungarian.library.kind == was {
		t.Error("the key where [ sits did not walk the kinds")
	}

	// And while a query is being typed on the search tab, they are letters.
	typing := m
	typing.tab = tabSearch
	typing.startTyping()
	after, _ := typing.handleKey(tea.KeyPressMsg{Code: ']', Text: "]"})
	if after.search.kind != typing.search.kind {
		t.Error("] changed the kind while the query was being typed")
	}
}
