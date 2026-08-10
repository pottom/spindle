package ui

// How much the record is giving, and how much the picture gives back.
//
// Everything on this screen is drawn from the spectrum, and the spectrum cannot
// tell a build from a lull: the daemon scales it to its own recent loudness so
// that a quiet passage is still a picture rather than a flat line, and a band
// reads 0.8 in a hush and 0.8 in a chorus. Measured over ninety seconds of a
// record, the mean of the bands moved between 0.50 and 0.60 and how full the
// spectrum was between 0.545 and 0.626 — and the quietest stretch of the record
// sat at the *top* of both. There is nothing in the bands to grow with.
//
// So the daemon hands out the one number that was taken away: where the top of
// its scale sits, in decibels. What is done with it here is the same trick the
// colours use — keep the range the record has been moving through lately and
// read where it is inside that. A record that gets louder moves more, and how
// much louder is measured against itself rather than against a number somebody
// picked for all music.

const (
	// swellRise is how quickly the range opens when the record goes somewhere it
	// has not been, and swellClose how slowly it comes back in. Quick to open so
	// that the first chorus is not clipped, slow to close so that the verse
	// after it is still measured against the chorus.
	swellRise  = 1
	swellClose = 0.0008

	// swellSpan is the narrowest range, in decibels, the movement is spread
	// over. Under this a record that barely changes would have its every breath
	// blown up into a build.
	swellSpan = 9

	// swellLeast is how much of its travel the picture keeps at the quietest the
	// record has been. Not nought: a lull is a quieter passage of music, not a
	// picture that has been switched off.
	swellLeast = 0.55
)

// swellFlow keeps the range of loudness the record has been moving through.
func (m *Model) swellFlow() {
	db := m.scope.beat.Loud
	if db == 0 {
		return // nothing has been heard yet
	}

	if m.words.swellHigh <= m.words.swellLow {
		m.words.swellLow, m.words.swellHigh = db-swellSpan/2, db+swellSpan/2
		return
	}

	if db < m.words.swellLow {
		m.words.swellLow += (db - m.words.swellLow) * swellRise
	} else {
		m.words.swellLow += (db - m.words.swellLow) * swellClose
	}
	if db > m.words.swellHigh {
		m.words.swellHigh += (db - m.words.swellHigh) * swellRise
	} else {
		m.words.swellHigh += (db - m.words.swellHigh) * swellClose
	}
}

// swell is how far the picture moves this frame, as a share of its travel: at
// the quietest the record has been lately, swellLeast of it; at the loudest, all
// of it.
func (m Model) swell() float32 {
	db := m.scope.beat.Loud
	if db == 0 || m.words.swellHigh <= m.words.swellLow {
		return 1
	}

	span := max(m.words.swellHigh-m.words.swellLow, swellSpan)
	at := float32((db - m.words.swellLow) / span)
	return swellLeast + (1-swellLeast)*min(max(at, 0), 1)
}
