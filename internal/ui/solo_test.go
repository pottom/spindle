package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// sung is a model playing a track with the words given, at the seconds given.
func sung(at ...int) Model {
	m := scopeModel(100, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.TrackID, m.ps.Title = "sung", "A Long Song"
	m.ps.Artists = []string{"The Band"}
	m.ps.Duration = 5 * time.Minute

	for i, s := range at {
		m.lyrics.lines = append(m.lyrics.lines, player.Lyric{
			At: int64(s) * 1000, Words: "line " + string(rune('a'+i%26)),
		})
	}
	m.lyrics.forTrack, m.lyrics.synced = m.ps.TrackID, true
	return m
}

// A line is not left standing over a solo. Once the singer has left it and the
// next line is a long way off, the words give way to the marks — which is what
// the bar actually is.
func TestALineGivesWayToTheMarks(t *testing.T) {
	m := sung(10, 14, 90, 94)

	// Two lines close together: the first is still up when the second is due.
	m.setProgress(12 * time.Second)
	if lines, _ := m.wordsComing(); len(lines) == 0 || strings.HasPrefix(lines[0], "♪") {
		t.Errorf("two seconds after a line the screen has %q, want the line", lines)
	}

	// The line at fourteen seconds is followed by seventy-six of nothing.
	m.setProgress(16 * time.Second)
	if lines, _ := m.wordsComing(); len(lines) != 1 || lines[0] != "line b" {
		t.Errorf("two seconds into the solo the screen has %q, want the line it is still holding", lines)
	}

	m.setProgress(30 * time.Second)
	lines, starts := m.wordsComing()
	if len(lines) != 1 || !wordsBeats(lines[0]) {
		t.Errorf("sixteen seconds into the solo the screen has %q, want the marks", lines)
	}
	if want := (14*time.Second + soloHold).Milliseconds(); starts != want {
		t.Errorf("the bar is stamped %d, want the moment the line gave way, %d", starts, want)
	}

	// And the line coming after it takes the screen back in time to gather.
	m.setProgress(90*time.Second - wordsGather/2)
	if lines, _ := m.wordsComing(); len(lines) != 1 || lines[0] != "line c" {
		t.Errorf("just before the singer comes back the screen has %q, want the line gathering", lines)
	}
}

// The card keeps time with everything else, and stays upright while it does.
//
// It goes up into a screen where the meter, the water and the marks are all
// answering the music, so a name standing rigid in the middle of that reads as
// a picture that has stopped. It does not lean, though: two lines of type set
// large are a card, and a card at an angle is a sticker.
func TestTheCardRidesButDoesNotLean(t *testing.T) {
	m := sung(30, 45, 90, 200, 210, 225)
	m.setProgress(150 * time.Second)
	m.words.forced = time.Now()

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.9
	}
	m.scope.bands = bands

	m.words.telling, m.words.text = true, strings.Join(m.soloName(), "\n")

	ride := m.wordsRiding(3)
	t.Logf("the card rides by %v", ride)
	if ride == nil {
		t.Error("the card stood still on a loud record")
	}
	var moved bool
	for _, by := range ride {
		if by != 0 {
			moved = true
		}
	}
	if !moved {
		t.Errorf("the card rides by %v, want it keeping time", ride)
	}

	if tilt, _ := m.wordsTilting(3); tilt != nil {
		t.Errorf("the card leans by %v, want it upright", tilt)
	}
}

