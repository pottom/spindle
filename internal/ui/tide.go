package ui

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/pottom/spindle/internal/ui/cover"
	"github.com/pottom/spindle/internal/ui/style"
)

// The colour of the next record, arriving before the sound of it.
//
// The change from one record to the next is the weakest moment this screen has.
// Everything about it is a cut: the picture belongs to one record and then to
// another, and with a crossfade set it changes over while the record you are
// still hearing is playing underneath. There is nothing to be done about that
// by drawing the change better, because the problem is that it is a change.
//
// So it stops being one. The next record's colour comes in over the last
// seconds of this one, a band of the spectrum at a time, from one side of the
// screen to the other — the water falls in it before the track that owns it has
// started, and by the time the sound arrives the picture is already its. What
// was a seam is the one moment of the record you cannot look away from.

const (
	// tideFor is how long the colour takes to cross. Long enough to read as
	// weather rather than as a wipe, short enough to belong to the record it is
	// leaving.
	tideFor = 12 * time.Second

	// tideSlot is the artwork slot the coming record's cover is loaded into. Its
	// picture is never drawn — only the colour it reads as is wanted — so it is
	// asked for at the smallest size that still says what that colour is.
	tideSlot = 2
	tideCell = 8

	// tideAhead is how far before the tide starts the artwork is sent for, so
	// the colour is in hand when it is wanted rather than arriving somewhere in
	// the middle of the crossing.
	tideAhead = 8 * time.Second
)

// tideState is the next record's colour, and which record it belongs to.
type tideState struct {
	// forTrack is the track the colour was taken from, and asked is the artwork
	// already sent for, so a record is not asked about on every frame.
	forTrack string
	asked    string

	accent color.RGBA
	has    bool

	// palette is the whole set of styles built from that colour, and buckets
	// how many bands of the spectrum have been handed over to it so far — the
	// merge is redone when that number changes rather than every frame.
	palette style.Styles
	buckets int
}

// tideComing is the track whose colour is on its way, or empty when there is
// none to know about.
func (m Model) tideComing() string {
	if len(m.queue) == 0 {
		return ""
	}
	return m.queue[0].ID
}

// tideAt is how far the colour has come, nought to one, and whether it is on
// its way at all.
//
// It runs out where the record does — with a crossfade set, where the next one
// starts underneath rather than where this one stops, which is the moment the
// picture is about to be answering the wrong record.
func (m Model) tideAt() (float32, bool) {
	if !m.tide.has || m.ps == nil || !m.ps.Playing || m.ps.Duration <= 0 {
		return 0, false
	}
	if m.tide.forTrack == "" || m.tide.forTrack != m.tideComing() {
		return 0, false
	}

	ends := m.ps.Duration
	if fade := m.settings.crossfade; fade > 0 && fade < ends {
		ends -= fade
	}

	left := ends - m.elapsed()
	if left <= 0 {
		return 1, true
	}
	if left >= tideFor {
		return 0, false
	}
	return float32(1 - float64(left)/float64(tideFor)), true
}

// tideCmd asks for the coming record's artwork, once, and only once the record
// playing is near enough its end for the colour to be wanted.
func (m *Model) tideCmd() tea.Cmd {
	next := m.tideComing()
	if next == "" || m.ps == nil || m.ps.Duration <= 0 || !m.ps.Playing {
		return nil
	}

	// Asked for a little before it is needed, so the colour is in hand when the
	// tide starts rather than arriving somewhere in the middle of it.
	ends := m.ps.Duration
	if fade := m.settings.crossfade; fade > 0 && fade < ends {
		ends -= fade
	}
	if ends-m.elapsed() > tideFor+tideAhead {
		return nil
	}

	if m.tide.asked == next || m.queue[0].CoverURL == "" {
		return nil
	}
	m.tide.asked = next
	return coverCmd(m.covers, m.queue[0].CoverURL, tideCell, tideCell, tideSlot)
}

// tideTook records the colour of the record that is coming.
func (m *Model) tideTook(art cover.Art) {
	m.tide.forTrack = m.tide.asked
	m.tide.accent, m.tide.has = art.Accent, art.HasAccent
	m.tide.buckets = -1

	if !m.tide.has {
		return
	}
	m.tide.palette = style.New(m.isDark, cover.Readable(m.tide.accent, m.isDark))
}

// tideFlow hands the bands of the spectrum over to the coming record's colour,
// as far as the tide has come.
//
// A band at a time, and only when the count changes: the merge is a handful of
// slice assignments, and doing it on every frame of every record would be a
// handful of slice assignments thirty times a second for nothing.
func (m *Model) tideFlow() {
	at, ok := m.tideAt()
	if !ok {
		if m.tide.buckets > 0 {
			m.restyle()
			m.tide.buckets = 0
		}
		return
	}

	want := int(at * float32(len(m.styles.Words)))
	if want == m.tide.buckets {
		return
	}
	m.tide.buckets = want

	// From the styles this record owns, with the low bands — the left of the
	// screen — given over first.
	m.restyle()
	for i := range want {
		if i < len(m.styles.Words) && i < len(m.tide.palette.Words) {
			m.styles.Words[i] = m.tide.palette.Words[i]
		}
		if i < len(m.styles.Bars) && i < len(m.tide.palette.Bars) {
			m.styles.Bars[i] = m.tide.palette.Bars[i]
		}
	}
}
