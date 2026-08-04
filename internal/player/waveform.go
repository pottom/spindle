package player

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
	// when nothing has played yet.
	Bands() []float32
}
