package player

// Device is a Spotify Connect endpoint playback can be transferred to.
type Device struct {
	ID     string
	Name   string
	Type   string
	Active bool
}
