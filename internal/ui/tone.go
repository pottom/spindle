package ui

import (
	"image/color"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/ui/cover"
)

// The colour the whole program is drawn in, and which record it belongs to.
//
// It belongs to the one that is playing, everywhere and always. That sounds like
// what it always did and it was not: the accent came off whichever cover was in
// the main slot, and on the browsing tabs that slot follows the cursor. Walking
// down a queue repainted the program a row at a time — the tabs, the playhead,
// the meter, the words — in the colours of records nobody was listening to.
//
// The picture still follows the cursor, because looking at what the cursor is on
// is what those screens are for. Only the colour is the music's. One is a
// question about what you are pointing at; the other is a question about what is
// sounding, and they are not the same question.
//
// So the sounding record's cover is asked for on its own, at the smallest size
// that still says what colour it is, the way the tide asks for the next
// record's — see tide.go, which this is the twin of.

const (
	// toneSlot is the artwork slot the sounding record's cover is loaded into.
	// Its picture is never drawn: only the colour it reads as is wanted.
	toneSlot = 3
	toneCell = 8
)

// toneState is the sounding record's colour, and which record it came from.
type toneState struct {
	// forTrack is the record the colour was taken from, and asked the one whose
	// cover has been sent for, so a record is not asked about on every frame.
	forTrack string
	asked    string

	accent color.RGBA
	has    bool
}

// toneFlow sends for the sounding record's colour when the record changes.
func (m *Model) toneFlow() tea.Cmd {
	if m.ps == nil || m.ps.TrackID == "" || m.ps.CoverURL == "" {
		return nil
	}
	if m.tone.asked == m.ps.TrackID {
		return nil
	}
	m.tone.asked = m.ps.TrackID
	return coverCmd(m.covers, m.ps.CoverURL, toneCell, toneCell, toneSlot)
}

// toneTook records the colour, and repaints everything in it.
func (m *Model) toneTook(art cover.Art) {
	m.tone.forTrack = m.tone.asked
	m.tone.accent, m.tone.has = art.Accent, art.HasAccent
	m.restyle()
}

// toneAccent is the colour every style is built from: the sounding record's,
// and the cover on screen only where there is no record sounding at all.
//
// The fallback matters on the screens that come up before anything is playing —
// the library and the search, where a cursor is resting on a cover and there is
// no music yet. A program with no accent at all there would go grey the moment
// it started, which reads as broken rather than as quiet.
func (m Model) toneAccent() (color.RGBA, bool) {
	if m.tone.has {
		return m.tone.accent, true
	}
	return m.cover.accent, m.cover.hasAccent
}
