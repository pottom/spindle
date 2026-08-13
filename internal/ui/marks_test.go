package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// A bar of marks is dealt its cast, and the notes are one of them.
//
// The drawn sets are another way of saying what a bar of music looks like, not
// a better one: the notes have been what it looks like here from the start, and
// they take their turn like everything else on this screen.
func TestTheNotesKeepTheirTurn(t *testing.T) {
	seen := map[string]int{}
	for starts := int64(0); starts < 4000; starts += 10 {
		seen[markCastFor("one", starts)]++
	}
	t.Logf("over 400 bars the casts came up %v (the empty one is the notes)", seen)

	if seen[""] == 0 {
		t.Error("the notes were never dealt, so the drawings have replaced them")
	}
	if seen[markMixed] == 0 {
		t.Error("the drawings were never dealt, so only the notes are left")
	}

	// And the deal is a deal: the same bar of the same record gets the same cast
	// twice.
	for starts := int64(0); starts < 500; starts += 70 {
		if a, b := markCastFor("one", starts), markCastFor("one", starts); a != b {
			t.Fatalf("the bar at %dms was dealt %q and then %q", starts, a, b)
		}
	}
}

// The opening bar of one record is not the opening bar of the next.
//
// A wordless bar is stamped at the top of the spell it is in, so the first half
// minute of every record is stamped nought. Dealt from the bar alone that is one
// row for the opening of every record ever played — watched by skipping through
// a list, which is the same row over and over. The record is in the deal too.
//
// What varies is the company rather than the cast. There are two casts now, the
// notes and everybody, and the crowd is dealt out of a pool of every drawing —
// so this asks the question where the answer lives, which is who came up.
func TestEveryRecordOpensWithItsOwnCompany(t *testing.T) {
	const w, rows = 200, 44
	records := []string{"3n3Ppam7vgaVa1iaRUc9Lp", "1301WleyT98MSxVHPZCA6M", "5ChkMS8OtdzJeqyybCc9R5",
		"7ouMYWpwJ422jRcDASZB7P", "0eGsygTp906u18L0Oimnem", "1lDWb6b6ieDQ2xT7ewTC3G",
		"2takcwOaAZWiXQijPHIx7B", "6habFhsOp2NvshLv26DqMb", "4u7EnebtmKWzUH433cf5Qv"}

	// The bar is stamped nought for all of them, so the record is the only thing
	// left to tell them apart.
	seen := map[string]int{}
	for i, id := range records {
		cast := markCastFor(id, 0)
		if cast == "" {
			continue // the notes, which have no company
		}
		_, layout, ok := markPicture(cast, w, rows, int64(i))
		if !ok {
			t.Fatalf("no row for %q", id)
		}
		seen[companyOf(layout)]++
	}
	t.Logf("the opening bar of nine records put up %d companies: %v", len(seen), seen)

	if len(seen) < 4 {
		t.Errorf("nine records opened with only %d companies between them: %v", len(seen), seen)
	}
}

// companyOf names a dealt row by where its marks stand, which is enough to tell
// one company from another without reaching into the drawings.
func companyOf(l msg.WordLayout) string {
	var b strings.Builder
	for i := range l.Count {
		if i < len(l.Lefts) && i < len(l.Rights) {
			fmt.Fprintf(&b, "%d-%d ", l.Lefts[i], l.Rights[i])
		}
	}
	return b.String()
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
		_, layout, ok := markPicture("band", w, rows, 7_000)
		if !ok {
			t.Errorf("%dx%d: no row at all", w, rows)
			continue
		}
		t.Logf("%3dx%-3d %d of the %d marks, standing %d..%d of %d dots",
			w, rows, layout.Count, len(set.sizes[0].marks), layout.Tops[0], layout.Bottoms[0], rows*dotsPerCellY)

		if layout.Count < markCrowdLeast {
			t.Errorf("%dx%d: the row is down to %d marks, which is under the %d that make a row", w, rows, layout.Count, markCrowdLeast)
		}
	}
}

