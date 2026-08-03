package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
)

// store keeps the device id and the session credentials between runs. Losing it
// means a new device id and another trip through the browser.
type store struct {
	path string

	mu    sync.Mutex
	state librespot.AppState
}

func newStore(path string) *store { return &store{path: path} }

func (s *store) Load() (*librespot.AppState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &s.state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read daemon state: %w", err)
	}

	// A file we cannot read is treated as absent: the daemon will earn a new
	// device id and ask for authorisation again, which beats refusing to start.
	if err := json.Unmarshal(data, &s.state); err != nil {
		return &s.state, nil
	}
	return &s.state, nil
}

func (s *store) Save(state *librespot.AppState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode daemon state: %w", err)
	}
	// It holds session credentials, so nobody else on the machine reads it.
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write daemon state: %w", err)
	}
	return nil
}
