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

// WaveformSamples is how many points a frame carries. It is comfortably more
// than the horizontal dots of a wide terminal, so the trace never has to be
// stretched — only reduced.
const WaveformSamples = 256
