package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"

	"github.com/pottom/spindle/internal/xdg"
)

// Store keeps the refresh token between runs. It is the whole reason a second
// launch does not open a browser.
type Store struct {
	path string
}

// NewStore places the token under the XDG config directory.
func NewStore() (*Store, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(dir, "token.json")}, nil
}

// Path is where the token lives, for diagnostics.
func (s *Store) Path() string { return s.path }

// Load returns the stored token, or nil when there is none. A file that cannot
// be parsed is treated as absent: the flow will simply run again.
func (s *Store) Load() (*oauth2.Token, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}

	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil || tok.RefreshToken == "" {
		return nil, nil
	}
	return &tok, nil
}

// Save writes the token so only its owner can read it.
func (s *Store) Save(tok *oauth2.Token) error {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write token: %w", err)
	}
	return nil
}

// Delete removes the token, which is what a revoked grant calls for.
func (s *Store) Delete() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}
