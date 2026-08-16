package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pottom/spindle/internal/xdg"
)

// clientIDPattern is what the Spotify dashboard hands out: 32 hex characters.
// Checking the shape catches the common mistakes — a pasted secret, a truncated
// copy, a whole URL — at the point where they can still be explained, rather
// than as an opaque rejection halfway through the browser flow.
var clientIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ErrMalformedClientID reports a client id that cannot be one.
var ErrMalformedClientID = errors.New("that does not look like a Spotify client id")

// settings is what spindle remembers between runs.
type settings struct {
	ClientID  string `json:"client_id"`
	Quality   string `json:"quality,omitempty"`
	Crossfade string `json:"crossfade,omitempty"`

	// Notify says whether the daemon announces each new track to the desktop.
	// Empty means off: a notification every three minutes is somebody else's
	// idea of useful, and it is asked for once with a command.
	Notify string `json:"notify,omitempty"`

	// LastFM is a key for last.fm's API, which spindle uses to say something
	// about an artist and a song that Spotify does not.
	//
	// Optional, and everything works without it: a source with no key is not in
	// the chain at all. It lives here rather than in the source because it is a
	// property of this installation, and it is asked for by name so that
	// somebody who does not want it never has to see it. See internal/notes.
	LastFM string `json:"lastfm_key,omitempty"`

	// CallbackPort is where the browser is sent back to when logging in.
	//
	// A setting rather than a number in the source, because the right value is a
	// property of the machine rather than of spindle: it has to be a port nothing
	// else on that machine wants, and which port that is nobody here can know. It
	// was 8888, and 8888 is one of the most contested numbers there is — Jupyter,
	// half the proxies, a good share of the development servers. A SOCKS proxy
	// sitting on it is enough to make logging in fail.
	//
	// Whatever it is set to has to match the Spotify application's own
	// configuration character for character. An application may list several, so
	// the old address can stay while a new one is added.
	CallbackPort int `json:"callback_port,omitempty"`
}

// load reads the settings file, treating an unreadable one as empty: spindle
// will ask again rather than refuse to start over a stray character.
func load() (settings, error) {
	var s settings

	path, err := settingsPath()
	if err != nil {
		return s, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return settings{}, nil
	}
	return s, nil
}

func save(s settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

// Quality is the audio quality spindle asks for, or "" when unset.
func Quality() (string, error) {
	s, err := load()
	if err != nil {
		return "", err
	}
	return s.Quality, nil
}

func settingsPath() (string, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// loadClientID returns the stored client id, or "" when there is none.
func loadClientID() (string, error) {
	s, err := load()
	if err != nil {
		return "", err
	}
	if !clientIDPattern.MatchString(s.ClientID) {
		return "", nil
	}
	return s.ClientID, nil
}

// SaveClientID validates and stores the client id, leaving the rest alone. The
// empty string forgets it, which is how one goes back to DefaultClientID.
func SaveClientID(id string) error {
	if id != "" && !clientIDPattern.MatchString(id) {
		return fmt.Errorf("%w: expected 32 hexadecimal characters, got %d", ErrMalformedClientID, len(id))
	}

	s, err := load()
	if err != nil {
		return err
	}
	s.ClientID = id
	return save(s)
}

// SaveQuality stores the audio quality, leaving the rest alone.
func SaveQuality(q string) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.Quality = q
	return save(s)
}

// SettingsPath is where the client id is kept, for diagnostics.
func SettingsPath() string {
	path, err := settingsPath()
	if err != nil {
		return ""
	}
	return path
}

// Crossfade is how long tracks overlap, as configured, or "" when unset.
func Crossfade() (string, error) {
	s, err := load()
	if err != nil {
		return "", err
	}
	return s.Crossfade, nil
}

// Notify reports whether track changes are announced.
func Notify() (bool, error) {
	s, err := load()
	if err != nil {
		return false, err
	}
	return s.Notify == "on", nil
}

// SaveNotify records whether they are, keeping everything else.
func SaveNotify(on bool) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.Notify = ""
	if on {
		s.Notify = "on"
	}
	return save(s)
}

// SaveCrossfade records the overlap, keeping everything else.
func SaveCrossfade(value string) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.Crossfade = value
	return save(s)
}
