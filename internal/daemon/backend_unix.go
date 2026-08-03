//go:build unix && !darwin

package daemon

// audioBackend on Linux and the BSDs is ALSA. PulseAudio is also available in
// go-librespot and needs no CGO, but ALSA is what is always there.
const audioBackend = "alsa"
