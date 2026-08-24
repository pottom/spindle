//go:build unix && !darwin

package daemon

// audioBackend on Linux and the BSDs is ALSA. PulseAudio is also available in
// go-librespot and needs no CGO, but ALSA is what is always there.
const audioBackend = "alsa"

// audioDevice is the PCM to open. "default" is resolved through the machine's
// own ALSA configuration, which is where PipeWire and PulseAudio put
// themselves, so it follows whatever the desktop is already playing through.
//
// It has to be said out loud. go-librespot hands the name straight to
// snd_pcm_open and has no default of its own inside the library: its command
// line fills this in with "default" before calling it, and spindle builds the
// configuration itself and did not. What reached ALSA was the empty string:
//
//	ALSA lib pcm.c:2722:(snd_pcm_open_noupdate) [error.pcm] Unknown PCM
//	failed seeking before play: ALSA error at snd_pcm_open: No such file or directory
//
// Which is a particularly quiet way to be broken. The device registered, took
// the transfer, became the active device, and drew a queue — everything except
// the sound. Nothing on the screen was wrong; the failure was two words in a
// log, from a C library, without a timestamp because it never went through the
// logger at all.
//
// On macOS the same empty string is harmless: AudioToolbox never reads the
// name, it asks CoreAudio for the default output and follows that. So this
// could only ever break on the machine nobody was developing on.
const audioDevice = "default"

// audioBufferTime and audioPeriodCount divide that device's buffer, and between
// them they decide how often the picture moves.
//
// go-librespot takes one period of samples per read, and the waveform, the
// spectrum and the beat all hang off that read: the tap wraps the reader, so
// what the screen shows changes exactly as often as the device asks for
// samples. Its own default is half a second in four periods, which is a read
// every 125ms — eight a second.
//
// Measured, by asking the daemon for the wave over and over and counting the
// times it came back different: 8.2 Hz, a median 125.2ms apart. The screen was
// drawing all sixty frames a second and seven or eight in a row were the same
// picture, so the trace advanced in visible steps while every frame counter in
// the program said it was keeping up. The tap is built for thirty a second, and
// macOS reaches that without being asked — AudioToolbox reads 1024 frames at a
// time, which is 23ms, or forty-three a second. That was the whole of the
// difference between the two machines, and none of it was the terminal: the
// same frames replayed into Ghostty here drew at 626 a second.
//
// The buffer time is left where it was. It is the margin against an underrun —
// how much audio stands queued ahead of the device — and shortening it to get
// shorter reads would buy a smoother picture and pay in crackling. Only the
// division changes: the same half second in thirty-two pieces instead of four
// is a read every 16ms, which is one the screen can draw on every frame.
//
// Measured at thirty-two over half a minute: 64.0 Hz, p95 18.6ms, worst 20.7ms,
// not one read of 1920 late enough to miss what the tap was built for, and no
// underrun in the log.
const (
	audioBufferTime  = 500_000
	audioPeriodCount = 32
)
