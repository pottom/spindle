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

// grant is the token together with what it was granted for. A token says
// nothing about which application asked for it or what it was allowed to do,
// and both change: a client id can be swapped, and a new screen can need a
// permission the last authorisation never requested. Keeping a token that no
// longer matches turns into a 403 somewhere far away from the cause.
type grant struct {
	*oauth2.Token
	ClientID string   `json:"granted_to,omitempty"`
	Scopes   []string `json:"granted_for,omitempty"`
}

// Load returns the stored token, or nil when there is none, when it was granted
// to another application, or when it lacks a permission now being asked for. A
// file that cannot be parsed is treated as absent: the flow will simply run
// again.
func (s *Store) Load() (*oauth2.Token, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}

	var saved grant
	if err := json.Unmarshal(data, &saved); err != nil || saved.Token == nil || saved.RefreshToken == "" {
		return nil, nil
	}

	id, err := ClientID()
	if err != nil {
		return nil, err
	}
	if saved.ClientID != id || !covers(saved.Scopes, Scopes) {
		return nil, nil
	}
	return saved.Token, nil
}

// covers reports whether a grant already carries every scope wanted.
func covers(granted, wanted []string) bool {
	have := make(map[string]bool, len(granted))
	for _, s := range granted {
		have[s] = true
	}
	for _, s := range wanted {
		if !have[s] {
			return false
		}
	}
	return true
}

// Save writes the token so only its owner can read it, along with the
// application and the permissions it belongs to.
func (s *Store) Save(tok *oauth2.Token) error {
	id, err := ClientID()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(grant{Token: tok, ClientID: id, Scopes: Scopes}, "", "  ")
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
