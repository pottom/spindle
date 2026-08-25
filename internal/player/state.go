package player

import "time"

// Repeat modes accepted by SetRepeat and reported in State.Repeat.
const (
	RepeatOff     = "off"
	RepeatContext = "context"
	RepeatTrack   = "track"
)

// State is a snapshot of what the active playback device is doing.
type State struct {
	// Stalled is the daemon answering out of a cupboard: its API is up and its
	// playback session has stopped moving, so what follows was true a while ago
	// rather than now. The position is left where the daemon last put it rather
	// than carried forward by this end's clock.
	Stalled bool

	TrackID  string
	Title    string
	Artists  []string
	Album    string
	CoverURL string // largest available, capped at 640px
	Progress time.Duration
	Duration time.Duration
	Playing  bool
	Shuffle  bool
	Repeat   string // "off" | "context" | "track"
	Volume   int    // 0–100
	DeviceID string

	// Unplayable is the last track the device could not play, whatever it was
	// asked to. Spotify refuses the key for some tracks inside the list they
	// belong to — measured, the same track plays when asked for on its own —
	// and the device moves past them. A list that skips a track on its own
	// reads as a fault in the player, so it is worth saying.
	Unplayable string

	// Deaf names what the device that is playing can no longer hear from — see
	// localStatus. Empty for a healthy one and for every backend that is not our
	// own daemon, which is why nothing may treat it as an error: it is a device
	// that still plays, and the only thing to do about it is start it again.
	Deaf []string

	// OutOfTouch is what the device has lost its connection to, against how
	// long it has been trying to get it back. Empty for a healthy device and
	// for every backend that is not our own daemon. Unlike Deaf this mends
	// itself, so nothing may treat it as a fault: it is a device that is still
	// playing what it has and still trying — see localStatus.
	OutOfTouch map[string]time.Duration

	DeviceName string

	// Bitrate is the stream actually playing, in kbps, or 0 when unknown. Only
	// the local daemon can say: the Web API never reports what a device chose.
	Bitrate int

	// Tempo is the measured beat rate in bpm, or zero while the analyser is
	// still listening or the recording has no steady beat.
	Tempo float64
}
