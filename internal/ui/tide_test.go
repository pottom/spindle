package ui

import (
	"image/color"
	"testing"
	"time"

	"github.com/pottom/spindle/internal/player"
	"github.com/pottom/spindle/internal/ui/cover"
)

// The colour of the record coming next arrives before the sound of it.
//
// The change from one record to the next is the weakest moment this screen has,
// because it is a change: the picture belongs to one record and then to
// another. This is the change happening over twelve seconds instead of one
// frame, a band of the spectrum at a time.
func TestTheNextRecordsColourComesInFirst(t *testing.T) {
	m := scopeModel(120, 44)
	m.stage.on = true
	m.ps.TrackID, m.ps.Duration, m.ps.Playing = "first", 3*time.Minute, true
	m.queue = []player.Track{{ID: "second", CoverURL: "http://example/next.jpg"}}

	// Nothing to hand over until the coming record's colour is in hand.
	m.setProgress(2 * time.Minute)
	if _, ok := m.tideAt(); ok {
		t.Error("the tide started before there was a colour to bring in")
	}

	// It is sent for near the end, and not before.
	if cmd := m.tideCmd(); cmd != nil {
		t.Error("the coming record's artwork was asked for a minute out")
	}
	m.setProgress(3*time.Minute - tideFor - time.Second)
	if cmd := m.tideCmd(); cmd == nil {
		t.Fatal("the artwork was never asked for")
	}
	if cmd := m.tideCmd(); cmd != nil {
		t.Error("the artwork was asked for twice")
	}

	// And once it is in, the tide crosses over the last seconds.
	m.tideTook(cover.Art{Accent: color.RGBA{R: 200, G: 40, B: 40, A: 255}, HasAccent: true})

	m.setProgress(3*time.Minute - tideFor + time.Second)
	early, ok := m.tideAt()
	if !ok {
		t.Fatal("the tide never started")
	}

	m.setProgress(3*time.Minute - 2*time.Second)
	late, _ := m.tideAt()
	t.Logf("the tide is %.2f across a second in and %.2f across two seconds from the end", early, late)

	if !(early < late) {
		t.Errorf("the tide went %.2f then %.2f, want it crossing", early, late)
	}
	if early > 0.2 || late < 0.7 {
		t.Errorf("the tide is %.2f then %.2f, want it starting near nought and ending near one", early, late)
	}

	// And the palette really changes: the low bands belong to the coming record
	// while the high ones still belong to this one.
	m.setProgress(3*time.Minute - tideFor/2)
	was := m.styles.Words[0][len(m.styles.Words[0])-1].GetForeground()
	m.tideFlow()
	now := m.styles.Words[0][len(m.styles.Words[0])-1].GetForeground()

	t.Logf("the lowest band went from %v to %v", was, now)
	if was == now {
		t.Error("the tide was half across and the lowest band had not changed colour")
	}

	last := len(m.styles.Words) - 1
	if m.styles.Words[last][0].GetForeground() != m.tide.palette.Words[last][0].GetForeground() {
		return // the top of the screen is still this record's, which is the point
	}
	t.Error("the tide was half across and the whole screen had already changed over")
}

// A track put in front while the colour is crossing does not send it back.
//
// Watched on a live record: the colour started over toward the record that was
// coming, and then a track was added from a phone. The crossing began with one
// colour and finished with another — and in between it snapped the whole way
// back to the record that was still playing, which is the one thing this exists
// to stop happening.
//
// Following the new track is right. Starting again from nothing is not: the last
// twelve seconds of a record are exactly when the screen must not flinch.
func TestATrackPutInFrontDoesNotSendTheColourBack(t *testing.T) {
	m := scopeModel(120, 44)
	m.stage.on = true
	m.ps.TrackID, m.ps.Duration, m.ps.Playing = "first", 3*time.Minute, true
	m.queue = []player.Track{{ID: "second", CoverURL: "http://example/second.jpg"}}

	m.setProgress(3*time.Minute - tideFor - time.Second)
	m.tideCmd()
	m.tideTook(cover.Art{Accent: color.RGBA{R: 200, G: 40, B: 40, A: 255}, HasAccent: true})

	// Half across.
	m.setProgress(3*time.Minute - tideFor/2)
	m.tideFlow()
	half := m.tide.buckets
	if half <= 0 {
		t.Fatalf("the tide had not started: %d bands handed over", half)
	}

	// And now a track arrives in front of it, from somewhere else entirely.
	m.queue = []player.Track{
		{ID: "jumped", CoverURL: "http://example/jumped.jpg"},
		{ID: "second", CoverURL: "http://example/second.jpg"},
	}
	m.setProgress(3*time.Minute - tideFor/2 + time.Second)
	m.tideFlow()

	if m.tide.buckets < half {
		t.Errorf("the colour went back from %d bands to %d when a track was put in front",
			half, m.tide.buckets)
	}

	// The new record's colour is sent for, and when it lands the crossing carries
	// on from where it was rather than starting again.
	if cmd := m.tideCmd(); cmd == nil {
		t.Error("the colour of the track that jumped the queue was never asked for")
	}
	m.tideTook(cover.Art{Accent: color.RGBA{R: 40, G: 40, B: 200, A: 255}, HasAccent: true})
	m.setProgress(3*time.Minute - tideFor/2 + 2*time.Second)
	m.tideFlow()

	if m.tide.buckets < half {
		t.Errorf("the colour went back from %d bands to %d once the new one was in hand",
			half, m.tide.buckets)
	}
}
