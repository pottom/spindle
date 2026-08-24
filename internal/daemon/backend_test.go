package daemon

import (
	"testing"
	"time"
)

// ALSA opens a PCM by name and has no notion of "whichever one the system is
// using": the name is the whole of the instruction. go-librespot passes it
// through untouched, so a configuration that leaves it out asks ALSA to open
// the device called "" — which registered a device, took the transfer, became
// the active player, and played nothing at all.
//
// The name is a constant per platform, so this only says anything on the
// platform that needs it. That is the platform it was broken on.
func TestTheBackendThatNeedsADeviceHasOne(t *testing.T) {
	cfg := playbackConfig(DefaultQuality, 0, t.TempDir())

	if cfg.AudioBackend == "" {
		t.Fatal("no audio backend was named")
	}
	if cfg.AudioBackend == "alsa" && cfg.AudioDevice == "" {
		t.Error("the alsa backend was given no device: it will open nothing and say so in C")
	}
}

// And the settings a listener can change reach the player, since none of them
// is checked anywhere between here and Spotify.
func TestTheSettingsReachThePlayer(t *testing.T) {
	cfg := playbackConfig(QualityHigh, 4500*time.Millisecond, "/tmp/nowhere")

	if want := QualityHigh.Bitrate(); cfg.Bitrate != want {
		t.Errorf("asking Spotify for %d, want %d", cfg.Bitrate, want)
	}
	// go-librespot counts the overlap in milliseconds.
	if cfg.CrossfadeDuration != 4500 {
		t.Errorf("crossfading for %d, want 4500 milliseconds", cfg.CrossfadeDuration)
	}
	if cfg.Cache.Dir != "/tmp/nowhere" {
		t.Errorf("the measured tempos would go to %q", cfg.Cache.Dir)
	}
	if cfg.Credentials.Interactive.CallbackPort != authCallbackPort {
		t.Errorf("the sign-in would come back to %d, want %d",
			cfg.Credentials.Interactive.CallbackPort, authCallbackPort)
	}
}

// How often the picture moves is how often the audio device asks for samples:
// the waveform, the spectrum and the beat all hang off that one read. So the
// period the buffer is divided into has to be short enough to feed a screen,
// and go-librespot's own default — half a second in four — is not.
//
// Thirty a second is what the tap is built for. A period longer than that is a
// trace that steps while every frame counter says it is keeping up, which is
// exactly how this was missed.
func TestTheAudioPeriodKeepsUpWithThePicture(t *testing.T) {
	cfg := playbackConfig(DefaultQuality, 0, t.TempDir())
	if cfg.AudioBackend != "alsa" {
		t.Skip("only alsa is handed its buffer in pieces")
	}
	if cfg.AudioPeriodCount <= 0 || cfg.AudioBufferTime <= 0 {
		t.Fatal("the buffer is left to go-librespot, which divides it eight a second")
	}

	period := time.Duration(cfg.AudioBufferTime/cfg.AudioPeriodCount) * time.Microsecond
	if want := 33 * time.Millisecond; period > want {
		t.Errorf("the device reads every %s, want %s or less: the picture would move %.0f times a second",
			period, want, float64(time.Second)/float64(period))
	}
}
