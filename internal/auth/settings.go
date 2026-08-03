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

// settings is what spindle remembers about the application it authenticates as.
type settings struct {
	ClientID string `json:"client_id"`
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
	path, err := settingsPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read settings: %w", err)
	}

	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		// A file we cannot read is treated as absent: spindle will ask again,
		// which beats refusing to start over a stray character.
		return "", nil
	}
	if !clientIDPattern.MatchString(s.ClientID) {
		return "", nil
	}
	return s.ClientID, nil
}

// SaveClientID validates and stores the client id for future runs.
func SaveClientID(id string) error {
	if !clientIDPattern.MatchString(id) {
		return fmt.Errorf("%w: expected 32 hexadecimal characters, got %d", ErrMalformedClientID, len(id))
	}

	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings{ClientID: id}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

// SettingsPath is where the client id is kept, for diagnostics.
func SettingsPath() string {
	path, err := settingsPath()
	if err != nil {
		return ""
	}
	return path
}
