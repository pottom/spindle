package daemon

// audioBackend on macOS is AudioToolbox, which is part of the system.
const audioBackend = "audio-toolbox"

// audioDevice is empty because AudioToolbox does not take one: it asks
// CoreAudio for the system's default output device and follows it wherever the
// listener moves it. See backend_unix.go, where the name is not optional.
const audioDevice = ""

// audioBufferTime and audioPeriodCount are how ALSA divides a buffer and mean
// nothing here: AudioToolbox is given a buffer size instead, and reads 1024
// frames — 23ms — at a time, which is already faster than the picture needs.
// See backend_unix.go, where it had to be asked for.
const (
	audioBufferTime  = 0
	audioPeriodCount = 0
)