// The row is what the rest of the screen already knows how to read: a field of
// dots, and a layout saying which piece each dot belongs to. Everything built on
// that — the ride, the lean, the figure knocking them over — asks the layout,
// and none of it knows a drawing from a note.
func TestADrawnRowReadsLikeAnyOther(t *testing.T) {
	const w, rows = 160, 44

	grain, layout, ok := markPicture("band", w, rows, 7_000)
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
	if !marksDrawn {
		t.Skip("the drawn sets are not being dealt — see marksDrawn")
	}
	m := scopeModel(160, 44)
	m.stage.on = true
	m.stage.mode = scopeWords
	m.ps.Duration = 4 * time.Minute
	m.lyrics.forTrack, m.lyrics.missing = m.ps.TrackID, true
	m.words.forTrack = m.ps.TrackID

	// A spell whose bar is dealt a drawn set.
	var spell int64 = -1
	for at := range int64(200) {
		if markCastFor(m.ps.TrackID, at*wordsSpell.Milliseconds()) != "" {
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
	if m.words.where.Count < markCrowdLeast {
		t.Errorf("the row has %d marks in its layout", m.words.where.Count)
	}
	t.Logf("the bar at %s was dealt %q and went up with %d marks in the frame it was asked in",
		time.Duration(spell)*wordsSpell, markCastFor(m.ps.TrackID, spell*wordsSpell.Milliseconds()), m.words.where.Count)

	// And it still counts as a bar of marks, which is what everything else on
	// the screen asks.
	if !m.words.beats {
		t.Error("a drawn row does not count as a bar of marks")
	}

	// The notes still go through the face, and that costs a command.
	var notes int64 = -1
	for at := range int64(200) {
		if markCastFor(m.ps.TrackID, at*wordsSpell.Milliseconds()) == "" {
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

	_, layout, ok := markPicture("band", w, rows, 7_000)
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

// The cast is dealt afresh every bar, not once a record.
//
// Every bar of marks is the same string of notes, so a picture already held was
// taken for the picture wanted and the row went up once and stayed — over a
// wordless record that is one deal in four minutes rather than one every half
// minute. Which set is up is part of what is on screen, so it belongs in that
// test.
func TestTheCastIsDealtEveryBar(t *testing.T) {
	if !marksDrawn {
		t.Skip("the drawn sets are not being dealt — see marksDrawn")
	}
	m := scopeModel(160, 44)
	m.stage.on = true
	m.stage.mode = scopeWords
	m.ps.Duration = 6 * time.Minute
	m.lyrics.forTrack, m.lyrics.missing = m.ps.TrackID, true
	m.words.forTrack = m.ps.TrackID

	seen := map[string]int{}
	for spell := range int64(12) {
		m.setProgress(time.Duration(spell)*wordsSpell + time.Second)
		m.wordsGrind()
		seen[m.words.cast]++
	}
	t.Logf("over twelve spells of one record the row was dealt %v (the empty one is the notes)", seen)

	if len(seen) < 2 {
		t.Errorf("twelve spells of one record only ever showed %v — the deal is being asked once", seen)
	}
}

// Silence the room and the row says so.
//
// A bar of marks is what a bar of music looks like here, so a bar with nothing
// coming out of it has to look like something — the same company with their
// fingers in their ears. It is a set kept out of the deal, so it can only ever
// mean the one thing.
func TestSilencingTheRoomPutsTheirFingersInTheirEars(t *testing.T) {
	if set, ok := markSets[markHush]; !ok {
		t.Fatal("there is no hush set")
	} else if !set.apart {
		t.Error("the hush is in the deal, so it can arrive when nothing is muted")
	}

	// Not dealt to a record, at any size, ever.
	for _, tall := range markHeights() {
		for _, one := range markEveryone(tall) {
			if one.set == markHush {
				t.Fatalf("%q was in the pool at %d dots", one.name, tall)
			}
		}
	}

	// Nor walked to by the key.
	m := New(player.NewMock(), nil, defaultTestCell)
	for range len(markSets) + 2 {
		m.marksWalk()
		if m.words.picked == markHush {
			t.Fatal("the key walked to the hush")
		}
	}

	// And put up by silence, however it was reached.
	m.ps = &player.State{TrackID: "one", Duration: time.Minute, Playing: true, Volume: 60}
	if m.muted() {
		t.Error("a room at sixty reads as silenced")
	}
	m.toggleMute()
	if !m.muted() {
		t.Error("the mute key did not silence the room")
	}
	m.toggleMute()
	if m.muted() {
		t.Error("unmuting left the room silenced")
	}
	// The arrows all the way down are the same silence.
	m.setVolume(0)
	if !m.muted() {
		t.Error("turning it down to nothing did not read as silenced")
	}
	m.setVolume(20)
	if m.muted() {
		t.Error("turning it back up left the room silenced")
	}
}
