package player

import (
	"context"
	"encoding/json"
	"net/http"
)

// Bands is how the local device's output is spread across the frequency range.
// Polled like the waveform: the daemon has no reason to send frames nobody is
// drawing.
func (l *Local) Bands() []float32 {
	if l.idle() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), waveformTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.addr+"/player/spectrum", nil)
	if err != nil {
		return nil
	}

	resp, err := l.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close() //nolint:errcheck // read-only request

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var out struct {
		Bands []float32 `json:"bands"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	return out.Bands
}
