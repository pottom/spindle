package ui

import (
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// A bar of marks is dealt its cast, and the notes are one of them.
//
// The drawn sets are another way of saying what a bar of music looks like, not
// a better one: the notes have been what it looks like here from the start, and
// they take their turn like everything else on this screen.
func TestTheNotesKeepTheirTurn(t *testing.T) {
	seen := map[string]int{}
	for starts := int64(0); starts < 4000; starts += 10 {
		seen[markCastFor(starts)]++
	}
	t.Logf("over 400 bars the casts came up %v (the empty one is the notes)", seen)

	if seen[""] == 0 {
		t.Error("the notes were never dealt, so a set of drawings has replaced them")
	}
	for name := range markSets {
		if seen[name] == 0 {
			t.Errorf("the %q set was never dealt", name)
		}
	}

	// And the deal is a deal: the same bar gets the same cast twice.
	for starts := int64(0); starts < 500; starts += 70 {
		if a, b := markCastFor(starts), markCastFor(starts); a != b {
			t.Fatalf("the bar at %dms was dealt %q and then %q", starts, a, b)
		}
	}
}

// The whole row at a smaller size beats part of it at a larger one.
//
// The order is what the row says — the kick at one end and the cymbals at the
// other, and the sound running between them — so four marks of seven is not a
// smaller version of the picture but a different one.
func TestTheWholeRowFitsBeforeItIsThinned(t *testing.T) {
	set, ok := markSets["band"]
	if !ok {
		t.Skip("no band set")
	}

	for _, size := range [][2]int{{200, 50}, {160, 44}, {120, 40}, {100, 30}} {
		w, rows := size[0], size[1]
		_, layout, ok := markPicture("band", w, rows)
		if !ok {
			t.Errorf("%dx%d: no row at all", w, rows)
			continue
		}
		t.Logf("%3dx%-3d %d of the %d marks, standing %d..%d of %d dots",
			w, rows, layout.Count, len(set.sizes[0].marks), layout.Tops[0], layout.Bottoms[0], rows*dotsPerCellY)

		if layout.Count < markLeast {
			t.Errorf("%dx%d: the row is down to %d marks, which is under the %d that make a row", w, rows, layout.Count, markLeast)
		}
	}
}

// The row is what the rest of the screen already knows how to read: a field of
// dots, and a layout saying which piece each dot belongs to. Everything built on
// that — the ride, the lean, the figure knocking them over — asks the layout,
// and none of it knows a drawing from a note.
func TestADrawnRowReadsLikeAnyOther(t *testing.T) {
	const w, rows = 160, 44

	grain, layout, ok := markPicture("band", w, rows)
	if !ok {
		t.Fatal("no row")
	}

	if grain.DotsX != w*dotsPerCellX || grain.DotsY != rows*dotsPerCellY {
		t.Errorf("the picture is %dx%d dots, want %dx%d", grain.DotsX, grain.DotsY, w*dotsPerCellX, rows*dotsPerCellY)
	}

	// Every mark has dots of its own, and they stand in the order they were
	// dealt: the low end at the left.
	first, last := make([]int, layout.Count), make([]int, layout.Count)
	for i := range first {
		first[i], last[i] = -1, -1
	}
	var lit int
	for y := layout.Tops[0]; y <= layout.Bottoms[0]; y++ {
		for x := range grain.DotsX {
			if grain.Lum[y*grain.DotsX+x] == 0 {
				continue
			}
			lit++
			at := layout.WordAt(x, y)
			if at < 0 || at >= layout.Count {
				t.Fatalf("a dot at %d,%d belongs to no mark (%d)", x, y, at)
			}
			if first[at] < 0 {
				first[at] = x
			}
			last[at] = x
		}
	}
	t.Logf("%d marks over %d lit dots, standing at %v", layout.Count, lit, first)

	for i := range first {
		if first[i] < 0 {
			t.Errorf("mark %d drew nothing", i)
		}
		if i > 0 && first[i] <= last[i-1] {
			t.Errorf("mark %d starts at %d, which is not past mark %d ending at %d", i, first[i], i-1, last[i-1])
		}
	}
}

// A drawn row goes up in the frame it was asked for: it is already dots, so
// there is nothing to send for and nothing to wait for. A line of type is set
// by the face off this goroutine and comes back as a message.
func TestADrawnRowNeedsNoRoundTrip(t *testing.T) {
	m := scopeModel(160, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.lyrics.forTrack, m.lyrics.missing = m.ps.TrackID, true
	m.words.forTrack = m.ps.TrackID

	// A spell whose bar is dealt a drawn set.
	var spell int64 = -1
	for at := range int64(200) {
		if markCastFor(at*wordsSpell.Milliseconds()) != "" {
			spell = at
			break
		}
	}
	if spell < 0 {
		t.Skip("no bar in two hundred was dealt a drawn set")
	}
	m.setProgress(time.Duration(spell)*wordsSpell + time.Second)

	if cmd := m.wordsGrind(); cmd != nil {
		t.Error("a drawn row sent for a picture instead of drawing itself")
	}
	if m.words.have.DotsX == 0 {
		t.Fatal("the row was not put up at all")
	}
	if m.words.where.Count < markLeast {
		t.Errorf("the row has %d marks in its layout", m.words.where.Count)
	}
	t.Logf("the bar at %s was dealt %q and went up with %d marks in the frame it was asked in",
		time.Duration(spell)*wordsSpell, markCastFor(spell*wordsSpell.Milliseconds()), m.words.where.Count)

	// And it still counts as a bar of marks, which is what everything else on
	// the screen asks.
	if !m.words.beats {
		t.Error("a drawn row does not count as a bar of marks")
	}

	// The notes still go through the face, and that costs a command.
	var notes int64 = -1
	for at := range int64(200) {
		if markCastFor(at*wordsSpell.Milliseconds()) == "" {
			notes = at
			break
		}
	}
	if notes < 0 {
		t.Skip("no bar in two hundred was dealt the notes")
	}
	m.words.text = "" // so the picture is not taken for one already held
	m.setProgress(time.Duration(notes)*wordsSpell + time.Second)
	if cmd := m.wordsGrind(); cmd == nil {
		t.Error("a row of notes did not send for a picture")
	}
}

// The marks are drawn where a line of type would be, so that what the meter is
// left is the same either way.
func TestADrawnRowStandsWhereTypeWould(t *testing.T) {
	const w, rows = 160, 44

	_, layout, ok := markPicture("band", w, rows)
	if !ok {
		t.Fatal("no row")
	}

	m := scopeModel(w, rows)
	m.words.beats, m.words.where = true, layout
	m.words.text = wordsNotes
	m.scope.beat = player.Beat{}

	dotsY := rows * dotsPerCellY
	middle := dotsY / 2
	at := (layout.Tops[0] + layout.Bottoms[0]) / 2
	t.Logf("the row stands about dot row %d of %d, and the middle is %d", at, dotsY, middle)

	if at < middle-dotsY/8 || at > middle+dotsY/8 {
		t.Errorf("the row sits at %d, want it about the middle at %d", at, middle)
	}
}
