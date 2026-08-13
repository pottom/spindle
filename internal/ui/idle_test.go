package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// wordless is a model playing a track the lyric database has nothing for.
func wordless(t *testing.T) Model {
	t.Helper()

	m := scopeModel(100, 44)
	m.stage.on = true
	m.stage.mode = scopeWords
	m.ps.TrackID, m.ps.Title, m.ps.Album = "instrumental", "Windowlicker", "Windowlicker EP"
	m.ps.Artists = []string{"Aphex Twin"}
	m.ps.Duration = 6 * time.Minute
	m.lyrics.forTrack, m.lyrics.missing = m.ps.TrackID, true
	return m
}

// A record with no words is one long solo and nothing else: the marks, from the
// first second to the last, with nothing written across them at any point.
//
// It used to take turns — the artist set over one stretch of the record and the
// album over the next. Both are gone. Nothing goes up on this screen unless it
// is sent for, and from where anybody is sitting there is no difference between
// the record's name and the artist's: both are writing that appeared by itself.
func TestAWordlessRecordShowsNothingButTheMarks(t *testing.T) {
	m := wordless(t)

	for at := time.Duration(0); at < 5*time.Minute; at += 3 * time.Second {
		m.setProgress(at)
		lines, _ := m.wordsIdle()
		if len(lines) != 1 || !wordsBeats(lines[0]) {
			t.Fatalf("%s into the record the screen has %q, want the marks and only the marks", at, lines)
		}
	}
	t.Log("five minutes of a wordless record, sampled every three seconds: the marks throughout")

	// The bar under them is still stamped afresh every spell, which is what deals
	// the row a new lean and a new chance of a visitor — a long instrumental is
	// several hands rather than one.
	m.setProgress(time.Second)
	_, first := m.wordsIdle()
	m.setProgress(wordsSpell + time.Second)
	_, second := m.wordsIdle()
	m.setProgress(wordsSpell + 20*time.Second)
	_, same := m.wordsIdle()

	t.Logf("the bar is stamped %d in the first spell and %d in the second", first, second)
	if first == second {
		t.Error("the second spell was stamped with the first one's bar, so the record deals itself once and never again")
	}
	if same != second {
		t.Errorf("the bar changed inside a spell, from %d to %d", second, same)
	}
}

// wordy is a backend that has words to give, which is what makes fetchLyrics
// worth calling at all.
type wordy struct{ *player.Mock }

func (wordy) Lyrics(context.Context, string) (*player.Lyrics, error) { return nil, nil }

// The words are sent for by whatever is going to show them. The pane on the
// player is one; the lyric picture on the big screen is the other, and it has
// its own key — before this, turning it on without the pane left it waiting for
// words nobody had asked for, and every record looked as though it had none.
func TestTheBigScreenSendsForTheWords(t *testing.T) {
	m := scopeModel(100, 44)
	m.player = wordy{player.NewMock()}
	m.lyrics.on = false

	if cmd := m.fetchLyrics(); cmd != nil {
		t.Fatal("the words were sent for with nothing on screen to show them")
	}

	m.stage.on = true
	m.stage.mode = scopeWords
	if cmd := m.fetchLyrics(); cmd == nil {
		t.Error("the lyric picture is on the big screen and the words were never sent for")
	}
}

// The marks keep the screen while the database has still to answer.
//
// This went the other way first, and the reasoning was that nothing should be
// put up on spec: a record whose sheet is a second late would get the marks for
// that second and the first line straight over them.
//
// What settled it was watching the alternative. Nothing to put up meant the
// picture drawn when nothing is set — the meter above and below and an empty
// band across the middle — for as long as the answer took, so the top of every
// record read as the marks of the record before, then a hole, then the marks
// again. A hole is a change of picture too, and a worse one. And the marks are
// barely a guess: nearly every record opens with a stretch nobody sings, and
// the intro is a gap the marks would keep anyway.
func TestTheMarksKeepTheScreenUntilTheSheetLands(t *testing.T) {
	m := scopeModel(100, 44)
	m.stage.on = true
	m.stage.mode = scopeWords
	m.ps.TrackID, m.ps.Title = "new", "A Song"
	m.ps.Artists = []string{"The Band"}
	m.ps.Duration = 3 * time.Minute

	// The record has just started and nothing has answered about it yet.
	for _, at := range []time.Duration{time.Second, wordsSpell, 2 * wordsSpell} {
		m.setProgress(at)
		lines, _ := m.wordsComing()
		if len(lines) != 1 || !wordsBeats(lines[0]) {
			t.Errorf("%s in, with no answer yet, the screen has %q, want the marks", at, lines)
		}
		if m.wordsSilent() {
			t.Errorf("%s in the screen counts as silent, so it draws the empty picture", at)
		}
	}

	// The sheet lands, a second before the first line: the line is what goes up.
	m.lyrics.forTrack, m.lyrics.synced = "new", true
	m.lyrics.lines = []player.Lyric{{At: 61_000, Words: "the first line"}}
	m.setProgress(61*time.Second - wordsGather/2)

	lines, _ := m.wordsComing()
	if len(lines) != 1 || lines[0] != "the first line" {
		t.Errorf("with the sheet in and the singer due, the picture is %q, want the line", lines)
	}

	// And a record it has answered about and has nothing for keeps them too.
	m.lyrics.synced, m.lyrics.missing = false, true
	m.setProgress(wordsSpell + time.Second)
	if lines, _ := m.wordsIdle(); len(lines) == 0 {
		t.Error("a record known to have no words was left with nothing")
	}
}

// The screen has one picture, and the moment nothing is set is not an excuse
// for another one.
//
// It used to be the mirrored spectrum there — a different picture from every
// other one this screen draws, filling exactly the moments a record changes
// over. And it hid the change it was covering: its columns run to white
// wherever the music is loud, so the accent everything else is coloured by had
// nowhere to show.
func TestNothingSetIsStillTheSamePicture(t *testing.T) {
	const w, rows = 100, 40

	m := scopeModel(w, rows)
	m.stage.on = true
	m.stage.mode = scopeWords

	bands := make([]float32, 28)
	for i := range bands {
		bands[i] = 0.9
	}
	m.scope.bands = bands

	idle := m.wordsIdleArt(w, rows)
	if len(idle) != rows {
		t.Fatalf("the idle picture is %d rows, want %d", len(idle), rows)
	}

	// The meter stands on the floor and hangs from the ceiling, which is what
	// every picture here does.
	if strings.TrimSpace(ansiOff(idle[0])) == "" {
		t.Error("nothing hangs from the ceiling")
	}
	if strings.TrimSpace(ansiOff(idle[rows-1])) == "" {
		t.Error("nothing stands on the floor")
	}

	// And the middle is empty, because there is nothing to put in it.
	var middle int
	for _, line := range idle[rows/2-2 : rows/2+2] {
		middle += len(strings.TrimSpace(ansiOff(line)))
	}
	t.Logf("the middle four rows carry %d cells", middle)

	// It is not the mirrored spectrum, which fills the middle and leaves the
	// edges bare — the other way round from this.
	mirrored := m.stageArt(w, rows)
	var was int
	for _, line := range mirrored[rows/2-2 : rows/2+2] {
		was += len(strings.TrimSpace(ansiOff(line)))
	}
	if middle >= was {
		t.Errorf("the middle carries %d cells against the mirrored picture's %d, want it left for what goes there", middle, was)
	}
}
