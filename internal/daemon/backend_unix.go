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