// The record's own moment comes once and may not come at all. The key is how it
// gets looked at on purpose: the same card, drawn the same way, wherever the
// record happens to be — and it does not put the big screen away.
func TestTheNameCanBeAskedFor(t *testing.T) {
	m := sung(30, 45, 90, 200, 210, 225)
	m.setProgress(5 * time.Second) // nowhere near a solo

	if lines, _ := m.wordsComing(); len(lines) == 1 && lines[0] == m.ps.Title {
		t.Fatal("the name was up before it was asked for")
	}

	var tm tea.Model = m
	tm, _ = tm.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	m = tm.(Model)

	if !m.stage.on {
		t.Fatal("asking what is playing put the big screen away")
	}
	if !m.soloTelling() {
		t.Fatal("the name was asked for and the model does not know it")
	}

	lines, _ := m.wordsComing()
	if len(lines) != 2 || lines[0] != m.ps.Title || lines[1] != "The Band" {
		t.Errorf("the screen has %q, want the record and the artist", lines)
	}

	// It goes again on its own, the way the record's own moment does.
	m.words.forced = time.Now().Add(-soloTells - time.Second)
	if m.soloTelling() {
		t.Error("the card asked for is still up long after it went up")
	}
}

// The card comes in the same way whether the record put it up or somebody
// asked for it.
//
// A lyric line is wound back so that it finishes gathering as it is sung, which
// is right for a line and wrong for a card: the card's moment is when it starts
// arriving, so wound back it was already there before anybody saw it move, and
// the one on the key was the livelier of the two for no reason.
func TestTheCardComesInTheSameWayTwice(t *testing.T) {
	gathering := func(m Model, telling bool) time.Duration {
		m.words.telling = telling
		var tm tea.Model = m
		tm, _ = tm.Update(msg.WordsReady{Text: "A Long Song\nThe Band", CellsX: m.width, CellsY: m.height})
		return time.Since(tm.(Model).words.since)
	}

	// Asked for in the middle of a solo, where the marks are up and a line's
	// picture would have been wound back to land on the beat.
	m := sung(30, 45, 200, 210, 225)
	m.setProgress(150 * time.Second)
	m.words.forced = time.Now()
	m.words.starts = 150_000
	itself := gathering(m, true)

	// And asked for over a line, wherever the record happens to be.
	m = sung(30, 45, 200, 210, 225)
	m.setProgress(5 * time.Second)
	m.words.forced = time.Now()
	m.words.starts = m.words.forced.UnixMilli()
	asked := gathering(m, true)

	t.Logf("in a solo the gathering was %s in, over a line %s in", itself, asked)
	if itself > wordsGather/4 {
		t.Errorf("the card was already %s into its arrival, want it starting", itself)
	}
	if asked > wordsGather/4 {
		t.Errorf("the card on the key was already %s into its arrival, want it starting", asked)
	}

	// A sung line is still wound back: it has somewhere to be.
	m = sung(30, 45, 200, 210, 225)
	m.setProgress(30 * time.Second)
	m.words.starts = 30_000
	if line := gathering(m, false); line < wordsGather/2 {
		t.Errorf("a line due now was %s into its arrival, want it nearly there", line)
	}
}

// A long name is broken rather than shrunk.
//
// Every other line on this screen is wrapped before it is set; the card was
// not, so a long title was left to be set at whatever size fitted it on one
// line — which on a narrow terminal is a row of specks.
func TestALongNameIsBrokenRatherThanShrunk(t *testing.T) {
	names := []struct{ title, artist string }{
		{"Dübörög a ház", "Emergency House"},
		{"Pattanások és szemüvegek", "Tankcsapda"},
		{"The Lamb Lies Down On Broadway - 2007 Stereo Mix", "Genesis, Peter Gabriel, Phil Collins"},
		{"Wouldn't It Be Nice (Remastered 1999 / Stereo Mix)", "The Beach Boys"},
	}

	// How tall the type comes out, which is the whole question: the same words
	// over more lines are set larger.
	tall := func(lines []string, w, rows int) int {
		img, _, ok := wordsImage(lines, w*dotsPerCellX, rows*dotsPerCellY)
		if !ok {
			return 0
		}
		top, bottom := -1, -1
		for y := range rows * dotsPerCellY {
			for x := range w * dotsPerCellX {
				if img.GrayAt(x, y).Y >= wordsLit {
					if top < 0 {
						top = y
					}
					bottom = y
					break
				}
			}
		}
		return bottom - top + 1
	}

	for _, size := range [][2]int{{60, 20}, {100, 30}, {180, 46}} {
		w, rows := size[0], size[1]
		for _, name := range names {
			m := scopeModel(100, 44)
			m.width, m.height = w, rows
			m.ps.Title, m.ps.Artists = name.title, strings.Split(name.artist, ", ")

			card := m.soloName()
			if len(card) == 0 {
				t.Fatalf("%dx%d: %q made no card", w, rows, name.title)
			}
			// Two parts, and neither is given more than its share of lines.
			if len(card) > 2*wordsCardMost {
				t.Errorf("%dx%d: %q came out as %d lines, want no more than %d",
					w, rows, name.title, len(card), 2*wordsCardMost)
			}

			straight := tall([]string{name.title, name.artist}, w, rows)
			broken := tall(card, w, rows)
			t.Logf("%3dx%-3d %-50.50q  one line each %2d dots, broken %2d", w, rows, name.title, straight, broken)

			if broken < straight {
				t.Errorf("%dx%d: %q is set at %d dots broken and %d straight, want breaking it to help",
					w, rows, name.title, broken, straight)
			}
		}
	}
}

