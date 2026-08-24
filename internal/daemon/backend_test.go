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
