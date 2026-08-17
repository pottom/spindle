package player

import "time"

// localStatus is what the daemon's /status endpoint answers with.
type localStatus struct {
	DeviceID   string `json:"device_id"`
	Unplayable string `json:"unplayable"`

	// Deaf names what the device can no longer hear from. Its inputs close for
	// good once it has given up reconnecting, and after that it plays and
	// answers this API while being out of Spotify Connect's reach — which from
	// here looks exactly like a device that is working.
	Deaf        []string `json:"deaf"`
	DeviceName  string   `json:"device_name"`
	Stopped     bool     `json:"stopped"`
	Paused      bool     `json:"paused"`
	Volume      int      `json:"volume"`
	VolumeSteps int      `json:"volume_steps"`

	RepeatContext bool `json:"repeat_context"`
	RepeatTrack   bool `json:"repeat_track"`
	ShuffleCtx    bool `json:"shuffle_context"`

	Track *localTrack `json:"track"`

	// Bitrate is the stream actually playing, which is not necessarily the one
	// configured: the best available format is picked per track.
	Bitrate int `json:"bitrate"`

	// Tempo is measured from the audio, since the endpoint that would report it
	// is closed to this application.
	Tempo float64 `json:"tempo"`
}

type localTrack struct {
	URI         string   `json:"uri"`
	Name        string   `json:"name"`
	ArtistNames []string `json:"artist_names"`
	AlbumName   string   `json:"album_name"`
	AlbumCover  string   `json:"album_cover_url"`
	Position    int64    `json:"position"`
	Duration    int64    `json:"duration"`
	ReleaseDate string   `json:"release_date"`
	TrackNumber int      `json:"track_number"`
	DiscNumber  int      `json:"disc_number"`
}

// toState converts a daemon status into the shape the UI works in.
func (s *localStatus) toState() *State {
	st := &State{
		Playing:    !s.Paused && !s.Stopped,
		Shuffle:    s.ShuffleCtx,
		Repeat:     repeatFromLocal(s.RepeatContext, s.RepeatTrack),
		Volume:     percentOf(s.Volume, s.VolumeSteps),
		DeviceID:   s.DeviceID,
		Unplayable: trackIDFromURI(s.Unplayable),
		Deaf:       s.Deaf,
		DeviceName: s.DeviceName,
		Bitrate:    s.Bitrate,
		Tempo:      s.Tempo,
	}

	if s.Track != nil {
		st.TrackID = trackIDFromURI(s.Track.URI)
		st.Title = s.Track.Name
		st.Artists = s.Track.ArtistNames
		st.Album = s.Track.AlbumName
		st.CoverURL = s.Track.AlbumCover
		st.Progress = time.Duration(s.Track.Position) * time.Millisecond
		st.Duration = time.Duration(s.Track.Duration) * time.Millisecond
	}
	return st
}

// percentOf rescales the daemon's volume, which counts in steps of its own
// choosing, onto the 0–100 the rest of spindle speaks.
func percentOf(value, steps int) int {
	if steps <= 0 {
		return 0
	}
	return min(max(value*100/steps, 0), 100)
}

// repeatFromLocal folds the daemon's two booleans back into one mode. Track
// wins: repeating one song inside a repeating context is still repeating one
// song.
func repeatFromLocal(context, track bool) string {
	switch {
	case track:
		return RepeatTrack
	case context:
		return RepeatContext
	default:
		return RepeatOff
	}
}

// trackIDFromURI turns "spotify:track:abc" into "abc". Everything else in
// spindle identifies tracks by bare id.
func trackIDFromURI(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == ':' {
			return uri[i+1:]
		}
	}
	return uri
}