// A picture that could not be set is not asked for over and over — but the
// refusal belongs to the size it happened at. Held without one, a name the
// screen had no room for stayed given up on after the window was made wider.
func TestARefusedPictureIsTriedAgainAtANewSize(t *testing.T) {
	m := sung(30, 45, 90, 200, 210, 225)
	m.words.forced = time.Now() // the card, whatever the record is doing

	if cmd := m.wordsGrind(); cmd == nil {
		t.Fatal("the card was never sent for")
	}
	if cmd := m.wordsGrind(); cmd != nil {
		t.Error("the same card at the same size was sent for twice")
	}

	m.width += 20
	if cmd := m.wordsGrind(); cmd == nil {
		t.Error("the screen changed shape and the card was not sent for again")
	}
}

// Whatever the name is, the card can be read.
//
// Breaking it in two is only half an answer: a podcast episode runs to ninety
// letters, and ninety letters over two lines is forty-five to a line, which on
// a wide terminal is seven dots a letter — a texture rather than a word. Past
// what another line would save, the rest is cut off and the cut is marked.
func TestACardIsAlwaysBigEnoughToRead(t *testing.T) {
	names := []string{
		"Dübörög a ház",
		"The Lamb Lies Down On Broadway - 2007 Stereo Mix",
		"Mikor a tű éri a lemezt, mi van Európában? Hogyan mutat a politikus // Fülelőbusz Podcast 131",
		"Donaudampfschifffahrtselektrizitätenhauptbetriebswerkbauunterbeamtengesellschaft",
	}

	for _, size := range [][2]int{{60, 20}, {100, 30}, {180, 46}, {240, 60}} {
		w, rows := size[0], size[1]
		for _, name := range names {
			m := scopeModel(100, 44)
			m.width, m.height = w, rows
			m.ps.Title, m.ps.Artists = name, []string{"Fülelőbusz Podcast"}

			card := m.soloName()
			if len(card) == 0 {
				t.Fatalf("%dx%d: %q made no card", w, rows, name)
			}
			if !wordsBigEnough(card, w*dotsPerCellX) {
				t.Errorf("%dx%d: %q came out too small to read: %q", w, rows, name, card)
			}
			if _, _, ok := wordsImage(card, w*dotsPerCellX, rows*dotsPerCellY); !ok {
				t.Errorf("%dx%d: %q could not be set at all", w, rows, name)
			}
			t.Logf("%3dx%-3d %q", w, rows, card)
		}
	}
}

