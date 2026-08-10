package player

import "time"

// Waveform is implemented by backends that can say what the audio actually
// looks like, not merely what is playing. Only a local device can: the Web API
// describes the track, never the sound.
//
// It is kept out of Player on purpose. Widening the interface would force every
// backend to pretend it has the samples, and the UI would lose the one honest
// way to ask.
type Waveform interface {
	// Waveform is the most recent frame, one value per sample in -1..1, or nil
	// when nothing has arrived. The caller resamples it to whatever width it
	// has; the length is whatever the source found convenient.
	Waveform() []float32
}

// WaveformWindow is how much of a frame is meant to be drawn. A frame carries
// more than this: the surplus is slack for the drawer to find a consistent
// starting point in the wave, without which the picture shimmers.
const WaveformWindow = 256

// Spectrum is implemented by backends that can say how the energy of what is
// playing is spread across the frequency range. Like the waveform, only a local
// device can: it has to hear the sound to measure it.
type Spectrum interface {
	// Bands is the current spectrum, lowest frequency first, each 0..1, or nil
	// when nothing has played yet — and the beat it was measured on.
	Bands() ([]float32, Beat)
}

// Beat is where the beats of what is playing are: how far apart they come, and
// how long ago the last one was heard.
//
// The pair is what it takes to keep time rather than to react. A picture given
// only the rate can move at exactly the right speed all night and never once be
// on the beat; given both, anything that draws can work out where the next one
// falls without asking again.
//
// Since may be negative, meaning the last beat found has not been heard yet:
// the daemon reads the samples on their way to the audio device. A zero Period
// means no beat was found, which is the honest answer for a recording that has
// none, and the answer for the first seconds of every recording.
type Beat struct {
	Period time.Duration
	Since  time.Duration

	// Loud is where the top of the spectrum's own scale sits, in decibels: -55
	// with nothing playing, up towards nought as the record gets louder, and
	// nought before anything has been heard. It rides along here because the
	// bands cannot say it — every one of them is measured against it, so a band
	// reads the same in a hush as in a chorus. Without it nothing drawn from the
	// spectrum can tell a build from a lull.
	Loud float64

	// Notes is which of the twelve pitch classes are sounding, C first, each
	// 0..1, or nil while nothing has been heard. It rides here for the same
	// reason Loud does: whatever draws to it is already asking for the spectrum
	// thirty times a second, and a second request on its own schedule would
	// arrive a frame out from the one it belongs to.
	//
	// The bands cannot say it. Two neighbouring semitones fall in one band
	// until well above where a tune lives, so the meter's window is too short to
	// tell a note from its neighbour; this comes off a window four times as
	// long. See the analyser in the fork.
	Notes []float32
}

// Found reports that there is a beat to keep.
func (b Beat) Found() bool { return b.Period > 0 }

// Next is how long until the next beat, given how long ago this was measured.
func (b Beat) Next(since time.Duration) time.Duration {
	if !b.Found() {
		return 0
	}

	gone := (b.Since + since) % b.Period
	if gone < 0 {
		gone += b.Period
	}
	return b.Period - gone
}
