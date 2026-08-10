package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/msg"
)

// The middle of the screen is never empty while a record is playing.
//
// Every one of these is a hole watched on the running interface, and every one
// of them is the same fault: something the screen could not set, drawn as
// nothing. See wordsComing.

// stageWords is the big screen with the words on it, playing a record.
func stageWords(id string) Model {
	m := scopeModel(160, 44)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords
	m.ps = &player.State{TrackID: id, Title: "a record", Artists: []string{"someone"},
		Playing: true, Duration: 4 * time.Minute}
	m.words.forTrack = id
	return m
}

// A run of skips leaves the daemon with no record to name.
//
// The status carries the track it is playing, and while it is loading the next
// one it carries no track at all — no id, no title, nothing. Held on ctrl+n,
// that is where the player spends most of its time, and the screen spent it
// with an empty band across the middle: the picture was thrown away for a state
// that named no record, and then nothing was put up because there was no record
// to put anything up for.
func TestTheDaemonNamingNoRecordIsNotAHole(t *testing.T) {
	m := stageWords("a")
	m.lyrics.forTrack, m.lyrics.missing = "a", true
	m.setProgress(20 * time.Second)
	m.wordsGrind()
	if m.wordsSilent() {
		t.Fatal("the record playing has nothing up before the skip even starts")
	}
	had := m.words.have

	// Held down: three records skipped past, and between each pair a stretch
	// with no record named at all.
	var frames, holes int
	for _, id := range []string{"b", "c", "d"} {
		for step := range 60 {
			switch {
			case step < 40: // the daemon loading, naming nothing
				m.ps = &player.State{Playing: true}
			default: // and the record it landed on
				m.ps = &player.State{TrackID: id, Title: "another", Playing: true,
					Duration: 4 * time.Minute}
				m.setProgress(time.Duration(step-40) * 30 * time.Millisecond)
			}
			if cmd := m.wordsGrind(); cmd != nil {
				m.wordsTake(cmd)
			}
			frames++
			if m.wordsSilent() || m.words.have.DotsX == 0 {
				holes++
			}
		}
	}
	t.Logf("%d frames of the %d over three skips had nothing in the middle", holes, frames)
	if holes > 0 {
		t.Errorf("the screen went empty for %d frames of a run of skips", holes)
	}
	if m.words.have.DotsX != had.DotsX {
		t.Error("the picture was thrown away for a status that says nothing about the record")
	}
}

// wordsTake runs a picture the setter was sent for and puts it up, the way the
// update loop does when the message comes back.
func (m *Model) wordsTake(cmd tea.Cmd) {
	ready, ok := cmd().(msg.WordsReady)
	if !ok {
		return
	}
	m.wordsAdopt(ready.Grain, ready.Words, ready.Text)
	m.words.cellsX, m.words.cellsY = ready.CellsX, ready.CellsY
}

// A record whose sheet arrives, and whose first line is sung early.
//
// The gaps are only kept by the marks when they are long enough to be worth a
// change of picture — twelve seconds. An intro shorter than that was neither a
// line nor a gap, so the top of the record was a hole, and how long it lasted
// depended on nothing but where the singer came in.
func TestAShortIntroIsNotAHole(t *testing.T) {
	m := stageWords("b")
	m.lyrics.forTrack, m.lyrics.synced = "b", true
	m.lyrics.lines = []player.Lyric{
		{At: 7000, Words: "the first line"},
		{At: 12000, Words: "the second"},
	}

	var holes int
	for at := range 70 { // the first seven seconds, ten frames a second
		m.setProgress(time.Duration(at) * 100 * time.Millisecond)
		m.wordsGrind()
		if m.wordsSilent() {
			holes++
		}
	}
	t.Logf("%d frames of the 70 before the first line had nothing in the middle", holes)
	if holes > 0 {
		t.Errorf("a seven second intro was a hole for %d frames", holes)
	}
}

// A sheet writes its own rests, and a rest is not nothing.
//
// Plenty of sheets put an entry with no words in it where the singer stops.
// Read as a line, it set nothing; read as a gap, it was too short to count. So
// the screen drew neither, which is the one thing it must never do.
func TestTheSheetsOwnRestIsNotAHole(t *testing.T) {
	m := stageWords("c")
	m.lyrics.forTrack, m.lyrics.synced = "c", true
	m.lyrics.lines = []player.Lyric{
		{At: 1000, Words: "a line"},
		{At: 9000, Words: ""}, // the singer stops
		{At: 16000, Words: "and back"},
	}

	var holes int
	for at := 90; at < 150; at++ { // nine seconds in to fifteen
		m.setProgress(time.Duration(at) * 100 * time.Millisecond)
		m.wordsGrind()
		if m.wordsSilent() {
			holes++
		}
	}
	t.Logf("%d frames of the 60 over the sheet's own rest had nothing in the middle", holes)
	if holes > 0 {
		t.Errorf("the sheet's own rest was a hole for %d frames", holes)
	}
}