// And what was cut says so.
func TestACutNameIsMarked(t *testing.T) {
	const name = "Mikor a tű éri a lemezt, mi van Európában? Hogyan mutat a politikus"

	if got := wordsShorten(name, 300); got != name {
		t.Errorf("a name that fits was cut to %q", got)
	}

	got := wordsShorten(name, 30)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("%q was cut without saying so", got)
	}
	if len([]rune(got)) > 30 {
		t.Errorf("%q is %d letters, want no more than 30", got, len([]rune(got)))
	}
	if !strings.HasPrefix(name, strings.TrimSuffix(got, "…")) {
		t.Errorf("%q is not the front of the name", got)
	}
	t.Logf("cut to thirty: %q", got)

	// A single word longer than the whole card is cut wherever it has got to,
	// there being no space to cut it on.
	if got := wordsShorten("Donaudampfschifffahrtselektrizitäten", 12); len([]rune(got)) > 12 {
		t.Errorf("one long word came out as %q", got)
	}
}

// The record never says its name on its own — only when it is sent for.
//
// It used to go up by itself, once a record, in the middle of the longest solo.
// By itself is exactly where it went wrong: a record can be taken for wordless
// while its sheet is on its way, say its name for that, and then say it again
// when the sheet turns up with a solo in the middle of it. And a name that
// appears on its own schedule appears over something you were reading.
func TestTheRecordNeverSaysItsNameUnasked(t *testing.T) {
	m := scopeModel(100, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps.TrackID, m.ps.Title = "late", "A Long Song"
	m.ps.Artists, m.ps.Album = []string{"The Band"}, "An Album"
	m.ps.Duration = 5 * time.Minute

	name := strings.Join(m.soloName(), "\n")
	said := 0

	for at := time.Second; at < 5*time.Minute; at += time.Second {
		// The sheet turns up a minute in, with a long solo after its first line
		// — which is where the card used to go.
		if at == time.Minute {
			m.lyrics.lines = []player.Lyric{
				{At: 65_000, Words: "the only line"},
				{At: 240_000, Words: "and the last"},
			}
			m.lyrics.forTrack, m.lyrics.synced = m.ps.TrackID, true
		}

		m.setProgress(at)
		if cmd := m.wordsGrind(); cmd != nil {
			if got := cmd(); got != nil {
				tm, _ := m.Update(got)
				m = tm.(Model)
			}
		}

		if lines, _ := m.wordsComing(); strings.Join(lines, "\n") == name {
			said++
			t.Logf("%5s the record said its name unasked", at)
		}
	}

	if said != 0 {
		t.Errorf("the record said its name %d times without being asked", said)
	}

	// And the key still puts it up, wherever the record is.
	m.setProgress(2 * time.Minute)
	m.words.forced = time.Now()
	if !m.soloTelling() {
		t.Error("the name could not be asked for at all")
	}
	if lines, _ := m.wordsComing(); strings.Join(lines, "\n") != name {
		t.Errorf("the key put up %q, want the record's name", lines)
	}
}

// And the marks are not put up twice in a row.
//
// They have the rest of every turn anyway, so dealing them as the card that
// opens one showed the same picture twice: marks arriving, and the very same
// marks arriving again a moment later by another road.
func TestTheMarksAreNotDealtOnTopOfThemselves(t *testing.T) {
	m := scopeModel(100, 44)
	m.ps.TrackID, m.ps.Title = "wordless", "An Instrumental"
	m.ps.Artists, m.ps.Album = []string{"The Band"}, "An Album"
	m.ps.Duration = 20 * time.Minute

	// Over a long record, no turn opens with the marks: what is dealt is a name
	// or nothing, and nothing is the marks having the whole turn.
	for spell := range 40 {
		lines, _ := func() ([]string, int64) {
			m.setProgress(time.Duration(spell)*wordsSpell + time.Second)
			return m.wordsIdle()
		}()
		if len(lines) == 1 && wordsBeats(lines[0]) {
			// The marks, which is what a turn dealt nothing looks like — and it
			// must be the same picture all the way through the turn.
			m.setProgress(time.Duration(spell)*wordsSpell + wordsTitle + time.Second)
			later, _ := m.wordsIdle()
			if len(later) != 1 || later[0] != lines[0] {
				t.Errorf("turn %d opened with marks and then changed to %q", spell, later)
			}
		}
	}
}
