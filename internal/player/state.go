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
	TrackID    string
	Title      string
	Artists    []string
	Album      string
	CoverURL   string // largest available, capped at 640px
	Progress   time.Duration
	Duration   time.Duration
	Playing    bool
	Shuffle    bool
	Repeat     string // "off" | "context" | "track"
	Volume     int    // 0–100
	DeviceID   string
	DeviceName string

	// Bitrate is the stream actually playing, in kbps, or 0 when unknown. Only
	// the local daemon can say: the Web API never reports what a device chose.
	Bitrate int
}
