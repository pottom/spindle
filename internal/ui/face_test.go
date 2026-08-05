package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Every face draws something, and they do not all draw the same thing.
func TestEveryFaceDrawsItsOwn(t *testing.T) {
	const w, rows = 60, 20

	seen := map[string]faceKind{}
	for kind := faceSmile; kind < faceKinds; kind++ {
		m := scopeModel(100, 44)
		m.width, m.height = w, rows
		m.face = faceState{kind: kind}

		bands := make([]float32, 28)
		for i := range bands {
			bands[i] = 0.8
		}
		m.scope.bands = bands
		for range 10 {
			m.faceFlow()
		}

		var sb strings.Builder
		for _, line := range m.faceLines(w, rows) {
			sb.WriteString(ansiOff(line))
		}
		drawn := sb.String()

		if strings.TrimSpace(drawn) == "" {
			t.Errorf("face %d drew nothing", kind)
		}
		if other, ok := seen[drawn]; ok {
			t.Errorf("face %d draws the same as face %d", kind, other)
		}
		seen[drawn] = kind
	}
}

// The eyes shut now and again, on their own rather than with the music.
func TestFacesBlink(t *testing.T) {
	m := scopeModel(100, 44)
	m.width, m.height = 60, 20
	m.face = faceState{kind: faceSmile}

	var shut int
	for range faceBlinkEvery * 3 {
		m.faceFlow()
		if m.face.blink > 0 {
			shut++
		}
	}
	t.Logf("the eyes were shut for %d of %d frames", shut, faceBlinkEvery*3)

	if shut == 0 {
		t.Error("it never blinked")
	}
	if shut > faceBlinkEvery {
		t.Errorf("it was shut for %d frames of %d, want a blink rather than a nap", shut, faceBlinkEvery*3)
	}
}

// A solo can be four minutes away, and these are worth looking at: the key puts
// one up now, and which one is a coin.
func TestTheKeyPullsAFace(t *testing.T) {
	m := scopeModel(100, 40)
	m.stage.on = true
	m.scope.modes[tabPlayer] = scopeWords

	seen := map[string]bool{}
	var tm tea.Model = m
	for range 30 {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: 'F', Text: "F"})
		got := tm.(Model)

		if !got.stage.on {
			t.Fatal("the key put the big screen away instead of pulling a face")
		}
		if !got.words.forced || got.words.until.Before(time.Now()) {
			t.Fatal("nothing was pulled")
		}
		if got.chase.on {
			seen["chase"] = true
		} else if got.words.drawn {
			seen[string(rune('a'+got.face.kind))] = true
		}
	}

	t.Logf("thirty presses pulled %d of the %d things", len(seen), int(faceKinds)+1)
	if len(seen) < 3 {
		t.Errorf("thirty presses only ever pulled %d of them", len(seen))
	}
}
