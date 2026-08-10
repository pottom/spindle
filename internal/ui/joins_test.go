package ui

import (
	"testing"
	"time"
)

// feedShape plays a made-up record past the detector: each entry is a stretch of
// seconds and the band shape held over it.
func feedShape(t *testing.T, m *Model, parts []struct {
	secs  float64
	shape []float32
}) []time.Duration {
	t.Helper()

	const fps = 30
	var joins []time.Duration
	var at time.Duration
	was := m.joinsAt()
	for _, part := range parts {
		for range int(part.secs * fps) {
			m.scope.bands = part.shape
			m.setProgress(at)
			m.joinsFlow(fps)
			if now := m.joinsAt(); now != was {
				joins = append(joins, now)
				was = now
			}
			at += time.Second / fps
		}
	}
	return joins
}

// A record that changes shape is a record with a join in it, and one that holds
// its shape is not.
//
// The two halves of the same test, because either alone proves nothing: a
// detector that fires on everything passes the first and a detector that never
// fires passes the second.
func TestAChangeOfShapeIsAJoin(t *testing.T) {
	quiet := make([]float32, 28)
	for i := range quiet {
		quiet[i] = 0.2 // flat: nothing standing out anywhere
	}
	loud := make([]float32, 28)
	for i := range loud {
		loud[i] = 0.2
		if i < 6 {
			loud[i] = 1 // the low end arrives, and nothing else changes
		}
	}
	// The same shape as quiet, only bigger. Loudness is not a join.
	same := make([]float32, 28)
	for i := range same {
		same[i] = 0.9
	}

	m := stageWords("a")
	got := feedShape(t, &m, []struct {
		secs  float64
		shape []float32
	}{
		{40, quiet}, // long enough to fill both windows and settle
		{40, loud},  // the drums come in
	})
	t.Logf("a record that changed shape gave joins at %v", got)
	if len(got) == 0 {
		t.Error("the shape changed completely and nothing was called a join")
	}

	m = stageWords("b")
	got = feedShape(t, &m, []struct {
		secs  float64
		shape []float32
	}{
		{40, quiet},
		{40, same}, // four and a half times as loud, the same shape
	})
	t.Logf("a record that only got louder gave joins at %v", got)
	for _, at := range got {
		if at < joinMost {
			t.Errorf("a change of loudness at the same shape was called a join, at %s", at)
		}
	}
}

// A record that never changes still turns the picture over.
//
// The whole point is to stop counting in seconds, and a record with nothing in
// it would then never change what is on screen at all — which is a screen that
// has stopped rather than a screen answering the music.
func TestARecordThatNeverChangesStillTurnsOver(t *testing.T) {
	flat := make([]float32, 28)
	for i := range flat {
		flat[i] = 0.3
	}

	m := stageWords("c")
	got := feedShape(t, &m, []struct {
		secs  float64
		shape []float32
	}{{float64(joinMost/time.Second) + 30, flat}})

	t.Logf("over %s of one unchanging shape the picture turned at %v", joinMost+30*time.Second, got)
	if len(got) == 0 {
		t.Errorf("nothing changed for %s and the picture never turned over", joinMost+30*time.Second)
	}
}

// Two joins are never closer together than a section is allowed to be.
func TestSectionsAreNotShorterThanTheyMayBe(t *testing.T) {
	m := stageWords("d")

	var parts []struct {
		secs  float64
		shape []float32
	}
	// A record that changes shape every three seconds, which is a record
	// nothing can be read off — the guard is what stops the picture chasing it.
	for i := range 30 {
		shape := make([]float32, 28)
		for j := range shape {
			shape[j] = 0.2
		}
		shape[(i*7)%28] = 1
		parts = append(parts, struct {
			secs  float64
			shape []float32
		}{3, shape})
	}
	got := feedShape(t, &m, parts)
	t.Logf("ninety seconds of a record changing every three gave %d joins: %v", len(got), got)

	for i := 1; i < len(got); i++ {
		if gap := got[i] - got[i-1]; gap < joinApart {
			t.Errorf("two joins %s apart, which is under the %s a section may be", gap, joinApart)
		}
	}
}
