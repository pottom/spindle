package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
)

// Silencing the room says so on whichever picture is up.
//
// The row of marks is the only picture with anybody in it, and the deal that
// puts the hush there only runs while that picture is the one showing. So on the
// waveform, the spectrum or the lamps, muting used to change nothing at all.
func TestTheHushShowsOverASungLine(t *testing.T) {
	m := New(player.NewMock(), nil, defaultTestCell)
	m.width, m.height = 120, 40
	m.stage.on = true
	m.scope.modes[m.tab] = scopeWords
	m.ps = &player.State{TrackID: "one", Title: "x", Duration: 3 * time.Minute, Playing: true, Volume: 60}
	m.setProgress(40 * time.Second)
	m.words.forTrack = "one"

	// A record with a sheet, mid-line: words on screen rather than marks.
	m.lyrics.synced, m.lyrics.forTrack = true, "one"
	// Lines close enough together that forty seconds in is mid-verse rather than
	// in a gap the sheet left, which the marks would have anyway.
	for at := 0; at < 60; at += 4 {
		m.lyrics.lines = append(m.lyrics.lines,
			player.Lyric{At: int64(at) * 1000, Words: "she came in through the window"})
	}

	m.wordsGrind()
	if m.words.beats {
		t.Fatal("the bar was marks with a line being sung")
	}

	m.toggleMute()
	m.wordsGrind()
	if !m.words.beats {
		t.Error("silencing the room left the sung line up")
	}
	if m.words.cast != markHush {
		t.Errorf("the row was %q while the room was silenced", m.words.cast)
	}
}

func TestTheHushShowsOnEveryPicture(t *testing.T) {
	for _, mode := range []scopeMode{scopeWave, scopeBars, scopeLadder, scopeWords} {
		m := New(player.NewMock(), nil, defaultTestCell)
		m.width, m.height = 120, 40
		m.stage.on = true
		m.scope.modes[m.tab] = mode
		m.ps = &player.State{TrackID: "one", Title: "x", Duration: 3 * time.Minute, Playing: true, Volume: 60}
		m.setProgress(40 * time.Second)
		m.words.forTrack = "one"
		m.toggleMute()
		m.wordsGrind()

		if m.words.cast != markHush {
			t.Errorf("%v: the row was %q while the room was silenced", mode, m.words.cast)
		}
		drawn := strings.Join(m.stagePicture(m.width, m.height), "")
		if strings.TrimSpace(drawn) == "" {
			t.Errorf("%v: nothing was drawn while muted", mode)
		}
	}
}
